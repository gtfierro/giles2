package archiver

// mongo provider for metadata store
import (
	"fmt"
	"github.com/karlseguin/ccache"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
	"sync/atomic"
	"time"
)

// default select clause to ignore internal variables
var ignoreDefault = bson.M{"_id": 0, "_api": 0}

type mongoStore struct {
	session     *mgo.Session
	db          *mgo.Database
	metadata    *mgo.Collection
	enforceKeys bool

	pool *mongoConnectionPool

	// caches for commonly queries items
	// UnitOfTime cache
	uotCache *ccache.Cache
	// UnitofMeasure cache
	uomCache *ccache.Cache
	// StreamType cache
	stCache *ccache.Cache
	// UUID cache
	uuidCache *ccache.Cache
	// expiry time for cache entries before they are automatically purged
	cacheExpiry time.Duration
}

type mongoConfig struct {
	address     *net.TCPAddr
	enforceKeys bool
}

func newMongoStore(c *mongoConfig) *mongoStore {
	var err error
	m := &mongoStore{}
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

	m.pool = newMongoConnectionPool(m.session, m.metadata, 20)

	// configure the unitoftime cache
	m.uotCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.uomCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.stCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.uuidCache = ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	m.cacheExpiry = 10 * time.Minute
	return m
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
	item, err := m.uotCache.Fetch(string(uuid), m.cacheExpiry, func() (uot interface{}, err error) {
		var (
			res interface{}
			c   int
		)
		uot = UOT_S
		query := m.metadata.Find(bson.M{"uuid": uuid}).Select(bson.M{"Properties.UnitofTime": 1})
		if c, err = query.Count(); err != nil {
			return
		} else if c == 0 {
			err = fmt.Errorf("no stream named %v", uuid)
			return
		}
		err = query.One(&res)
		if props, found := res.(bson.M)["Properties"]; found {
			if entry, found := props.(bson.M)["UnitofTime"]; found {
				uot = UnitOfTime(entry.(int))
			}
		}
		return
	})
	if item != nil && err == nil {
		return item.Value().(UnitOfTime), err
	}
	return UOT_S, err
}

func (m *mongoStore) GetStreamType(uuid UUID) (StreamType, error) {
	item, err := m.stCache.Fetch(string(uuid), m.cacheExpiry, func() (entry interface{}, err error) {
		var (
			res interface{}
			c   int
		)
		entry = NUMERIC_STREAM
		query := m.metadata.Find(bson.M{"uuid": uuid}).Select(bson.M{"Properties.StreamType": 1})
		if c, err = query.Count(); err != nil {
			return
		} else if c == 0 {
			err = fmt.Errorf("no stream named %v", uuid)
			return
		}
		err = query.One(&res)
		if props, found := res.(bson.M)["Properties"]; found {
			if entry, found := props.(bson.M)["StreamType"]; found {
				entry = StreamType(entry.(int))
			}
		}
		return
	})
	if item != nil && err == nil {
		return item.Value().(StreamType), err
	}
	return NUMERIC_STREAM, err
}

func (m *mongoStore) GetUnitOfMeasure(uuid UUID) (string, error) {
	item, err := m.uotCache.Fetch(string(uuid), m.cacheExpiry, func() (entry interface{}, err error) {
		var (
			res interface{}
			c   int
		)
		entry = ""
		query := m.metadata.Find(bson.M{"uuid": uuid}).Select(bson.M{"Properties.UnitofMeasure": 1})
		if c, err = query.Count(); err != nil {
			err = err
			return
		} else if c == 0 {
			err = fmt.Errorf("no stream named %v", uuid)
			return
		}
		err = query.One(&res)
		if props, found := res.(bson.M)["Properties"]; found {
			entry = props.(bson.M)["UnitofMeasure"]
		}
		return
	})
	if item != nil && err == nil {
		return item.Value().(string), err
	}
	return "", err
}

