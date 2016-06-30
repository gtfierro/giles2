package archiver

import (
	"fmt"
	btrdb "github.com/SoftwareDefinedBuildings/btrdb-go"
	"github.com/gtfierro/giles2/common"
	uuid "github.com/pborman/uuid"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

type btrdbConfig struct {
	addr           *net.TCPAddr
	mdStore        MetadataStore
	maxConnections int
}

var BtrDBReadErr = errors.New("Error receiving data from BtrDB")

const MaximumTime = (48 << 56)

type btrIface struct {
	addr    *net.TCPAddr
	mdStore MetadataStore
	client  *btrdb.BTrDBConnection
	clients []*btrdb.BTrDBConnection
	sync.RWMutex
}

func newBtrIface(c *btrdbConfig) *btrIface {
	rand.Seed(time.Now().UnixNano())
	var err error
	b := &btrIface{
		addr:    c.addr,
		mdStore: c.mdStore,
		clients: make([]*btrdb.BTrDBConnection, 10),
	}
	log.Noticef("Connecting to BtrDB at %v...", b.addr.String())

	if b.client, err = btrdb.NewBTrDBConnection(c.addr.String()); err != nil {
		log.Fatalf("Could not connect to btrdb: %v", err)
	}

	for i := 0; i < 10; i++ {
		c, err := btrdb.NewBTrDBConnection(c.addr.String())
		if err != nil {
			log.Fatalf("Could not connect to btrdb: %v", err)
		}
		b.clients[i] = c
	}

	return b
}

func (bdb *btrIface) getClient() *btrdb.BTrDBConnection {
	bdb.RLock()
	defer bdb.RUnlock()
	return bdb.clients[rand.Intn(10)]
}

func (bdb *btrIface) AddMessage(msg *common.SmapMessage) error {
	var (
		parsed_uuid uuid.UUID
		err         error
	)

	// turn the string representation into UUID bytes
	parsed_uuid = uuid.Parse(string(msg.UUID))

	records := make([]btrdb.StandardValue, len(msg.Readings))
	for i, rdg := range msg.Readings {
		rdg.ConvertTime(common.UOT_NS)
		num, ok := rdg.GetValue().(float64)
		if !ok {
			return fmt.Errorf("Bad number in message %v %v", msg.UUID, rdg)
		}
		records[i] = btrdb.StandardValue{Time: int64(rdg.GetTime()), Value: num}
	}
	client := bdb.getClient()
	c, err := client.InsertValues(parsed_uuid, records, false)
	<-c // wait for response
	return err
}

func (bdb *btrIface) numberResponseFromChan(c chan btrdb.StandardValue) common.SmapNumbersResponse {
	var sr = common.SmapNumbersResponse{
		Readings: []*common.SmapNumberReading{},
	}
	for val := range c {
		sr.Readings = append(sr.Readings, &common.SmapNumberReading{Time: uint64(val.Time), Value: val.Value, UoT: common.UOT_NS})
	}
	return sr
}

func (bdb *btrIface) statisticalResponseFromChan(c chan btrdb.StatisticalValue) common.StatisticalNumbersResponse {
	var sr = common.StatisticalNumbersResponse{
		Readings: []*common.StatisticalNumberReading{},
	}
	for val := range c {
		sr.Readings = append(sr.Readings, &common.StatisticalNumberReading{Time: uint64(val.Time), Count: val.Count, Min: val.Min, Max: val.Max, Mean: val.Mean, UoT: common.UOT_NS})
	}
	return sr
}

func (bdb *btrIface) queryNearestValue(uuids []common.UUID, start uint64, backwards bool) ([]common.SmapNumbersResponse, error) {
	var ret = make([]common.SmapNumbersResponse, len(uuids))
	var results []chan btrdb.StandardValue
	client := bdb.getClient()
	for _, uu := range uuids {
		uuid := uuid.Parse(string(uu))
		values, _, _, err := client.QueryNearestValue(uuid, int64(start), backwards, 0)
		if err != nil {
			return ret, err
		}
		results = append(results, values)
	}
	for i, c := range results {
		sr := bdb.numberResponseFromChan(c)
		sr.UUID = uuids[i]
		ret[i] = sr
	}
	return ret, nil
}

func (bdb *btrIface) Prev(uuids []common.UUID, start uint64) ([]common.SmapNumbersResponse, error) {
	return bdb.queryNearestValue(uuids, start, true)
}

func (bdb *btrIface) Next(uuids []common.UUID, start uint64) ([]common.SmapNumbersResponse, error) {
	return bdb.queryNearestValue(uuids, start, false)
}

func (bdb *btrIface) GetData(uuids []common.UUID, start, end uint64) ([]common.SmapNumbersResponse, error) {
	var ret = make([]common.SmapNumbersResponse, len(uuids))
	var results []chan btrdb.StandardValue
	client := bdb.getClient()
	for _, uu := range uuids {
		uuid := uuid.Parse(string(uu))
		values, _, _, err := client.QueryStandardValues(uuid, int64(start), int64(end), 0)
		if err != nil {
			return ret, err
		}
		results = append(results, values)
	}
	for i, c := range results {
		sr := bdb.numberResponseFromChan(c)
		sr.UUID = uuids[i]
		ret[i] = sr
	}
	return ret, nil
}

func (bdb *btrIface) StatisticalData(uuids []common.UUID, pointWidth int, start, end uint64) ([]common.StatisticalNumbersResponse, error) {
	var ret = make([]common.StatisticalNumbersResponse, len(uuids))
	var results []chan btrdb.StatisticalValue
	client := bdb.getClient()
	for _, uu := range uuids {
		uuid := uuid.Parse(string(uu))
		values, _, _, err := client.QueryStatisticalValues(uuid, int64(start), int64(end), uint8(pointWidth), 0)
		if err != nil {
			return ret, err
		}
		results = append(results, values)
	}
	for i, c := range results {
		sr := bdb.statisticalResponseFromChan(c)
		sr.UUID = uuids[i]
		ret[i] = sr
	}
	return ret, nil
}

func (bdb *btrIface) WindowData(uuids []common.UUID, width, start, end uint64) ([]common.StatisticalNumbersResponse, error) {
	var ret = make([]common.StatisticalNumbersResponse, len(uuids))
	var results []chan btrdb.StatisticalValue
	client := bdb.getClient()
	for _, uu := range uuids {
		uuid := uuid.Parse(string(uu))
		values, _, _, err := client.QueryWindowValues(uuid, int64(start), int64(end), width, 0, 0)
		if err != nil {
			return ret, err
		}
		results = append(results, values)
	}
	for i, c := range results {
		sr := bdb.statisticalResponseFromChan(c)
		sr.UUID = uuids[i]
		ret[i] = sr
	}
	return ret, nil
}

func (bdb *btrIface) DeleteData(uuids []common.UUID, start uint64, end uint64) error {
	client := bdb.getClient()
	for _, uu := range uuids {
		uuid := uuid.Parse(string(uu))
		if _, err := client.DeleteValues(uuid, int64(start), int64(end)); err != nil {
			return err
		}
	}
	return nil
}

func (bdb *btrIface) ValidTimestamp(time uint64, uot common.UnitOfTime) bool {
	var err error
	if uot != common.UOT_NS {
		time, err = common.ConvertTime(time, uot, common.UOT_NS)
	}
	return time >= 0 && time <= MaximumTime && err == nil
}
