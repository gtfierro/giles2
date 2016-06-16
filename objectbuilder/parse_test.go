package objectbuilder

import (
	"reflect"
	"testing"
)

func compareUintSlice(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, aa := range a {
		if aa != b[i] {
			return false
		}
	}
	return true
}

func tokenListToNames(tokens []uint32) []string {
	names := []string{}
	for _, tok := range tokens {
		names = append(names, getName(tok))
	}
	return names
}

func TestParseTokens(t *testing.T) {
	for _, test := range []struct {
		expr   string
		tokens []uint32
	}{
		{
			"key",
			[]uint32{KEY},
		},
		{
			"key1.key2",
			[]uint32{KEY, DOT, KEY},
		},
		{
			"[0]",
			[]uint32{LBRACKET, NUMBER, RBRACKET},
		},
		{
			"[:]",
			[]uint32{LBRACKET, COLON, RBRACKET},
		},
		{
			"key[:]",
			[]uint32{KEY, LBRACKET, COLON, RBRACKET},
		},
		{
			"key.key2[:]",
			[]uint32{KEY, DOT, KEY, LBRACKET, COLON, RBRACKET},
		},
		{
			"[0][1][2]",
			[]uint32{LBRACKET, NUMBER, RBRACKET, LBRACKET, NUMBER, RBRACKET, LBRACKET, NUMBER, RBRACKET},
		},
		{
			"[0].key1",
			[]uint32{LBRACKET, NUMBER, RBRACKET, DOT, KEY},
		},
		{
			"[0].key1",
			[]uint32{LBRACKET, NUMBER, RBRACKET, DOT, KEY},
		},
		{
			"[0].key1[1].key2",
			[]uint32{LBRACKET, NUMBER, RBRACKET, DOT, KEY, LBRACKET, NUMBER, RBRACKET, DOT, KEY},
		},
	} {
		l := NewExprLexer(test.expr)
		exParse(l)
		if !compareUintSlice(test.tokens, l.lextokens) {
			t.Errorf("TOKENS wrong for: %s -> Got %+v but wanted %+v", test.expr, tokenListToNames(l.lextokens), tokenListToNames(test.tokens))
		}
	}
}

func TestEvalArray(t *testing.T) {
	for _, test := range []struct {
		op     ArrayOperator
		data   interface{}
		result interface{}
	}{
		{
			ArrayOperator{index: 0, all: false, slice: false},
			[]int{1, 2, 3, 4},
			1,
		},
		{
			ArrayOperator{index: 3, all: false, slice: false},
			[]int{1, 2, 3, 4},
			4,
		},
		{
			ArrayOperator{index: 5, all: false, slice: false},
			[]int{1, 2, 3, 4},
			4,
		},
		{
			ArrayOperator{index: -1, all: false, slice: false},
			[]int{1, 2, 3, 4},
			1,
		},
		{
			ArrayOperator{all: true, slice: false},
			[]int{1, 2, 3, 4},
			[]int{1, 2, 3, 4},
		},
		{
			ArrayOperator{all: false, slice: true, slice_start: 0, slice_end: 4},
			[]int{1, 2, 3, 4},
			[]int{1, 2, 3, 4},
		},
		{
			ArrayOperator{all: false, slice: true, slice_start: 1, slice_end: 4},
			[]int{1, 2, 3, 4},
			[]int{2, 3, 4},
		},
		{
			ArrayOperator{all: false, slice: true, slice_start: 1, slice_end: 40},
			[]int{1, 2, 3, 4},
			[]int{2, 3, 4},
		},
		{
			ArrayOperator{all: false, slice: true, slice_start: 1, slice_end: 2},
			[]int{1, 2, 3, 4},
			[]int{2},
		},
	} {
		res := test.op.Eval(test.data)
		if !reflect.DeepEqual(res, test.result) {
			t.Errorf("Operator %+v on %+v gave %+v but wanted %+v", test.op, test.data, res, test.result)
		}
	}
}

func BenchmarkArrayIndex(b *testing.B) {
	op := ArrayOperator{index: 0, all: false, slice: false}
	data := []uint32{1, 2, 3, 4}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		op.Eval(data)
	}
}

func BenchmarkArraySlice(b *testing.B) {
	op := ArrayOperator{slice_start: 0, slice_end: 4, all: false, slice: true}
	data := []uint32{1, 2, 3, 4}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		op.Eval(data)
	}
}

func BenchmarkArrayAll(b *testing.B) {
	op := ArrayOperator{all: true, slice: false}
	data := []uint32{1, 2, 3, 4}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		op.Eval(data)
	}
}

func TestEvalMap(t *testing.T) {
	for _, test := range []struct {
		op     ObjectOperator
		data   interface{}
		result interface{}
	}{
		{
			ObjectOperator{"key1"},
			map[string]interface{}{"key1": "val1"},
			"val1",
		},
		{
			ObjectOperator{"key1"},
			map[string]interface{}{"key1": 12345},
			12345,
		},
		{
			ObjectOperator{"key1"},
			map[string]interface{}{"key1": []string{"a", "b"}},
			[]string{"a", "b"},
		},
	} {
		res := test.op.Eval(test.data)
		if !reflect.DeepEqual(res, test.result) {
			t.Errorf("Operator %+v on %+v gave %+v but wanted %+v", test.op, test.data, res, test.result)
		}
	}
}

