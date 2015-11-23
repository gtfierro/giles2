//go:generate go tool yacc -o query.go -p SQ query.y
package archiver

import (
	"fmt"
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
	txc *transactionCoalescer
	qp  *queryProcessor
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
			log.Fatal("Error parsing Mongo address: %v", err)
		}
		config := &mongoConfig{
			address:     mongoaddr,
			enforceKeys: c.Archiver.EnforceKeys,
		}
		mdStore = newMongoStore(config)
		pm = newMongoPermissionsManager(config)
	default:
		log.Fatal(*c.Archiver.MetadataStore, " is not a recognized metadata store")
	}

	a.mdStore = mdStore
	a.pm = pm

	switch *c.Archiver.TimeseriesStore {
	case "quasar":
		qsraddr, err := net.ResolveTCPAddr("tcp4", *c.Quasar.Address+":"+*c.Quasar.Port)
		if err != nil {
			log.Fatal("Error parsing Quasar address: %v", err)
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
			log.Fatal("Error parsing BtrDB address: %v", err)
		}
		config := &btrdbConfig{
			addr:           btrdbaddr,
			mdStore:        a.mdStore,
			maxConnections: *c.Archiver.MaxConnections,
		}
		tsStore = newBtrDB(config)
	default:
		log.Fatal(*c.Archiver.TimeseriesStore, " is not a recognized timeseries store")
	}

	a.tsStore = tsStore

	a.txc = newTransactionCoalescer(&a.tsStore, &a.mdStore)
	a.qp = &queryProcessor{a}

	a.metrics = make(metricMap)
	a.metrics.addMetric("adds")

	a.startReport()
	return
}

func (a *Archiver) startReport() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		for {
			log.Info("Adds:%d", a.metrics["adds"].GetAndReset())
			<-t.C
		}
	}()
}

// Takes an incoming SmapMessage object (from a client) and does the following:
//  - Checks the incoming message against the ApiKey to verify it is valid to write
//  - Saves the attached metadata (if any) to the metadata store
//  - Reevaluates any dynamic subscriptions and pushes to republish clients
//  - Saves the attached readings (if any) to the timeseries database
// These last 2 steps happen in parallel
func (a *Archiver) AddData(msg *SmapMessage, ephkey EphemeralKey) (err error) {
	if a.enforceKeys && !a.pm.ValidEphemeralKey(ephkey) {
		return fmt.Errorf("Ephemeral key %v is not valid", ephkey)
	}

	// save metadata
	err = a.mdStore.SaveTags(msg)
	if err != nil {
		return err
	}

	//TODO reevaluate subscriptions, push to clients
	//save timeseries data
	a.txc.AddSmapMessage(msg)
	a.metrics["adds"].Mark(1)
	//a.tsStore.AddMessage(msg)
	return nil
}

// Need to think about how to transfer the results of these queries to the handlers that are
// asking for them and need to transform them into their own internal representations (e.g.
// JSON, MsgPack, etc). What are the data patterns we are seeing?
// Basically everything fits into SmapMessageList
func (a *Archiver) HandleQuery(querystring string, ephkey EphemeralKey) (SmapMessageList, error) {
	var (
		err    error
		result SmapMessageList
	)

	if a.enforceKeys && !a.pm.ValidEphemeralKey(ephkey) {
		return result, fmt.Errorf("Ephemeral key %v is not valid", ephkey)
	}

	// parse the query
	parsed := a.qp.Parse(querystring)
	if parsed.err != nil {
		return result, fmt.Errorf("Error (%v) in query \"%v\" (error at %v)\n", parsed.err, querystring, parsed.errPos)
	}

	// execute the query
	switch parsed.queryType {
	case SELECT_TYPE:
		return a.handleSelect(parsed, ephkey)
	case DELETE_TYPE:
		return result, a.handleDelete(parsed, ephkey)
	case SET_TYPE:
		return result, a.handleSet(parsed, ephkey)
	case DATA_TYPE:
		return a.handleData(parsed, ephkey)
	default:
		return result, fmt.Errorf("Could not decide query type %v", querystring)
	}

	return result, err
}

func (a *Archiver) HandleNewSubscriber(subscriber Subscriber, querystring string, ephkey EphemeralKey) error {
	// first evaluate the query when we receive a subscriber
	result, err := a.HandleQuery(querystring, ephkey)
	// if there was an error during that evaluation, notify the client and return
	if err != nil {
		subscriber.SendError(err)
		return err
	}
	// make sure the client gets the first result
	subscriber.BlockSend(result)
	subscriber.query = a.qp.Parse(querystring)

	// TODO: handle the actual subscription

	return nil
}

func (a *Archiver) handleSelect(parsed *parsedQuery, ephkey EphemeralKey) (SmapMessageList, error) {
	//TODO: filter results by EphKey
	if parsed.distinct {
		return a.mdStore.GetDistinct(parsed.target[0], parsed.where)
	}
	return a.mdStore.GetTags(parsed.target, parsed.where)
}

func (a *Archiver) handleData(parsed *parsedQuery, ephkey EphemeralKey) (SmapMessageList, error) {
	var (
		result   SmapMessageList
		readings []SmapNumbersResponse
	)

	uuids, err := a.mdStore.GetUUIDs(parsed.where)
	if err != nil {
		return result, err
	}

	if parsed.data.limit.streamlimit > 0 && len(uuids) > 0 {
		uuids = uuids[:parsed.data.limit.streamlimit]
	}

	start := uint64(parsed.data.start.UnixNano())
	end := uint64(parsed.data.end.UnixNano())

	switch parsed.data.dtype {
	case IN_TYPE:
		log.Debug("Data in start %v end %v", start, end)
		if start < end {
			readings, err = a.tsStore.GetData(uuids, start, end)
		} else {
			readings, err = a.tsStore.GetData(uuids, end, start)
		}
	case BEFORE_TYPE:
		log.Debug("Data before time %v (%v ns)", parsed.data.start, start)
		readings, err = a.tsStore.Prev(uuids, start)
	case AFTER_TYPE:
		log.Debug("Data after time %v (%v ns)", parsed.data.start, start)
		readings, err = a.tsStore.Next(uuids, start)
	}

	log.Debug("readings %v", readings)
	for _, resp := range readings {
		if len(resp.Readings) > 0 {
			msg := &SmapMessage{UUID: resp.UUID}
			for _, rdg := range resp.Readings {
				rdg.ConvertTime(UOT_NS, parsed.data.timeconv)
				msg.Readings = append(msg.Readings, rdg)
			}
			result = append(result, msg)
		}
	}
	log.Debug("readings %v", result)

	return result, nil
}

func (a *Archiver) handleDelete(parsed *parsedQuery, ephkey EphemeralKey) error {
	if len(parsed.target) > 0 {
		// remove tags
		log.Debug("Removing tags %v docs where %v", parsed.target, parsed.where)
		return a.mdStore.RemoveTags(parsed.target, parsed.where)
	}
	log.Debug("Removing all docs where %v", parsed.where)
	return a.mdStore.RemoveDocs(parsed.where)
}

func (a *Archiver) handleSet(parsed *parsedQuery, ephkey EphemeralKey) error {
	log.Debug("Apply updates %v where %v", parsed.set, parsed.where)
	if len(parsed.set) == 0 {
		return nil
	}
	return a.mdStore.UpdateDocs(parsed.set, parsed.where)
}
