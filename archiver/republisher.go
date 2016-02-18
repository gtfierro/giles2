package archiver

import (
	"github.com/gtfierro/giles2/archiver/internal/querylang"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

type queryHash querylang.QueryHash

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
	queries     map[queryHash]*querylang.ParsedQuery
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
		queries:      make(map[queryHash]*querylang.ParsedQuery),
		queryConcern: make(map[queryHash][](*Subscriber)),
		keyConcern:   make(map[string][]queryHash),
		uuidConcern:  make(map[UUID][]queryHash)}
	return
}

// passthrough to our MetadataStore.GetUUIDs
func (r *Republisher) evaluateWhereClause(where bson.M) ([]UUID, error) {
	return r.a.mdStore.GetUUIDs(where)
}

// Adds subscriber to queryConcern using subscriber.query.hash
func (r *Republisher) addSubscriber(sub *Subscriber) {
	var q = sub.query
	r.subscribersLock.Lock()
	if subscribers, found := r.queryConcern[queryHash(q.Hash)]; found {
		subscribers = append(subscribers, sub)
		r.queryConcern[queryHash(q.Hash)] = subscribers
	} else {
		r.queryConcern[queryHash(q.Hash)] = [](*Subscriber){sub}
	}
	r.subscribersLock.Unlock()
}

// When we receive a *Subscriber instance, we get access to its parsed query.
// When a subscriber instance is started, we evaluate the WHERE clause to get
// the set of streams that need to be forwarded, and also evaluate the SELECT
// clause so that the subscriber gets an immediate confirmation and some data to work with.
func (r *Republisher) handleSubscriber(subscriber *Subscriber) error {
	//var (
	//	q *querylang.ParsedQuery
	//)
	// first, evaluate the WHERE clause to get the set of UUIDs
	// TODO: fetch this from a cache if its already done
	//q = subscriber.query
	//initialUUIDs, fetchErr := r.evaluateWhereClause(q.where)
	//if fetchErr != nil {
	//	subscriber.errorHandler(fetchErr)
	//	return fetchErr // abort!
	//}
	//log.Debugf("got subscriber %v", subscriber)

	//r.addSubscriber(subscriber)

	//for _, uuid := range initialUUIDs {
	//	q.matchedUUIDs[uuid] = OLD
	//}

	//r.queries[q.hash] = q

	//// for each matched UUID, store the query that matched it
	//r.uuidConcernLock.Lock()
	//for uuid, _ := range q.matchedUUIDs {
	//	if list, found := r.uuidConcern[uuid]; found {
	//		list = append(list, q.hash)
	//		r.uuidConcern[uuid] = list
	//	} else {
	//		r.uuidConcern[uuid] = []queryHash{q.hash}
	//	}
	//}
	//r.uuidConcernLock.Unlock()

	//// for each key in the query, store the query that mentions it
	//for _, key := range q.keys {
	//	if queries, found := r.keyConcern[key]; found {
	//		queries = append(queries, q.hash)
	//		r.keyConcern[key] = queries
	//	} else {
	//		r.keyConcern[key] = []queryHash{q.hash}
	//	}
	//}

	//// evaluate the SELECT clause and deliver it
	//result, evalErr := r.a.evaluateQuery(q, NewEphemeralKey())
	//if evalErr != nil {
	//	subscriber.errorHandler(evalErr)
	//	return evalErr
	//}

	//log.Debugf("SEND INIT %v", result)
	//subscriber.BlockSend(result)

	//log.Debug("waiting...")
	//<-subscriber.closed
	//r.subscribersLock.Lock()
	//hash := subscriber.query.Hash
	//clients := r.queryConcern[hash]
	//for i, pubclient := range clients {
	//	if subscriber == pubclient {
	//		log.Debugf("removing subscriber for %v", hash)
	//		subscriber.Close()
	//		clients = append(clients[:i], clients[i+1:]...)
	//		r.queryConcern[hash] = clients
	//		break
	//	}
	//}
	//r.subscribersLock.Unlock()
	//log.Debug("CLOSED!")
	return nil
}

