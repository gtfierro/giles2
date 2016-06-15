package objectbuilder

func Parse(s string) []Operation {
	l := NewExprLexer(s)
	exParse(l)
	return l.operations
}
