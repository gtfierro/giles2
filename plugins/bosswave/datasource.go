package bosswave

import (
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v5"
)

type DataSource struct {
	// the source we are listening to
	uri      string
	data     chan *bw.SimpleMessage
	client   *bw.BW2Client
	archiver *giles.Archiver
}

func NewSource(uri string, client *bw.BW2Client, a *giles.Archiver) *DataSource {
	var err error
	src := &DataSource{
		uri:      uri,
		client:   client,
		archiver: a,
	}
	src.data, err = client.Subscribe(&bw.SubscribeParams{
		URI: uri,
	})
	if err != nil {
		log.Error(err)
		return nil
	}

	go func() {
		for msg := range src.data {
			src.handleIncomingMessage(msg)
		}
	}()

	return src
}

func (src *DataSource) handleIncomingMessage(msg *bw.SimpleMessage) {
	var (
		ts Timeseries
		md KeyValueMetadata
	)
	for _, po := range msg.POs {
		if po.IsTypeDF(GilesTimeseriesPIDString) {
			if err := po.(bw.MsgPackPayloadObject).ValueInto(&ts); err != nil {
				log.Error(errors.Wrap(err, "Could not unmarshal Timeseries"))
			}
			log.Noticef("TS: %+v", ts)
			smap := &common.SmapMessage{
				UUID:     common.UUID(ts.UUID),
				Readings: ts.ToReadings(),
				Metadata: common.Dict{"SourceName": "testmebw"},
				Path:     msg.URI,
			}
			src.archiver.AddData(smap, common.NewEphemeralKey())
		} else if po.IsTypeDF(GilesKeyValueMetadataPIDString) {
			if err := po.(bw.MsgPackPayloadObject).ValueInto(&md); err != nil {
				log.Error(errors.Wrap(err, "Could not unmarshal KeyValueMetadata"))
			}
			log.Noticef("MD: %+v", md)
		} else {
			log.Noticef("Got unrecognized type: %s", po.TextRepresentation())
		}
	}
}
