package archiver

import (
	"gopkg.in/mgo.v2/bson"
	"testing"
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
