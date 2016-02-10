package websocket

import (
	"github.com/gorilla/websocket"
	giles "github.com/gtfierro/giles2/archiver"
	"time"
)

const (
	pongPeriod      = 60 * time.Second
	pingPeriod      = 10 * time.Second
	writeWait       = 10 * time.Second
	maxMessageSize  = 1024
	clientQueueSize = 2048
)

type WebSocketSubscriber struct {
	ws           *websocket.Conn
	closed       bool
	closeC       chan bool
	outbound     chan []byte
	notify       chan bool
	subscription *giles.Subscriber
}

func (wss *WebSocketSubscriber) handleError(e error) {
	if e == nil {
		return
	}
	log.Error("WS error", e.Error())
}

func StartSubscriber(ws *websocket.Conn) *giles.Subscriber {
	wss := &WebSocketSubscriber{ws: ws, closed: false, outbound: make(chan []byte, clientQueueSize), closeC: make(chan bool), notify: make(chan bool)}
	wss.subscription = giles.NewSubscriber(wss.closeC, 10, wss.handleError)
	m.initialize <- wss

	go func(wss *WebSocketSubscriber) {
		log.Debug("start repub for %v", wss)
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			m.remove <- wss
			wss.ws.Close()
		}()
		for {
			select {
			case val := <-wss.subscription.C:
				log.Debug("repub %v", val)
				wss.ws.WriteJSON(val)
			case <-ticker.C:
				if err := wss.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
			}
		}
	}(wss)

	return wss.subscription
}
