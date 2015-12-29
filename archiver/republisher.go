package archiver

import (
	"gopkg.in/mgo.v2/bson"
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
	queryConcern    map[queryHash][](*Subscriber)
	subscribersLock sync.RWMutex

	// key -> list of queries
	keyConcern     map[string][]queryHash
	keyConcernLock sync.RWMutex

	// uuid -> queries concerning uuid
	uuidConcern     map[UUID][]queryHash
	uuidConcernLock sync.RWMutex
}

func NewRepublisher(a *Archiver) (r *Republisher) {
	r = &Republisher{
		a:            a,
		clients:      [](*Subscriber){},
		queries:      make(map[queryHash]*parsedQuery),
		queryConcern: make(map[queryHash][](*Subscriber)),
		keyConcern:   make(map[string][]queryHash),
		uuidConcern:  make(map[UUID][]queryHash)}
	return
}

// passthrough to our MetadataStore.GetUUIDs
func (r *Republisher) evaluateWhereClause(where bson.M) ([]UUID, error) {
	return r.a.mdStore.GetUUIDs(where)
}

// When we receive a *Subscriber instance, we get access to its parsed query.
// When a subscriber instance is started, we evaluate the WHERE clause to get
// the set of streams that need to be forwarded, and also evaluate the SELECT
// clause so that the subscriber gets an immediate confirmation and some data to work with.
func (r *Republisher) handleSubscriber(subscriber *Subscriber) {
	var (
		q *parsedQuery
	)
	// first, evaluate the WHERE clause to get the set of UUIDs
	// TODO: fetch this from a cache if its already done
	q = subscriber.query
	initialUUIDs, fetchErr := r.evaluateWhereClause(q.where)
	if fetchErr != nil {
		subscriber.errorHandler(fetchErr)
		return // abort!
	}

	r.subscribersLock.Lock()
	if subscribers, found := r.queryConcern[q.hash]; found {
		subscribers = append(subscribers, subscriber)
		r.queryConcern[q.hash] = subscribers
	} else {
		r.queryConcern[q.hash] = [](*Subscriber){subscriber}
	}
	r.subscribersLock.Unlock()

	for _, uuid := range initialUUIDs {
		q.matchedUUIDs[uuid] = OLD
	}

	r.queries[q.hash] = q

	// for each matched UUID, store the query that matched it
	for uuid, _ := range q.matchedUUIDs {
		if list, found := r.uuidConcern[uuid]; found {
			list = append(list, q.hash)
			r.uuidConcern[uuid] = list
		} else {
			r.uuidConcern[uuid] = []queryHash{q.hash}
		}
	}

	// for each key in the query, store the query that mentions it
	for _, key := range q.keys {
		if queries, found := r.keyConcern[key]; found {
			queries = append(queries, q.hash)
			r.keyConcern[key] = queries
		} else {
			r.keyConcern[key] = []queryHash{q.hash}
		}
	}

	// evaluate the SELECT clause and deliver it
	result, evalErr := r.a.evaluateQuery(q, NewEphemeralKey())
	if evalErr != nil {
		subscriber.errorHandler(evalErr)
		return
	}

	subscriber.BlockSend(result)

	log.Debug("waiting...")
	<-subscriber.closed
	log.Debug("CLOSED!")
}

func (r *Republisher) Republish(msg *SmapMessage) {
	if queries, found := r.uuidConcern[msg.UUID]; found {
		log.Debug("found %v", queries)
		for _, hash := range queries {
			log.Debug("hash %v", hash)
			for _, client := range r.queryConcern[hash] {
				log.Debug("push to client %v", msg)
				client.QueueToSend(msg)
			}
		}
	}
}
