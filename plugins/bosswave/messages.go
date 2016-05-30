package bosswave

import (
	"github.com/gtfierro/giles2/common"
	bw "gopkg.in/immesys/bw2bind.v5"
)

const (
	GilesKeyValueQueryPIDString    = "2.0.9.1"
	GilesQueryErrorPIDString       = "2.0.9.2"
	GilesKeyValueMetadataPIDString = "2.0.9.3"
	GilesTimeseriesPIDString       = "2.0.9.4"

	GilesQueryListResultPIDString       = "2.0.9.5"
	GilesQueryMetadataResultPIDString   = "2.0.9.6"
	GilesQueryTimeseriesResultPIDString = "2.0.9.7"
)

var (
	GilesKeyValueQueryPID    = bw.FromDotForm(GilesKeyValueQueryPIDString)
	GilesQueryErrorPID       = bw.FromDotForm(GilesQueryErrorPIDString)
	GilesKeyValueMetadataPID = bw.FromDotForm(GilesKeyValueMetadataPIDString)
	GilesTimeseriesPID       = bw.FromDotForm(GilesTimeseriesPIDString)

	GilesQueryListResultPID       = bw.FromDotForm(GilesQueryListResultPIDString)
	GilesQueryMetadataResultPID   = bw.FromDotForm(GilesQueryMetadataResultPIDString)
	GilesQueryTimeseriesResultPID = bw.FromDotForm(GilesQueryTimeseriesResultPIDString)
)

//TODO: put VK and URi and "format" in these messages, but don't put them in the
// manifest
type KeyValueQuery struct {
	Query string
	Nonce uint32
}

func (msg KeyValueQuery) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesKeyValueQueryPID, msg)
	return
}

type QueryError struct {
	Query string
	Nonce uint32
	Error string
}

func (msg QueryError) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryErrorPID, msg)
	return
}

type QueryListResult struct {
	Nonce uint32
	Data  []string
}

func (msg QueryListResult) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryListResultPID, msg)
	return
}

type QueryMetadataResult struct {
	Nonce uint32
	Data  []KeyValueMetadata
}

func (msg QueryMetadataResult) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryMetadataResultPID, msg)
	return
}

type QueryTimeseriesResult struct {
	Nonce uint32
	Data  []Timeseries
}

func (msg QueryTimeseriesResult) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryTimeseriesResultPID, msg)
	return
}

// do we need a "query result" struct?
type KeyValueMetadata struct {
	Nonce    uint32
	UUID     string
	Metadata map[string]interface{}
}

func (msg KeyValueMetadata) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesKeyValueMetadataPID, msg)
	return
}

type Timeseries struct {
	Nonce uint32
	UUID  string
	Data  []Point
}

type Point struct {
	Time  uint64
	Value float64
}

func (msg Timeseries) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesTimeseriesPID, msg)
	return
}

func (msg Timeseries) ToReadings() []common.Reading {
	var res = make([]common.Reading, len(msg.Data))
	for idx, point := range msg.Data {
		res[idx] = &common.SmapNumberReading{Time: point.Time, Value: point.Value, UoT: common.GuessTimeUnit(point.Time)}
	}
	return res
}

type BWavable interface {
	ToMsgPackBW() bw.PayloadObject
}
