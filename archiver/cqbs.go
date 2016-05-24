package archiver

import (
	"github.com/gtfierro/giles2/archiver/internal/querylang"
	"github.com/gtfierro/giles2/common"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

type UUIDSTATE uint

const (
	OLD UUIDSTATE = iota
	NEW
	SAME
)

type QueryResult interface {
	IsResult()
}

type Query struct {
	// query string
	Query string
	// list of keys in this query
	Keys []string
	// where clause in BSON
	WhereClause bson.M
	// uuids
	Streams     map[common.UUID]UUIDSTATE
	subscribers *subscriberList
	// most recent evaluation of this query
	Initial QueryResult
	sync.RWMutex
}

func NewQuery(pq *querylang.ParsedQuery) *Query {
	return &Query{
		Query:       pq.Querystring,
		Keys:        pq.Keys,
		WhereClause: pq.Where,
		Streams:     make(map[common.UUID]UUIDSTATE),
		subscribers: new(subscriberList),
	}
}

// updates internal list of qualified streams. Returns the lists of added and removed UUIDs
func (q *Query) changeUUIDs(uuids []common.UUID) (added, removed []common.UUID) {
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
	subscribers     map[common.UUID]*subscriberList
	subscribersLock sync.RWMutex

	// key -> list of queries
	keys     map[string]*queryList
	keysLock sync.RWMutex
}

func NewBroker(a *Archiver) *Broker {
	return &Broker{
		a:           a,
		queries:     make(map[string]*Query),
		subscribers: make(map[common.UUID]*subscriberList),
		keys:        make(map[string]*queryList),
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
	result, evalErr := b.a.evaluateQuery(pq, common.NewEphemeralKey())
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

	// add key pointers too
	b.keysLock.Lock()
	var (
		list *queryList
	)
	for _, key := range q.Keys {
		if list, found = b.keys[key]; !found {
			list = new(queryList)
		}
		list.addQuery(q)
		b.keys[key] = list
	}
	b.keysLock.Unlock()
	b.queryLock.Unlock()

	return q, nil
}

//TODO: reevaluate query.Initial
// first adjust all subscriptions based on metadata in this message,
// then forward it out to all subscribed clients
func (b *Broker) HandleMessage(msg *common.SmapMessage) {
	var toReevaluate = make(map[*Query]bool)
	b.keysLock.RLock()
	if msg.Metadata != nil {
		for key, _ := range msg.Metadata {
			if queries, found := b.keys["Metadata."+key]; found {
				for _, query := range *queries {
					toReevaluate[query] = true
				}
			}
		}
	}
	if msg.Actuator != nil {
		for key, _ := range msg.Actuator {
			if queries, found := b.keys["Actuator."+key]; found {
				for _, query := range *queries {
					toReevaluate[query] = true
				}
			}
		}
	}
	if msg.Properties != nil {
		if queries, found := b.keys["Properties.UnitofMeasure"]; found {
			for _, query := range *queries {
				toReevaluate[query] = true
			}
		}
		if queries, found := b.keys["Properties.UnitofTime"]; found {
			for _, query := range *queries {
				toReevaluate[query] = true
			}
		}
		if queries, found := b.keys["Properties.StreamType"]; found {
			for _, query := range *queries {
				toReevaluate[query] = true
			}
		}
	}
	if queries, found := b.keys["uuid"]; found {
		for _, query := range *queries {
			toReevaluate[query] = true
		}
	}
	if queries, found := b.keys["Path"]; found {
		for _, query := range *queries {
			toReevaluate[query] = true
		}
	}
	b.keysLock.RUnlock()
	for query, _ := range toReevaluate {
		b.reevaluateQuery(query)
	}
	b.ForwardMessage(msg)
}

// What does it take to do the reevaluation correctly?
// Information we are given: which queries could be affected. For each query, we have the list of previously matching UUIDs.
// When we reevaluate the query, we get a list of REMOVED uuids and ADDED uuids
// For each of the REMOVED uuids, we go through the list of clients for that uuid. If their query is equal to the removed query, then
// we delete the client from the list of UUIDs
func (b *Broker) reevaluateQuery(q *Query) {
	var (
		list  *subscriberList
		found bool
	)
	log.Debugf("reevalute %v", q)
	uuids, err := b.a.mdStore.GetUUIDs(q.WhereClause)
	if err != nil {
		log.Criticalf("Error fetching UUIDs for (%v) from metadata store (%v)", q.WhereClause, err)
		return
	}
	// added, removed
	added, removed := q.changeUUIDs(uuids)
	if len(removed) > 0 {
		b.subscribersLock.Lock()
		for _, rm_uuid := range removed {
			if list, found = b.subscribers[rm_uuid]; !found {
				// no subscribers for this uuid
				continue
			}
			// if there is a list of subscribers, iterate through and see if they are subscribed to *this* query
			for _, client := range *list {
				if client.query.Querystring == q.Query { // remove!
					list.removeSubscriber(client)
					continue
				}
			}
			b.subscribers[rm_uuid] = list
		}
		b.subscribersLock.Unlock()
	}

	if len(added) > 0 {
		for _, add_uuid := range added {
			for _, sub := range *q.subscribers {
				b.addSubscriberToStream(add_uuid, sub)
			}
		}
	}

}

func (b *Broker) NewSubscriber(sub *Subscriber) error {
	query, err := b.GetQuery(sub.query)
	if err != nil {
		sub.errorHandler(err)
		return err
	}
	log.Debugf("NEW Subscriber %v with query %v", sub, sub.query)
	query.subscribers.addSubscriber(sub)
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

func (b *Broker) addSubscriberToStream(uuid common.UUID, sub *Subscriber) {
	var (
		list  *subscriberList
		found bool
	)
	// check if uuid in stream map
	b.subscribersLock.Lock()
	if list, found = b.subscribers[uuid]; !found {
		list = new(subscriberList)
	}
	list.addSubscriber(sub)
	b.subscribers[uuid] = list
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
	query.Lock()
	query.subscribers.removeSubscriber(sub)
	if len(*query.subscribers) == 0 {
		// remove ourselves from key references
		b.keysLock.Lock()
		for _, key := range query.Keys {
			b.keys[key].removeQuery(query)
		}
		b.keysLock.Unlock()
	}
	query.Unlock()
}

// finds all clients subscribed to the uuid for this message
// can calls client.QueueToSend(msg) on them
func (b *Broker) ForwardMessage(msg *common.SmapMessage) {
	b.subscribersLock.RLock()
	if list, found := b.subscribers[msg.UUID]; found {
		b.subscribersLock.RUnlock()
		if len(*list) == 0 {
			return
		}
		log.Debugf("Found list of subscribers for msg %v (%v)", msg, list)
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

type queryList []*Query

func (ql *queryList) addQuery(q *Query) {
	for _, oldSub := range *ql {
		if oldSub == q {
			return
		}
	}

	*ql = append(*ql, q)
}

func (ql *queryList) removeQuery(q *Query) {
	for i, oldSub := range *ql {
		if oldSub == q {
			*ql = append((*ql)[:i], (*ql)[i+1:]...)
			return
		}
	}
}
