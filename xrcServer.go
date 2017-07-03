package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"xgo"

	"golang.org/x/net/websocket"
)

var disp *xgo.Display

var port = *flag.String("p", "8888", "http service port")
var assets = *flag.String("d", "", "working dir")

func init() {
	flag.Parse()
}

var httpHomeTemplateFilename = assets + "templates/home.html"
var homeTempl = template.Must(template.ParseFiles(httpHomeTemplateFilename))

func homeHandler(c http.ResponseWriter, req *http.Request) {
	log.Printf("homeHandler: %v", req.URL.Path)
	if req.URL.Path != "/" {
		//log.Printf("Send him file: %v", http.Dir(req.URL.Path))
		//http.ServeFile(c, req, req.URL.Path)
		return
	}
	log.Printf("homeHandler: template \"%s\"", httpHomeTemplateFilename)
	homeTempl.Execute(c, req.Host)
}

func wsHandler(ws *websocket.Conn) {
	log.Printf("wsHandler: start")
	defer log.Printf("wsHandler: stop")
	defer ws.Close()
	var msg string
	for {
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			log.Printf("wsHandler: recieve error: %s", err)
			break
		}
		log.Printf("wsHandler: recieved message: %s", msg)
	}
}

func main() {
	var err error
	disp, err = xgo.OpenDisplay("")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(assets+"js/"))))
	http.Handle("/ws", websocket.Handler(wsHandler))
	log.Printf("xrcServer listens on port :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("http.ListenAndServe:", err)
	}
}
