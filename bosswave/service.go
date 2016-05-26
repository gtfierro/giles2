package bosswave

import (
	"fmt"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	bw "gopkg.in/immesys/bw2bind.v5"
	"os"
	"sync"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("bosswave")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type BOSSWaveHandler struct {
	a         *giles.Archiver
	bw        *bw.BW2Client
	svc       *bw.Service
	iface     *bw.Interface
	namespace string
	vk        string
	subsLock  sync.RWMutex
	subs      map[string]string
	stop      chan bool
}

func NewHandler(a *giles.Archiver, entityfile, namespace string) *BOSSWaveHandler {
	bwh := &BOSSWaveHandler{
		a:         a,
		bw:        bw.ConnectOrExit(""),
		namespace: namespace,
		stop:      make(chan bool),
	}
	bwh.bw.OverrideAutoChainTo(true)
	bwh.vk = bwh.bw.SetEntityFileOrExit(entityfile)
	bwh.svc = bwh.bw.RegisterService(bwh.namespace, "s.giles")
	bwh.iface = bwh.svc.RegisterInterface("0", "i.archiver")
	bwh.iface.SubscribeSlot("query", bwh.listenQueries)
	log.Infof("iface: %s", bwh.iface.FullURI())
	return bwh
}

func Handle(a *giles.Archiver, entityfile, namespace string) {
	bwh := NewHandler(a, entityfile, namespace)
	<-bwh.stop
}

func (bwh *BOSSWaveHandler) listenQueries(msg *bw.SimpleMessage) {
	var (
		// the publisher of the message. We incorporate this into the signal URI
		fromVK string
		// the computed signal based on the VK and query nonce
		signalURI string
		// query message
		query KeyValueQuery
	)
	fromVK = msg.From
	po := msg.GetOnePODF(GilesKeyValueQueryPIDString)
	if po == nil { // no query found
		return
	}

	if obj, ok := po.(bw.MsgPackPayloadObject); !ok {
		log.Error("Received query was not msgpack")
	} else if err := obj.ValueInto(&query); err != nil {
		log.Error(errors.Wrap(err, "Could not unmarshal received query"))
	}

	signalURI = fmt.Sprintf("%s/queries", fromVK[:len(fromVK)-1])

	log.Infof("Got query %+v", query)
	res, err := bwh.a.HandleQuery(query.Query, common.NewEphemeralKey())
	if err != nil {
		msg := QueryError{
			Query: query.Query,
			Nonce: query.Nonce,
			Error: err.Error(),
		}
		po, _ := bw.CreateMsgPackPayloadObject(GilesQueryErrorPID, msg)
		log.Error(errors.Wrap(err, "Error evaluating query"))
		bwh.iface.PublishSignal(signalURI, po)
	}

	var reply []bw.PayloadObject

	switch t := res.(type) {
	case common.SmapMessageList:
		log.Debug("smap messages list")
		pos := POsFromSmapMessageList(query.Nonce, t)
		reply = append(reply, pos...)
	case common.DistinctResult:
		log.Debug("distinct list")
		reply = append(reply, POFromDistinctResult(query.Nonce, t))
	default:
		log.Debug("type %T", res)
	}

	bwh.iface.PublishSignal(signalURI, reply...)
}
