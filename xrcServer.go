package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
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
	"sync"
	"syscall"
	"time"
	"xgo"
)

type application struct {
	display              *xgo.Display
	port, assets, config string
	noTLS                bool
	homeTemplate         *template.Template
	certs                []tls.Certificate

	authMx       *sync.Mutex
	authPassword []byte
	authExpire   time.Time
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
	defer log.Printf("bye")

	app := application{
		authMx: &sync.Mutex{},
	}

	// Flags
	flag.StringVar(&app.port, "p", "10905", "http service port")
	flag.StringVar(&app.assets, "d", "assets", "working dir")
	flag.StringVar(&app.config, "c", "~/.config/xrcServer", "configuration dir")
	flag.BoolVar(&app.noTLS, "notls", false, "do not use TLS encrypted connection (not recomended)")
	flag.Parse()
	app.homeTemplate = template.Must(template.ParseFiles(filepath.Join(app.assets, "index.tmpl")))

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

	nl, err := net.Listen("tcp", ":"+app.port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("xrcServer listens on port :%s", app.port)

	// load or generate certificates
	certFile := filepath.Join(app.config, "cert.pem")
	keyFile := filepath.Join(app.config, "key.pem")

	if err := app.certificates(certFile, keyFile); err != nil {
		log.Fatal(err)
	}

	interruptCancel := make(chan struct{})

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
					// use tls
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
		runner{
			func() error {
				interrupt(interruptCancel)
				return nil
			},
			func() error {
				close(interruptCancel)
				return nil
			},
		},
	); len(errors) > 0 {
		for _, err := range errors {
			log.Print(err)
		}
	}
}

func interrupt(cancel <-chan struct{}) {
	log.Print("Press Ctrl-c to quit")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(c)
	select {
	case sig := <-c:
		fmt.Println() // Prevent un-terminated ^C character in terminal
		log.Printf("received signal: %s", sig)
	case <-cancel:
	}
}

type runner struct {
	run  func() error
	stop func() error
}
type runnererror struct {
	index int
	err   error
}

func run(runners ...runner) []error {
	if len(runners) == 0 {
		return nil
	}

	if len(runners) == 1 {
		return []error{runners[0].run()}
	}

	res := make([]error, 0)
	errors := make(chan runnererror, len(runners))

	for i, r := range runners {
		go func(i int, f func() error) {
			errors <- runnererror{i, f()}
		}(i, r.run)
	}

	rerr := <-errors
	if rerr.err != nil {
		res = append(res, rerr.err)
	}

	for i, r := range runners {
		if i == rerr.index {
			continue
		}
		if serr := r.stop(); serr != nil {
			//TODO diferentiate between run errors and stop errors
			res = append(res, serr)
		}
	}

	for i := 0; i < cap(runners)-1; i++ {
		rerr = <-errors
		if rerr.err != nil {
			res = append(res, rerr.err)
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
	//TODO cache
	switch key := app.certs[0].PrivateKey.(type) {
	case *rsa.PrivateKey:
		return x509.MarshalPKCS1PrivateKey(key), nil
	case *ecdsa.PrivateKey:
		return x509.MarshalECPrivateKey(key)
	}
	return nil, errors.New("tls: found unknown private key type in PKCS#8 wrapping")
}

func (app *application) publicKey() ([]byte, error) {
	//TODO cache
	switch key := app.certs[0].PrivateKey.(type) {
	case *rsa.PrivateKey:
		return x509.MarshalPKCS1PrivateKey(key), nil
	case *ecdsa.PrivateKey:
		return x509.MarshalECPrivateKey(key)
	}
	return nil, errors.New("tls: found unknown private key type in PKCS#8 wrapping")
}

func (app *application) authNewPassword() ([]byte, error) {
	app.authMx.Lock()
	defer app.authMx.Unlock()

	// expired
	if app.authPassword != nil && app.authExpire.Before(time.Now()) {
		app.authPassword = nil
		log.Print("authNewPassword: expired")
	}

	if app.authPassword == nil {
		app.authPassword = make([]byte, 8)
		_, err := rand.Read(app.authPassword)
		if err != nil {
			return nil, err
		}
		log.Print("authNewPassword: new created")
	}

	app.authExpire = time.Now().Add(time.Minute)
	return app.authPassword, nil
}

func (app *application) authClearPassword() {
	app.authMx.Lock()
	defer app.authMx.Unlock()
	app.authPassword = nil
	log.Print("authClearPassword: cleared")
}

func (app *application) auth(b []byte) bool {
	defer app.authClearPassword()
	app.authMx.Lock()
	defer app.authMx.Unlock()

	if app.authPassword == nil {
		return false
	}
	if app.authExpire.Before(time.Now()) {
		// expired
		log.Print("auth: expired")
		return false
	}
	return bytes.Equal(app.authPassword, b)
}
