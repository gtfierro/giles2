package common

import (
	"gopkg.in/mgo.v2/bson"
	"testing"
	"time"
)

func BenchmarkFlatten1x1(b *testing.B) {
	test := bson.M{"test": "test"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flatten(test)
	}
}

func BenchmarkFlatten1x2(b *testing.B) {
	test := bson.M{"Metadata": bson.M{"System": "HVAC"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flatten(test)
	}
}

func BenchmarkFlatten1x3(b *testing.B) {
	test := bson.M{"Metadata": bson.M{"Point": bson.M{"Type": "Sensor"}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flatten(test)
	}
}

func BenchmarkFlatten2x3(b *testing.B) {
	test := bson.M{"Metadata": bson.M{"Point": bson.M{"Type": "Sensor"}},
		"Properties": bson.M{"UnitofTime": bson.M{"X": "Y"}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flatten(test)
	}
}

func TestCompareStringSliceAsSet(t *testing.T) {
	for _, test := range []struct {
		s1    []string
		s2    []string
		equal bool
	}{
		{
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
			true,
		},
		{
			[]string{"b", "a", "c"},
			[]string{"a", "b", "c"},
			true,
		},
		{
			[]string{"a", "c"},
			[]string{"a", "b", "c"},
			false,
		},
		{
			[]string{"a", "c", "d"},
			[]string{"a", "b", "c"},
			false,
		},
		{
			[]string{"a", "b", "c"},
			[]string{"a", "c"},
			false,
		},
		{
			[]string{"a", "b", "c"},
			[]string{"a", "d", "c"},
			false,
		},
	} {
		res := compareStringSliceAsSet(test.s1, test.s2)
		if res != test.equal {
			t.Errorf("Slices \n%v\n \n%v\n should be equal? Got %v but should be %v", test.s1, test.s2, res, test.equal)
		}
	}
}

func TestGetPrefixes(t *testing.T) {
	var x string
	var y, z []string
	x = "/a/b/c"
	y = getPrefixes(x)
	z = []string{"/", "/a", "/a/b"}
	if !isStringSliceEqual(y, z) {
		t.Error("Got ", y, " should be ", z)
	}

	x = "/a/b/c/"
	y = getPrefixes(x)
	z = []string{"/", "/a", "/a/b"}
	if !isStringSliceEqual(y, z) {
		t.Error("Got ", y, " should be ", z)
	}

	x = "a/b/c/"
	y = getPrefixes(x)
	z = []string{"/", "/a", "/a/b"}
	if !isStringSliceEqual(y, z) {
		t.Error("Got ", y, " should be ", z)
	}
}

func TestGuessTimeUnit(t *testing.T) {
	for _, test := range []struct {
		inp        uint64
		actualUnit UnitOfTime
	}{
		{
			uint64(time.Date(2079, 1, 1, 1, 1, 1, 0, time.UTC).UnixNano()),
			UOT_NS,
		},
		{
			uint64(time.Date(2079, 1, 1, 1, 1, 1, 0, time.UTC).Unix()),
			UOT_S,
		},
		{
			0,
			UOT_S,
		},
	} {
		computed := GuessTimeUnit(test.inp)
		if computed != test.actualUnit {
			t.Error("Time %v unit %s computed as %s", test.inp, test.actualUnit.String(), computed.String())
		}
	}
}
