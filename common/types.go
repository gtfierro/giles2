package common

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

// internal unique identifier
type UUID string

// generates a new random v4 UUID
func NewUUID() UUID {
	return UUID(uuid.NewV4().String())
}

type DistinctResult []string

func (dr DistinctResult) IsResult() {
}

// a flat map for storing key-value pairs
type Dict map[string]interface{}

func NewDict() *Dict {
	return new(Dict)
}

func DictFromBson(m bson.M) Dict {
	d := Dict{}
	for k, v := range m {
		key := FixMongoKey(k)
		d[key] = v
	}
	return d
}

func (d Dict) ToBson() bson.M {
	m := bson.M{}
	for k, v := range d {
		m[k] = v
	}
	return m
}

func (d Dict) MarshalJSON() ([]byte, error) {
	var m = make(map[string]interface{})
	if len(d) == 0 {
		return json.Marshal(m)
	}

	var ok bool = false
	for dk, dv := range d {
		pieces := strings.Split(dk, "|")
		plen := len(pieces)
		var cur = m
		for _, token := range pieces[:plen-1] {
			if _, found := cur[token]; !found {
				cur[token] = make(map[string]interface{})
			}
			if cur, ok = cur[token].(map[string]interface{}); !ok {
				return []byte{}, fmt.Errorf("Could not convert cur to map[string]interface{} was %v", cur[token])
			}
		}
		cur[pieces[plen-1]] = dv
	}
	return json.Marshal(m)
}

// Takes a timestamp with accompanying unit of time 'stream_uot' and
// converts it to the unit of time 'target_uot'
func convertTime(time uint64, stream_uot, target_uot UnitOfTime) (uint64, error) {
	var returnTime uint64
	if stream_uot == target_uot {
		return time, nil
	}
	if target_uot < stream_uot { // target/stream is > 1, so we can use uint64
		returnTime = time * (unitmultiplier[target_uot] / unitmultiplier[stream_uot])
		if returnTime < time {
			return time, TimeConvertErr
		}
	} else {
		returnTime = time / uint64(unitmultiplier[stream_uot]/unitmultiplier[target_uot])
		if returnTime > time {
			return time, TimeConvertErr
		}
	}
	return returnTime, nil
}

// stream type indicators
type StreamType uint

const (
	OBJECT_STREAM StreamType = iota + 1
	NUMERIC_STREAM
)

func (st StreamType) String() string {
	switch st {
	case OBJECT_STREAM:
		return "object"
	case NUMERIC_STREAM:
		return "numeric"
	default:
		return ""
	}
}

func (st StreamType) MarshalJSON() ([]byte, error) {
	switch st {
	case OBJECT_STREAM:
		return []byte(`"object"`), nil
	case NUMERIC_STREAM:
		return []byte(`"numeric"`), nil
	default:
		return []byte(`"numeric"`), nil
	}
}

func (st *StreamType) UnmarshalJSON(b []byte) (err error) {
	str := strings.Trim(string(b), `"`)
	switch str {
	case "numeric":
		*st = NUMERIC_STREAM
	case "object":
		*st = OBJECT_STREAM
	default:
		return fmt.Errorf("%v is not a valid StreamType", str)
	}
	return nil
}

type ApiKey string

type EphemeralKey [32]byte

func NewEphemeralKey() EphemeralKey {
	var key []byte
	var ekey EphemeralKey
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	copy(ekey[:], key)
	return ekey
}
