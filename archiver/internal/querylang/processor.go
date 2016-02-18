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

func (qp *QueryProcessor) Parse(querystring string) *ParsedQuery {
	pq, _ := qp.parsedQueries.Fetch(querystring, 10*time.Minute, func() (interface{}, error) {
		if !strings.HasSuffix(querystring, ";") {
			querystring = querystring + ";"
		}
		l := NewSQLex(querystring)
		sqParse(l)
		pq := ParsedQuery{
			QueryType: l.query.qtype,
			Keys:      make([]string, len(l._keys)),
			Target:    l.query.Contents,
			Where:     l.query.WhereBson(),
			Set:       l.query.SetBson(),
			Distinct:  l.query.distinct,
			Data:      l.query.data,
			Err:       l.error,
			ErrPos:    l.lasttoken,
			//TODO: have a more robust hash function
			Hash:        QueryHash(querystring),
			Querystring: querystring,
		}
		i := 0
		for key, _ := range l._keys {
			pq.Keys[i] = cleantagstring(key)
			i += 1
		}
		return &pq, nil
	})
	return pq.Value().(*ParsedQuery)
}

type ParsedQuery struct {
	QueryType QueryType
	// all the keys contained in this query
	Keys []string
	// list of tags to target for deletion or selection
	Target []string
	//TODO: replace these with with something that's not bson.M
	// where clause for query
	Where bson.M
	// key-value pairs to add
	Set bson.M
	// are we querying distinct values?
	Distinct bool
	// a unique representation of this query used to compare two different query objects
	Hash QueryHash
	Data *DataQuery
	// any error that arose during parsing
	Err error
	// token where the error in parsing took place
	ErrPos      string
	Querystring string
}

type QueryType uint8

const (
	SELECT_TYPE QueryType = iota + 1
	DELETE_TYPE
	SET_TYPE
	DATA_TYPE
	APPLY_TYPE
)

type QueryHash string

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
