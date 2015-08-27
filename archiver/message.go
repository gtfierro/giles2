package archiver

import ()

type smapProperties struct {
	unitOfTime    UnitOfTime
	unitOfMeasure string
	streamType    StreamType
}

type SmapMessage struct {
	Path       string
	UUID       UUID `json:"uuid"`
	Properties *smapProperties
	Actuator   Dict
	Metadata   Dict
	Readings   []Reading
}
