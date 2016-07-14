package bosswave

import (
	"github.com/gtfierro/giles2/common"
	bw "gopkg.in/immesys/bw2bind.v5"
)

func POFromDistinctResult(nonce uint32, msg common.DistinctResult) bw.PayloadObject {
	res := QueryListResult{
		Nonce: nonce,
		Data:  msg,
	}
	return res.ToMsgPackBW()
}

func POsFromSmapMessageList(nonce uint32, list common.SmapMessageList) []bw.PayloadObject {
	replies := make([]bw.PayloadObject, 2)
	mdRes := QueryMetadataResult{
		Nonce: nonce,
		Data:  []KeyValueMetadata{},
	}
	tsRes := QueryTimeseriesResult{
		Nonce: nonce,
		Data:  []Timeseries{},
		Stats: []Statistics{},
	}
	for _, msg := range list {
		if len(msg.Metadata) > 0 || msg.Properties != nil {
			mdRes.Data = append(mdRes.Data, ExtractMetadataToBW(msg))
		}
		if len(msg.Readings) > 0 && !msg.Readings[0].IsStats() {
			tsRes.Data = append(tsRes.Data, ExtractTimeseriesToBW(msg))
		}
		if len(msg.Readings) > 0 && msg.Readings[0].IsStats() {
			tsRes.Stats = append(tsRes.Stats, ExtractStatisticsToBW(msg))
		}
	}
	replies[0] = mdRes.ToMsgPackBW()
	replies[1] = tsRes.ToMsgPackBW()

	return replies
}

func ExtractMetadataToBW(msg *common.SmapMessage) KeyValueMetadata {
	md := KeyValueMetadata{
		UUID:     string(msg.UUID),
		Metadata: make(map[string]interface{}),
		Path:     msg.Path,
	}
	md.Metadata["Metadata"] = map[string]interface{}(msg.Metadata)
	md.Metadata["Properties"] = msg.Properties
	return md
}

func ExtractTimeseriesToBW(msg *common.SmapMessage) Timeseries {
	ts := Timeseries{
		UUID:   string(msg.UUID),
		Times:  make([]uint64, len(msg.Readings)),
		Values: make([]float64, len(msg.Readings)),
	}
	for i, rdg := range msg.Readings {
		if !rdg.IsObject() && !rdg.IsStats() {
			d := rdg.(*common.SmapNumberReading)
			ts.Times[i] = d.Time
			ts.Values[i] = d.Value
		}
	}
	return ts
}

func ExtractStatisticsToBW(msg *common.SmapMessage) Statistics {
	stats := Statistics{
		UUID:  string(msg.UUID),
		Times: make([]uint64, len(msg.Readings)),
		Count: make([]uint64, len(msg.Readings)),
		Min:   make([]float64, len(msg.Readings)),
		Mean:  make([]float64, len(msg.Readings)),
		Max:   make([]float64, len(msg.Readings)),
	}
	for i, rdg := range msg.Readings {
		if rdg.IsStats() {
			d := rdg.(*common.StatisticalNumberReading)
			stats.Times[i] = d.Time
			stats.Count[i] = d.Count
			stats.Min[i] = d.Min
			stats.Mean[i] = d.Mean
			stats.Max[i] = d.Max
		}
	}
	return stats
}
