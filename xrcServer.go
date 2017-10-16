package main

import (
	"crypto/tls"
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
	"xgo"

	"github.com/gorilla/securecookie"
	"golang.org/x/net/websocket"
)

type application struct {
	display              *xgo.Display
	port, assets, config string
	homeTemplate         *template.Template
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

	app := application{}

	// Flags
	flag.StringVar(&app.port, "p", "10905", "http service port")
	flag.StringVar(&app.assets, "d", "assets", "working dir")
	flag.StringVar(&app.config, "c", "~/.config/xrcServer", "configuration dir")
	flag.Parse()
	app.homeTemplate = template.Must(template.ParseFiles(filepath.Join(app.assets, "index.html")))

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

	interruptCancel := make(chan struct{})

	if errors := run(
		runner{
			func() error {
				certFile := filepath.Join(app.config, "cert.pem")
				keyFile := filepath.Join(app.config, "key.pem")

				certs, err := app.certs(certFile, keyFile)
				if err != nil {
					log.Print(err)
					return err
				}

				// secure cookie
				//TODO comon sc name, value, hash, block
				sc := securecookie.New([]byte(""), nil)

				authenticate := func(h http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						cookie, err := r.Cookie("cookie-name")
						if err != nil {
							//TODO auth page redirect
							http.Redirect(w, r, "auth", http.StatusUnauthorized)
							return
						}
						value := ""
						if err := sc.Decode("cookie-name", cookie.Value, &value); err != nil {
							http.Redirect(w, r, "auth", http.StatusUnauthorized)
							http.Error(w, err.Error(), http.StatusUnauthorized)
							return
						}
						//TODO expiration check, user agent check
						h.ServeHTTP(w, r)
					})
				}

				mux := http.NewServeMux()
				mux.Handle("/", authenticate(http.HandlerFunc(app.homeHandler)))
				mux.Handle("/auth", http.HandlerFunc(authHandler))
				mux.Handle("/favicon.ico", http.FileServer(http.Dir(app.assets)))
				mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(filepath.Join(app.assets, "js")))))
				mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(filepath.Join(app.assets, "css")))))
				mux.Handle("/ws", authenticate(websocket.Handler(app.websocketHandler)))

				s := &http.Server{
					Addr:    ":" + app.port,
					Handler: https.EnforceTLS(mux),
					TLSConfig: &tls.Config{
						Certificates: certs,
					},
				}

				log.Printf("starting http server with TLS")
				return s.ServeTLS(nl, "", "")
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

func (app application) certs(certFile, keyFile string) ([]tls.Certificate, error) {
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
			return nil, fmt.Errorf("couldn't create certificates: %s", err)
		}
		cert, err := tls.X509KeyPair(certBlob, keyBlob)
		if err != nil {
			return nil, fmt.Errorf("couldn't load created certificates: %s", err)
		}

		if err := certificates.Save(certFile, keyFile, certBlob, keyBlob); err != nil {
			log.Printf("failed to save https certificates (cert.pem, key.pem) to %s: %s", app.config, err)
			log.Print("certificates will be lost on exit")
		} else {
			log.Printf("created https certificates (cert.pem, key.pem) to %s", app.config)
		}

		certs = append(certs, cert)
	}

	return certs, nil
}
