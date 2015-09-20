package archiver

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
)

// internal unique identifier
type UUID string

// generates a new random v4 UUID
func NewUUID() UUID {
	return UUID(uuid.NewV4().String())
}

// a flat map for storing key-value pairs
type Dict map[string]string

func NewDict() *Dict {
	return new(Dict)
}

func DictFromBson(m bson.M) Dict {
	d := Dict{}
	for k, v := range m {
		key := fixMongoKey(k)
		if vs, ok := v.(string); ok {
			d[key] = vs
		} else if vs, ok := v.(int64); ok {
			d[key] = strconv.FormatInt(vs, 10)
		} else if vs, ok := v.(float64); ok {
			d[key] = strconv.FormatFloat(vs, 'f', -1, 64)
		}
	}
	return d
}

func (d Dict) MarshalJSON() ([]byte, error) {
	var m = make(map[string]interface{})
	if len(d) == 0 {
		return json.Marshal(m)
	}

	for dk, dv := range d {
		pieces := strings.Split(dk, "|")
		plen := len(pieces)
		var cur = m
		for _, token := range pieces[:plen-1] {
			if _, found := cur[token]; !found {
				cur[token] = make(map[string]interface{})
			}
			cur = cur[token].(map[string]interface{})
		}
		cur[pieces[plen-1]] = dv
	}
	return json.Marshal(m)
}

// unit of time indicators
type UnitOfTime uint

const (
	// nanoseconds 1000000000
	UOT_NS UnitOfTime = 1
	// microseconds 1000000
	UOT_US UnitOfTime = 2
	// milliseconds 1000
	UOT_MS UnitOfTime = 3
	// seconds 1
	UOT_S UnitOfTime = 4
)

var unitmultiplier = map[UnitOfTime]uint64{
	UOT_NS: 1000000000,
	UOT_US: 1000000,
	UOT_MS: 1000,
	UOT_S:  1,
}

// Takes a timestamp with accompanying unit of time 'stream_uot' and
// converts it to the unit of time 'target_uot'
func convertTime(time uint64, stream_uot, target_uot UnitOfTime) uint64 {
	if stream_uot == target_uot {
		return time
	}
	if target_uot < stream_uot { // target/stream is > 1, so we can use uint64
		return time * (unitmultiplier[target_uot] / unitmultiplier[stream_uot])
	} else {
		return time / uint64(unitmultiplier[stream_uot]/unitmultiplier[target_uot])
	}
}

func (u UnitOfTime) String() string {
	switch u {
	case UOT_NS:
		return "ns"
	case UOT_US:
		return "us"
	case UOT_MS:
		return "ms"
	case UOT_S:
		return "s"
	default:
		return ""
	}
}

func (u UnitOfTime) MarshalJSON() ([]byte, error) {
	switch u {
	case UOT_NS:
		return []byte(`"ns"`), nil
	case UOT_US:
		return []byte(`"us"`), nil
	case UOT_MS:
		return []byte(`"ms"`), nil
	case UOT_S:
		return []byte(`"s"`), nil
	default:
		return []byte(`"s"`), nil
	}
}

func (u *UnitOfTime) UnmarshalJSON(b []byte) (err error) {
	str := strings.Trim(string(b), `"`)
	switch str {
	case "ns":
		*u = UOT_NS
	case "us":
		*u = UOT_US
	case "ms":
		*u = UOT_MS
	case "s":
		*u = UOT_S
	default:
		return fmt.Errorf("%v is not a valid UnitOfTime", str)
	}
	return nil
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

type queryType uint8

const (
	SELECT_TYPE queryType = iota + 1
	DELETE_TYPE
	SET_TYPE
	DATA_TYPE
	APPLY_TYPE
)