func (r *Republisher) Republish(msg *SmapMessage) {
	r.uuidConcernLock.RLock()
	log.Debug("found queries %v", r.uuidConcern[msg.UUID])
	if queries, found := r.uuidConcern[msg.UUID]; found {
		r.uuidConcernLock.RUnlock()
		for _, hash := range queries {
			//log.Debug("hash %v", hash)
			r.subscribersLock.RLock()
			log.Debug("clients: %v", r.queryConcern[hash])
			for _, client := range r.queryConcern[hash] {
				log.Debug("push to client %v", client)
				client.QueueToSend(msg)
			}
			r.subscribersLock.RUnlock()
		}
	} else {
		r.uuidConcernLock.RUnlock()
	}
}

func (r *Republisher) TriggerChangesMessage(msg *SmapMessage) error {
	var toReevaluate = make(map[queryHash]bool)
	// first check if contents of this message match any of the current subscriptions
	r.keyConcernLock.RLock()
	if msg.Metadata != nil {
		for key, _ := range msg.Metadata {
			for _, query := range r.keyConcern["Metadata."+key] {
				toReevaluate[query] = true
			}
		}
	}
	if msg.Actuator != nil {
		for key, _ := range msg.Actuator {
			for _, query := range r.keyConcern["Actuator."+key] {
				toReevaluate[query] = true
			}
		}
	}
	if msg.Properties != nil {
		for _, query := range r.keyConcern["Properties.UnitofMeasure"] {
			toReevaluate[query] = true
		}
		for _, query := range r.keyConcern["Properties.UnitofTime"] {
			toReevaluate[query] = true
		}
		for _, query := range r.keyConcern["Properties.StreamType"] {
			toReevaluate[query] = true
		}
	}
	for _, query := range r.keyConcern["uuid"] {
		toReevaluate[query] = true
	}
	for _, query := range r.keyConcern["Path"] {
		toReevaluate[query] = true
	}

	r.keyConcernLock.RUnlock()
	for qh, _ := range toReevaluate {
		r.ReevaluateQuery(qh)
	}
	return nil
}

//TODO: deliveries that could be affected by this should be paused until this finishes
func (r *Republisher) ReevaluateQuery(qh queryHash) error {
	//var (
	//	q     *parsedQuery
	//	err   error
	//	found bool
	//)
	//if q, found = r.queries[qh]; !found {
	//	log.Error("Could not find query hash %v", qh)
	//	return fmt.Errorf("Republisher asked to re-evaluate query with hash \"%v\" but it does not exist", qh)
	//}

	//newUUIDs, err := r.evaluateWhereClause(q.where)
	//if err != nil {
	//	log.Error("Received error when evaluating where clause %v (%v)", q.where, err)
	//	return err
	//}

	//for _, uuid := range newUUIDs {
	//	if _, found := q.matchedUUIDs[uuid]; found {
	//		q.matchedUUIDs[uuid] = SAME
	//	} else {
	//		q.matchedUUIDs[uuid] = NEW
	//	}
	//}

	//// store the query by its hash
	//r.queries[q.hash] = q

	//// remove old UUIDs
	//for uuid, status := range q.matchedUUIDs {
	//	// uuids that used to match should be marked as SAME
	//	// if this uuid is marked as OLD, then we remove the current query
	//	// from the list of queries related to the UUID
	//	if status == OLD {
	//		r.uuidConcernLock.RLock()
	//		concerned := r.uuidConcern[uuid]
	//		r.uuidConcernLock.RUnlock()
	//		for i, chash := range concerned {
	//			if chash == q.hash {
	//				concerned = append(concerned[:i], concerned[i+1:]...)
	//				break
	//			}
	//		}
	//		r.uuidConcernLock.Lock()
	//		r.uuidConcern[uuid] = concerned
	//		r.uuidConcernLock.Unlock()
	//		q.matchedUUIDs[uuid] = DEL
	//	} else if status == NEW {
	//		r.uuidConcernLock.Lock()
	//		r.uuidConcern[uuid] = append(r.uuidConcern[uuid], q.hash)
	//		r.uuidConcernLock.Unlock()
	//		q.matchedUUIDs[uuid] = OLD
	//	} else {
	//		q.matchedUUIDs[uuid] = OLD
	//	}
	//}

	//// now delete the DEL ones
	//for uuid, status := range q.matchedUUIDs {
	//	if status == DEL {
	//		delete(q.matchedUUIDs, uuid)
	//	}
	//}

	//return err
	return nil
}
