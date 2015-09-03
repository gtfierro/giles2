package archiver

import (
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
	// transaction coalescer
	coalescer *coalescer
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
			log.Fatal("Error parsing Mongo address: %v", err)
		}
		config := &mongoConfig{
			address:     mongoaddr,
			enforceKeys: c.Archiver.EnforceKeys,
		}
		mdStore = newMongoStore(config)
	default:
		log.Fatal(*c.Archiver.MetadataStore, " is not a recognized metadata store")
	}

	a.mdStore = mdStore

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
	default:
		log.Fatal(*c.Archiver.TimeseriesStore, " is not a recognized timeseries store")
	}

	a.tsStore = tsStore

	a.coalescer = newCoalescer(a.tsStore, a.mdStore)

	a.metrics = make(metricMap)
	a.metrics.addMetric("adds")

	a.startReport()
	return
}

func (a *Archiver) startReport() {
	go func() {
		t := time.NewTicker(1 * time.Second)
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
func (a *Archiver) AddData(msg *SmapMessage, apikey ApiKey) (err error) {
	//TODO: check api key

	// save metadata
	err = a.mdStore.SaveTags(msg)
	if err != nil {
		return err
	}

	//TODO reevaluate subscriptions, push to clients
	//save timeseries data
	a.coalescer.add(msg)
	a.metrics["adds"].Mark(1)
	//a.tsStore.AddMessage(msg)
	return nil
}
