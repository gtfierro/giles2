//go:generate go tool yacc -o query.go -p SQ query.y
package archiver

import (
	"fmt"
	"github.com/gtfierro/giles2/archiver/internal/querylang"
	"github.com/gtfierro/giles2/common"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"net"
	"os"
	"time"
)

// logger
var log *logging.Logger

// set up logging facilities
func init() {
	log = logging.MustGetLogger("archiver")
	var format = "%{color}%{level} %{time:Jan 02 15:04:05} %{shortfile}%{color:reset} ▶ %{message}"
	var logBackend = logging.NewLogBackend(os.Stderr, "", 0)
	logBackendLeveled := logging.AddModuleLevel(logBackend)
	logging.SetBackend(logBackendLeveled)
	logging.SetFormatter(logging.MustStringFormatter(format))
}

type Archiver struct {
	// timeseries database interface
	tsStore TimeseriesStore
	// metadata store
	mdStore MetadataStore
	// transaction coalescer
	qp *querylang.QueryProcessor
	// broker
	broker *Broker
	// metrics
	metrics metricMap
}

// Returns a new archiver object from a configuration. Will Fatal out of the
// program if there is an error in setting up connections to databases or reading
// the config file
func NewArchiver(c *Config) (a *Archiver) {
	var (
		mdStore MetadataStore
		tsStore TimeseriesStore
	)

	a = &Archiver{}

	switch *c.Archiver.MetadataStore {
	case "mongo":
		mongoaddr, err := net.ResolveTCPAddr("tcp4", *c.Mongo.Address+":"+*c.Mongo.Port)
		if err != nil {
			log.Fatalf("Error parsing Mongo address: %v", err)
		}
		config := &mongoConfig{
			address: mongoaddr,
		}
		mdStore = newMongoStore(config)
	default:
		log.Fatalf(*c.Archiver.MetadataStore, " is not a recognized metadata store")
	}

	a.mdStore = mdStore

	switch *c.Archiver.TimeseriesStore {
	case "quasar":
		qsraddr, err := net.ResolveTCPAddr("tcp4", *c.Quasar.Address+":"+*c.Quasar.Port)
		if err != nil {
			log.Fatalf("Error parsing Quasar address: %v", err)
		}
		config := &quasarConfig{
			addr:    qsraddr,
			mdStore: a.mdStore,
		}
		tsStore = newQuasarDB(config)
	case "btrdb":
		btrdbaddr, err := net.ResolveTCPAddr("tcp4", *c.BtrDB.Address+":"+*c.BtrDB.Port)
		if err != nil {
			log.Fatalf("Error parsing BtrDB address: %v", err)
		}
		config := &btrdbConfig{
			addr:    btrdbaddr,
			mdStore: a.mdStore,
		}
		tsStore = newBtrIface(config)
	default:
		log.Fatalf(*c.Archiver.TimeseriesStore, " is not a recognized timeseries store")
	}

	a.tsStore = tsStore

	a.qp = querylang.NewQueryProcessor()

	a.broker = NewBroker(a)

	a.metrics = make(metricMap)
	a.metrics.addMetric("adds")

	if c.Archiver.PeriodicReport {
		a.startReport()
	}
	return
}

func (a *Archiver) startReport() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		for {
			log.Infof("Adds:%d", a.metrics["adds"].GetAndReset())
			<-t.C
		}
	}()
}

// Takes an incoming common.SmapMessage object (from a client) and does the following:
//  - Checks the incoming message against the ApiKey to verify it is valid to write
//  - Saves the attached metadata (if any) to the metadata store
//  - Reevaluates any dynamic subscriptions and pushes to republish clients
//  - Saves the attached readings (if any) to the timeseries database
func (a *Archiver) AddData(msg *common.SmapMessage) (err error) {
	// save metadata
	err = a.mdStore.SaveTags(msg)
	if err != nil {
		return err
	}

	// fix inconsistencies
	var (
		uot common.UnitOfTime
		uom string
	)
	if uot, err = a.mdStore.GetUnitOfTime(msg.UUID); uot == 0 && err == nil {
		if len(msg.Readings) > 0 {
			uot = common.GuessTimeUnit(msg.Readings[0].GetTime())
		}
	} else if err != nil {
		return err
	}
	for _, rdg := range msg.Readings {
		rdg.SetUOT(uot)
	}

	if uom, err = a.mdStore.GetUnitOfMeasure(msg.UUID); uom == "" && err == nil {
		if msg.Properties == nil {
			msg.Properties = &common.SmapProperties{StreamType: common.NUMERIC_STREAM}
		}
		msg.Properties.UnitOfMeasure = "n/a"
		err = a.mdStore.SaveTags(msg)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	//save timeseries data
	a.metrics["adds"].Mark(1)
	a.tsStore.AddMessage(msg)
	a.broker.HandleMessage(msg)
	return err
}

// Need to think about how to transfer the results of these queries to the handlers that are
// asking for them and need to transform them into their own internal representations (e.g.
// JSON, MsgPack, etc). What are the data patterns we are seeing?
// Basically everything fits into common.SmapMessageList
func (a *Archiver) HandleQuery(querystring string) (QueryResult, error) {
	var result QueryResult
	// parse the query
	parsed := a.qp.Parse(querystring)
	if parsed.Err != nil {
		return result, fmt.Errorf("Error (%v) in query \"%v\" (error at %v)\n", parsed.Err, querystring, parsed.ErrPos)
	}
	return a.evaluateQuery(parsed)
}

func (a *Archiver) evaluateQuery(parsed *querylang.ParsedQuery) (QueryResult, error) {
	var result QueryResult
	switch parsed.QueryType {
	case querylang.SELECT_TYPE:
		if parsed.Distinct {
			params := parsed.GetParams().(*common.DistinctParams)
			return a.DistinctTag(params)
		}
		params := parsed.GetParams().(*common.TagParams)
		return a.SelectTags(params)
	case querylang.DELETE_TYPE:
		params := parsed.GetParams()
		switch t := params.(type) {
		case *common.TagParams:
			return result, a.DeleteTags(t)
		case *common.DataParams:
			return result, a.DeleteData(t)
		default:
			return result, errors.New("Invalid DELETE type")
		}
	case querylang.SET_TYPE:
		params := parsed.GetParams().(*common.SetParams)
		return result, a.SetTags(params)
	case querylang.DATA_TYPE:
		params := parsed.GetParams().(*common.DataParams)
		if params.IsStatistical || params.IsWindow {
			return a.SelectStatisticalData(params)
		}
		switch parsed.Data.Dtype {
		case querylang.IN_TYPE:
			return a.SelectDataRange(params)
		case querylang.BEFORE_TYPE:
			return a.SelectDataBefore(params)
		case querylang.AFTER_TYPE:
			return a.SelectDataAfter(params)
		}

	}
	return result, nil
}

func (a *Archiver) HandleNewSubscriber(subscriber *Subscriber, querystring string) error {
	subscriber.query = a.qp.Parse(querystring)
	return a.broker.NewSubscriber(subscriber)
}
