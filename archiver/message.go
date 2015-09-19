package archiver

import (
	"encoding/json"
	"gopkg.in/mgo.v2/bson"
	"sort"
)

type smapProperties struct {
	UnitOfTime    UnitOfTime
	UnitOfMeasure string
	StreamType    StreamType
}

func (sp smapProperties) IsEmpty() bool {
	return sp.UnitOfTime == 0 &&
		sp.UnitOfMeasure == "" &&
		sp.StreamType == 0
}

type SmapMessage struct {
	Path       string
	UUID       UUID           `json:"uuid"`
	Properties smapProperties `json:",omitempty"`
	Actuator   Dict           `json:",omitempty"`
	Metadata   Dict           `json:",omitempty"`
	Readings   []Reading      `json:",omitempty"`
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
	if !msg.Properties.IsEmpty() {
		ret["Properties.UnitofTime"] = msg.Properties.UnitOfTime
		ret["Properties.UnitofMeasure"] = msg.Properties.UnitOfMeasure
		ret["Properties.StreamType"] = msg.Properties.StreamType
	}
	return ret
}

func (sm *SmapMessage) UnmarshalJSON(b []byte) (err error) {
	var (
		incoming  = new(incomingSmapMessage)
		time      uint64
		value_num float64
		value_obj interface{}
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
	sm.Metadata = *DictFromBson(flatten(incoming.Metadata))
	sm.Properties = incoming.Properties
	sm.Actuator = *DictFromBson(flatten(incoming.Actuator))

	// convert the readings depending if they are numeric or object
	sm.Readings = make([]Reading, len(incoming.Readings))
	for idx, reading := range incoming.Readings {
		// time should be a uint64 no matter what
		err = json.Unmarshal(reading[0], &time)
		if err != nil {
			return
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
	}
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
		ret.Metadata = *DictFromBson(md.(bson.M))
	}

	if md, found := m["Actuator"]; found {
		ret.Actuator = *DictFromBson(md.(bson.M))
	}

	if md, found := m["Properties"]; found {
		if props, ok := md.(bson.M); ok {
			ret.Properties = smapProperties{}
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
		(!msg.Properties.IsEmpty())
}

func (msg *SmapMessage) IsTimeseries() bool {
	return msg.UUID != ""
}

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
			if !prefixMsg.Properties.IsEmpty() {
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
	Properties smapProperties `json:",omitempty"`
	// Unique identifier for this stream. Should be empty for Collections
	UUID UUID `json:"uuid"`
	// Path of this stream (thus far)
	Path string
}
