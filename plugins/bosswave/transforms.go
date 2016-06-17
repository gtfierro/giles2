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
	}
	for _, msg := range list {
		mdRes.Data = append(mdRes.Data, ExtractMetadataToBW(msg))
		tsRes.Data = append(tsRes.Data, ExtractTimeseriesToBW(msg))
	}
	replies[0] = mdRes.ToMsgPackBW()
	replies[1] = tsRes.ToMsgPackBW()

	return replies
}

func ExtractMetadataToBW(msg *common.SmapMessage) KeyValueMetadata {
	md := KeyValueMetadata{
		UUID:     string(msg.UUID),
		Metadata: make(map[string]interface{}),
	}
	md.Metadata["Metadata"] = msg.Metadata
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
		if !rdg.IsObject() {
			d := rdg.(*common.SmapNumberReading)
			ts.Times[i] = d.Time
			ts.Values[i] = d.Value
		}
	}
	return ts
}
