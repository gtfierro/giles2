package bosswave

import (
	"fmt"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	ob "github.com/gtfierro/giles2/objectbuilder"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	bw "gopkg.in/immesys/bw2bind.v5"
	"strings"
	"time"
)

var NAMESPACE_UUID = uuid.FromStringOrNil("b26d2e62-333e-11e6-b557-0cc47a0f7eea")

// This object is a set of instructions for how to create an archivable message
// from some received PayloadObject, though really this should be able to
// operate on any object. Each ArchiveRequest acts as a translator for received
// messages into a single timeseries stream
type ArchiveRequest struct {
	// AUTOPOPULATED. The entity that requested the URI to be archived.
	FromVK string
	// OPTIONAL. the URI to subscribe to. Requires building a chain on the URI
	// from the .FromVK field. If not provided, uses the base URI of where this
	// ArchiveRequest was stored. For example, if this request was published
	// on <uri>/!meta/giles, then if the URI field was elided it would default
	// to <uri>.
	URI string
	// Extracts objects of the given Payload Object type from all messages
	// published on the URI. If elided, operates on all PO types.
	PO int
	// OPTIONAL. If provided, this is used as the stream UUID.  If not
	// provided, then a UUIDv3 with NAMESPACE_UUID and the URI, PO type and
	// Value are used.
	UUID string
	// the real UUID when we get it
	uuidActual common.UUID
	uuid       []ob.Operation
	// expression determining how to extract the value from the received
	// message
	Value string
	value []ob.Operation
	// OPTIONAL. Expression determining how to extract the value from the
	// received message. If not included, it uses the time the message was
	// received on the server.
	Time string
	time []ob.Operation
	// OPTIONAL. Golang time parse string
	TimeParse string

	// OPTIONAL. a base URI to scan for metadata. If `<uri>` is provided, we
	// scan `<uri>/!meta/+` for metadata keys/values
	MetadataURI string

	// OPTIONAL. a URI terminating in a metadata key that contains some kv
	// structure of metadata, for example `/a/b/c/!meta/metadatahere`
	MetadataBlock string

	// OPTIONAL. a ObjectBuilder expression to search in the current message
	// for metadata
	MetadataExpr string
	metadataExpr []ob.Operation
}

func (req *ArchiveRequest) GetSmapMessage(thing interface{}) *common.SmapMessage {
	var msg = new(common.SmapMessage)
	var rdg = new(common.SmapNumberReading)

	value := ob.Eval(req.value, thing)
	switch t := value.(type) {
	case int64:
		rdg.Value = float64(t)
	case uint64:
		rdg.Value = float64(t)
	case float64:
		rdg.Value = t
	}

	rdg.Time = req.getTime(thing)

	if len(req.uuid) > 0 && req.uuidActual == "" {
		req.uuidActual = common.UUID(ob.Eval(req.uuid, thing).(string))
	} else if req.uuidActual == "" {
		req.uuidActual = common.UUID(req.UUID)
	}
	msg.UUID = req.uuidActual
	msg.Path = req.URI + "/" + req.Value
	msg.Readings = []common.Reading{rdg}

	if len(req.metadataExpr) > 0 {
		msg.Metadata = make(common.Dict)
		if md, ok := ob.Eval(req.metadataExpr, thing).(map[string]interface{}); ok {
			for k, v := range md {
				msg.Metadata[k] = fmt.Sprintf("%s", v)
			}
		}
	}

	return msg
}

func (req *ArchiveRequest) GetMetadata(msg *bw.SimpleMessage) *common.SmapMessage {
	var ret = new(common.SmapMessage)
	ret.UUID = req.uuidActual
	ret.Path = req.URI + "/" + req.Value
	ret.Metadata = make(common.Dict)

	for _, po := range msg.POs {
		var md map[string]interface{}
		if po.IsTypeDF(bw.PODFMsgPack) {
			err := po.(bw.MsgPackPayloadObject).ValueInto(&md)
			if err != nil {
				log.Error(errors.Wrap(err, "Could not unmarshal msgpack metadata"))
				return nil
			}
		}
		for k, v := range md {
			ret.Metadata[k] = fmt.Sprintf("%s", v)
		}
	}
	return ret
}

func (req *ArchiveRequest) getTime(thing interface{}) uint64 {
	if len(req.time) == 0 {
		return uint64(time.Now().UnixNano())
	}
	timeString, ok := ob.Eval(req.time, thing).(string)
	if ok {
		parsedTime, err := time.Parse(req.TimeParse, timeString)
		if err != nil {
			return uint64(time.Now().UnixNano())
		}
		return uint64(parsedTime.UnixNano())
	}
	return uint64(time.Now().UnixNano())
}

