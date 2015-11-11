package archiver

// AUTO GENERATED - DO NOT EDIT

import (
	C "github.com/glycerine/go-capnproto"
	"math"
	"unsafe"
)

type SmapMessage2Capn C.Struct

func NewSmapMessage2Capn(s *C.Segment) SmapMessage2Capn { return SmapMessage2Capn(s.NewStruct(0, 6)) }
func NewRootSmapMessage2Capn(s *C.Segment) SmapMessage2Capn {
	return SmapMessage2Capn(s.NewRootStruct(0, 6))
}
func AutoNewSmapMessage2Capn(s *C.Segment) SmapMessage2Capn {
	return SmapMessage2Capn(s.NewStructAR(0, 6))
}
func ReadRootSmapMessage2Capn(s *C.Segment) SmapMessage2Capn {
	return SmapMessage2Capn(s.Root(0).ToStruct())
}
func (s SmapMessage2Capn) Path() string     { return C.Struct(s).GetObject(0).ToText() }
func (s SmapMessage2Capn) SetPath(v string) { C.Struct(s).SetObject(0, s.Segment.NewText(v)) }
func (s SmapMessage2Capn) UUID() string     { return C.Struct(s).GetObject(1).ToText() }
func (s SmapMessage2Capn) SetUUID(v string) { C.Struct(s).SetObject(1, s.Segment.NewText(v)) }
func (s SmapMessage2Capn) Properties() SmapProperties2Capn {
	return SmapProperties2Capn(C.Struct(s).GetObject(2).ToStruct())
}
func (s SmapMessage2Capn) SetProperties(v SmapProperties2Capn) { C.Struct(s).SetObject(2, C.Object(v)) }
func (s SmapMessage2Capn) Actuator() DictEntryCapn_List {
	return DictEntryCapn_List(C.Struct(s).GetObject(3))
}
func (s SmapMessage2Capn) SetActuator(v DictEntryCapn_List) { C.Struct(s).SetObject(3, C.Object(v)) }
func (s SmapMessage2Capn) Metadata() DictEntryCapn_List {
	return DictEntryCapn_List(C.Struct(s).GetObject(4))
}
func (s SmapMessage2Capn) SetMetadata(v DictEntryCapn_List) { C.Struct(s).SetObject(4, C.Object(v)) }
func (s SmapMessage2Capn) Readings() SmapNumberReadingCapn_List {
	return SmapNumberReadingCapn_List(C.Struct(s).GetObject(5))
}
func (s SmapMessage2Capn) SetReadings(v SmapNumberReadingCapn_List) {
	C.Struct(s).SetObject(5, C.Object(v))
}

// capn.JSON_enabled == false so we stub MarshallJSON().
func (s SmapMessage2Capn) MarshalJSON() (bs []byte, err error) { return }

type SmapMessage2Capn_List C.PointerList

func NewSmapMessage2CapnList(s *C.Segment, sz int) SmapMessage2Capn_List {
	return SmapMessage2Capn_List(s.NewCompositeList(0, 6, sz))
}
func (s SmapMessage2Capn_List) Len() int { return C.PointerList(s).Len() }
func (s SmapMessage2Capn_List) At(i int) SmapMessage2Capn {
	return SmapMessage2Capn(C.PointerList(s).At(i).ToStruct())
}
func (s SmapMessage2Capn_List) ToArray() []SmapMessage2Capn {
	return *(*[]SmapMessage2Capn)(unsafe.Pointer(C.PointerList(s).ToArray()))
}
func (s SmapMessage2Capn_List) Set(i int, item SmapMessage2Capn) {
	C.PointerList(s).Set(i, C.Object(item))
}

type SmapNumberReadingCapn C.Struct

func NewSmapNumberReadingCapn(s *C.Segment) SmapNumberReadingCapn {
	return SmapNumberReadingCapn(s.NewStruct(16, 0))
}
func NewRootSmapNumberReadingCapn(s *C.Segment) SmapNumberReadingCapn {
	return SmapNumberReadingCapn(s.NewRootStruct(16, 0))
}
func AutoNewSmapNumberReadingCapn(s *C.Segment) SmapNumberReadingCapn {
	return SmapNumberReadingCapn(s.NewStructAR(16, 0))
}
func ReadRootSmapNumberReadingCapn(s *C.Segment) SmapNumberReadingCapn {
	return SmapNumberReadingCapn(s.Root(0).ToStruct())
}
func (s SmapNumberReadingCapn) Time() uint64       { return C.Struct(s).Get64(0) }
func (s SmapNumberReadingCapn) SetTime(v uint64)   { C.Struct(s).Set64(0, v) }
func (s SmapNumberReadingCapn) Value() float64     { return math.Float64frombits(C.Struct(s).Get64(8)) }
func (s SmapNumberReadingCapn) SetValue(v float64) { C.Struct(s).Set64(8, math.Float64bits(v)) }

// capn.JSON_enabled == false so we stub MarshallJSON().
func (s SmapNumberReadingCapn) MarshalJSON() (bs []byte, err error) { return }

type SmapNumberReadingCapn_List C.PointerList

