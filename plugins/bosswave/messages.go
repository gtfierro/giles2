package bosswave

import (
	"github.com/gtfierro/giles2/common"
	bw "gopkg.in/immesys/bw2bind.v5"
	"math"
)

const (
	GilesKeyValueQueryPIDString    = "2.0.9.1"
	GilesQueryErrorPIDString       = "2.0.9.2"
	GilesKeyValueMetadataPIDString = "2.0.9.3"
	GilesTimeseriesPIDString       = "2.0.9.4"
	GilesStatisticsPIDString       = "2.0.9.5"

	GilesQueryListResultPIDString       = "2.0.9.6"
	GilesQueryMetadataResultPIDString   = "2.0.9.7"
	GilesQueryTimeseriesResultPIDString = "2.0.9.8"

	GilesArchiveRequestPIDString = "2.0.8.0"
)

var (
	GilesKeyValueQueryPID    = bw.FromDotForm(GilesKeyValueQueryPIDString)
	GilesQueryErrorPID       = bw.FromDotForm(GilesQueryErrorPIDString)
	GilesKeyValueMetadataPID = bw.FromDotForm(GilesKeyValueMetadataPIDString)
	GilesTimeseriesPID       = bw.FromDotForm(GilesTimeseriesPIDString)

	GilesQueryListResultPID       = bw.FromDotForm(GilesQueryListResultPIDString)
	GilesQueryMetadataResultPID   = bw.FromDotForm(GilesQueryMetadataResultPIDString)
	GilesQueryTimeseriesResultPID = bw.FromDotForm(GilesQueryTimeseriesResultPIDString)
	GilesArchiveRequestPID        = bw.FromDotForm(GilesArchiveRequestPIDString)
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
	Stats []Statistics
}

func (msg QueryTimeseriesResult) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryTimeseriesResultPID, msg)
	return
}

type KeyValueMetadata struct {
	UUID     string
	Metadata map[string]interface{}
}

func (msg KeyValueMetadata) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesKeyValueMetadataPID, msg)
	return
}

type Timeseries struct {
	UUID   string
	Times  []uint64
	Values []float64
}

func (msg Timeseries) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesTimeseriesPID, msg)
	return
}

func (msg Timeseries) ToReadings() []common.Reading {
	lesserLength := int(math.Min(float64(len(msg.Times)), float64(len(msg.Values))))
	var res = make([]common.Reading, lesserLength)
	for idx := 0; idx < lesserLength; idx++ {
		res[idx] = &common.SmapNumberReading{Time: msg.Times[idx], Value: msg.Values[idx], UoT: common.GuessTimeUnit(msg.Times[idx])}
	}
	return res
}

type Statistics struct {
	UUID  string
	Times []uint64
	Count []uint64
	Min   []float64
	Mean  []float64
	Max   []float64
}

func (msg Statistics) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesTimeseriesPID, msg)
	return
}

func (msg Statistics) ToReadings() []common.Reading {
	lesserLength := int(math.Min(float64(len(msg.Times)), float64(len(msg.Count))))
	var res = make([]common.Reading, lesserLength)
	for idx := 0; idx < lesserLength; idx++ {
		res[idx] = &common.StatisticalNumberReading{Time: msg.Times[idx], UoT: common.GuessTimeUnit(msg.Times[idx]), Count: msg.Count[idx], Min: msg.Min[idx], Max: msg.Max[idx], Mean: msg.Mean[idx]}
	}
	return res
}

type BWavable interface {
	ToMsgPackBW() bw.PayloadObject
}