// When we receive a metadata message with the right key (currently !meta/giles), then
// we parse out the list of contained ObjectTemplates
func (bwh *BOSSWaveHandler) ExtractArchiveRequests(msg *bw.SimpleMessage) []*ArchiveRequest {
	var requests []*ArchiveRequest
	for _, po := range msg.POs {
		if !po.IsTypeDF(GilesArchiveRequestPIDString) {
			continue
		}
		var request = new(ArchiveRequest)
		err := po.(bw.MsgPackPayloadObject).ValueInto(request)
		if err != nil {
			log.Error(errors.Wrap(err, "Could not parse Archive Request"))
			continue
		}
		if request.PO == 0 {
			log.Error(errors.Wrap(err, "Request contained no PO"))
			continue
		}
		if request.Value == "" {
			log.Error(errors.Wrap(err, "Request contained no Value expression"))
			continue
		}
		request.FromVK = msg.From
		if request.URI == "" { // no URI supplied
			request.URI = strings.TrimSuffix(request.URI, "!meta/giles")
			request.URI = strings.TrimSuffix(request.URI, "/")
		}
		//TODO: build a chain here to check if they have da permissiones
		requests = append(requests, request)
	}

	return requests
}

// First, we check that all the fields are valid and the necessary ones are populated.
// This also involves filling in the optional ones with sane values.
// Then we build a chain on the URI to the VK -- if this fails, then we stop
// Then we build the operator chains for the expressions required
// Then we subscribe to the URI indicated.
func (bwh *BOSSWaveHandler) ParseArchiveRequest(request *ArchiveRequest) (*URIArchiver, error) {
	if request.FromVK == "" {
		return nil, errors.New("VK was empty in ArchiveRequest")
	}
	request.value = ob.Parse(request.Value)

	if request.UUID == "" {
		request.UUID = uuid.NewV3(NAMESPACE_UUID, request.URI+request.Value).String()
	} else {
		request.uuid = ob.Parse(request.UUID)
	}

	if request.Time != "" {
		request.time = ob.Parse(request.Time)
	}

	if request.MetadataExpr != "" {
		request.metadataExpr = ob.Parse(request.MetadataExpr)
	}

	var metadataChan = make(chan *bw.SimpleMessage)
	if request.MetadataURI != "" {
		sub1, err := bwh.bw.Subscribe(&bw.SubscribeParams{
			URI: strings.TrimSuffix(request.MetadataURI, "/") + "/!meta/+",
		})
		if err != nil {
			return nil, err
		}
		go func() {
			for msg := range sub1 {
				metadataChan <- msg
			}
		}()

		q1, err := bwh.bw.Query(&bw.QueryParams{
			URI: strings.TrimSuffix(request.MetadataURI, "/") + "/!meta/+",
		})
		if err != nil {
			return nil, err
		}
		go func() {
			for msg := range q1 {
				metadataChan <- msg
			}
		}()
	}
	//TODO: subscribe then query MetadataBlock
	//TODO: subscribe then query MetadataURI

	log.Debugf("Subscribing to %s", request.URI)
	sub, err := bwh.bw.Subscribe(&bw.SubscribeParams{
		URI: request.URI,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not subscribe")
	}
	log.Debugf("Got archive request %+v", request)

	archiver := &URIArchiver{sub, metadataChan, request}
	go archiver.Listen(bwh.a)

	return archiver, nil
}

// this struct handles incoming messages from a source. It is defined by an ArchiveRequest.
// For each message, we apply the operator chains where appropriate and form a set of SmapMessage
// that get sent to the archiver instance
type URIArchiver struct {
	subscription chan *bw.SimpleMessage
	metadataChan chan *bw.SimpleMessage
	*ArchiveRequest
}

func (uri *URIArchiver) Listen(a *giles.Archiver) {
	go func() {
		for msg := range uri.metadataChan {
			a.AddData(uri.GetMetadata(msg))
		}
	}()
	for msg := range uri.subscription {
		for _, po := range msg.POs {
			if !po.IsType(uri.PO, uri.PO) {
				continue
			}
			// for each of the major types, unmarshal it into some generic type
			// and apply the object builder stuff.
			// For now, assume it is msgpack
			var thing interface{}
			err := po.(bw.MsgPackPayloadObject).ValueInto(&thing)
			if err != nil {
				log.Error(errors.Wrap(err, "Could not unmarshal msgpack object"))
			}
			err = a.AddData(uri.GetSmapMessage(thing))
			if err != nil {
				log.Error(errors.Wrap(err, "Could not add data"))
			}
		}
	}
}
