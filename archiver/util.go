package archiver

import (
	"gopkg.in/mgo.v2/bson"
	"strings"
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
