package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"html/template"
	"https"
	"https/certificates"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"xgo"
)

type application struct {
	display      *xgo.Display
	homeTemplate *template.Template
	pairTemplate *template.Template
	certs        []tls.Certificate

	//config
	port, assets, config string
	noTLS                bool
	clientDebug          bool

	pair *pair

	//cache
	privateKeyCached []byte
	publicKeyCached  []byte
}

func main() {
	logFile, err := os.OpenFile("xrcServer.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
	if err == nil {
		defer logFile.Close()
		logMW := io.MultiWriter(os.Stderr, logFile)
		log.SetOutput(logMW)
		log.Printf("hello")
	} else {
		log.SetOutput(os.Stderr)
		log.Printf("hello")
		log.Printf("logging only to Stderr")
	}
	//TODO this and other defers are not runned after log.Fatal or os.Exit
	defer log.Printf("bye")

	app := application{
		pair: &pair{
			pairOpenDuration: 20 * time.Second,
			passwordDuration: 60 * time.Second,
			cookieDuration:   365 * 24 * time.Hour,
		},
	}

	//Flags
	flag.StringVar(&app.port, "port", "10905", "http(s) service `port number`")
	flag.StringVar(&app.config, "config", "~/.config/xrcServer", "`path` to configuration directory")
	flag.BoolVar(&app.noTLS, "notls", false, "do not use TLS encrypted connection (not recomended)")
	flag.BoolVar(&app.clientDebug, "debug-client", false, "show debuging info in served client app")
	flag.IntVar(&app.pair.passwordLen, "password", 8, "`length` of generated authentication password string. 0 means no password.")

	assets := ""
	flag.StringVar(&assets, "assets", "", "`path` to assets directory for http serving. embeded assets are used if empty")
	flag.Parse()

	cleanUpAssets, err := app.parseAssets(assets)
	if err != nil {
		log.Fatalf("Error parsing assets: %v", err)
	}
	defer cleanUpAssets()

	if strings.HasPrefix(app.config, "~") {
		user, err := user.Current()
		if err == nil {
			app.config = filepath.Join(user.HomeDir, app.config[1:])
		} else {
			log.Print(err)
			log.Printf("can't get user, using current dir for config")
		}
	}
	log.Printf("config dir: %s", app.config)

	d, err := xgo.OpenDisplay("")
	if err != nil {
		log.Fatal(err)
	}
	app.display = d

	//load or generate certificates
	certFile := filepath.Join(app.config, "cert.pem")
	keyFile := filepath.Join(app.config, "key.pem")
	if err := app.certificates(certFile, keyFile); err != nil {
		log.Fatal(err)
	}

	//start net listener
	nl, err := net.Listen("tcp", ":"+app.port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("xrcServer listens on port :%s", app.port)

	interruptHandlerCancel := make(chan struct{})
	appPairUnlockHandlerCancel := make(chan struct{})

	if errors := run(
		runner{
			func() error {

				mux := http.NewServeMux()
				mux.Handle("/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Write([]byte("pong"))
				}))
				mux.Handle("/", app.authenticate(http.HandlerFunc(app.homeHandler)))
				mux.Handle("/pair/", http.StripPrefix("/pair/", http.HandlerFunc(app.pairHandler)))
				mux.Handle("/favicon.ico", http.FileServer(http.Dir(app.assets)))
				mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(filepath.Join(app.assets, "js")))))
				mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(filepath.Join(app.assets, "css")))))
				mux.Handle("/ws", app.authenticate(http.HandlerFunc(app.websocketHandler)))

				s := &http.Server{
					Addr:    ":" + app.port,
					Handler: mux,
				}

				if app.noTLS == false {
					//use tls
					s.Handler = https.EnforceTLS(s.Handler)
					s.TLSConfig = &tls.Config{
						Certificates: app.certs,
					}
					log.Printf("starting http server with TLS")
					return s.ServeTLS(nl, "", "")
				}
				log.Printf("starting http server !!! without TLS !!!")
				return s.Serve(nl)

			},
			func() error {
				return nl.Close()
			},
		},
		runner{ // handle SIGTERM & SIGINT
			func() error {
				interruptHandler(interruptHandlerCancel)
				return nil
			},
			func() error {
				close(interruptHandlerCancel)
				return nil
			},
		},
		runner{ // handle SIGUSR1 as application pairing unlock
			func() error {
				app.pair.UnlockHandle(appPairUnlockHandlerCancel)
				return nil
			},
			func() error {
				close(appPairUnlockHandlerCancel)
				return nil
			},
		},
	); len(errors) > 0 {
		for _, err := range errors {
			if _, ok := err.(runStopErr); !ok {
				log.Print(err)
			}
		}
	}
	log.Printf("application stopped")

	app.pair.clearPassword()
}

