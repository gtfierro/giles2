%{
package objectbuilder

import (
	"github.com/taylorchu/toki"
    "errors"
)

%}

%union{
    str string
    int int
}

%token <str> LBRACKET RBRACKET DOT KEY COLON
%token NUMBER

%type <int> NUMBER
%type <str> KEY


%%
expression  : array expression
            | object expression
            |
            ;

array       : LBRACKET index RBRACKET
            ;

index       : NUMBER
            | NUMBER COLON NUMBER
            | COLON
            ;

object      : KEY
            | object DOT KEY
            | array DOT KEY
            ;
%%

const eof = 0

func getName(tok uint32) string {
    switch tok {
    case LBRACKET:
        return "LBRACKET"
    case RBRACKET:
        return "RBRACKET"
    case DOT:
        return "DOT"
    case KEY:
        return "KEY"
    case COLON:
        return "COLON"
    case NUMBER:
        return "NUMBER"
    }
    return "UNKNOWN"
}

type lexer struct {
    expression string
    scanner *toki.Scanner
    tokens  []string
    lextokens []uint32
    operations []Operation
    error   error
}

func NewExprLexer(s string) *lexer {
    scanner := toki.NewScanner(
        []toki.Def{
            {Token: DOT, Pattern: "\\."},
            {Token: COLON, Pattern: ":"},
            {Token: LBRACKET, Pattern: "\\["},
            {Token: RBRACKET, Pattern: "\\]"},
			{Token: NUMBER, Pattern: "([+-]?([0-9]*\\.)?[0-9]+)"},
			{Token: KEY, Pattern: "[a-zA-Z\\~\\$\\_][a-zA-Z0-9\\/\\%_\\-]*"},
        })
    scanner.SetInput(s)
    return &lexer{
        expression: s,
        scanner: scanner,
        operations: []Operation{},
        tokens: []string{},
    }
}

func (l *lexer) Lex(lval *exSymType) int {
	r := l.scanner.Next()
	if r.Pos.Line == 2 || len(r.Value) == 0 {
		return eof
	}
	lval.str = string(r.Value)
    l.tokens = append(l.tokens, lval.str)
    l.lextokens = append(l.lextokens, uint32(r.Token))
	return int(r.Token)
}

func (l *lexer) Error(s string) {
    l.error = errors.New(s)
}
