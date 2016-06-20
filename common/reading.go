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
	IsStats() bool
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
	guess := GuessTimeUnit(s.Time)
	if to_uot != guess {
		s.Time, err = convertTime(s.Time, guess, to_uot)
		s.UoT = guess
	}
	return
}

func (s *SmapNumberReading) IsObject() bool {
	return false
}

func (s *SmapNumberReading) IsStats() bool {
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
	guess := GuessTimeUnit(s.Time)
	if to_uot != guess {
		s.Time, err = convertTime(s.Time, guess, to_uot)
		s.UoT = guess
	}
	return
}

func (s *SmapObjectReading) IsObject() bool {
	return true
}

func (s *SmapObjectReading) IsStats() bool {
	return false
}

func (s *SmapObjectReading) GetValue() interface{} {
	return s.Value
}

func (s *SmapObjectReading) SetUOT(uot UnitOfTime) {
	s.UoT = uot
}

type StatisticalNumberReading struct {
	Time  uint64
	UoT   UnitOfTime
	Count uint64
	Min   float64
	Mean  float64
	Max   float64
}

func (s *StatisticalNumberReading) IsObject() bool {
	return false
}

func (s *StatisticalNumberReading) IsStats() bool {
	return true
}

func (s *StatisticalNumberReading) GetValue() interface{} {
	return map[string]interface{}{"Count": s.Count, "Min": s.Min, "Mean": s.Mean, "Max": s.Max}
}

func (s *StatisticalNumberReading) SetUOT(uot UnitOfTime) {
	s.UoT = uot
}

func (s *StatisticalNumberReading) ConvertTime(to_uot UnitOfTime) (err error) {
	guess := GuessTimeUnit(s.Time)
	if to_uot != guess {
		s.Time, err = convertTime(s.Time, guess, to_uot)
		s.UoT = guess
	}
	return
}

func (s *StatisticalNumberReading) MarshalJSON() ([]byte, error) {
	timeString := strconv.FormatUint(s.Time, 10)
	return json.Marshal([]interface{}{json.Number(timeString), s.Count, s.Min, s.Mean, s.Max})
}

func (s *StatisticalNumberReading) GetTime() uint64 {
	return s.Time
}

type SmapNumbersResponse struct {
	Readings []*SmapNumberReading
	UUID     UUID `json:"uuid"`
}

type SmapObjectResponse struct {
	Readings []*SmapObjectReading
	UUID     UUID `json:"uuid"`
}

type StatisticalNumbersResponse struct {
	Readings []*StatisticalNumberReading
	UUID     UUID `json:"uuid"`
}
