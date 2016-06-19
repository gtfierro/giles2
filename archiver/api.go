package archiver

import (
	"github.com/gtfierro/giles2/common"
)

type SelectParams struct {
	Tags  []string
	Where common.Dict
}

type DistinctParams struct {
	Tag   string
	Where common.Dict
}

type DataParams struct {
	// clause to evaluate for which streams to fetch.
	// If this is empty, uses the UUIDs
	Where common.Dict
	// UUIDs from which to fetch data. Superceded by Where
	UUIDs []common.UUID
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
	ConvertToUnit common.UnitOfTime
}

func (a *Archiver) SelectTags(params *SelectParams) (QueryResult, error) {
	return a.mdStore.GetTags(params.Tags, params.Where.ToBson())
}

func (a *Archiver) DistinctTag(params *DistinctParams) (QueryResult, error) {
	return a.mdStore.GetDistinct(params.Tag, params.Where.ToBson())
}

// selects data for the matching streams within the range given
// by Begin/End
func (a *Archiver) SelectDataRange(params *DataParams) (common.SmapMessageList, error) {
	var (
		err      error
		result   = common.SmapMessageList{}
		readings []common.SmapNumbersResponse
	)
	if err = a.prepareDataParams(params); err != nil {
		return result, err
	}

	// switch order so its consistent
	if params.End < params.Begin {
		params.Begin, params.End = params.End, params.Begin
	}

	// fetch readings
	readings, err = a.tsStore.GetData(params.UUIDs, params.Begin, params.End)
	if err != nil {
		return result, err
	}

	// convert readings into the correct unit of time
	result = a.packResults(params, readings)

	return result, nil
}

// selects the data point most immediately before the Start parameter for all matching streams
func (a *Archiver) SelectDataBefore(params *DataParams) (result common.SmapMessageList, err error) {
	var readings []common.SmapNumbersResponse
	if err = a.prepareDataParams(params); err != nil {
		return
	}
	readings, err = a.tsStore.Prev(params.UUIDs, params.Begin)
	result = a.packResults(params, readings)
	return
}

// selects the data point most immediately after the Start parameter for all matching streams
func (a *Archiver) SelectDataAfter(params *DataParams) (result common.SmapMessageList, err error) {
	var readings []common.SmapNumbersResponse
	if err = a.prepareDataParams(params); err != nil {
		return
	}
	readings, err = a.tsStore.Next(params.UUIDs, params.Begin)
	result = a.packResults(params, readings)
	return
}

func (a *Archiver) DeleteData(params *DataParams) (err error) {
	if err = a.prepareDataParams(params); err != nil {
		return
	}
	// switch order so its consistent
	if params.End < params.Begin {
		params.Begin, params.End = params.End, params.Begin
	}
	return a.tsStore.DeleteData(params.UUIDs, params.Begin, params.End)
}

func (a *Archiver) prepareDataParams(params *DataParams) (err error) {
	// parse and evaluate the where clause if we need to
	if len(params.Where) > 0 {
		params.UUIDs, err = a.mdStore.GetUUIDs(params.Where.ToBson())
		if err != nil {
			return err
		}
	}

	// apply the streamlimit if it exists
	if params.StreamLimit > 0 && len(params.UUIDs) > params.StreamLimit {
		params.UUIDs = params.UUIDs[:params.StreamLimit]
	}

	// make sure that Begin/End are both in nanoseconds
	if begin_uot := common.GuessTimeUnit(params.Begin); begin_uot != common.UOT_NS {
		params.Begin, err = common.ConvertTime(params.Begin, begin_uot, common.UOT_NS)
		if err != nil {
			return err
		}
	}
	if end_uot := common.GuessTimeUnit(params.End); end_uot != common.UOT_NS {
		params.End, err = common.ConvertTime(params.End, end_uot, common.UOT_NS)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Archiver) packResults(params *DataParams, readings []common.SmapNumbersResponse) common.SmapMessageList {
	var result = common.SmapMessageList{}
	for _, resp := range readings {
		if len(resp.Readings) > 0 {
			msg := &common.SmapMessage{UUID: resp.UUID}
			for _, rdg := range resp.Readings {
				rdg.ConvertTime(common.UnitOfTime(params.ConvertToUnit))
				msg.Readings = append(msg.Readings, rdg)
			}
			// apply data limit if exists
			if params.DataLimit > 0 && len(msg.Readings) > params.DataLimit {
				msg.Readings = msg.Readings[:params.DataLimit]
			}
			result = append(result, msg)
		}
	}
	log.Debugf("Returning %d readings", len(result))
	return result
}