package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type pair struct {
	passwordLen      int
	passwordDuration time.Duration
	cookieDuration   time.Duration

	mx            sync.Mutex
	expireControl chan func() (password []byte, result chan<- bool)
	expireDone    <-chan struct{}
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
			if err := p.generatePassword(); err != nil {
				if err == errPairPasswordAllreadyGenerated {
					log.Printf("Pairing allready unlocked")
				} else {
					log.Printf("pair.UnlockHandle: password generate error: %s", err.Error())
				}
				break
			}
			log.Printf("Unlocking application for pairing for %v", p.passwordDuration)
			lock = time.After(p.passwordDuration)
		case <-lock:
			log.Printf("Locking application for pairing")
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

	// encode to string via hex
	passphrase := hex.EncodeToString(passwordBytes)
	//log.Printf("pairHandler: passphrase generated: %s", passphrase)

	// encode passphrase to human readable form
	passphraseServerHumanReadable := insertEveryN(" ", passphrase, 4)

	if expire == nil {
		return errors.New("generatePassword: password expire channel is nil")
	}

	log.Printf("generatePassword: new password generated")

	// show human readable passphrase to stdout
	fmt.Println(strings.Repeat("*", 30))
	fmt.Println("Passphrase:", passphraseServerHumanReadable)
	fmt.Println(strings.Repeat("*", 30))

	return nil

	////do all stuff with password
	//fileWithPathPrefix := filepath.Join(os.TempDir(), "xrcServer."+app.port+".")

	////store passphrase to tmp file
	//if err := app.pair.expirableFile([]byte(passphraseServerHumanReadable), fileWithPathPrefix+"passphrase.txt", expire); err != nil {
	//	log.Printf("generatePassword: can't create tmp text file: %v", err)
	//}

	//ip, err := externalIP()
	//if err != nil {
	//	log.Printf("generatePassword: obtaining external ip address error: %v", err)
	//} else {
	//	//create passphrase url for LAN
	//	protocol := "http"
	//	if app.noTLS == false {
	//		protocol += "s"
	//	}
	//	passphraseServerURL := protocol + "://" + ip.String() + ":" + app.port + "/pair/" + passphraseServer

	//	//store passphrase url to tmp file
	//	if err := app.pair.expirableFile([]byte(passphraseServerURL), fileWithPathPrefix+"passphrase.url", expire); err != nil {
	//		log.Printf("generatePassword: can't create tmp url file: %v", err)
	//	}

	//	//create and store passphrase url as qr code in png file
	//	if png, err := qrcode.Encode(passphraseServerURL, qrcode.Medium, 256); err != nil {
	//		log.Printf("generatePassword: qr code encoding error: %v", err)
	//	} else {
	//		if err := app.pair.expirableFile(png, fileWithPathPrefix+"passphrase.png", expire); err != nil {
	//			log.Printf("generatePassword: qr code saving error: %v", err)
	//		}
	//	}
	//}

}
