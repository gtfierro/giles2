package archiver

import (
	"gopkg.in/mgo.v2/bson"
)

type smapProperties struct {
	unitOfTime    UnitOfTime `json:"UnitofTime,omitempty"`
	unitOfMeasure string     `json:"UnitofMeasure,omitempty"`
	streamType    StreamType `json:"StreamType,omitempty"`
}

func (sp smapProperties) IsEmpty() bool {
	return sp.unitOfTime == 0 &&
		sp.unitOfMeasure == "" &&
		sp.streamType == 0
}

type SmapMessage struct {
	Path       string
	UUID       UUID `json:"uuid"`
	Properties smapProperties
	Actuator   Dict
	Metadata   Dict
	Readings   []Reading
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
		ret["Properties.UnitofTime"] = msg.Properties.unitOfTime
		ret["Properties.UnitofMeasure"] = msg.Properties.unitOfMeasure
		ret["Properties.StreamType"] = msg.Properties.streamType
	}
	return ret
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
			log.Debug("props %v %v", props, ret.Properties)
			if uot, fnd := props["UnitofTime"]; fnd {
				ret.Properties.unitOfTime = uot.(UnitOfTime)
			}
			if uom, fnd := props["UnitofMeasure"]; fnd {
				ret.Properties.unitOfMeasure = uom.(string)
			}
			if uot, fnd := props["StreamType"]; fnd {
				ret.Properties.streamType = uot.(StreamType)
			}
			log.Debug("props %v %v", props, ret.Properties)
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

type SmapMessageList []*SmapMessage

func (sml *SmapMessageList) ToBson() []bson.M {
	ret := make([]bson.M, len(*sml))
	for idx, msg := range *sml {
		ret[idx] = msg.ToBson()
	}
	return ret
}

func SmapMessageListFromBson(m []bson.M) *SmapMessageList {
	ret := make(SmapMessageList, len(m))
	for idx, doc := range m {
		ret[idx] = SmapMessageFromBson(doc)
	}
	return &ret
}
