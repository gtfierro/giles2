package archiver

import (
	"encoding/json"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"strings"
)

type QueryResult interface {
	IsResult()
}

type SmapProperties struct {
	UnitOfTime    UnitOfTime
	UnitOfMeasure string
	StreamType    StreamType
}

func (sp SmapProperties) MarshalJSON() ([]byte, error) {
	// watch capitals
	var (
		m     map[string]string
		empty bool = true
	)
	if sp.UnitOfTime != 0 {
		empty = false
		if len(m) == 0 {
			m = make(map[string]string)
		}
		m["UnitofTime"] = sp.UnitOfTime.String()
	}
	if sp.StreamType != 0 {
		empty = false
		if len(m) == 0 {
			m = make(map[string]string)
		}
		m["StreamType"] = sp.StreamType.String()
	}
	if len(sp.UnitOfMeasure) != 0 {
		empty = false
		if len(m) == 0 {
			m = make(map[string]string)
		}
		m["UnitofMeasure"] = sp.UnitOfMeasure
	}
	if !empty {
		return json.Marshal(m)
	} else {
		return json.Marshal(nil)
	}
}

func (sp SmapProperties) IsEmpty() bool {
	return sp.UnitOfTime == 0 &&
		sp.UnitOfMeasure == "" &&
		sp.StreamType == 0
}

type SmapMessage struct {
	Path       string          `json:",omitempty" msgpack:",omitempty"`
	UUID       UUID            `json:"uuid,omitempty" msgpack:",omitempty"`
	Properties *SmapProperties `json:",omitempty" msgpack:",omitempty"`
	Actuator   Dict            `json:",omitempty" msgpack:",omitempty"`
	Metadata   Dict            `json:",omitempty" msgpack:",omitempty"`
	Readings   []Reading       `json:",omitempty" msgpack:",omitempty"`
}

// will insert a key string e.g. "Metadata.KeyName" and value e.g. "Value"
// into a SmapMessage instance, creating one if necessary
func (msg *SmapMessage) AddTag(key string, value string) {
	if msg == nil {
		msg = &SmapMessage{}
	}
	switch {
	case strings.HasPrefix(key, "Metadata."):
		if len(msg.Metadata) == 0 {
			msg.Metadata = Dict{}
		}
		// len("Metadata.") == 9
		msg.Metadata[key[9:]] = value
	case strings.HasPrefix(key, "Actuator."):
		if len(msg.Actuator) == 0 {
			msg.Actuator = Dict{}
		}
		// len("Actuator.") == 9
		msg.Actuator[key[9:]] = value
	case key == "Path":
		msg.Path = value
	case key == "UUID":
		msg.UUID = UUID(value)
		//case strings.HasPrefix(key, "Properties."):
		//    if msg.Properties == nil {
		//        msg.Properties = &SmapProperties{}
		//    }
		//    switch key {
		//    case "Properties.UnitofTime":
		//        msg.Properties.UnitOfTime = value
		//    }
		//    msg.Actuator[key[11:]] = value
	}
}

// returns this struct as BSON for storing the metadata. We ignore Readings
// because they are not part of the metadata store
//TODO: explore putting this in the mongo-specific file? This isn't general purpose
func (msg *SmapMessage) ToBson() (ret bson.M) {
	ret = bson.M{
		"uuid": msg.UUID,
		"Path": msg.Path,
	}
	if msg.Metadata != nil && len(msg.Metadata) > 0 {
		for k, v := range msg.Metadata {
			ret["Metadata."+fixKey(k)] = v
		}
	}
	if msg.Actuator != nil && len(msg.Actuator) > 0 {
		for k, v := range msg.Actuator {
			ret["Actuator."+fixKey(k)] = v
		}
	}
	if msg.Properties != nil && !msg.Properties.IsEmpty() {
		ret["Properties.UnitofTime"] = msg.Properties.UnitOfTime
		ret["Properties.UnitofMeasure"] = msg.Properties.UnitOfMeasure
		ret["Properties.StreamType"] = msg.Properties.StreamType
	}
	return ret
}

func (sm *SmapMessage) UnmarshalJSON(b []byte) (err error) {
	var (
		incoming   = new(incomingSmapMessage)
		time       uint64
		time_weird float64
		value_num  float64
		value_obj  interface{}
	)

	// unmarshal to an intermediary struct that matches the format
	// of the incoming messages
	err = json.Unmarshal(b, incoming)
	if err != nil {
		return
	}

	// copy the values over that we don't need to translate
	sm.UUID = incoming.UUID
	sm.Path = incoming.Path
	if len(incoming.Metadata) > 0 {
		sm.Metadata = DictFromBson(flatten(incoming.Metadata))
	}
	if !incoming.Properties.IsEmpty() {
		sm.Properties = &incoming.Properties
	}
	if len(incoming.Actuator) > 0 {
		sm.Actuator = DictFromBson(flatten(incoming.Actuator))
	}

	// convert the readings depending if they are numeric or object
	sm.Readings = make([]Reading, len(incoming.Readings))
	idx := 0
	for _, reading := range incoming.Readings {
		if len(reading) == 0 {
			continue
		}
		// time should be a uint64 no matter what
		err = json.Unmarshal(reading[0], &time)
		if err != nil {
			err = json.Unmarshal(reading[0], &time_weird)
			if err != nil {
				return
			} else {
				time = uint64(time_weird)
			}
		}

		// check if we have a numerical value
		err = json.Unmarshal(reading[1], &value_num)
		if err != nil {
			// if we don't, then we treat as an object reading
			err = json.Unmarshal(reading[1], &value_obj)
			sm.Readings[idx] = &SmapObjectReading{time, value_obj}
		} else {
			sm.Readings[idx] = &SmapNumberReading{time, value_num}
		}
		idx += 1
	}
	sm.Readings = sm.Readings[:idx]
	return
}

