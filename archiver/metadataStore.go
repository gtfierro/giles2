package archiver

import (
	"gopkg.in/mgo.v2/bson"
)

type MetadataStore interface {
	GetUnitOfTime(uuid UUID) (UnitOfTime, error)
	GetStreamType(uuid UUID) (StreamType, error)
	GetUnitOfMeasure(uuid UUID) (string, error)

	GetTags(tags []string, where bson.M) (SmapMessageList, error)
	GetDistinct(tag string, where bson.M) (DistinctResult, error)
	GetUUIDs(where bson.M) ([]UUID, error)

	SaveTags(msg *SmapMessage) error

	UpdateDocs(updates, where bson.M) error

	//TODO: add feedback on how many docs changed/removed/etc. This requires a specialized struct
	RemoveTags(tags []string, where bson.M) error
	RemoveDocs(where bson.M) error
}
