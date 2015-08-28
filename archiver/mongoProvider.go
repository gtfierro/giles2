package archiver

// mongo provider for metadata store
import (
	"github.com/karlseguin/ccache"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
	"time"
)

// default select clause to ignore internal variables
var ignoreDefault = bson.M{"_id": 0, "_api": 0}

type mongoStore struct {
	session     *mgo.Session
	db          *mgo.Database
	metadata    *mgo.Collection
	enforceKeys bool

	// caches for commonly queries items
	// UnitOfTime cache
	uotCache *ccache.Cache
	// UnitofMeasure cache
	uomCache *ccache.Cache
	// StreamType cache
	stCache *ccache.Cache
	// expiry time for cache entries before they are automatically purged
	cacheExpiry time.Duration
}

type mongoConfig struct {
	address     *net.TCPAddr
	enforceKeys bool
}

func newMongoStore(c *mongoConfig) (m *mongoStore) {
	var err error
	log.Notice("Connecting to MongoDB at %v...", c.address.String())
	m.session, err = mgo.Dial(c.address.String())
	if err != nil {
		log.Critical("Could not connect to MongoDB: %v", err)
		return nil
	}
	log.Notice("...connected!")
	// fetch/create collections and db reference
	m.db = m.session.DB("archiver")
	m.metadata = m.db.C("metadata")

	// add indexes. This will fail Fatal
	m.addIndexes()

	m.enforceKeys = c.enforceKeys

	// configure the unitoftime cache
	m.uotCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.uomCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.stCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.cacheExpiry = 10 * time.Minute
	return
}

func (m *mongoStore) addIndexes() {
	var err error
	// create indexes
	index := mgo.Index{
		Key:        []string{"uuid"},
		Unique:     true,
		DropDups:   false,
		Background: false,
		Sparse:     false,
	}
	err = m.metadata.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on metadata.uuid (%v)", err)
	}

	//TODO: check capitalization UnitofTime vs UnitOfTime. same with UnitofMeasure
	index.Key = []string{"Properties.UnitofTime"}
	index.Unique = false
	err = m.metadata.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on metadata.properties.unitofmeasure (%v)", err)
	}

	index.Key = []string{"Properties.UnitofMeasure"}
	err = m.metadata.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on metadata.properties.unitofmeasure (%v)", err)
	}

	index.Key = []string{"Properties.StreamType"}
	err = m.metadata.EnsureIndex(index)
	if err != nil {
		log.Fatalf("Could not create index on metadata.properties.streamtype (%v)", err)
	}
}

func (m *mongoStore) GetUnitOfTime(uuid UUID) (UnitOfTime, error) {
	item, err := m.uotCache.Fetch(string(uuid), m.cacheExpiry, func() (res interface{}, err error) {
		err = m.metadata.Find(bson.M{"uuid": uuid}).
			Select(bson.M{"Properties.UnitofTime": 1}).
			One(&res)
		return
	})
	if item != nil && err == nil {
		return item.Value().(UnitOfTime), err
	}
	return UOT_S, err
}

func (m *mongoStore) GetStreamType(uuid UUID) (StreamType, error) {
	item, err := m.stCache.Fetch(string(uuid), m.cacheExpiry, func() (res interface{}, err error) {
		err = m.metadata.Find(bson.M{"uuid": uuid}).
			Select(bson.M{"Properties.StreamType": 1}).
			One(&res)
		return
	})
	if item != nil && err == nil {
		return item.Value().(StreamType), err
	}
	return NUMERIC_STREAM, err
}

func (m *mongoStore) GetUnitOfMeasure(uuid UUID) (string, error) {
	item, err := m.uotCache.Fetch(string(uuid), m.cacheExpiry, func() (res interface{}, err error) {
		err = m.metadata.Find(bson.M{"uuid": uuid}).
			Select(bson.M{"Properties.UnitofMeasure": 1}).
			One(&res)
		return
	})
	if item != nil && err == nil {
		return item.Value().(string), err
	}
	return "", err
}

func (m *mongoStore) GetTags(tags []string, where bson.M) (interface{}, error) {
	return nil, nil
}

func (m *mongoStore) GetDistinct(tag string, where bson.M) (interface{}, error) {
	return nil, nil
}

func (m *mongoStore) GetUUIDs(where bson.M) ([]UUID, error) {
	return nil, nil
}

func (m *mongoStore) SaveTags(msg *SmapMessage) error {
	return nil
}

func (m *mongoStore) SaveTagsBulk(msgs []*SmapMessage) error {
	return nil
}

func (m *mongoStore) RemoveTags(tags []string, where bson.M) error {
	return nil
}

func (m *mongoStore) RemoveDocs(where bson.M) error {
	return nil
}
