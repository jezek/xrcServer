package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"xgo"
)

type pair struct {
	app              *application
	passwordLen      int
	passwordDuration time.Duration
	cookieDuration   time.Duration

	mx            sync.Mutex
	expireControl chan func() (password []byte, result chan<- bool)
	wg            *sync.WaitGroup
}

func (_ pairPassword) hashPasswordBytes(pbytes []byte) []byte {
	hash := sha512.New()
	hash.Write(pbytes)
	return hash.Sum(nil)
}

var errPairPasswordAllreadyGenerated error = errors.New("Pair password allready generated")

// Returns read only channel, that closes after password is expired or forced to expire.
// Also returns the generated password. That is if everything gooes well.
// If not, the appropiate error is returned and the channel and password bytes are nil
func (p *pair) newPassword() (<-chan struct{}, []byte, error) {
	p.mx.Lock()
	defer p.mx.Unlock()
	//log.Printf("start pair.newPassword()")
	//defer log.Printf("end pair.newPassword()")

	if p.expireControl != nil {
		// password is allready generated, return error
		return nil, nil, errPairPasswordAllreadyGenerated
	}

	passwordBytes := make([]byte, p.passwordLen)
	if p.passwordLen > 0 {
		if _, err := rand.Read(passwordBytes); err != nil {
			return nil, nil, err
		}
	}
	//TODO use some constructor, to have proper hashing everyvhere
	password := pairPassword{pairPassword{}.hashPasswordBytes(passwordBytes)}

	control := make(chan func() ([]byte, chan<- bool))
	done := password.expire(control, p.passwordDuration)
	p.expireControl = control

	// launch gourutine, to wait for password to expire and clean up
	if p.wg == nil {
		p.wg = &sync.WaitGroup{}
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		//log.Printf("\tstart pair.newPassword(): go func()")
		//defer log.Printf("\tend pair.newPassword(): go func()")

		// wait for password expire or be forced to expire
		for done != nil {
			if _, ok := <-done; !ok {
				done = nil
			}
		}

		// password expired, no need for p.expireControl
		p.mx.Lock()
		defer p.mx.Unlock()
		p.expireControl = nil

		// also make wg nil, don't worry, about loosing it before wg.Wait() is done, it is handled upon closing expireControl
		p.wg = nil
		//log.Printf("\tpair.newPassword(): go func(): pair.expireControl = nil")
	}()

	return done, passwordBytes, nil
}

func (p *pair) clearPassword() {
	//log.Printf("start pair.clearPassword()")
	//defer log.Printf("end pair.clearPassword()")
	p.mx.Lock()
	if p.expireControl != nil {
		//log.Printf("pair.clearPassword(): closing control")
		close(p.expireControl)
	}
	wg := p.wg
	p.mx.Unlock()

	if wg != nil {
		log.Printf("pair.clearPassword(): waiting for expirable pair processes to stop")
		wg.Wait()
		log.Printf("pair.clearPassword(): all expirable pair processes stopped")
	}
}

func (p *pair) authorize(b []byte) bool {
	// right after authorization attempt, clear generated password whatever the result is
	defer p.clearPassword()

	p.mx.Lock()
	defer p.mx.Unlock()

	if p.passwordLen == 0 {
		return true
	}

	if p.expireControl == nil {
		return false
	}

	resultChannel := make(chan bool)

	p.expireControl <- func() ([]byte, chan<- bool) {
		return b, resultChannel
	}
	return <-resultChannel
}

func (p *pair) expirableFile(what []byte, where string, expire <-chan struct{}) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.expireControl == nil {
		return fmt.Errorf("expirableFile: allready expired")
	}

	file, err := os.OpenFile(where, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	if _, err := file.Write(what); err != nil {
		file.Close()
		if err := os.Remove(where); err == nil {
			log.Printf("expirableFile: after write error, file removed: %s", where)
		} else {
			log.Printf("expirableFile: after write error, file remove FAILED: %s, %v", where, err)
		}
		return err
	}
	file.Close()
	log.Printf("expirable file saved: %s", where)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		//log.Printf("\tstart pair.expirableFile(): go func(): waiting for password expire to cleanup")
		//defer log.Printf("\tend pair.expirableFile(): go func(): everything cleaned up")

		for expire != nil {
			if _, ok := <-expire; !ok {
				//log.Printf("\tpair.expirableFile(): go func(): got expire")
				expire = nil
			}
		}
		//password expired, do all stuff after expired passwd
		if err := os.Remove(where); err != nil {
			log.Printf("\tpair.expirableFile(): go func(): after expire, file remove FAILED: %s, %v", where, err)
		} else {
			log.Printf("expirable file removed: %s", where)
		}
	}()
	return nil
}

