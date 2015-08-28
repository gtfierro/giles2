package archiver

import (
	"gopkg.in/mgo.v2/bson"
)

type smapProperties struct {
	unitOfTime    UnitOfTime `json:"UnitofTime,omitempty"`
	unitOfMeasure string     `json:"UnitofMeasure,omitempty"`
	streamType    StreamType `json:"StreamType,omitempty"`
}

type SmapMessage struct {
	Path       string
	UUID       UUID `json:"uuid"`
	Properties *smapProperties
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
			ret["Metadata."+k] = v
		}
	}
	if msg.Actuator != nil && len(msg.Actuator) > 0 {
		for k, v := range msg.Actuator {
			ret["Actuator."+k] = v
		}
	}
	if msg.Properties != nil {
		ret["Properties.UnitofTime"] = msg.Properties.unitOfTime
		ret["Properties.UnitofMeasure"] = msg.Properties.unitOfMeasure
		ret["Properties.StreamType"] = msg.Properties.streamType
	}
	return ret
}