func NewSmapNumberReadingCapnList(s *C.Segment, sz int) SmapNumberReadingCapn_List {
	return SmapNumberReadingCapn_List(s.NewCompositeList(16, 0, sz))
}
func (s SmapNumberReadingCapn_List) Len() int { return C.PointerList(s).Len() }
func (s SmapNumberReadingCapn_List) At(i int) SmapNumberReadingCapn {
	return SmapNumberReadingCapn(C.PointerList(s).At(i).ToStruct())
}
func (s SmapNumberReadingCapn_List) ToArray() []SmapNumberReadingCapn {
	return *(*[]SmapNumberReadingCapn)(unsafe.Pointer(C.PointerList(s).ToArray()))
}
func (s SmapNumberReadingCapn_List) Set(i int, item SmapNumberReadingCapn) {
	C.PointerList(s).Set(i, C.Object(item))
}

type SmapProperties2Capn C.Struct

func NewSmapProperties2Capn(s *C.Segment) SmapProperties2Capn {
	return SmapProperties2Capn(s.NewStruct(16, 1))
}
func NewRootSmapProperties2Capn(s *C.Segment) SmapProperties2Capn {
	return SmapProperties2Capn(s.NewRootStruct(16, 1))
}
func AutoNewSmapProperties2Capn(s *C.Segment) SmapProperties2Capn {
	return SmapProperties2Capn(s.NewStructAR(16, 1))
}
func ReadRootSmapProperties2Capn(s *C.Segment) SmapProperties2Capn {
	return SmapProperties2Capn(s.Root(0).ToStruct())
}
func (s SmapProperties2Capn) UnitOfTime() uint64     { return C.Struct(s).Get64(0) }
func (s SmapProperties2Capn) SetUnitOfTime(v uint64) { C.Struct(s).Set64(0, v) }
func (s SmapProperties2Capn) UnitOfMeasure() string  { return C.Struct(s).GetObject(0).ToText() }
func (s SmapProperties2Capn) SetUnitOfMeasure(v string) {
	C.Struct(s).SetObject(0, s.Segment.NewText(v))
}
func (s SmapProperties2Capn) StreamType() uint64     { return C.Struct(s).Get64(8) }
func (s SmapProperties2Capn) SetStreamType(v uint64) { C.Struct(s).Set64(8, v) }

// capn.JSON_enabled == false so we stub MarshallJSON().
func (s SmapProperties2Capn) MarshalJSON() (bs []byte, err error) { return }

type SmapProperties2Capn_List C.PointerList

func NewSmapProperties2CapnList(s *C.Segment, sz int) SmapProperties2Capn_List {
	return SmapProperties2Capn_List(s.NewCompositeList(16, 1, sz))
}
func (s SmapProperties2Capn_List) Len() int { return C.PointerList(s).Len() }
func (s SmapProperties2Capn_List) At(i int) SmapProperties2Capn {
	return SmapProperties2Capn(C.PointerList(s).At(i).ToStruct())
}
func (s SmapProperties2Capn_List) ToArray() []SmapProperties2Capn {
	return *(*[]SmapProperties2Capn)(unsafe.Pointer(C.PointerList(s).ToArray()))
}
func (s SmapProperties2Capn_List) Set(i int, item SmapProperties2Capn) {
	C.PointerList(s).Set(i, C.Object(item))
}

type DictEntryCapn C.Struct

func NewDictEntryCapn(s *C.Segment) DictEntryCapn      { return DictEntryCapn(s.NewStruct(0, 2)) }
func NewRootDictEntryCapn(s *C.Segment) DictEntryCapn  { return DictEntryCapn(s.NewRootStruct(0, 2)) }
func AutoNewDictEntryCapn(s *C.Segment) DictEntryCapn  { return DictEntryCapn(s.NewStructAR(0, 2)) }
func ReadRootDictEntryCapn(s *C.Segment) DictEntryCapn { return DictEntryCapn(s.Root(0).ToStruct()) }
func (s DictEntryCapn) Key() string                    { return C.Struct(s).GetObject(0).ToText() }
func (s DictEntryCapn) SetKey(v string)                { C.Struct(s).SetObject(0, s.Segment.NewText(v)) }
func (s DictEntryCapn) Value() string                  { return C.Struct(s).GetObject(1).ToText() }
func (s DictEntryCapn) SetValue(v string)              { C.Struct(s).SetObject(1, s.Segment.NewText(v)) }

// capn.JSON_enabled == false so we stub MarshallJSON().
func (s DictEntryCapn) MarshalJSON() (bs []byte, err error) { return }

type DictEntryCapn_List C.PointerList

func NewDictEntryCapnList(s *C.Segment, sz int) DictEntryCapn_List {
	return DictEntryCapn_List(s.NewCompositeList(0, 2, sz))
}
func (s DictEntryCapn_List) Len() int { return C.PointerList(s).Len() }
func (s DictEntryCapn_List) At(i int) DictEntryCapn {
	return DictEntryCapn(C.PointerList(s).At(i).ToStruct())
}
func (s DictEntryCapn_List) ToArray() []DictEntryCapn {
	return *(*[]DictEntryCapn)(unsafe.Pointer(C.PointerList(s).ToArray()))
}
func (s DictEntryCapn_List) Set(i int, item DictEntryCapn) { C.PointerList(s).Set(i, C.Object(item)) }
