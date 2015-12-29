package archiver

import (
	"gopkg.in/mgo.v2/bson"
	"strings"
)

type parsedQuery struct {
	queryType queryType
	// all the keys contained in this query
	keys []string
	// list of tags to target for deletion or selection
	target []string
	//TODO: replace these with with something that's not bson.M
	// where clause for query
	where bson.M
	// key-value pairs to add
	set bson.M
	// are we querying distinct values?
	distinct bool
	// a unique representation of this query used to compare two different query objects
	hash queryHash
	// Track state transitions for the UUIDs that match this query
	matchedUUIDs map[UUID]UUIDSTATE
	// reference to data query
	//  type dataquery struct {
	//  	dtype		dataqueryType
	//  	start		_time.Time
	//  	end			_time.Time
	//  	limit		datalimit
	//      timeconv  UnitOfTime
	//  }
	data *dataquery
	// any error that arose during parsing
	err error
	// token where the error in parsing took place
	errPos      string
	querystring string
}

type queryProcessor struct {
	a *Archiver
}

func (qp *queryProcessor) Parse(querystring string) *parsedQuery {
	if !strings.HasSuffix(querystring, ";") {
		querystring = querystring + ";"
	}
	l := NewSQLex(querystring)
	log.Debug("Parsing query: %v\n", querystring)
	SQParse(l)

	pq := &parsedQuery{
		queryType: l.query.qtype,
		keys:      make([]string, len(l._keys)),
		target:    l.query.Contents,
		where:     l.query.WhereBson(),
		set:       l.query.SetBson(),
		distinct:  l.query.distinct,
		data:      l.query.data,
		err:       l.error,
		errPos:    l.lasttoken,
		//TODO: have a more robust hash function
		hash:         queryHash(querystring),
		matchedUUIDs: make(map[UUID]UUIDSTATE),
		querystring:  querystring,
	}
	i := 0
	for key, _ := range l._keys {
		pq.keys[i] = cleantagstring(key)
		i += 1
	}
	return pq
}
