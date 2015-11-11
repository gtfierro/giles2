package archiver

import (
	capn "github.com/glycerine/go-capnproto"
	"io"
)

func (s *SmapMessage2) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	SmapMessage2GoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *SmapMessage2) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := ReadRootSmapMessage2Capn(capMsg)
	SmapMessage2CapnToGo(z, s)
	return nil
}

func DictCapnToGo(src DictEntryCapn_List, dest *Dict) *Dict {
	if dest == nil {
		dest = &Dict{}
	}
	for i := 0; i < src.Len(); i++ {
		entry := src.At(i)
		(*dest)[entry.Key()] = entry.Value()
	}
	return dest
}

func DictGoToCapn(seg *capn.Segment, src *Dict) DictEntryCapn_List {
	dest := NewDictEntryCapnList(seg, len(*src))
	plist := capn.PointerList(dest)
	i := 0
	for key, val := range *src {
		obj := NewDictEntryCapn(seg)
		obj.SetKey(key)
		obj.SetValue(val)
		plist.Set(i, capn.Object(obj))
		i++
	}
	return dest
}

func SmapMessage2CapnToGo(src SmapMessage2Capn, dest *SmapMessage2) *SmapMessage2 {
	if dest == nil {
		dest = &SmapMessage2{}
	}
	dest.Path = src.Path()
	dest.UUID = src.UUID()
	dest.Properties = SmapProperties2CapnToGo(src.Properties(), nil)
	dest.Actuator = *DictCapnToGo(src.Actuator(), nil)
	dest.Metadata = *DictCapnToGo(src.Metadata(), nil)

	var n int

	// Readings
	n = src.Readings().Len()
	dest.Readings = make([]*SmapNumberReading, n)
	for i := 0; i < n; i++ {
		dest.Readings[i] = SmapNumberReadingCapnToGo(src.Readings().At(i), nil)
	}

	return dest
}

func SmapMessage2GoToCapn(seg *capn.Segment, src *SmapMessage2) SmapMessage2Capn {
	dest := AutoNewSmapMessage2Capn(seg)
	dest.SetPath(src.Path)
	dest.SetUUID(src.UUID)
	dest.SetProperties(SmapProperties2GoToCapn(seg, src.Properties))
	dest.SetActuator(DictGoToCapn(seg, &src.Actuator))
	dest.SetMetadata(DictGoToCapn(seg, &src.Metadata))

	// Readings -> SmapNumberReadingCapn (go slice to capn list)
	if len(src.Readings) > 0 {
		typedList := NewSmapNumberReadingCapnList(seg, len(src.Readings))
		plist := capn.PointerList(typedList)
		i := 0
		for _, ele := range src.Readings {
			plist.Set(i, capn.Object(SmapNumberReadingGoToCapn(seg, ele)))
			i++
		}
		dest.SetReadings(typedList)
	}

	return dest
}

func (s *SmapNumberReading) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	SmapNumberReadingGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *SmapNumberReading) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := ReadRootSmapNumberReadingCapn(capMsg)
	SmapNumberReadingCapnToGo(z, s)
	return nil
}

func SmapNumberReadingCapnToGo(src SmapNumberReadingCapn, dest *SmapNumberReading) *SmapNumberReading {
	if dest == nil {
		dest = &SmapNumberReading{}
	}
	dest.Time = src.Time()
	dest.Value = src.Value()

	return dest
}

func SmapNumberReadingGoToCapn(seg *capn.Segment, src *SmapNumberReading) SmapNumberReadingCapn {
	dest := AutoNewSmapNumberReadingCapn(seg)
	dest.SetTime(src.Time)
	dest.SetValue(src.Value)

	return dest
}

func (s *SmapProperties2) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	SmapProperties2GoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *SmapProperties2) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := ReadRootSmapProperties2Capn(capMsg)
	SmapProperties2CapnToGo(z, s)
	return nil
}

func SmapProperties2CapnToGo(src SmapProperties2Capn, dest *SmapProperties2) *SmapProperties2 {
	if dest == nil {
		dest = &SmapProperties2{}
	}
	dest.UnitOfTime = src.UnitOfTime()
	dest.UnitOfMeasure = src.UnitOfMeasure()
	dest.StreamType = src.StreamType()

	return dest
}

func SmapProperties2GoToCapn(seg *capn.Segment, src *SmapProperties2) SmapProperties2Capn {
	dest := AutoNewSmapProperties2Capn(seg)
	if src != nil && !src.IsEmpty() {
		dest.SetUnitOfTime(src.UnitOfTime)
		dest.SetUnitOfMeasure(src.UnitOfMeasure)
		dest.SetStreamType(src.StreamType)
	}

	return dest
}
