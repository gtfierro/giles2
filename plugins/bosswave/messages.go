package bosswave

import (
	"encoding/json"
	"fmt"
	"github.com/gtfierro/giles2/common"
	bw "gopkg.in/immesys/bw2bind.v5"
	"math"
)

const (
	GilesArchiveRequestPIDString        = "2.0.8.0"
	GilesKeyValueQueryPIDString         = "2.0.8.1"
	GilesQueryMetadataResultPIDString   = "2.0.8.2"
	GilesKeyValueMetadataPIDString      = "2.0.8.3"
	GilesQueryTimeseriesResultPIDString = "2.0.8.4"
	GilesTimeseriesPIDString            = "2.0.8.5"
	GilesStatisticsPIDString            = "2.0.8.6"
	GilesQueryListResultPIDString       = "2.0.8.7"
	GilesQueryErrorPIDString            = "2.0.8.9"
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

func (msg QueryListResult) Dump() string {
	if bytes, err := json.MarshalIndent(msg.Data, "", "  "); err != nil {
		return fmt.Sprintf("%+v", msg)
	} else {
		return string(bytes)
	}
}

type QueryMetadataResult struct {
	Nonce uint32
	Data  []KeyValueMetadata
}

func (msg QueryMetadataResult) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesQueryMetadataResultPID, msg)
	return
}

func (msg QueryMetadataResult) Dump() string {
	var res string
	for _, kv := range msg.Data {
		res += kv.Dump()
	}
	return res
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

func (msg QueryTimeseriesResult) Dump() string {
	var res string
	for _, ts := range msg.Data {
		res += ts.Dump()
	}
	return res
}

type KeyValueMetadata struct {
	UUID     string
	Path     string
	Metadata map[string]interface{}
}

func (msg KeyValueMetadata) ToMsgPackBW() (po bw.PayloadObject) {
	po, _ = bw.CreateMsgPackPayloadObject(GilesKeyValueMetadataPID, msg)
	return
}

func (msg KeyValueMetadata) Dump() string {
	var md = make(map[string]interface{})
	for k, v := range msg.Metadata {
		if vmap, ok := v.(map[interface{}]interface{}); ok {
			for kk, vv := range vmap {
				md[k+"/"+kk.(string)] = vv
			}
		} else {
			md[k] = v
		}
	}
	msg.Metadata = md
	if bytes, err := json.MarshalIndent(msg, "", "  "); err != nil {
		log.Error(err)
		return fmt.Sprintf("%+v", msg)
	} else {
		return string(bytes)
	}
}

type Timeseries struct {
	UUID   string
	Path   string
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

func (msg Timeseries) Dump() string {
	var res [][]interface{}
	for i, time := range msg.Times {
		res = append(res, []interface{}{time, msg.Values[i]})
	}
	if bytes, err := json.MarshalIndent(map[string]interface{}{"UUID": msg.UUID, "Timeseries": res}, "", "  "); err != nil {
		return fmt.Sprintf("%+v", res)
	} else {
		return string(bytes)
	}
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
