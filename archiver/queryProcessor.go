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
	errPos string
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
	}
	i := 0
	for key, _ := range l._keys {
		pq.keys[i] = cleantagstring(key)
		i += 1
	}
	return pq
}
