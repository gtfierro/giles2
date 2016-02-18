package querylang

import (
	"github.com/gtfierro/giles2/archiver/internal/unitoftime"
	"time"
)

type DataQuery struct {
	Dtype    DataQueryType
	Start    time.Time
	End      time.Time
	Limit    Limit
	Timeconv unitoftime.UnitOfTime
}

type Limit struct {
	Limit       int64
	Streamlimit int64
}

type DataQueryType uint

const (
	IN_TYPE DataQueryType = iota
	BEFORE_TYPE
	AFTER_TYPE
)

func (dt DataQueryType) String() string {
	ret := ""
	switch dt {
	case IN_TYPE:
		ret = "in"
	case BEFORE_TYPE:
		ret = "before"
	case AFTER_TYPE:
		ret = "after"
	}
	return ret
}