// Retrieves all tags in the provided list that match the provided where clause.
func (m *mongoStore) GetTags(tags []string, where bson.M) (SmapMessageList, error) {
	var (
		staged      *mgo.Query
		selectTags  bson.M
		whereClause bson.M
		x           []bson.M
	)
	if len(where) != 0 {
		whereClause = make(bson.M)
		for wk, wv := range where {
			whereClause[fixMongoKey(wk)] = wv
		}
	}
	staged = m.metadata.Find(whereClause)
	if len(tags) == 0 { // select all
		selectTags = bson.M{"_id": 0, "_api": 0}
	} else {
		selectTags = bson.M{"_id": 0}
		for _, tag := range tags {
			selectTags[fixMongoKey(tag)] = 1
		}
	}
	log.Debug("select Tags %v", selectTags)
	err := staged.Select(selectTags).All(&x)
	// trim down empty rows
	filtered := x[:0]
	for _, doc := range x {
		if len(doc) != 0 {
			filtered = append(filtered, doc)
		}
	}
	return SmapMessageListFromBson(filtered), err
}

func (m *mongoStore) GetDistinct(tag string, where bson.M) (SmapMessageList, error) {
	var (
		strResult   []string
		result      SmapMessageList
		whereClause bson.M
		fixedTag    = fixMongoKey(tag)
	)
	if len(where) != 0 {
		whereClause = make(bson.M)
		for wk, wv := range where {
			whereClause[fixMongoKey(wk)] = wv
		}
	}
	err := m.metadata.Find(where).Distinct(fixedTag, &strResult)
	result = SmapMessageList{}
	i := 0
	for _, val := range strResult {
		if len(val) == 0 {
			continue
		}
		result = append(result, &SmapMessage{})
		result[i].AddTag(fixedTag, val)
		i += 1
	}
	return result, err
}

func (m *mongoStore) GetUUIDs(where bson.M) ([]UUID, error) {
	var results []UUID
	var x []bson.M
	var whereClause bson.M
	if len(where) != 0 {
		whereClause = make(bson.M)
		for wk, wv := range where {
			whereClause[fixMongoKey(wk)] = wv
		}
	}
	selectClause := bson.M{"_id": 0, "uuid": 1}
	err := m.metadata.Find(where).Select(selectClause).All(&x)
	results = make([]UUID, len(x))
	for i, doc := range x {
		results[i] = UUID(doc["uuid"].(string))
	}
	return results, err
}

func (m *mongoStore) SaveTags(msg *SmapMessage) error {
	// if the message has no metadata and is already in cache, then skip writing
	if !msg.HasMetadata() && m.uuidCache.Get(string(msg.UUID)) != nil {
		return nil
	}
	// save to the metadata database
	_, err := m.metadata.Upsert(bson.M{"uuid": msg.UUID}, bson.M{"$set": msg.ToBson()})
	// and save to the uuid cache
	m.uuidCache.Set(string(msg.UUID), struct{}{}, m.cacheExpiry)
	//TODO: invalidate special properties keys in the cache or reset them to latest value
	return err
}

func (m *mongoStore) RemoveTags(tags []string, where bson.M) error {
	return nil
}

func (m *mongoStore) RemoveDocs(where bson.M) error {
	return nil
}

type mongoSession struct {
	s *mgo.Session
	c *mgo.Collection
}

// connection pool for MongoStore
type mongoConnectionPool struct {
	s     *mgo.Session
	c     *mgo.Collection
	pool  chan *mongoSession
	count int64
}

func newMongoConnectionPool(session *mgo.Session, collection *mgo.Collection, maxConnections int) *mongoConnectionPool {
	return &mongoConnectionPool{s: session, c: collection, pool: make(chan *mongoSession, maxConnections), count: 0}
}

func (mpool *mongoConnectionPool) get() *mongoSession {
	var s *mongoSession
	select {
	case s = <-mpool.pool:
	default:
		s = &mongoSession{s: mpool.s.Copy()}
		s.c = mpool.c.With(s.s)
		atomic.AddInt64(&mpool.count, 1)
		//log.Info("Creating new Mongo connection in pool (%v)", mpool.count)
	}
	return s
}

func (mpool *mongoConnectionPool) put(s *mongoSession) {
	select {
	case mpool.pool <- s:
	default:
		s.s.Close()
		atomic.AddInt64(&mpool.count, -1)
		//log.Info("Releasing connection in pool, now %v", mpool.count)
	}
}
