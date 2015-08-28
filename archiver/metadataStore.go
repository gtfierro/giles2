package archiver

import (
	"gopkg.in/mgo.v2/bson"
)

type MetadataStore interface {
	GetUnitOfTime(uuid UUID) UnitOfTime
	GetStreamType(uuid UUID) StreamType
	GetUnitOfMeasure(uuid UUID) string

	GetTags(tags []string, where bson.M) (interface{}, error)
	GetDistinct(tag string, where bson.M) (interface{}, error)
	GetUUIDs(where bson.M) ([]UUID, error)

	SaveTags(msg *SmapMessage) error

	//TODO: add feedback on how many docs changed/removed/etc. This requires
	//      a specialized struct
	RemoveTags(tags []string, where bson.M) error
	RemoveDocs(where bson.M) error
}
