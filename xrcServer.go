package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"xgo"

	"golang.org/x/net/websocket"
)

var disp *xgo.Display

var port = flag.String("p", "10905", "http service port")
var assets = flag.String("d", "assets/", "working dir")

var homeTempl *template.Template

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

	flag.Parse()
	homeTempl = template.Must(template.ParseFiles(filepath.Join(*assets, "index.html")))

	d, err := xgo.OpenDisplay("")
	if err != nil {
		log.Fatal(err)
	}
	disp = d

	nl, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("xrcServer listens on port :%s", *port)

	interruptCancel := make(chan struct{})

	if errors := concurent(
		runner{
			func() error {

				mux := http.NewServeMux()
				mux.HandleFunc("/", homeHandler)
				mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(*assets+"js/"))))
				mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(*assets+"css/"))))
				mux.Handle("/ws", websocket.Handler(wsHandler))

				return http.Serve(nl, mux)
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

func concurent(runners ...runner) []error {
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
