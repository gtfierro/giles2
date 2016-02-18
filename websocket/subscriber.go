package websocket

import (
	"github.com/gorilla/websocket"
	giles "github.com/gtfierro/giles2/archiver"
	"sync"
	"time"
)

const (
	pongPeriod      = 60 * time.Second
	pingPeriod      = 2 * time.Second
	writeWait       = 10 * time.Second
	maxMessageSize  = 1024
	clientQueueSize = 2048
)

type WebSocketSubscriber struct {
	ws           *websocket.Conn
	closeC       chan bool
	outbound     chan []byte
	notify       chan bool
	subscription *giles.Subscriber
	sync.Mutex
}

func (wss *WebSocketSubscriber) handleError(e error) {
	if e == nil {
		return
	}
	log.Errorf("WS error %s", e.Error())
}

func StartSubscriber(ws *websocket.Conn) *giles.Subscriber {
	wss := &WebSocketSubscriber{ws: ws, outbound: make(chan []byte, clientQueueSize), closeC: make(chan bool), notify: make(chan bool)}
	wss.subscription = giles.NewSubscriber(wss.closeC, 10, wss.handleError)
	m.initialize <- wss

	go func(wss *WebSocketSubscriber) {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			m.remove <- wss
		}()
		for {
			select {
			case val := <-wss.subscription.C:
				wss.Lock()
				wss.ws.WriteJSON(val)
				wss.Unlock()
			case <-ticker.C:
				wss.Lock()
				if err := wss.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					wss.Unlock()
					log.Errorf("web socket error %v", err)
					return
				}
				wss.Unlock()
			}
		}
	}(wss)

	return wss.subscription
}
