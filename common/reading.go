package common

import (
	"encoding/json"
	"gopkg.in/vmihailenco/msgpack.v2"
	"strconv"
)

// interface for sMAP readings
type Reading interface {
	GetTime() uint64
	ConvertTime(to UnitOfTime) error
	SetUOT(uot UnitOfTime)
	GetValue() interface{}
	IsObject() bool
}

// Reading implementation for numerical data
type SmapNumberReading struct {
	// uint64 timestamp
	Time uint64
	UoT  UnitOfTime
	// value associated with this timestamp
	Value float64
}

func (s *SmapNumberReading) MarshalJSON() ([]byte, error) {
	floatString := strconv.FormatFloat(s.Value, 'f', -1, 64)
	timeString := strconv.FormatUint(s.Time, 10)
	return json.Marshal([]json.Number{json.Number(timeString), json.Number(floatString)})
}

func (s *SmapNumberReading) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.Encode(s.Time, s.Value)
}

func (s *SmapNumberReading) DecodeMsgpack(enc *msgpack.Decoder) error {
	return enc.Decode(&s.Time, &s.Value)
}

func (s *SmapNumberReading) GetTime() uint64 {
	return s.Time
}

func (s *SmapNumberReading) SetUOT(uot UnitOfTime) {
	s.UoT = uot
}

func (s *SmapNumberReading) ConvertTime(to_uot UnitOfTime) (err error) {
	if to_uot != s.UoT {
		s.Time, err = convertTime(s.Time, s.UoT, to_uot)
	}
	return
}

func (s *SmapNumberReading) IsObject() bool {
	return false
}

func (s *SmapNumberReading) GetValue() interface{} {
	return s.Value
}

// Reading implementation for object data
type SmapObjectReading struct {
	// uint64 timestamp
	Time uint64
	UoT  UnitOfTime
	// value associated with this timestamp
	Value interface{}
}

func (s *SmapObjectReading) MarshalJSON() ([]byte, error) {
	timeString := strconv.FormatUint(s.Time, 10)
	return json.Marshal([]interface{}{json.Number(timeString), s.Value})
}

func (s *SmapObjectReading) GetTime() uint64 {
	return s.Time
}

func (s *SmapObjectReading) ConvertTime(to_uot UnitOfTime) (err error) {
	if to_uot != s.UoT {
		s.Time, err = convertTime(s.Time, s.UoT, to_uot)
	}
	return
}

func (s *SmapObjectReading) IsObject() bool {
	return true
}

func (s *SmapObjectReading) GetValue() interface{} {
	return s.Value
}

func (s *SmapObjectReading) SetUOT(uot UnitOfTime) {
	s.UoT = uot
}

type SmapNumbersResponse struct {
	Readings []*SmapNumberReading
	UUID     UUID `json:"uuid"`
}

type SmapObjectResponse struct {
	Readings []*SmapObjectReading
	UUID     UUID `json:"uuid"`
}
