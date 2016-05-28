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
		mdRes.Data = append(mdRes.Data, ExtractMetadata(msg))
		tsRes.Data = append(tsRes.Data, ExtractTimeseries(msg))
	}
	replies[0] = mdRes.ToMsgPackBW()
	replies[1] = tsRes.ToMsgPackBW()

	return replies
}

func ExtractMetadata(msg *common.SmapMessage) KeyValueMetadata {
	md := KeyValueMetadata{
		UUID:     string(msg.UUID),
		Metadata: make(map[string]interface{}),
	}
	md.Metadata["Metadata"] = msg.Metadata
	md.Metadata["Properties"] = msg.Properties
	return md
}

func ExtractTimeseries(msg *common.SmapMessage) Timeseries {
	ts := Timeseries{
		UUID: string(msg.UUID),
		Data: make([]Point, len(msg.Readings)),
	}
	for i, rdg := range msg.Readings {
		if !rdg.IsObject() {
			d := rdg.(*common.SmapNumberReading)
			ts.Data[i] = Point{d.Time, d.Value}
		}
	}
	return ts
}