func interruptHandler(cancel <-chan struct{}) {
	log.Print("Press Ctrl-c to quit")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(c)
	select {
	case sig := <-c:
		fmt.Println() //Prevent un-terminated ^C character in terminal
		log.Printf("received signal: %s", sig)
	case <-cancel:
	}
}

//TODO? to new package with evolution's paralel & serial
type runner struct {
	run  func() error
	stop func() error
}
type runnererror struct {
	index int
	err   error
}

type runStopErr struct {
	error
}

func (runStopErr) runStopError() {}

//runs all runners .run function concurently and waits for first to terminate.
//then closes left runners with .stop function and waits for all to finish.
//if closing fails on runner, runner will not be waited for finish
func run(runners ...runner) map[int]error {
	if len(runners) == 0 {
		return nil
	}

	if len(runners) == 1 {
		res := map[int]error{}
		if err := runners[0].run(); err != nil {
			res[0] = err
		}
		return res
	}

	res := map[int]error{}
	errors := make(chan runnererror, len(runners))

	active := make(map[int]runner, len(runners))

	for i, r := range runners {
		active[i] = r
		go func(i int, r runner) {
			err := r.run()
			errors <- runnererror{i, err}
		}(i, r)
	}

	runerr := <-errors
	delete(active, runerr.index)
	if runerr.err != nil {
		res[runerr.index] = runerr.err
	}

	for i, r := range active {
		if err := r.stop(); err != nil {
			res[i] = err
			delete(active, i)
		}
	}

	for len(active) > 0 {
		runerr = <-errors
		delete(active, runerr.index)
		_, inres := res[runerr.index]
		if runerr.err != nil && inres == false {
			res[runerr.index] = runStopErr{runerr.err}
		}
	}
	return res
}

func (app *application) certificates(certFile, keyFile string) error {
	certs := []tls.Certificate{}
	if err := certificates.Check(certFile, keyFile); err == nil {
		log.Printf("using https certificates (cert.pem, key.pem) from %s", app.config)
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Printf("failed to load https certificates (cert.pem, key.pem) from %s: %s", app.config, err)
		} else {
			certs = append(certs, cert)
		}
	}

	if len(certs) == 0 {
		//no certificates, try to generate
		log.Printf("trying to create https certificates (cert.pem, key.pem) to %s", app.config)
		c := certificates.Config{
			Hosts: []string{"127.0.0.1:" + app.port, "localhost:" + app.port},
			Subject: &pkix.Name{
				Organization: []string{"jEzCorp"},
				CommonName:   "xrcServer",
			},
		}
		certBlob, keyBlob, err := certificates.GenerateArrays(c)
		if err != nil {
			return fmt.Errorf("couldn't create certificates: %s", err)
		}
		cert, err := tls.X509KeyPair(certBlob, keyBlob)
		if err != nil {
			return fmt.Errorf("couldn't load created certificates: %s", err)
		}

		if err := certificates.Save(certFile, keyFile, certBlob, keyBlob); err != nil {
			log.Printf("failed to save https certificates (cert.pem, key.pem) to %s: %s", app.config, err)
			log.Print("certificates will be lost on exit")
		} else {
			log.Printf("created https certificates (cert.pem, key.pem) to %s", app.config)
		}

		certs = append(certs, cert)
	}

	app.certs = certs

	return nil
}

func (app *application) privateKey() ([]byte, error) {
	if app.privateKeyCached == nil {
		switch key := app.certs[0].PrivateKey.(type) {
		case *rsa.PrivateKey:
			app.privateKeyCached = x509.MarshalPKCS1PrivateKey(key)
		case *ecdsa.PrivateKey:
			pk, err := x509.MarshalECPrivateKey(key)
			if err != nil {
				return nil, fmt.Errorf("privateKey: %s", err.Error())
			}
			app.privateKeyCached = pk
		default:
			return nil, fmt.Errorf("privateKey: found unknown private key type in PKCS#8 wrapping: %T", key)
		}
	}

	return app.privateKeyCached, nil
}

func (app *application) publicKey() ([]byte, error) {
	if app.publicKeyCached == nil {
		var publicKey interface{}
		switch key := app.certs[0].PrivateKey.(type) {
		case *rsa.PrivateKey:
			publicKey = &key.PublicKey
		case *ecdsa.PrivateKey:
			publicKey = &key.PublicKey
		default:
			return nil, fmt.Errorf("publicKey: found unknown private key type in PKCS#8 wrapping: %T", key)
		}
		pk, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, fmt.Errorf("publicKey: %s", err.Error())
		}
		app.publicKeyCached = pk
	}

	return app.publicKeyCached, nil
}
