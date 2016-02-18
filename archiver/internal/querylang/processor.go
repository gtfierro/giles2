//go:generate go tool yacc -o query.go -p sq query.y
package querylang

import (
	"github.com/karlseguin/ccache"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
)

type QueryProcessor struct {
	parsedQueries *ccache.Cache
}

// TODO: add caching?
func NewQueryProcessor() *QueryProcessor {
	return &QueryProcessor{
		parsedQueries: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(10)),
	}
}

func (qp *QueryProcessor) Parse(querystring string) ParsedQuery {
	pq, _ := qp.parsedQueries.Fetch(querystring, 10*time.Minute, func() (interface{}, error) {
		if !strings.HasSuffix(querystring, ";") {
			querystring = querystring + ";"
		}
		l := NewSQLex(querystring)
		sqParse(l)
		pq := ParsedQuery{
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
			hash:        queryHash(querystring),
			querystring: querystring,
		}
		i := 0
		for key, _ := range l._keys {
			pq.keys[i] = cleantagstring(key)
			i += 1
		}
		return pq, nil
	})
	return pq.Value().(ParsedQuery)
}

type ParsedQuery struct {
	queryType QueryType
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
	data *dataquery
	// any error that arose during parsing
	err error
	// token where the error in parsing took place
	errPos      string
	querystring string
}

type QueryType uint8

const (
	SELECT_TYPE QueryType = iota + 1
	DELETE_TYPE
	SET_TYPE
	DATA_TYPE
	APPLY_TYPE
)

type queryHash string

func fixMongoKey(key string) string {
	switch {
	case strings.HasPrefix(key, "Metadata"):
		return "Metadata." + strings.Replace(key[9:], ".", "|", -1)
	case strings.HasPrefix(key, "Properties"):
		return "Properties." + strings.Replace(key[11:], ".", "|", -1)
	case strings.HasPrefix(key, "Actuator"):
		return "Actuator." + strings.Replace(key[9:], ".", "|", -1)
	}
	return key
}
