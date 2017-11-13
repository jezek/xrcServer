package main

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gorilla/securecookie"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/net/websocket"
)

type appUser struct {
	Agent     string `json:"-"`
	ActiveTab string `json:"activeTab"`
	KFocus    string `json:"keyinputsFocusing"`
}

func (app *application) newAuthSecureCookie() (string, *securecookie.SecureCookie, error) {
	pub, err := app.publicKey()
	if err != nil {
		return "", nil, err
	}
	cookieHash := sha1.Sum(pub)
	cookieName := base64.RawURLEncoding.EncodeToString(cookieHash[:])

	priv, err := app.privateKey()
	if err != nil {
		return cookieName, nil, err
	}
	//TODO? user password via securecookie block encryption
	sCookie := securecookie.New(priv, nil)
	sCookie.MaxAge(int(app.authCookeieDuration.Seconds()))
	return cookieName, sCookie, nil
}

func (app *application) newPairSecureCookie() (string, *securecookie.SecureCookie, error) {
	pub, err := app.publicKey()
	if err != nil {
		return "", nil, err
	}
	cookieHash := sha1.Sum(pub)
	cookieName := base64.RawURLEncoding.EncodeToString(cookieHash[:])

	priv, err := app.privateKey()
	if err != nil {
		return cookieName, nil, err
	}
	privHash := sha256.Sum256(priv)
	sCookie := securecookie.New(priv, privHash[:])
	sCookie.MaxAge(int(app.authPassDuration.Seconds()))
	return cookieName, sCookie, nil
}

