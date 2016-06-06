package common

import (
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
)

// Takes a dictionary that contains nested dictionaries and
// transforms it to a 1-level map with fields separated by periods k.kk.kkk = v
func flatten(m bson.M) bson.M {
	var ret = make(bson.M)
	for k, v := range m {
		if vb, ok := v.(map[string]interface{}); ok {
			for kk, vv := range flatten(vb) {
				ret[k+"."+kk] = vv
			}
		} else {
			ret[k] = v
		}
	}
	return ret
}

func fixKey(k string) string {
	return strings.Replace(k, ".", "|", -1)
}
func FixMongoKey(key string) string {
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

func compareStringSliceAsSet(s1, s2 []string) bool {
	var (
		found bool
	)

	if len(s1) != len(s2) {
		return false
	}

	for _, val1 := range s1 {
		found = false
		for _, val2 := range s2 {
			if val1 == val2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Given a forward-slash delimited path, returns a slice of prefixes, e.g.:
// input: /a/b/c/d
// output: ['/', '/a','/a/b','/a/b/c']
func getPrefixes(s string) []string {
	ret := []string{"/"}
	root := ""
	s = "/" + s
	for _, prefix := range strings.Split(s, "/") {
		if len(prefix) > 0 { //skip empty strings created by Split
			root += "/" + prefix
			ret = append(ret, root)
		}
	}
	if len(ret) > 1 {
		return ret[:len(ret)-1]
	}
	return ret
}

// returns true if the two slices are equal
func isStringSliceEqual(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	for idx := range x {
		if x[idx] != y[idx] {
			return false
		}
	}
	return true
}

// get current time
func GetNow(units UnitOfTime) uint64 {
	now, err := convertTime(uint64(time.Now().UnixNano()), UOT_NS, units)
	if err != nil {
		panic(err) // "This should never ever happen" - Gabe Fierro, 8 February 2016
	}
	return now
}

const (
	S_LOW  uint64 = 2 << 30
	MS_LOW uint64 = 2 << 39
	US_LOW uint64 = 2 << 50
	NS_LOW uint64 = 2 << 58
)

func GuessTimeUnit(val uint64) UnitOfTime {
	if val < MS_LOW {
		return UOT_S
	} else if val < US_LOW {
		return UOT_MS
	} else if val < NS_LOW {
		return UOT_US
	}
	return UOT_NS
}