func (p *pair) expirableWindowWithPassphrase(expireDone <-chan struct{}, passphraseHumanReadable, passphraseUrl string) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.expireControl == nil {
		return fmt.Errorf("pair.expirableWindowWithPassphrase: allready expired")
	}
	if p.wg == nil {
		return fmt.Errorf("pair.expirableWindowWithPassphrase: WTF? pair expireControl is NOT nil, but waitgroup is nil!")
	}

	screen := p.app.display.DefaultScreen()
	gcc := xgo.GraphicsContextComponents{}
	gc, err := p.app.display.NewGraphicsContext(
		gcc.BackgroundPixel(screen.BlackPixel),
		gcc.ForegroundPixel(screen.WhitePixel),
		gcc.NewFontIfMatch("*fixed*-20-*"),
	)
	if err != nil {
		return fmt.Errorf("pair.expirableWindowWithPassphrase: graphics context for window texts creation error: %v", err)
	}

	windowSize := image.Point{}

	components := []struct {
		name     string
		drawData [3]image.Point // 0: bounds.Min, 1: bounds.Max, 2: drawPoint
		drawFunc func(*xgo.Pixmap, image.Point) error
	}{
		{
			name: "human readable text",
			drawData: func() [3]image.Point {
				if passphraseHumanReadable == "" {
					return [3]image.Point{}
				}
				info, err := gc.TextExtents(passphraseHumanReadable)
				if err != nil {
					log.Printf("pair.expirableWindowWithPassphrase: getting text extents info error: %v", err)
					return [3]image.Point{}
				}
				return [3]image.Point{
					image.Pt(0, 0),
					image.Pt(int(info.OverallWidth), int(info.OverallAscent+info.OverallDescent)),
					image.Pt(0, int(info.OverallAscent)),
				}
			}(),
			drawFunc: func(p *xgo.Pixmap, point image.Point) error {
				return p.Draw(xgo.PixmapDrawers{}.Text(passphraseHumanReadable, point, gc))
			},
		},
	}
	log.Println(windowSize)
	log.Println(components)
	{ // arange components stacked from up to down and get window size
		offset := 50
		windowSize.Y += offset
		for i, c := range components {
			for j, p := range c.drawData {
				components[i].drawData[j] = p.Add(image.Pt(0, windowSize.Y))
			}
			c = components[i]

			bounds := image.Rectangle{c.drawData[0], c.drawData[1]}.Bounds()
			if windowSize.X < bounds.Dx() {
				windowSize.X = bounds.Dx()
			}
			windowSize.Y += bounds.Dy() + offset
		}
		windowSize.X += 2 * offset
	}

	componentsDrawers := []xgo.PixmapDrawer{}
	{ // center components horizontaly (on x axis) and get pixmap drawers out of them
		for i, c := range components {
			width := image.Rectangle{c.drawData[0], c.drawData[1]}.Bounds().Dx()
			for j, p := range c.drawData {
				components[i].drawData[j] = p.Add(image.Pt((windowSize.X-width)/2, 0))
			}
			c = components[i]

			pt := c.drawData[2]
			componentsDrawers = append(
				componentsDrawers,
				func(p *xgo.Pixmap) error {
					return c.drawFunc(p, pt)
				},
			)
		}
	}

	// create pixmap & draw components to pixmap
	pixmap, err := screen.NewPixmap(
		windowSize,
		xgo.PixmapOperations{}.Draw(componentsDrawers...),
	)

	////create and store passphrase url as qr code in png file
	//if png, err := qrcode.Encode(passphraseServerURL, qrcode.Medium, 256); err != nil {
	//	log.Printf("pair.expirableWindowWithPassphrase: qr code encoding error: %v", err)
	//} else {
	//	if err := app.pair.expirableFile(png, fileWithPathPrefix+"passphrase.png", expire); err != nil {
	//		log.Printf("pair.expirableWindowWithPassphrase: qr code saving error: %v", err)
	//	}
	//}

	wo := xgo.WindowOperations{}
	win, err := p.app.display.NewWindow(
		wo.Size(windowSize),
		wo.Attributes(
			xgo.WindowAttributes{}.BackgroundPixmap(pixmap),
		),
		wo.Clear(),
		wo.Map(),
	)
	if err != nil {
		return fmt.Errorf("pair.expirableWindowWithPassphrase: can't create window: %v", err)
	}

	stopWindowCloseNotify := make(chan struct{})
	windowCloseRequest, err := win.CloseNotify(stopWindowCloseNotify)
	if err != nil {
		win.Destroy()
		return fmt.Errorf("pair.expirableWindowWithPassphrase: can't listen to window close request: %v", err)
	}

	// after this point window is successfuly opened
	// launch a gouroutine that handles window closing
	p.wg.Add(1) // append this goroutine to pair waitgroup. Waiting is done on password expire
	go func() {
		defer p.wg.Done()

		log.Printf("\tstart pair.expirableWindowWithPassphrase(): go func(): waiting for password expire to cleanup, or window closing")
		defer log.Printf("\tend pair.expirableWindowWithPassphrase(): go func(): everything cleaned up")

		defer win.Destroy() // destroy created window for sure if this routine exits

		// wait for passphrase to expire or window is closed
		for windowCloseRequest != nil {
			log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): select waiting signal")
			select {
			case _, ok := <-expireDone:
				log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): select got password expireDone signal")
				if !ok {
					log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): expireDone closed")
					// password expired, close window, etc...

					// stop window close listening
					close(stopWindowCloseNotify)
					stopWindowCloseNotify = nil
					// do not wait for expire in select
					expireDone = nil
				}
			case _, ok := <-windowCloseRequest:
				log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): select got window windowCloseRequest signal")
				if !ok {
					log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): windowCloseRequest closed")
					// somebody is trying to close the window, or listening to close is finished
					// however, expire password
					windowCloseRequest = nil
					//TODO expire password upon window closing
				}
			}
		}
		log.Printf("\tpair.expirableWindowWithPassphrase(): go func(): windowCloseRequest are nil")
	}()
	return nil
}

