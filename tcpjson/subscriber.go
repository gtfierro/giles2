package tcpjson

import (
	"encoding/json"
	giles "github.com/gtfierro/giles2/archiver"
	"net"
	"sync"
)

type TCPJSONSubscriber struct {
	conn         net.Conn
	subscription *giles.Subscriber
	closeC       chan bool
	closed       bool
	sync.Mutex
}

func (tsub *TCPJSONSubscriber) handleError(e error) {
	if e == nil {
		return
	}
	tsub.Lock()
	log.Error(e)
	tsub.conn.Write([]byte(e.Error()))
	tsub.closed = true
	tsub.Unlock()
	return
}

func StartTCPJSONSubscriber(conn net.Conn) *giles.Subscriber {
	tsub := &TCPJSONSubscriber{conn: conn, closed: false, closeC: make(chan bool)}
	tsub.subscription = giles.NewSubscriber(tsub.closeC, 10, tsub.handleError)
	writer := json.NewEncoder(tsub.conn)
	go func(tsub *TCPJSONSubscriber, writer *json.Encoder) {
		var err error
		for val := range tsub.subscription.C {
			tsub.Lock()
			if tsub.closed {
				tsub.conn.Close()
				tsub.closeC <- true
				tsub.Unlock()
				break
			}
			log.Debugf("repub %v", val)
			err = writer.Encode(val)
			tsub.Unlock()
			tsub.handleError(err)
		}
	}(tsub, writer)
	return tsub.subscription
}
