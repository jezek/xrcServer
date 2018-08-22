package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type pair struct {
	mx               sync.Mutex
	pairOpened       bool
	pairOpenDuration time.Duration
	passwordLen      int
	passwordDuration time.Duration
	cookieDuration   time.Duration
	expireControl    chan<- interface{}
	expireDone       <-chan struct{}
	wg               *sync.WaitGroup
}

func (p *pair) newPassword() (<-chan struct{}, []byte, error) {
	p.mx.Lock()
	defer p.mx.Unlock()
	//log.Printf("start pair.newPassword()")
	//defer log.Printf("end pair.newPassword()")

	if p.expireControl != nil {
		//prolong password
		passwordChannel := make(chan []byte)

		p.expireControl <- passwordChannel
		return nil, <-passwordChannel, nil
		//p.expireControl <- nil
		//return nil, nil, nil
	}

	passwordBytes := make([]byte, p.passwordLen)
	if p.passwordLen > 0 {
		if _, err := rand.Read(passwordBytes); err != nil {
			return nil, nil, err
		}
	}

	password := pairPassword{passwordBytes}
	control := make(chan interface{})
	done := password.expire(control, p.passwordDuration)

	p.expireControl = control
	p.wg = &sync.WaitGroup{}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		//log.Printf("\tstart pair.newPassword(): go func()")
		//defer log.Printf("\tend pair.newPassword(): go func()")

		for done != nil {
			if _, ok := <-done; !ok {
				done = nil
			}
		}

		p.mx.Lock()
		defer p.mx.Unlock()
		p.expireControl = nil
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
		//log.Printf("pair.clearPassword(): wait for wg")
		log.Printf("waiting for expirable pair processes to stop")
		wg.Wait()
		log.Printf("all expirable pair processes stopped")
	}
}

func (p *pair) authorize(b []byte) bool {
	defer p.clearPassword()

	p.mx.Lock()
	defer p.mx.Unlock()

	if p.passwordLen == 0 {
		return true
	}

	if p.expireControl == nil {
		return false
	}

	passwordBytes := []byte{}
	passwordChannel := make(chan []byte)

	p.expireControl <- passwordChannel
	passwordBytes = <-passwordChannel

	return bytes.Equal(passwordBytes, b)
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

func (p pairPassword) expire(control <-chan interface{}, d time.Duration) <-chan struct{} {
	//log.Printf("start pairPassword.expire()")
	//defer log.Printf("end pairPassword.expire()")

	done := make(chan struct{})
	go func() {
		defer close(done)

		//log.Printf("\tstart pairPassword.expire(): go func()")
		//defer log.Printf("\tend pairPassword.expire(): go func()")

		for control != nil {
			//log.Printf("\tpairPassword.expire(): go func(): waiting for timer or control")
			select {
			case i, ok := <-control:
				//log.Printf("\tpairPassword.expire(): go func(): got control")
				if !ok {
					//log.Printf("\tpairPassword.expire(): go func(): control closed")
					log.Printf("pair password forced expire")
					control = nil
					break
				}
				switch it := i.(type) {
				case chan []byte:
					//log.Printf("\tpairPassword.expire(): go func(): requesting password")
					log.Printf("pair password requested")
					it <- p.bytes
				case nil:
					//log.Printf("\tpairPassword.expire(): go func(): requesting prolong")
					log.Printf("pair password expiration prolonged")
				}
			case <-time.After(d):
				//log.Printf("\tpairPassword.expire(): go func(): got timer")
				log.Printf("pair password expired by timer")
				control = nil
			}
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
			p.mx.Lock()
			if p.pairOpened == false {
				log.Printf("Unlocking application for pairing for %v", p.pairOpenDuration)
				p.pairOpened = true
				lock = time.After(p.pairOpenDuration)
			} else {
				log.Printf("Pairing allready unlocked")
			}
			p.mx.Unlock()
		case <-lock:
			p.mx.Lock()
			log.Printf("Locking application for pairing")
			p.pairOpened = false
			lock = nil
			p.mx.Unlock()
		case <-cancel:
			unlock = nil
		}
	}
}