func TestEvalStruct(t *testing.T) {
	for _, test := range []struct {
		op     ObjectOperator
		data   interface{}
		result interface{}
	}{
		{
			ObjectOperator{"Key1"},
			struct{ Key1 string }{Key1: "val1"},
			"val1",
		},
		{
			ObjectOperator{"Key2"},
			struct{ Key2 map[string]string }{Key2: map[string]string{"a": "b"}},
			map[string]string{"a": "b"},
		},
	} {
		res := test.op.Eval(test.data)
		if !reflect.DeepEqual(res, test.result) {
			t.Errorf("Operator %+v on %+v gave %+v but wanted %+v", test.op, test.data, res, test.result)
		}
	}
}

func BenchmarkObjectMap(b *testing.B) {
	op := ObjectOperator{"key1"}
	data := map[string]interface{}{"key1": "val1"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		op.Eval(data)
	}
}
func BenchmarkObjectStruct(b *testing.B) {
	op := ObjectOperator{"Key1"}
	data := struct{ Key1 string }{Key1: "val1"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		op.Eval(data)
	}
}

func TestParseOperatorChain(t *testing.T) {
	for _, test := range []struct {
		expr      string
		operators []Operation
	}{
		{
			"key",
			[]Operation{ObjectOperator{"key"}},
		},
		{
			"key1.key2",
			[]Operation{ObjectOperator{"key1"}, ObjectOperator{"key2"}},
		},
		{
			"[0]",
			[]Operation{ArrayOperator{index: 0, slice: false, all: false}},
		},
		{
			"[:]",
			[]Operation{ArrayOperator{slice: false, all: true}},
		},
		{
			"key[:]",
			[]Operation{ObjectOperator{"key"}, ArrayOperator{slice: false, all: true}},
		},
		{
			"key.key2[:]",
			[]Operation{ObjectOperator{"key"}, ObjectOperator{"key2"}, ArrayOperator{slice: false, all: true}},
		},
		{
			"[0][1][2]",
			[]Operation{ArrayOperator{index: 0, slice: false, all: false}, ArrayOperator{index: 1, slice: false, all: false}, ArrayOperator{index: 2, slice: false, all: false}},
		},
		{
			"[0].key1",
			[]Operation{ArrayOperator{index: 0, slice: false, all: false}, ObjectOperator{"key1"}},
		},
		{
			"[0].key1[1]",
			[]Operation{ArrayOperator{index: 0, slice: false, all: false}, ObjectOperator{"key1"}, ArrayOperator{index: 1, slice: false, all: false}},
		},
		{
			"[0].key1[1].key2",
			[]Operation{ArrayOperator{index: 0, slice: false, all: false}, ObjectOperator{"key1"}, ArrayOperator{index: 1, slice: false, all: false}, ObjectOperator{"key2"}},
		},
	} {
		parsedOps := Parse(test.expr)
		if !reflect.DeepEqual(parsedOps, test.operators) {
			t.Errorf("Operations wrong for: %s -> Got %+v but wanted %+v", test.expr, parsedOps, test.operators)
		}
	}
}

func TestEvalOperatorChain(t *testing.T) {
	for _, test := range []struct {
		expr   string
		data   interface{}
		result interface{}
	}{
		{
			"[0]",
			[]int{1, 2, 3, 4},
			1,
		},
		{
			"[3]",
			[]int{1, 2, 3, 4},
			4,
		},
		{
			"[5]",
			[]int{1, 2, 3, 4},
			4,
		},
		{
			"[-1]",
			[]int{1, 2, 3, 4},
			1,
		},
		{
			"[:]",
			[]int{1, 2, 3, 4},
			[]int{1, 2, 3, 4},
		},
		{
			"[0:4]",
			[]int{1, 2, 3, 4},
			[]int{1, 2, 3, 4},
		},
		{
			"[1:4]",
			[]int{1, 2, 3, 4},
			[]int{2, 3, 4},
		},
		{
			"[1:40]",
			[]int{1, 2, 3, 4},
			[]int{2, 3, 4},
		},
		{
			"[1:2]",
			[]int{1, 2, 3, 4},
			[]int{2},
		},
		{
			"key1",
			map[string]interface{}{"key1": "val1"},
			"val1",
		},
		{
			"key1",
			map[string]interface{}{"key1": 12345},
			12345,
		},
		{
			"key1",
			map[string]interface{}{"key1": []string{"a", "b"}},
			[]string{"a", "b"},
		},
		{
			"key1",
			map[string]interface{}{"key1": "val1"},
			"val1",
		},
		{
			"key1",
			map[string]interface{}{"key1": 12345},
			12345,
		},
		{
			"key1[:]",
			map[string]interface{}{"key1": []string{"a", "b"}},
			[]string{"a", "b"},
		},
		{
			"key1[0]",
			map[string]interface{}{"key1": []string{"a", "b"}},
			"a",
		},
		{
			"[0].key1[1]",
			[]map[string]interface{}{map[string]interface{}{"key1": []string{"a", "b"}}},
			"b",
		},
	} {
		res := Eval(Parse(test.expr), test.data)
		if !reflect.DeepEqual(res, test.result) {
			t.Errorf("Expr %+v on %+v gave %+v but wanted %+v", test.expr, test.data, res, test.result)
		}
	}
}
