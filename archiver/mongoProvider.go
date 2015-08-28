package archiver

// mongo provider for metadata store
import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
)

// default select clause to ignore internal variables
var ignoreDefault = bson.M{"_id": 0, "_api": 0}

type mongoStore struct {
	session     *mgo.Session
	db          *mgo.Database
	metadata    *mgo.Collection
	enforceKeys bool
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

func (m *mongoStore) GetUnitOfTime(uuid UUID) UnitOfTime {
	return UOT_MS
}

func (m *mongoStore) GetStreamType(uuid UUID) StreamType {
	return NUMERIC_STREAM
}

func (m *mongoStore) GetUnitOfMeasure(uuid UUID) string {
	return ""
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

func (m *mongoStore) RemoveTags(tags []string, where bson.M) error {
	return nil
}

func (m *mongoStore) RemoveDocs(where bson.M) error {
	return nil
}
