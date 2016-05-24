package archiver

import (
	"github.com/gtfierro/giles2/common"
	"gopkg.in/mgo.v2/bson"
)

type MetadataStore interface {
	GetUnitOfTime(uuid common.UUID) (common.UnitOfTime, error)
	GetStreamType(uuid common.UUID) (common.StreamType, error)
	GetUnitOfMeasure(uuid common.UUID) (string, error)

	GetTags(tags []string, where bson.M) (common.SmapMessageList, error)
	GetDistinct(tag string, where bson.M) (common.DistinctResult, error)
	GetUUIDs(where bson.M) ([]common.UUID, error)

	SaveTags(msg *common.SmapMessage) error

	UpdateDocs(updates, where bson.M) error

	//TODO: add feedback on how many docs changed/removed/etc. This requires a specialized struct
	RemoveTags(tags []string, where bson.M) error
	RemoveDocs(where bson.M) error
}
