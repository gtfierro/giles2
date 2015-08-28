package archiver

import (
	"gopkg.in/mgo.v2/bson"
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