func (app *application) authenticate(h http.Handler) http.Handler {
	//log.Print("authenticate")
	cookieName, sCookie, err := app.newAuthSecureCookie()
	if err != nil {
		log.Printf("authenticate: new auth securecookie error: %s", err.Error())
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//log.Print("authenticate: handler")
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			log.Print("authenticate: handler: user has no auth cookie")
			http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
			return
		}

		user := appUser{}
		if err := sCookie.Decode(cookieName, cookie.Value, &user); err != nil {
			log.Printf("authenticate: handler: auth cookie decode error: %s", err.Error())
			http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
			return
		}

		if user.Agent != r.UserAgent() {
			//TODO? user agent is veery similiar ... maybe update (if user uses password, update agent)
			log.Printf("authenticate: handler: auth cookie UserAgent changed!")

			//delete auth cookie
			cookie.Value = ""
			cookie.Expires = time.Unix(0, 0)
			http.SetCookie(w, cookie)

			//TODO? message via flash cookie or parameter
			http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
			return
		}

		ctx := context.WithValue(r.Context(), appUser{}, user)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) pairHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("pairHandler")
	defer log.Printf("pairHandler end")

	ascn, asc, err := app.newAuthSecureCookie()
	if err != nil {
		log.Printf("pairHandler: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pscn, psc, err := app.newPairSecureCookie()
	if err != nil {
		log.Printf("pairHandler: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ac, err := r.Cookie(ascn)
	if err == nil {
		//log.Printf("pairHandler: got auth cookie")
		user := appUser{}
		err := asc.Decode(ascn, ac.Value, &user)
		if err == nil {
			log.Print("pairHandler: allready authenticated")
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		//log.Printf("pairHandler: have auth cookie but cant decode: %s", err.Error())
		if ascn != pscn {
			log.Printf("pairHandler: auth cookie can NOT be pair cookie")
			//delete auth cookie
			ac = &http.Cookie{
				Name:    ascn,
				Value:   "",
				Expires: time.Unix(0, 0),
				Path:    "/",
			}
			http.SetCookie(w, ac)
			//log.Print("pairHandler: corrupt auth cookie set to delete")
		}
		//log.Printf("pairHandler: auth cookie can also be pair cookie")
	} else {
		//log.Printf("pairHandler: don't have auth cookie: %s", err.Error())
	}

	if app.authPassLen > 0 {
		passphraseUrl := r.URL.Path
		pc, err := r.Cookie(pscn)

		//Generates or prolong existing password.
		//Converts password to hex string and splits into half.
		//First half is printed to server stdout
		//Second half goes to client via cookie
		generateOrProlongPassword := func() error {
			if err := app.authNewPassword(); err != nil {
				return err
			}
			passphrase := hex.EncodeToString(app.authPassBytes)
			//log.Printf("pairHandler: passphrase generated: %s", passphrase)
			passphraseServer := passphrase[:len(passphrase)/2]
			passphraseClient := passphrase[len(passphrase)/2:]

			//set pair cookie
			encoded, err := psc.Encode(pscn, passphraseClient)
			if err != nil {
				return fmt.Errorf("generateOrProlongPassword: pair cookie encoding error: %s", err.Error())
			}
			pc = &http.Cookie{
				Name:    pscn,
				Value:   encoded,
				Expires: time.Now().Add(app.authPassDuration),
				Path:    "/pair/",
			}
			//log.Printf("pairHandler: generateOrProlongPassword: pair cookie: %#v", pc)
			http.SetCookie(w, pc)
			//log.Printf("pairHandler: generateOrProlongPassword: pair cookie sent")
			passphraseServerHumanReadable := insertEveryN(" ", passphraseServer, 4)
			//show passphrase
			fmt.Println(strings.Repeat("*", 30))
			fmt.Println("Passphrase:", passphraseServerHumanReadable)
			fmt.Println(strings.Repeat("*", 30))

			//store passphrase to tmp file
			if err := app.pairPassphraseToFile([]byte(passphraseServerHumanReadable), "passphrase.txt"); err != nil {
				log.Printf("generateOrProlongPassword: can't create tmp text file: %v", err)
			}

			ip, err := externalIP()
			if err != nil {
				log.Printf("generateOrProlongPassword: obtaining external ip address error: %v", err)
			} else {
				//create passphrase url for LAN
				protocol := "http"
				if app.noTLS == false {
					protocol += "s"
				}
				passphraseServerURL := protocol + "://" + ip.String() + ":" + app.port + "/pair/" + passphraseServer

				//store passphrase url to tmp file
				if err := app.pairPassphraseToFile([]byte(passphraseServerURL), "passphrase.url"); err != nil {
					log.Printf("generateOrProlongPassword: can't create tmp url file: %v", err)
				}

				//create and store passphrase url as qr code in png file
				if png, err := qrcode.Encode(passphraseServerURL, qrcode.Medium, 256); err != nil {
					log.Printf("generateOrProlongPassword: qr code encoding error: %v", err)
				} else {
					if err := app.pairPassphraseToFile(png, "passphrase.png"); err != nil {
						log.Printf("generateOrProlongPassword: qr code saving error: %v", err)
					}
				}
			}

			return nil
		}

		if err != nil {
			//no pair cookie in requet (or some other error)
			//log.Printf("pairHandler: no request pair cookie: %s", err.Error())
			if passphraseUrl != "" {
				log.Print("pairHandler: url not empty")
				http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
				return
			}

			if err := generateOrProlongPassword(); err != nil {
				log.Printf("pairHandler: password generate or prolong error: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			//show pair form
			if err := app.pairTemplate.Execute(w, struct{}{}); err != nil {
				log.Printf("pairHandler: pairTemplate execute error: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("pairHandler: pair page served: %s", r.RemoteAddr)
			return
		}

		//got pair cookie, try decode it
		//log.Printf("pairHandler: got pair cookie")
		pcp := ""
		if err := psc.Decode(pscn, pc.Value, &pcp); err != nil {
			log.Printf("pairHandler: can't decode request pair cookie: %s", err)
			//delete pair cookie
			pc = &http.Cookie{
				Name:    pscn,
				Value:   "",
				Expires: time.Unix(0, 0),
				Path:    "/pair/",
			}
			http.SetCookie(w, pc)
			//log.Print("pairHandler: corrupt pair cookie set to delete")

			http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
			//TODO? show why invalid. expired edited?
			//TODO? message via flash cookie
			//TODO? attempt to guess cookie do something!
			return
		}

		//log.Printf("pairHandler: request pair cookie decoded, got pair cookie passphrase: %s", pcp)
		if passphraseUrl == "" {
			log.Printf("pairHandler: url passphrase missing")

			if err := generateOrProlongPassword(); err != nil {
				log.Printf("pairHandler: password generate or prolong error: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			//show pair form
			if err := app.pairTemplate.Execute(w, struct{}{}); err != nil {
				log.Printf("pairHandler: pairTemplate execute error: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("pairHandler: pair page served: %s", r.RemoteAddr)
			return
		}

		passphrase := stripWhiteSpaces(passphraseUrl)
		//log.Printf("pairHandler: got url passphrase: %s", passphrase)
		//log.Printf("pairHandler: got combined passphrase: %s", passphrase+pcp)
		password, err := hex.DecodeString(passphrase + pcp)
		if err != nil {
			log.Printf("pairHandler: decoding passphrase error: %s", err.Error())
			app.authClearPassword()
		} else {
			//log.Printf("pairHandler: got password from passphrase")
		}

		if err != nil || !app.auth(password) {
			log.Print("pairHandler: wrong passphrase")
			//delete pair cookie
			pc = &http.Cookie{
				Name:    pscn,
				Value:   "",
				Expires: time.Unix(0, 0),
				Path:    "/pair/",
			}
			http.SetCookie(w, pc)
			//log.Print("pairHandler: invalid pair cookie set to delete")

			//TODO user misses passphrase for more times, block
			//TODO? message via flash cookie
			http.Redirect(w, r, "/pair/", http.StatusTemporaryRedirect)
			return
		}

		//delete pair cookie
		pc = &http.Cookie{
			Name:    pscn,
			Value:   "",
			Expires: time.Unix(0, 0),
			Path:    "/pair/",
		}
		http.SetCookie(w, pc)
		//log.Print("pairHandler: pair cookie set to delete")

		log.Print("pairHandler: passphrase is correct")
	} else {
		log.Print("pairHandler: no password required")
	}

	//log.Print("pairHandler: pair user")

	user := appUser{
		Agent: r.UserAgent(),
	}
	encoded, err := asc.Encode(ascn, user)
	if err != nil {
		log.Printf("pairHandler: auth cookie encoding error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//set auth cookie
	ac = &http.Cookie{
		Name:    ascn,
		Value:   encoded,
		Expires: time.Now().Add(app.authCookeieDuration),
		Path:    "/",
	}
	http.SetCookie(w, ac)
	//log.Print("pairHandler: auth cookie set")

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	log.Print("pairHandler: user paired")
}

func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	//log.Printf("homeHandler: %v", r.URL.Path)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user, ok := r.Context().Value(appUser{}).(appUser)
	if !ok {
		txt := "homeHandler: no user context"
		log.Print(txt)
		http.Error(w, txt, http.StatusInternalServerError)
		return
	}

	userConfig, err := json.Marshal(user)
	if err != nil {
		log.Printf("homeHandler: user config to json error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		UserConfig template.JS
		Debug      bool
	}{
		template.JS(string(userConfig)),
		app.clientDebug,
	}

	//log.Printf("homeHandler: template \"%s\"", app.homeTemplate.Name())
	//log.Printf("homeHandler: template data: %#v", data)
	if err := app.homeTemplate.Execute(w, data); err != nil {
		log.Printf("homeHandler: homeTemplat execute error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("home page served to athorized user: %s", r.RemoteAddr)
}

type message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func (app *application) websocketHandler(w http.ResponseWriter, r *http.Request) {
	//log.Printf("websocketHandler: start")
	user, ok := r.Context().Value(appUser{}).(appUser)
	if !ok {
		txt := "websocketHandler: no user context"
		log.Print(txt)
		http.Error(w, txt, http.StatusInternalServerError)
		return
	}

	websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("websocket.Handler: start")
		defer log.Printf("websocket.Handler: stop")
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
						log.Printf("websocket.Handler: send: closed")
						return
					}
					if _, err := ws.Write(msg); err != nil {
						log.Printf("websocket.Handler: send: error %s", err.Error())
					}
				}
			}
		}()

		var msg []byte
		for {
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				log.Printf("websocket.Handler: recieve error: %s", err.Error())
				break
			}
			//log.Printf("websocket.Handler: recieved message: %s", msg)
			m := message{}
			if err := json.Unmarshal(msg, &m); err != nil {
				log.Printf("websocket.Handler: message json unmarshal error: %s", err)
				break
			}
			//log.Printf("websocket.Handler: m: %#v", m)
			switch m.Type {
			case "moverelative":
				x, y := int(2*m.Data["x"].(float64)), int(2*m.Data["y"].(float64))
				log.Printf("websocket.Handler: move relative x: %d, y:  %d", x, y)
				app.display.DefaultScreen().Window().Pointer().Control().MoveRelative(x, y)
			case "down":
				b := m.Data["button"].(string)
				log.Printf("websocket.Handler: down: %s", b)
				switch b {
				case "left":
					app.display.DefaultScreen().Window().Pointer().Control().DownLeft()
				case "right":
					app.display.DefaultScreen().Window().Pointer().Control().DownRight()
				default:
					log.Printf("websocket.Handler: down: %s unknown", b)
				}
			case "up":
				b := m.Data["button"].(string)
				log.Printf("websocket.Handler: up: %s", b)
				switch b {
				case "left":
					app.display.DefaultScreen().Window().Pointer().Control().UpLeft()
				case "right":
					app.display.DefaultScreen().Window().Pointer().Control().UpRight()
				default:
					log.Printf("websocket.Handler: up: %s unknown", b)
				}
			case "click":
				b := m.Data["button"].(string)
				log.Printf("websocket.Handler: click: %s", b)
				switch b {
				case "left":
					app.display.DefaultScreen().Window().Pointer().Control().ClickLeft()
				case "right":
					app.display.DefaultScreen().Window().Pointer().Control().ClickRight()
				default:
					log.Printf("websocket.Handler: click: %s unknown", b)
				}
			case "scroll":
				dir := m.Data["dir"].(string)
				log.Printf("websocket.Handler: scroll: %s", dir)
				switch dir {
				case "down":
					app.display.DefaultScreen().Window().Pointer().Control().ScrollDown()
				case "up":
					app.display.DefaultScreen().Window().Pointer().Control().ScrollUp()
				default:
					log.Printf("websocket.Handler: scroll: %s unknown", dir)
				}
			case "keyinput", "key":
				log.Printf("websocket.Handler: %s: %v", m.Type, m.Data)
				text, ok := m.Data["text"].(string)
				log.Printf("websocket.Handler: key: text codes %v", []byte(text))
				if !ok {
					log.Printf("websocket.Handler: key: no text, or not string: %v", m.Data["text"])
					break
				}
				//TODO sending alt+tab results in s+ a+ t+ t- a-

				if err := app.display.DefaultScreen().Window().Keyboard().Control().Write(text); err != nil {
					//log.Printf("websocket.Handler: key: x keyboard write error: %v", err)
					msg := message{
						m.Type,
						map[string]interface{}{
							"sender": m.Data["sender"],
							"error":  err.Error(),
						},
					}
					msgBytes, err := json.Marshal(msg)
					if err != nil {
						log.Printf("websocket.Handler: key: error message marshal error: %v", err)
						break
					}
					send <- msgBytes
					log.Printf("websocket.Handler: key: x keyboard write error sent: %#v", msg)
					break
				}
				send <- []byte(msg)
			case "cookieConfig":
				log.Printf("websocket.Handler: cookieConfig: %s", m.Data["updates"])
				sendError := func(ch chan<- []byte, errStr string) {
					//log.Printf("websocket.Handler: cookieConfig: sendError: %s", errStr)
					msg := message{m.Type, make(map[string]interface{})}
					if u, ok := m.Data["updates"]; ok {
						msg.Data["updates"] = u
					}
					msg.Data["error"] = errStr
					msgBytes, err := json.Marshal(msg)
					if err != nil {
						log.Printf("websocket.Handler: cookieConfig: sendError: message marshal error: %v", err)
						return
					}
					send <- msgBytes
					log.Printf("websocket.Handler: cookieConfig: sendError: sent: %#v", msg)
				}

				cfgStr, ok := m.Data["config"].(string)
				if !ok {
					sendError(send, fmt.Sprintf("cookieConfig.config is not a string: %s", reflect.TypeOf(m.Data["config"])))
					break
				}
				if err := json.Unmarshal([]byte(cfgStr), &user); err != nil {
					sendError(send, fmt.Sprintf("cookieConfig.config json unmarshal error: %s", err))
					break
				}

				cookieName, sCookie, err := app.newAuthSecureCookie()
				if err != nil {
					sendError(send, err.Error())
					break
				}
				encoded, err := sCookie.Encode(cookieName, user)
				if err != nil {
					sendError(send, err.Error())
					break
				}
				userConfig, err := json.Marshal(user)
				if err != nil {
					sendError(send, err.Error())
					break
				}

				msg := message{
					Type: m.Type,
					Data: map[string]interface{}{
						"config": userConfig,
						"cookie": map[string]interface{}{
							"name":    cookieName,
							"value":   encoded,
							"expires": time.Now().Add(app.authCookeieDuration).Format("Mon, 2 Jan 2006 15:04:05 MST"),
							"path":    "/",
						},
					},
				}
				if u, ok := m.Data["updates"]; ok {
					msg.Data["updates"] = u
				}
				//log.Printf("websocket.Handler: cookieConfig: returning: %#v", msg)
				msgBytes, err := json.Marshal(msg)
				if err != nil {
					log.Printf("websocket.Handler: cookieConfig: message marshal error: %s", err)
					break
				}

				send <- msgBytes
				//log.Printf("websocket.Handler: cookieConfig: returned config: %s", string(msg.Data["config"].([]byte)))

				//TODO if mate-settings-daemon detected, give an option to restart it
			default:
				log.Printf("websocket.Handler: unknown type: %s", m.Type)
			}
		}
	}).ServeHTTP(w, r)
}

func insertEveryN(what, where string, n int) string {
	if what == "" || where == "" || n == 0 || n >= len(where) {
		return where
	}
	res := []rune{}
	whereRunes := []rune(where)
	for i := 0; i < len(whereRunes)/n; i++ {
		res = append(res, whereRunes[i*n:i*n+n]...)
		if i != len(whereRunes)/n-1 {
			res = append(res, []rune(what)...)
		}
	}
	if len(whereRunes)%n > 0 {
		res = append(res, []rune(what)...)
		res = append(res, whereRunes[len(whereRunes)/n*n:]...)
	}
	return string(res)
}

func stripWhiteSpaces(where string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, where)
}

//https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip, nil
		}
	}
	return nil, errors.New("externalIP: are you connected to the network?")
}

func (app *application) pairPassphraseToFile(what []byte, where string) error {
	fileName := filepath.Join(os.TempDir(), "xrcServer."+app.port+"."+where)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(what); err != nil {
		return err
	}
	//TODO? remove tmp file after expire
	log.Printf("saved passphrase to: %s", fileName)
	return nil
}
