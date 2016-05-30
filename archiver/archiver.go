//go:generate go tool yacc -o query.go -p SQ query.y
package archiver

import (
	"fmt"
	"github.com/gtfierro/giles2/archiver/internal/querylang"
	"github.com/gtfierro/giles2/common"
	"github.com/op/go-logging"
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
	// permissions manager
	pm permissionsManager
	// transaction coalescer
	qp *querylang.QueryProcessor
	// broker
	broker *Broker
	// metrics
	metrics metricMap
	// enforce ephemral key checks
	enforceKeys bool
}

// Returns a new archiver object from a configuration. Will Fatal out of the
// program if there is an error in setting up connections to databases or reading
// the config file
func NewArchiver(c *Config) (a *Archiver) {
	var (
		mdStore MetadataStore
		tsStore TimeseriesStore
		pm      permissionsManager
	)

	a = &Archiver{}

	a.enforceKeys = c.Archiver.EnforceKeys

	switch *c.Archiver.MetadataStore {
	case "mongo":
		mongoaddr, err := net.ResolveTCPAddr("tcp4", *c.Mongo.Address+":"+*c.Mongo.Port)
		if err != nil {
			log.Fatalf("Error parsing Mongo address: %v", err)
		}
		config := &mongoConfig{
			address:     mongoaddr,
			enforceKeys: c.Archiver.EnforceKeys,
		}
		mdStore = newMongoStore(config)
		pm = newMongoPermissionsManager(config)
	default:
		log.Fatalf(*c.Archiver.MetadataStore, " is not a recognized metadata store")
	}

	a.mdStore = mdStore
	a.pm = pm

	switch *c.Archiver.TimeseriesStore {
	case "quasar":
		qsraddr, err := net.ResolveTCPAddr("tcp4", *c.Quasar.Address+":"+*c.Quasar.Port)
		if err != nil {
			log.Fatalf("Error parsing Quasar address: %v", err)
		}
		config := &quasarConfig{
			addr:           qsraddr,
			mdStore:        a.mdStore,
			maxConnections: *c.Archiver.MaxConnections,
		}
		tsStore = newQuasarDB(config)
	case "btrdb":
		btrdbaddr, err := net.ResolveTCPAddr("tcp4", *c.BtrDB.Address+":"+*c.BtrDB.Port)
		if err != nil {
			log.Fatalf("Error parsing BtrDB address: %v", err)
		}
		config := &btrdbConfig{
			addr:           btrdbaddr,
			mdStore:        a.mdStore,
			maxConnections: *c.Archiver.MaxConnections,
		}
		tsStore = newBtrDB(config)
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
func (a *Archiver) AddData(msg *common.SmapMessage, ephkey common.EphemeralKey) (err error) {
	if a.enforceKeys && !a.pm.ValidEphemeralKey(ephkey) {
		return fmt.Errorf("Ephemeral key %v is not valid", ephkey)
	}

	// save metadata
	err = a.mdStore.SaveTags(msg)
	if err != nil {
		return err
	}

	// fix inconsistencies
	var (
		uot common.UnitOfTime
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
func (a *Archiver) HandleQuery(querystring string, ephkey common.EphemeralKey) (QueryResult, error) {
	var result QueryResult
	if a.enforceKeys && !a.pm.ValidEphemeralKey(ephkey) {
		return result, fmt.Errorf("Ephemeral key %v is not valid", ephkey)
	}

	// parse the query
	parsed := a.qp.Parse(querystring)
	if parsed.Err != nil {
		return result, fmt.Errorf("Error (%v) in query \"%v\" (error at %v)\n", parsed.Err, querystring, parsed.ErrPos)
	}
	return a.evaluateQuery(parsed, ephkey)

}

func (a *Archiver) evaluateQuery(parsed *querylang.ParsedQuery, ephkey common.EphemeralKey) (QueryResult, error) {
	var result QueryResult
	// execute the query
	switch parsed.QueryType {
	case querylang.SELECT_TYPE:
		return a.handleSelect(parsed, ephkey)
	case querylang.DELETE_TYPE:
		return result, a.handleDelete(parsed, ephkey)
	case querylang.SET_TYPE:
		return result, a.handleSet(parsed, ephkey)
	case querylang.DATA_TYPE:
		return a.handleData(parsed, ephkey)
	default:
		return result, fmt.Errorf("Could not decide query type %v", parsed.Querystring)
	}
}

func (a *Archiver) HandleNewSubscriber(subscriber *Subscriber, querystring string, ephkey common.EphemeralKey) error {
	subscriber.query = a.qp.Parse(querystring)
	return a.broker.NewSubscriber(subscriber)
}

func (a *Archiver) handleSelect(parsed *querylang.ParsedQuery, ephkey common.EphemeralKey) (QueryResult, error) {
	//TODO: filter results by EphKey
	if parsed.Distinct {
		return a.mdStore.GetDistinct(parsed.Target[0], parsed.Where)
	}
	return a.mdStore.GetTags(parsed.Target, parsed.Where)
}

func (a *Archiver) handleData(parsed *querylang.ParsedQuery, ephkey common.EphemeralKey) (common.SmapMessageList, error) {
	var (
		result   = common.SmapMessageList{}
		readings []common.SmapNumbersResponse
	)

	uuids, err := a.mdStore.GetUUIDs(parsed.Where)
	if err != nil {
		return result, err
	}

	if parsed.Data.Limit.Streamlimit > 0 && len(uuids) > 0 {
		uuids = uuids[:parsed.Data.Limit.Streamlimit]
	}

	start := uint64(parsed.Data.Start.UnixNano())
	end := uint64(parsed.Data.End.UnixNano())

	switch parsed.Data.Dtype {
	case querylang.IN_TYPE:
		log.Debugf("Data in start %v end %v", start, end)
		if start < end {
			readings, err = a.tsStore.GetData(uuids, start, end)
		} else {
			readings, err = a.tsStore.GetData(uuids, end, start)
		}
	case querylang.BEFORE_TYPE:
		log.Debugf("Data before time %v (%v ns)", parsed.Data.Start, start)
		readings, err = a.tsStore.Prev(uuids, start)
	case querylang.AFTER_TYPE:
		log.Debugf("Data after time %v (%v ns)", parsed.Data.Start, start)
		readings, err = a.tsStore.Next(uuids, start)
	}

	for _, resp := range readings {
		if len(resp.Readings) > 0 {
			msg := &common.SmapMessage{UUID: resp.UUID}
			for _, rdg := range resp.Readings {
				rdg.ConvertTime(common.UnitOfTime(parsed.Data.Timeconv))
				msg.Readings = append(msg.Readings, rdg)
			}
			result = append(result, msg)
		}
	}
	log.Debugf("Returning %d readings", len(result))

	return result, nil
}

func (a *Archiver) handleDelete(parsed *querylang.ParsedQuery, ephkey common.EphemeralKey) error {
	if len(parsed.Target) > 0 {
		// remove tags
		log.Debugf("Removing tags %v docs where %v", parsed.Target, parsed.Where)
		return a.mdStore.RemoveTags(parsed.Target, parsed.Where)
	}
	log.Debugf("Removing all docs where %v", parsed.Where)
	return a.mdStore.RemoveDocs(parsed.Where)
}

func (a *Archiver) handleSet(parsed *querylang.ParsedQuery, ephkey common.EphemeralKey) error {
	log.Debugf("Apply updates %v where %v", parsed.Set, parsed.Where)
	if len(parsed.Set) == 0 {
		return nil
	}
	return a.mdStore.UpdateDocs(parsed.Set, parsed.Where)
}
