package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/securecookie"
	"golang.org/x/net/websocket"
)

func authHandler(w http.ResponseWriter, r *http.Request) {
	//TODO start new passphrase timeout, is allready running, reset timer
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	secret := r.FormValue("s")
	//TODO confront passphrase timer with phrase
	if secret == "1234" {
		sc := securecookie.New([]byte(""), nil)
		//TODO expiration, user agent
		encoded, err := sc.Encode("cookie-name", "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cookie := &http.Cookie{
			Name:  "cookie-name",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func homeHandler(c http.ResponseWriter, req *http.Request) {
	log.Printf("homeHandler: %v", req.URL.Path)
	if req.URL.Path != "/" {
		//log.Printf("Send him file: %v", http.Dir(req.URL.Path))
		//http.ServeFile(c, req, req.URL.Path)
		return
	}
	log.Printf("homeHandler: template \"%s\"", homeTempl.Name())
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
		case "keyinput", "key":
			log.Printf("wsHandler: %s: %v", m.Type, m.Data)
			text, ok := m.Data["text"].(string)
			log.Printf("wsHandler: key: text codes %v", []byte(text))
			if !ok {
				log.Printf("wsHandler: key: no text, or not string: %v", m.Data["text"])
				break
			}
			if err := disp.DefaultScreen().Window().Keyboard().Control().Write(text); err != nil {
				log.Printf("wsHandler: key: x keyboard write error: %v", err)
				//TODO send error back
				break
			}
			send <- []byte(msg)
		default:
			log.Printf("wsHandler: unknown type: %s", m.Type)
		}
	}
}
