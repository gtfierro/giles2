package objectbuilder

import (
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
