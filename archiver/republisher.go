package archiver

import (
	"sync"
)

type queryHash string

type UUIDSTATE uint

const (
	OLD UUIDSTATE = iota
	NEW
	SAME
	DEL
)

// This is a more thought-out version of the republisher that was first
// included in Giles.  The focus of this version of the republisher is SPEED:
// efficient discovery of who to deliver a new message to, and efficient
// reevaluation of queries in the face of new commands + data
type Republisher struct {
	sync.RWMutex

	// Pointer to archiver
	a *Archiver

	// list of all republish clients (unique)
	clients [](*Subscriber)

	// stores hash -> query object
	queries     map[queryHash]*parsedQuery
	queriesLock sync.RWMutex

	// query -> list of clients
	queryConcern map[queryHash][](*Subscriber)

	// key -> list of queries
	keyConcern     map[string][]queryHash
	keyConcernLock sync.RWMutex

	// uuid -> queries concerning uuid
	uuidConcern     map[string][]queryHash
	uuidConcernLock sync.RWMutex
}

func NewRepublisher(a *Archiver) (r *Republisher) {
	r = &Republisher{
		a:            a,
		clients:      [](*Subscriber){},
		queries:      make(map[queryHash]*parsedQuery),
		queryConcern: make(map[queryHash][](*Subscriber)),
		keyConcern:   make(map[string][]queryHash),
		uuidConcern:  make(map[string][]queryHash)}
	return
}

func (r *Republisher) handleSubscriber(subscriber *Subscriber) {
}
