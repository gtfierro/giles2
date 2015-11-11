package archiver

// bambam -o . -p archiver paper.go
// capnpc -ogo schema.capnp

type SmapProperties2 struct {
	UnitOfTime    uint64 `capid:"0"`
	UnitOfMeasure string `capid:"1"`
	StreamType    uint64 `capid:"2"`
}

type SmapMessage2 struct {
	Path       string               `json:",omitempty" msgpack:",omitempty" capid:"0"`
	UUID       string               `json:"uuid,omitempty" msgpack:",omitempty" capid:"1"`
	Properties *SmapProperties2     `json:",omitempty" msgpack:",omitempty" capid:"2"`
	Actuator   Dict                 `json:",omitempty" msgpack:",omitempty" capid:"3"`
	Metadata   Dict                 `json:",omitempty" msgpack:",omitempty" capid:"4"`
	Readings   []*SmapNumberReading `json:",omitempty" msgpack:",omitempty" capid:"5"`
}

func (sp SmapProperties2) IsEmpty() bool {
	return sp.UnitOfTime == 0 &&
		sp.UnitOfMeasure == "" &&
		sp.StreamType == 0
}
