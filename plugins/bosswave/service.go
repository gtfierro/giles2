package bosswave

import (
	"fmt"
	giles "github.com/gtfierro/giles2/archiver"
	"github.com/gtfierro/giles2/common"
	"github.com/gtfierro/giles2/plugins/bosswave/views"
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

	stop chan bool
	// subscribe to */!meta/archive to see what we subscribe to
	incomingData chan *bw.SimpleMessage

	// archive requests. Map of hash -> request struct
	requests     map[string]*ArchiveRequest
	requestsLock sync.RWMutex
}

func NewHandler(a *giles.Archiver, entityfile, namespace string) *BOSSWaveHandler {
	bwh := &BOSSWaveHandler{
		a:         a,
		bw:        bw.ConnectOrExit(""),
		namespace: namespace,
		stop:      make(chan bool),
		requests:  make(map[string]*ArchiveRequest),
	}
	bwh.bw.OverrideAutoChainTo(true)
	bwh.vk = bwh.bw.SetEntityFileOrExit(entityfile)
	bwh.svc = bwh.bw.RegisterService(bwh.namespace, "s.giles")
	bwh.iface = bwh.svc.RegisterInterface("0", "i.archiver")
	bwh.iface.SubscribeSlot("query", bwh.listenQueries)
	bwh.iface.SubscribeSlot("subscribe", bwh.listenCQBS)

	v, e := views.CreateView(bwh.bw, views.Expression{
		NamespaceList: []string{"gabe.pantry", "scratch.ns"},
		N:             &views.EqualsNode{Key: views.String("giles")},
	})
	if e != nil {
		log.Error(errors.Wrap(e, "Could not create view"))
	}

	go func() {
		for msg := range v.Subscribe() {
			go bwh.handleArchiveRequest(msg)
		}
	}()

	return bwh
}

func Handle(a *giles.Archiver, entityfile, namespace string) {
	bwh := NewHandler(a, entityfile, namespace)
	<-bwh.stop
}

func (bwh *BOSSWaveHandler) handleArchiveRequest(msg *bw.SimpleMessage) {
	requests := bwh.ExtractArchiveRequests(msg)
	for _, req := range requests {
		if bwh.addRequest(req) {
			bwh.ParseArchiveRequest(req)
		}
	}
}

// Adds request if it does not already exist. Returns true if the request is new.
func (bwh *BOSSWaveHandler) addRequest(req *ArchiveRequest) bool {
	bwh.requestsLock.Lock()
	defer bwh.requestsLock.Unlock()
	hash := req.Hash()
	if _, found := bwh.requests[req.Hash()]; !found {
		bwh.requests[hash] = req
		return true
	}
	return false
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

	signalURI = fmt.Sprintf("%s,queries", fromVK[:len(fromVK)-1])

	log.Infof("Got query %+v", query)
	res, err := bwh.a.HandleQuery(query.Query)
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

func (bwh *BOSSWaveHandler) listenCQBS(msg *bw.SimpleMessage) {
	var (
		// the publisher of the message. We incorporate this into the signal URI
		fromVK string
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

	subscription := bwh.StartSubscriber(fromVK, query)
	go bwh.a.HandleNewSubscriber(subscription, query.Query)
}

func (bwh *BOSSWaveHandler) StartSubscriber(vk string, query KeyValueQuery) *giles.Subscriber {
	bws := &BWSubscriber{
		bw:      bwh.bw,
		nonce:   query.Nonce,
		closeC:  make(chan bool),
		baseURI: fmt.Sprintf("%s,", vk[:len(vk)-1]),
	}
	bws.allURI = bws.baseURI + "all"
	bws.timeseriesURI = bws.baseURI + "timeseries"
	bws.metadataURI = bws.baseURI + "metadata"
	bws.diffURI = bws.baseURI + "diff"
	bws.subscription = giles.NewSubscriber(bws.closeC, 10, bws.handleError)

	go func(bws *BWSubscriber) {
		for val := range bws.subscription.C {
			var reply []bw.PayloadObject
			log.Debugf("subscription got val %+v", val)
			switch t := val.(type) {
			case common.SmapMessageList:
				log.Debugf("smap messages list %+v", t)
				pos := POsFromSmapMessageList(query.Nonce, t)
				reply = append(reply, pos...)
			case common.DistinctResult:
				log.Debugf("distinct list %+v", t)
				reply = append(reply, POFromDistinctResult(query.Nonce, t))
			default:
				log.Debug("type %T", val)
			}
			if err := bwh.iface.PublishSignal(bws.allURI, reply...); err != nil {
				log.Error(errors.Wrap(err, "Could not publish reply"))
			}
		}
	}(bws)

	return bws.subscription
}

type BWSubscriber struct {
	bw            *bw.BW2Client
	subscription  *giles.Subscriber
	closeC        chan bool
	baseURI       string
	allURI        string
	timeseriesURI string
	metadataURI   string
	diffURI       string
	nonce         uint32
}

func (bws *BWSubscriber) handleError(e error) {
	log.Error("sub got error", e)
}
