package views

import (
	bw "gopkg.in/immesys/bw2bind.v5"
	"sync"
)

type forwarder struct {
	uri         string
	incoming    chan *bw.SimpleMessage
	forwardList map[*View]struct{}
	sync.RWMutex
}

func newForwarder(incoming chan *bw.SimpleMessage, uri string) *forwarder {
	f := new(forwarder)
	f.uri = uri
	f.incoming = incoming
	f.forwardList = make(map[*View]struct{})

	go func() {
		for msg := range incoming {
			f.RLock()
			for view := range f.forwardList {
				select {
				case view.C <- msg:
				default:
				}
			}
			f.RUnlock()
		}
	}()

	return f
}

func (f *forwarder) addViews(vs ...*View) {
	f.Lock()
	for _, view := range vs {
		f.forwardList[view] = struct{}{}
	}
	f.Unlock()
}

func (f *forwarder) removeViews(vs ...*View) {
	f.Lock()
	for _, view := range vs {
		delete(f.forwardList, view)
	}
	f.Unlock()
}