type pairPassword struct {
	bytes []byte
}

func (p pairPassword) expire(control <-chan func() ([]byte, chan<- bool), d time.Duration) <-chan struct{} {
	//log.Printf("start pairPassword.expire()")
	//defer log.Printf("end pairPassword.expire()")

	done := make(chan struct{})
	go func() {
		defer close(done)

		//log.Printf("\tstart pairPassword.expire(): go func()")
		//defer log.Printf("\tend pairPassword.expire(): go func()")

		//log.Printf("\tpairPassword.expire(): go func(): waiting for timer or control")
		select {
		case controlDataFunction, ok := <-control:
			//log.Printf("\tpairPassword.expire(): go func(): got control")
			if !ok {
				//log.Printf("\tpairPassword.expire(): go func(): control closed")
				log.Printf("Pair password forced to expire")
				break
			}
			passwordBytes, resultChannel := controlDataFunction()
			resultChannel <- bytes.Equal(p.bytes, p.hashPasswordBytes(passwordBytes))
			log.Printf("pair password authentification result sent")
		case <-time.After(d):
			//log.Printf("\tpairPassword.expire(): go func(): got timer")
			log.Printf("pair password expired by timer")
		}
	}()
	return done
}

func (p *pair) UnlockHandle(cancel <-chan struct{}) {
	log.Print("To unlock app for pairing, send SIGUSR1")

	unlock := make(chan os.Signal, 1)
	signal.Notify(unlock, syscall.SIGUSR1)
	defer signal.Stop(unlock)

	lock := (<-chan time.Time)(nil)
	for unlock != nil {
		select {
		case <-unlock:
			fmt.Printf("Unlocking application for pairing for %v", p.passwordDuration)
			if err := p.generatePassword(); err != nil {
				if err == errPairPasswordAllreadyGenerated {
					fmt.Printf("Pairing allready unlocked")
				} else {
					log.Printf("pair.UnlockHandle: password generate error: %s", err.Error())
				}
				break
			}
			lock = time.After(p.passwordDuration)
		case <-lock:
			fmt.Printf("Locking application for pairing")
			p.clearPassword()
			lock = nil
		case <-cancel:
			unlock = nil
		}
	}
}

//TODO open window with password text and qrcode
//Generates password.
//Converts password to hex string and is printed to server stdout
func (p *pair) generatePassword() error {
	expire, passwordBytes, err := p.newPassword()
	if err != nil {
		return err
	}
	if expire == nil {
		return errors.New("generatePassword: password expire channel is nil")
	}
	log.Printf("generatePassword: new password generated")

	passphrase := hex.EncodeToString(passwordBytes)

	passphraseUrl := ""
	if urlBase, err := p.app.GetBaseUrlLAN(); err != nil {
		log.Printf("generatePassword: cant get base url: %v", err)
	} else {
		passphraseUrl = urlBase + "pair/" + passphrase
	}

	passphraseHumanReadable := insertEveryN(" ", passphrase, 4)

	fmt.Println(strings.Repeat("*", 30))
	fmt.Println("Passphrase:", passphraseHumanReadable)
	if passphraseUrl != "" {
		fmt.Println("Url:", passphraseUrl)
	}
	fmt.Println(strings.Repeat("*", 30))

	if err := p.expirableWindowWithPassphrase(expire, passphraseHumanReadable, passphraseUrl); err != nil {
		log.Printf("generatePassword: can't create window with passphrase: %v", err)
	}

	return nil
}