func SmapMessageFromBson(m bson.M) *SmapMessage {
	ret := &SmapMessage{}
	if uuid, found := m["uuid"]; found {
		ret.UUID = UUID(uuid.(string))
	}

	if path, found := m["Path"]; found {
		ret.Path = path.(string)
	}

	if md, found := m["Metadata"]; found {
		ret.Metadata = DictFromBson(md.(bson.M))
	}

	if md, found := m["Actuator"]; found {
		ret.Actuator = DictFromBson(md.(bson.M))
	}

	if md, found := m["Properties"]; found {
		if props, ok := md.(bson.M); ok {
			ret.Properties = &SmapProperties{}
			if uot, fnd := props["UnitofTime"]; fnd {
				if ret.Properties.UnitOfTime, ok = uot.(UnitOfTime); !ok {
					ret.Properties.UnitOfTime = UOT_MS
				}
			}
			if uom, fnd := props["UnitofMeasure"]; fnd {
				ret.Properties.UnitOfMeasure = uom.(string)
			}
			if st, fnd := props["StreamType"]; fnd {
				if ret.Properties.StreamType, ok = st.(StreamType); !ok {
					ret.Properties.StreamType = NUMERIC_STREAM
				}
			}
		}
	}

	return ret
}

// returns True if the message contains anything beyond Path, UUID, Readings
func (msg *SmapMessage) HasMetadata() bool {
	return (msg.Actuator != nil && len(msg.Actuator) > 0) ||
		(msg.Metadata != nil && len(msg.Metadata) > 0) ||
		(msg.Properties != nil && !msg.Properties.IsEmpty())
}

func (msg *SmapMessage) IsTimeseries() bool {
	return msg.UUID != ""
}

func (msg SmapMessage) IsResult() {}

type SmapMessageList []*SmapMessage

func (sml *SmapMessageList) ToBson() []bson.M {
	ret := make([]bson.M, len(*sml))
	for idx, msg := range *sml {
		ret[idx] = msg.ToBson()
	}
	return ret
}

func SmapMessageListFromBson(m []bson.M) SmapMessageList {
	ret := make(SmapMessageList, len(m))
	for idx, doc := range m {
		ret[idx] = SmapMessageFromBson(doc)
	}
	return ret
}

func (sml SmapMessageList) IsResult() {}

type TieredSmapMessage map[string]*SmapMessage

// This performs the metadata inheritance for the paths and messages inside
// this collection of SmapMessages. Inheritance starts from the root path "/"
// can progresses towards the leaves.
// First, get a list of all of the potential timeseries (any path that contains a UUID)
// Then, for each of the prefixes for the path of that timeserie (util.getPrefixes), grab
// the paths from the TieredSmapMessage that match the prefixes. Sort these in "decreasing" order
// and apply to the metadata.
// Finally, delete all non-timeseries paths
func (tsm *TieredSmapMessage) CollapseToTimeseries() {
	var (
		prefixMsg *SmapMessage
		found     bool
	)
	for path, msg := range *tsm {
		if !msg.IsTimeseries() {
			continue
		}
		prefixes := getPrefixes(path)
		sort.Sort(sort.Reverse(sort.StringSlice(prefixes)))
		for _, prefix := range prefixes {
			// if we don't find the prefix OR it exists but doesn't have metadata, we skip
			prefixMsg, found = (*tsm)[prefix]
			if !found || prefixMsg == nil || (prefixMsg != nil && !prefixMsg.HasMetadata()) {
				continue
			}
			// otherwise, we apply keys from paths higher up if our timeseries doesn't already have the key
			// (this is reverse inheritance)
			if prefixMsg.Metadata != nil && len(prefixMsg.Metadata) > 0 {
				for k, v := range prefixMsg.Metadata {
					if _, hasKey := msg.Metadata[k]; !hasKey {
						if msg.Metadata == nil {
							msg.Metadata = make(Dict)
						}
						msg.Metadata[k] = v
					}
				}
			}
			if prefixMsg.Properties != nil && !prefixMsg.Properties.IsEmpty() {
				if msg.Properties.UnitOfTime != 0 {
					msg.Properties.UnitOfTime = prefixMsg.Properties.UnitOfTime
				}
				if msg.Properties.UnitOfMeasure != "" {
					msg.Properties.UnitOfMeasure = prefixMsg.Properties.UnitOfMeasure
				}
				if msg.Properties.StreamType != 0 {
					msg.Properties.StreamType = prefixMsg.Properties.StreamType
				}
			}

			if prefixMsg.Actuator != nil && len(prefixMsg.Actuator) > 0 {
				for k, v := range prefixMsg.Actuator {
					if _, hasKey := msg.Actuator[k]; !hasKey {
						if msg.Actuator == nil {
							msg.Actuator = make(Dict)
						}
						msg.Actuator[k] = v
					}
				}
			}
			(*tsm)[path] = msg
		}
	}
	// when done, delete all non timeseries paths
	for path, msg := range *tsm {
		if !msg.IsTimeseries() {
			delete(*tsm, path)
		}
	}
}

type incomingSmapMessage struct {
	// Readings for this message
	Readings [][]json.RawMessage
	// If this struct corresponds to a sMAP collection,
	// then Contents contains a list of paths contained within
	// this collection
	Contents []string `json:",omitempty"`
	// Map of the metadata
	Metadata bson.M `json:",omitempty"`
	// Map containing the actuator reference
	Actuator bson.M `json:",omitempty"`
	// Map of the properties
	Properties SmapProperties `json:",omitempty"`
	// Unique identifier for this stream. Should be empty for Collections
	UUID UUID `json:"uuid"`
	// Path of this stream (thus far)
	Path string
}
