package archiver

import (
	"fmt"
	"github.com/gtfierro/giles2/archiver/internal/querylang"
)

type Subscriber struct {
	C            chan interface{}
	closed       <-chan bool
	errorHandler func(error)
	query        *querylang.ParsedQuery
}

// The [closed] argument is a channel provided by the protocol adapter
// for a client. When a value is sent on this channel, the client is
// considered dead, so we clean up
func NewSubscriber(closed <-chan bool, bufferSize int, handleError func(error)) *Subscriber {
	return &Subscriber{
		C:            make(chan interface{}, bufferSize),
		closed:       closed,
		errorHandler: handleError,
	}
}

// Attempts to send a message on the subscribers channel. If this
// fails (e.g. queue is full), then the message is dropped
func (s *Subscriber) QueueToSend(v interface{}) error {
	select {
	case s.C <- v:
		return nil
	default:
		return fmt.Errorf("Did not deliver %v", v)
	}
}

// Like QueueToSend, but blocks until sent
func (s *Subscriber) BlockSend(v interface{}) {
	s.C <- v
}

// sends error to the client
func (s *Subscriber) SendError(e error) {
	s.errorHandler(e)
}

func (s *Subscriber) Close() {
	close(s.C)
}
