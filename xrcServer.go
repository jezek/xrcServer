package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"xgo"

	"golang.org/x/net/websocket"
)

var disp *xgo.Display

var port = flag.String("p", "8888", "http service port")
var assets = flag.String("d", "assets/", "working dir")

var httpHomeTemplateFilename string
var homeTempl *template.Template

func init() {
	flag.Parse()

	httpHomeTemplateFilename = *assets + "templates/home.html"
	homeTempl = template.Must(template.ParseFiles(httpHomeTemplateFilename))

}

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

type message struct {
	Type string                 `json:type`
	Data map[string]interface{} `json:data`
}

func wsHandler(ws *websocket.Conn) {
	log.Printf("wsHandler: start")
	defer log.Printf("wsHandler: stop")
	defer ws.Close()
	var msg string
	for {
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			log.Printf("wsHandler: recieve error: %s", err)
			break
		}
		//log.Printf("wsHandler: recieved message: %s", msg)
		m := message{}
		if err := json.Unmarshal([]byte(msg), &m); err != nil {
			log.Printf("wsHandler: unmarshal error: %s", err)
			break
		}
		//log.Printf("wsHandler: m: %#v", m)
		switch m.Type {
		case "moverelative":
			x, y := int(m.Data["x"].(float64)), int(m.Data["y"].(float64))
			log.Printf("wsHandler: move relative x: %d, y:  %d", x, y)
			disp.DefaultScreen().Window().Pointer().Control().MoveRelative(x, y)
		case "clickleft":
			log.Printf("wsHandler: click left")
			disp.DefaultScreen().Window().Pointer().Control().ClickLeft()
		default:
			log.Printf("wsHandler: unknown type: %s", m.Type)
		}
	}
}

func main() {
	var err error
	disp, err = xgo.OpenDisplay("")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", homeHandler)
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(*assets+"js/"))))
	http.Handle("/ws", websocket.Handler(wsHandler))
	log.Printf("xrcServer listens on port :%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("http.ListenAndServe:", err)
	}
}
