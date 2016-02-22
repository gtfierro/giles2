package http

import (
	"encoding/json"
	giles "github.com/gtfierro/giles2/archiver"
	"net/http"
	"sync"
)

type HTTPSubscriber struct {
	rw           http.ResponseWriter
	subscription *giles.Subscriber
	closed       bool
	_closeC      <-chan bool
	closeC       chan bool
	sync.Mutex
}

func (hs *HTTPSubscriber) handleError(e error) {
	if e == nil {
		return
	}
	hs.Lock()
	hs.rw.WriteHeader(500)
	hs.rw.Write([]byte(e.Error()))
	hs.closed = true
	hs.Unlock()
	return
}

func (hs *HTTPSubscriber) watchForClose() {
	go func() {
		<-hs._closeC
		log.Debug("closing")
		hs.Lock()
		hs.closed = true
		hs.Unlock()
		hs.closeC <- true
	}()
}

func StartHTTPSubscriber(rw http.ResponseWriter) *giles.Subscriber {
	var err error
	_closeC := rw.(http.CloseNotifier).CloseNotify()
	hs := &HTTPSubscriber{rw: rw, closed: false, _closeC: _closeC, closeC: make(chan bool)}
	hs.watchForClose()
	hs.subscription = giles.NewSubscriber(hs.closeC, 10, hs.handleError)
	writer := json.NewEncoder(rw)
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	go func(hs *HTTPSubscriber, writer *json.Encoder) {
		log.Debugf(">>> NEW HTTP REPUB %v", hs)
		for val := range hs.subscription.C {
			hs.Lock()
			if hs.closed {
				hs.Unlock()
				break
			}
			log.Debugf("repub %v", val)
			err = writer.Encode(val)
			hs.Unlock()
			hs.handleError(err)
			hs.Lock()
			hs.rw.Write([]byte{'\n', '\n'})
			if flusher, ok := hs.rw.(http.Flusher); ok && !hs.closed {
				flusher.Flush()
			}
			hs.Unlock()
		}
	}(hs, writer)

	return hs.subscription
}
