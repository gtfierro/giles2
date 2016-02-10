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
		for val := range wss.subscription.C {
			log.Debug("repub %v", val)
			wss.ws.WriteJSON(val)
		}
	}(wss)

	return wss.subscription
}
