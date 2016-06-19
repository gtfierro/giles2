package common

import (
	"fmt"
)

type QueryParams interface {
	Dump() string
}

type TagParams struct {
	Tags  []string
	Where Dict
}

func (params TagParams) Dump() string {
	ret := fmt.Sprintf("SELECT\nTags:\n")
	for _, tag := range params.Tags {
		ret += fmt.Sprintf("-> %s\n", tag)
	}
	ret += fmt.Sprintf("WHERE\n%+v", params.Where)
	return ret
}

type DistinctParams struct {
	Tag   string
	Where Dict
}

func (params DistinctParams) Dump() string {
	ret := fmt.Sprintf("SELECT DISTINCT\nTag: %s\n", params.Tag)
	ret += fmt.Sprintf("WHERE\n%+v", params.Where)
	return ret
}

type DataParams struct {
	// clause to evaluate for which streams to fetch.
	// If this is empty, uses the UUIDs
	Where Dict
	// UUIDs from which to fetch data. Superceded by Where
	UUIDs []UUID
	// restrict the number of streams returned
	StreamLimit int
	// restrict the number of data points per stream returned.
	// Defaults to the most recent
	DataLimit int
	// time to start fetching data from (inclusive)
	Begin uint64
	// time to stop fetching data from (inclusive)
	End uint64
	// converts all readings to this unit of time when finished
	ConvertToUnit UnitOfTime
}

func (params DataParams) Dump() string {
	ret := fmt.Sprintf("DATA\n%d UUIDs\nWHERE:\n%+v", len(params.UUIDs), params.Where)
	ret += fmt.Sprintf("Begin: %d\nEnd: %d\n", params.Begin, params.End)
	ret += fmt.Sprintf("Convert to : %s", params.ConvertToUnit.String())
	return ret
}

type SetParams struct {
	Set   Dict
	Where Dict
}

func (params SetParams) Dump() string {
	return fmt.Sprintf("SET\n%+v\nWHERE:\n%+v", params.Set, params.Where)
}
