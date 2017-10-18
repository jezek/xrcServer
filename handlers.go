package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/securecookie"
	"golang.org/x/net/websocket"
)

type appUser struct {
	Agent string
}

func (app application) newCookie() (string, *securecookie.SecureCookie, error) {
	//TODO cache
	pub, err := app.publicKey()
	if err != nil {
		return "", nil, err
	}
	cookieHash := sha1.Sum(pub)
	cookieName := base64.RawStdEncoding.EncodeToString(cookieHash[:])

	priv, err := app.privateKey()
	if err != nil {
		return cookieName, nil, err
	}
	//TODO user password via securecookie block encryption
	sCookie := securecookie.New(priv, nil)
	//TODO what if i want different max age?
	sCookie.MaxAge(3600 * 24 * 365)
	return cookieName, sCookie, nil
}

func (app application) authenticate(h http.Handler) http.Handler {
	cookieName, sCookie, err := app.newCookie()
	if err != nil {
		log.Print(err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			//TODO auth page redirect
			http.Redirect(w, r, "pair", http.StatusUnauthorized)
			return
		}

		user := appUser{}
		if err := sCookie.Decode(cookieName, cookie.Value, &user); err != nil {
			http.Redirect(w, r, "pair", http.StatusUnauthorized)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if user.Agent != r.UserAgent() {
			//TODO user agent is veery similiar ... maybe update? (if user uses password, update agent)
			http.Redirect(w, r, "pair", http.StatusUnauthorized)
			http.Error(w, "UserAgent changed!", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (app application) pairHandler(w http.ResponseWriter, r *http.Request) {
	cookieName, sCookie, err := app.newCookie()
	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//TODO start new passphrase timeout, is allready running, reset timer
	secret := r.URL.Path

	//TODO confront passphrase timer with phrase
	if secret != "" {
		if secret == "1234" {

			user := appUser{
				Agent: r.UserAgent(),
			}
			encoded, err := sCookie.Encode(cookieName, user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cookie := &http.Cookie{
				Name:  cookieName,
				Value: encoded,
				Path:  "/",
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			log.Print("pairHandler: paired")
			return
		}
		http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
		log.Print("pairHandler: wrong secret")
		return
	}

	w.Write([]byte("TODO pair form"))
}

func (app application) homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("homeHandler: %v", r.URL.Path)
	if r.URL.Path != "/" {
		//log.Printf("Send him file: %v", http.Dir(req.URL.Path))
		//http.ServeFile(c, req, req.URL.Path)
		return
	}
	log.Printf("homeHandler: template \"%s\"", app.homeTemplate.Name())
	app.homeTemplate.Execute(w, r.Host)
}

type message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func (app application) websocketHandler(ws *websocket.Conn) {
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
			app.display.DefaultScreen().Window().Pointer().Control().MoveRelative(x, y)
		case "down":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: down: %s", b)
			switch b {
			case "left":
				app.display.DefaultScreen().Window().Pointer().Control().DownLeft()
			case "right":
				app.display.DefaultScreen().Window().Pointer().Control().DownRight()
			default:
				log.Printf("wsHandler: down: %s unknown", b)
			}
		case "up":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: up: %s", b)
			switch b {
			case "left":
				app.display.DefaultScreen().Window().Pointer().Control().UpLeft()
			case "right":
				app.display.DefaultScreen().Window().Pointer().Control().UpRight()
			default:
				log.Printf("wsHandler: up: %s unknown", b)
			}
		case "click":
			b := m.Data["button"].(string)
			log.Printf("wsHandler: click: %s", b)
			switch b {
			case "left":
				app.display.DefaultScreen().Window().Pointer().Control().ClickLeft()
			case "right":
				app.display.DefaultScreen().Window().Pointer().Control().ClickRight()
			default:
				log.Printf("wsHandler: click: %s unknown", b)
			}
		case "scroll":
			dir := m.Data["dir"].(string)
			log.Printf("wsHandler: scroll: %s", dir)
			switch dir {
			case "down":
				app.display.DefaultScreen().Window().Pointer().Control().ScrollDown()
			case "up":
				app.display.DefaultScreen().Window().Pointer().Control().ScrollUp()
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
			if err := app.display.DefaultScreen().Window().Keyboard().Control().Write(text); err != nil {
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
