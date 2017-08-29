package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"xgo"

	"golang.org/x/net/websocket"
)

var disp *xgo.Display

var port = flag.String("p", "10905", "http service port")
var assets = flag.String("d", "assets/", "working dir")

var httpHomeTemplateFilename string
var homeTempl *template.Template

func init() {
	flag.Parse()

	httpHomeTemplateFilename = *assets + "index.html"
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
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func wsHandler(ws *websocket.Conn) {
	log.Printf("wsHandler: start")
	defer log.Printf("wsHandler: stop")
	defer ws.Close()

	wg := sync.WaitGroup{}
	defer wg.Wait()

	send := make(chan []byte, 1)
	defer close(send)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case msg, ok := <-send:
				if !ok {
					log.Printf("wsHandler: send: closed")
					return
				}
				if _, err := ws.Write(msg); err != nil {
					log.Printf("wsHandler: send: error %v", err)
				}
			}
		}
	}()

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
			x, y := int(2*m.Data["x"].(float64)), int(2*m.Data["y"].(float64))
			log.Printf("wsHandler: move relative x: %d, y:  %d", x, y)
			disp.DefaultScreen().Window().Pointer().Control().MoveRelative(x, y)
		case "down":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: down: %s", b)
			switch b {
			case "left":
				disp.DefaultScreen().Window().Pointer().Control().DownLeft()
			case "right":
				disp.DefaultScreen().Window().Pointer().Control().DownRight()
			default:
				log.Printf("wsHandler: down: %s unknown", b)
			}
		case "up":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: up: %s", b)
			switch b {
			case "left":
				disp.DefaultScreen().Window().Pointer().Control().UpLeft()
			case "right":
				disp.DefaultScreen().Window().Pointer().Control().UpRight()
			default:
				log.Printf("wsHandler: up: %s unknown", b)
			}
		case "click":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: click: %s", b)
			switch b {
			case "left":
				disp.DefaultScreen().Window().Pointer().Control().ClickLeft()
			case "right":
				disp.DefaultScreen().Window().Pointer().Control().ClickRight()
			default:
				log.Printf("wsHandler: click: %s unknown", b)
			}
		case "scroll":
			dir := m.Data["dir"].(string)
			log.Printf("wsHandler: scroll: %s", dir)
			switch dir {
			case "down":
				disp.DefaultScreen().Window().Pointer().Control().ScrollDown()
			case "up":
				disp.DefaultScreen().Window().Pointer().Control().ScrollUp()
			default:
				log.Printf("wsHandler: scroll: %s unknown", dir)
			}
		case "keyinput":
			log.Printf("wsHandler: keyinput: %v", m.Data)
			text, ok := m.Data["text"].(string)
			log.Printf("wsHandler: key: text codes %v", []byte(text))
			if !ok {
				log.Printf("wsHandler: key: no text, or not string: %v", m.Data["text"])
				break
			}
			if err := disp.DefaultScreen().Window().Keyboard().Control().Write(""); err != nil {
				log.Printf("wsHandler: key: x keyboard write error: %v", err)
				break
			}
			send <- []byte(msg)
		case "modifier", "key":
			log.Printf("wsHandler: %s: %v", m.Type, m.Data)
			name, ok := m.Data["name"].(string)
			if !ok {
				log.Printf("wsHandler: %s: no name, or not string: %v", m.Type, m.Data["name"])
				break
			}
			down, ok := m.Data["down"].(bool)
			if !ok {
				log.Printf("wsHandler: %s: no down, or not bool: %v", m.Type, m.Data["down"])
				break
			}
			s := "-"
			if down {
				s = "+"
			}
			text := fmt.Sprintf("%%%s%%\"%s\"", s, name)
			log.Printf("wsHandler: %s: text %s", m.Type, text)
			if err := disp.DefaultScreen().Window().Keyboard().Control().Write(""); err != nil {
				log.Printf("wsHandler: %s: x keyboard write error: %v", m.Type, err)
				break
			}
			send <- []byte(msg)
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
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(*assets+"css/"))))
	http.Handle("/ws", websocket.Handler(wsHandler))
	log.Printf("xrcServer listens on port :%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("http.ListenAndServe:", err)
	}
}
