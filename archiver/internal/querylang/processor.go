//go:generate go tool yacc -o query.go -p sq query.y
package querylang

import (
	"github.com/gtfierro/giles2/common"
	"github.com/karlseguin/ccache"
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
			Where:     l.query.where,
			Set:       l.query.set,
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
	Where common.Dict
	// key-value pairs to add
	Set common.Dict
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

func (parsed *ParsedQuery) GetParams() common.QueryParams {
	switch parsed.QueryType {
	case SELECT_TYPE:
		if parsed.Distinct {
			return &common.DistinctParams{
				Tag:   parsed.Target[0],
				Where: parsed.Where,
			}
		}
		return &common.TagParams{
			Tags:  parsed.Target,
			Where: parsed.Where,
		}
	case DELETE_TYPE:
		if parsed.Data == nil {
			return &common.TagParams{
				Tags:  parsed.Target,
				Where: parsed.Where,
			}
		} else {
			return &common.DataParams{
				Where:         parsed.Where,
				Begin:         uint64(parsed.Data.Start.UnixNano()),
				End:           uint64(parsed.Data.End.UnixNano()),
				IsStatistical: false,
				IsWindow:      false,
			}
		}
	case SET_TYPE:
		return &common.SetParams{
			Set:   parsed.Set,
			Where: parsed.Where,
		}
	case DATA_TYPE:
		return &common.DataParams{
			Where:         parsed.Where,
			StreamLimit:   int(parsed.Data.Limit.Streamlimit),
			DataLimit:     int(parsed.Data.Limit.Limit),
			Begin:         uint64(parsed.Data.Start.UnixNano()),
			End:           uint64(parsed.Data.End.UnixNano()),
			ConvertToUnit: parsed.Data.Timeconv,
			IsStatistical: parsed.Data.IsStatistical,
			IsWindow:      parsed.Data.IsWindow,
			Width:         parsed.Data.Width,
			PointWidth:    int(parsed.Data.PointWidth),
		}
	default:
		return nil
	}
	return nil
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
