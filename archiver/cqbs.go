package archiver

import (
	"github.com/gtfierro/giles2/archiver/internal/querylang"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

type UUIDSTATE uint

const (
	OLD UUIDSTATE = iota
	NEW
	SAME
)

type Query struct {
	// query string
	Query string
	// list of keys in this query
	Keys []string
	// where clause in BSON
	WhereClause bson.M
	// uuids
	Streams map[UUID]UUIDSTATE
	// most recent evaluation of this query
	Initial QueryResult
	sync.RWMutex
}

func NewQuery(pq *querylang.ParsedQuery) *Query {
	return &Query{
		Query:       pq.Querystring,
		Keys:        pq.Keys,
		WhereClause: pq.Where,
		Streams:     make(map[UUID]UUIDSTATE),
	}
}

// updates internal list of qualified streams. Returns the lists of added and removed UUIDs
func (q *Query) changeUUIDs(uuids []UUID) (added, removed []UUID) {
	// mark the UUIDs that are new
	q.Lock()
	for _, uuid := range uuids {
		if _, found := q.Streams[uuid]; found {
			q.Streams[uuid] = SAME
		} else {
			q.Streams[uuid] = NEW
			added = append(added, uuid)
		}
	}

	// remove the old ones
	for uuid, status := range q.Streams {
		if status == OLD {
			removed = append(removed, uuid)
			delete(q.Streams, uuid)
		}
	}

	for uuid, _ := range q.Streams {
		q.Streams[uuid] = OLD
	}
	q.Unlock()
	return
}

type Broker struct {
	a *Archiver
	// map of query string -> query struct
	queries   map[string]*Query
	queryLock sync.RWMutex

	// uuid -> subscriber
	subscribers     map[UUID]*subscriberList
	subscribersLock sync.RWMutex
}

func NewBroker(a *Archiver) *Broker {
	return &Broker{
		a:           a,
		queries:     make(map[string]*Query),
		subscribers: make(map[UUID]*subscriberList),
	}
}

// Given a parsed query (output from query processor), return the
// broker's representation of it to use in broker calls. If this query
// does not exist yet, this adds it.
func (b *Broker) GetQuery(pq *querylang.ParsedQuery) (*Query, error) {
	var (
		q     *Query
		found bool
	)
	b.queryLock.RLock()
	if q, found = b.queries[pq.Querystring]; found {
		b.queryLock.RUnlock()
		return q, nil
	}
	b.queryLock.RUnlock()
	// if we couldn't find the query, then theres a chance
	// it hasn't been evaluated. So, we evaluate it to get
	// the initial UUIDs
	q = NewQuery(pq)
	uuids, err := b.a.mdStore.GetUUIDs(q.WhereClause)
	if err != nil {
		return q, err
	}
	q.changeUUIDs(uuids)

	// also get initial result for query and cache it
	result, evalErr := b.a.evaluateQuery(pq, NewEphemeralKey())
	if evalErr != nil {
		return q, nil
	}
	q.Initial = result

	// now check if someone else did this
	b.queryLock.Lock()
	if oldq, found := b.queries[pq.Querystring]; found {
		b.queryLock.Unlock()
		return oldq, nil
	}
	// if not, then we add the one we just did
	b.queries[pq.Querystring] = q
	b.queryLock.Unlock()
	return q, nil
}

// Updates the internal mapping of UUIDs a query has and adjusts
// the mapping of query to uuids w/n the broker
func (b *Broker) AddEvaluation(q *Query, uuids []UUID) {
	added, removed := q.changeUUIDs(uuids)
	if len(added) == 0 && len(removed) == 0 {
		log.Debugf("Query %v saw no changes", q)
	}
}

func (b *Broker) NewSubscriber(sub *Subscriber) error {
	query, err := b.GetQuery(sub.query)
	if err != nil {
		sub.errorHandler(err)
		return err
	}
	log.Debugf("NEW Subscriber %v", sub)
	for uuid, _ := range query.Streams {
		b.addSubscriberToStream(uuid, sub)
	}

	// send initial results of query
	log.Debugf("SEND INIT %v", query.Initial)
	sub.BlockSend(query.Initial)
	log.Debug("waiting for client to leave...")
	<-sub.closed
	b.removeSubscriber(sub)
	log.Debug("client left!")

	return err
}

func (b *Broker) addSubscriberToStream(uuid UUID, sub *Subscriber) {
	// check if uuid in stream map
	b.subscribersLock.Lock()
	if list, found := b.subscribers[uuid]; found {
		list.addSubscriber(sub)
	} else {
		list = new(subscriberList)
		list.addSubscriber(sub)
		b.subscribers[uuid] = list
	}
	b.subscribersLock.Unlock()
}

func (b *Broker) removeSubscriber(sub *Subscriber) {
	var (
		query *Query
		found bool
	)
	b.queryLock.RLock()
	if query, found = b.queries[sub.query.Querystring]; !found {
		b.queryLock.RUnlock()
		log.Criticalf("Removing subscriber with non existant query %v", sub.query.Querystring)
		return
	}
	b.queryLock.RUnlock()

	query.RLock()
	b.subscribersLock.Lock()
	for uuid, _ := range query.Streams {
		if list, found := b.subscribers[uuid]; found {
			list.removeSubscriber(sub)
		}
	}
	b.subscribersLock.Unlock()
	query.RUnlock()
}

// finds all clients subscribed to the uuid for this message
// can calls client.QueueToSend(msg) on them
func (b *Broker) ForwardMessage(msg *SmapMessage) {
	b.subscribersLock.RLock()
	if list, found := b.subscribers[msg.UUID]; found {
		b.subscribersLock.RUnlock()
		for _, sub := range *list {
			sub.QueueToSend(msg)
		}
	} else {
		b.subscribersLock.RUnlock()
	}
}

// TODO: more efficient implementation?
type subscriberList []*Subscriber

func (sl *subscriberList) addSubscriber(sub *Subscriber) {
	for _, oldSub := range *sl {
		if oldSub == sub {
			return
		}
	}

	*sl = append(*sl, sub)
}

func (sl *subscriberList) removeSubscriber(sub *Subscriber) {
	for i, oldSub := range *sl {
		if oldSub == sub {
			*sl = append((*sl)[:i], (*sl)[i+1:]...)
			return
		}
	}
}
