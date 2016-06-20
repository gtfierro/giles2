//line query.y:2
package querylang

import __yyfmt__ "fmt"

//line query.y:3
import (
	"bufio"
	"fmt"
	"github.com/gtfierro/giles2/common"
	"github.com/taylorchu/toki"
	"strconv"
	_time "time"
)

/**
Notes here
**/

//line query.y:20
type sqSymType struct {
	yys      int
	str      string
	dict     common.Dict
	data     *DataQuery
	limit    Limit
	timeconv common.UnitOfTime
	list     List
	time     _time.Time
	timediff _time.Duration
}

const SELECT = 57346
const DISTINCT = 57347
const DELETE = 57348
const SET = 57349
const APPLY = 57350
const STATISTICAL = 57351
const WINDOW = 57352
const WHERE = 57353
const DATA = 57354
const BEFORE = 57355
const AFTER = 57356
const LIMIT = 57357
const STREAMLIMIT = 57358
const NOW = 57359
const LVALUE = 57360
const QSTRING = 57361
const EQ = 57362
const NEQ = 57363
const COMMA = 57364
const ALL = 57365
const LEFTPIPE = 57366
const LIKE = 57367
const AS = 57368
const AND = 57369
const OR = 57370
const HAS = 57371
const NOT = 57372
const IN = 57373
const TO = 57374
const LPAREN = 57375
const RPAREN = 57376
const LBRACK = 57377
const RBRACK = 57378
const NUMBER = 57379
const SEMICOLON = 57380
const NEWLINE = 57381
const TIMEUNIT = 57382

var sqToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"SELECT",
	"DISTINCT",
	"DELETE",
	"SET",
	"APPLY",
	"STATISTICAL",
	"WINDOW",
	"WHERE",
	"DATA",
	"BEFORE",
	"AFTER",
	"LIMIT",
	"STREAMLIMIT",
	"NOW",
	"LVALUE",
	"QSTRING",
	"EQ",
	"NEQ",
	"COMMA",
	"ALL",
	"LEFTPIPE",
	"LIKE",
	"AS",
	"AND",
	"OR",
	"HAS",
	"NOT",
	"IN",
	"TO",
	"LPAREN",
	"RPAREN",
	"LBRACK",
	"RBRACK",
	"NUMBER",
	"SEMICOLON",
	"NEWLINE",
	"TIMEUNIT",
}
var sqStatenames = [...]string{}

const sqEofCode = 1
const sqErrCode = 2
const sqInitialStackSize = 16

//line query.y:407

const eof = 0

var supported_formats = []string{"1/2/2006",
	"1-2-2006",
	"1/2/2006 03:04:05 PM MST",
	"1-2-2006 03:04:05 PM MST",
	"1/2/2006 15:04:05 MST",
	"1-2-2006 15:04:05 MST",
	"2006-1-2 15:04:05 MST"}

type List []string

func (qt QueryType) String() string {
	ret := ""
	switch qt {
	case SELECT_TYPE:
		ret = "select"
	case DELETE_TYPE:
		ret = "delete"
	case SET_TYPE:
		ret = "set"
	case DATA_TYPE:
		ret = "data"
	}
	return ret
}

type query struct {
	// the type of query we are doing
	qtype QueryType
	// information about a data query if we are one
	data *DataQuery
	// key-value pairs to add
	set common.Dict
	// where clause for query
	where common.Dict
	// are we querying distinct values?
	distinct bool
	// list of tags to target for deletion, selection
	Contents []string
}

func (q *query) Print() {
	fmt.Printf("Type: %v\n", q.qtype.String())
	if q.qtype == DATA_TYPE {
		fmt.Printf("Data Query Type: %v\n", q.data.Dtype.String())
		fmt.Printf("Start: %v\n", q.data.Start)
		fmt.Printf("End: %v\n", q.data.End)
		fmt.Printf("Limit: %v\n", q.data.Limit.Limit)
		fmt.Printf("Streamlimit: %v\n", q.data.Limit.Streamlimit)
	}
	fmt.Printf("Contents: %v\n", q.Contents)
	fmt.Printf("Distinct? %v\n", q.distinct)
	fmt.Printf("where: %v\n", q.where)
}

type sqLex struct {
	querystring string
	query       *query
	scanner     *toki.Scanner
	lasttoken   string
	tokens      []string
	error       error
	// all keys that we encounter. Used for republish concerns
	_keys map[string]struct{}
	keys  []string
}

func NewSQLex(s string) *sqLex {
	scanner := toki.NewScanner(
		[]toki.Def{
			{Token: WHERE, Pattern: "where"},
			{Token: SELECT, Pattern: "select"},
			{Token: APPLY, Pattern: "apply"},
			{Token: DELETE, Pattern: "delete"},
			{Token: DISTINCT, Pattern: "distinct"},
			{Token: STATISTICAL, Pattern: "statistical"},
			{Token: WINDOW, Pattern: "window"},
			{Token: LIMIT, Pattern: "limit"},
			{Token: STREAMLIMIT, Pattern: "streamlimit"},
			{Token: ALL, Pattern: "\\*"},
			{Token: NOW, Pattern: "now"},
			{Token: SET, Pattern: "set"},
			{Token: BEFORE, Pattern: "before"},
			{Token: AFTER, Pattern: "after"},
			{Token: COMMA, Pattern: ","},
			{Token: AND, Pattern: "and"},
			{Token: AS, Pattern: "as"},
			{Token: TO, Pattern: "to"},
			{Token: DATA, Pattern: "data"},
			{Token: OR, Pattern: "or"},
			{Token: IN, Pattern: "in"},
			{Token: HAS, Pattern: "has"},
			{Token: NOT, Pattern: "not"},
			{Token: NEQ, Pattern: "!="},
			{Token: EQ, Pattern: "="},
			{Token: LEFTPIPE, Pattern: "<"},
			{Token: LPAREN, Pattern: "\\("},
			{Token: RPAREN, Pattern: "\\)"},
			{Token: LBRACK, Pattern: "\\["},
			{Token: RBRACK, Pattern: "\\]"},
			{Token: SEMICOLON, Pattern: ";"},
			{Token: NEWLINE, Pattern: "\n"},
			{Token: LIKE, Pattern: "(like)|~"},
			{Token: NUMBER, Pattern: "([+-]?([0-9]*\\.)?[0-9]+)"},
			{Token: LVALUE, Pattern: "[a-zA-Z\\~\\$\\_][a-zA-Z0-9\\/\\%_\\-]*"},
			{Token: QSTRING, Pattern: "(\"[^\"\\\\]*?(\\.[^\"\\\\]*?)*?\")|('[^'\\\\]*?(\\.[^'\\\\]*?)*?')"},
		})
	scanner.SetInput(s)
	q := &query{Contents: []string{}, distinct: false, data: &DataQuery{}}
	return &sqLex{query: q, querystring: s, scanner: scanner, error: nil, lasttoken: "", _keys: map[string]struct{}{}, tokens: []string{}}
}

func (sq *sqLex) Lex(lval *sqSymType) int {
	r := sq.scanner.Next()
	sq.lasttoken = r.String()
	if r.Pos.Line == 2 || len(r.Value) == 0 {
		return eof
	}
	lval.str = string(r.Value)
	sq.tokens = append(sq.tokens, lval.str)
	return int(r.Token)
}

func (sq *sqLex) Error(s string) {
	sq.error = fmt.Errorf(s)
}

func readline(fi *bufio.Reader) (string, bool) {
	fmt.Printf("smap> ")
	s, err := fi.ReadString('\n')
	if err != nil {
		return "", false
	}
	return s, true
}

// Parse has been moved to query_processor.go

//line yacctab:1
var sqExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const sqNprod = 59
const sqPrivate = 57344

var sqTokenNames []string
var sqStates []string

const sqLast = 169

var sqAct = [...]int{

	105, 82, 51, 40, 48, 79, 13, 16, 13, 75,
	15, 39, 53, 24, 64, 20, 20, 52, 52, 53,
	53, 53, 63, 59, 46, 45, 36, 126, 44, 14,
	61, 54, 55, 47, 13, 80, 60, 50, 50, 95,
	41, 38, 32, 22, 43, 71, 44, 76, 14, 108,
	67, 107, 77, 57, 56, 74, 100, 85, 140, 41,
	137, 87, 124, 43, 42, 44, 111, 99, 86, 131,
	127, 93, 94, 96, 29, 28, 97, 91, 92, 26,
	27, 73, 72, 103, 104, 7, 109, 128, 122, 98,
	17, 106, 16, 16, 16, 65, 66, 25, 62, 112,
	113, 114, 115, 136, 76, 118, 133, 117, 69, 70,
	119, 116, 102, 68, 101, 19, 58, 90, 89, 125,
	88, 21, 23, 78, 30, 33, 129, 53, 120, 14,
	132, 31, 130, 34, 35, 81, 134, 121, 135, 139,
	141, 138, 142, 143, 9, 83, 84, 123, 11, 12,
	110, 10, 11, 12, 20, 10, 20, 14, 1, 6,
	49, 14, 8, 2, 18, 4, 3, 5, 37,
}
var sqPact = [...]int{

	159, -1000, 139, 111, 143, 5, 145, -1000, -1000, 111,
	66, 42, 41, 102, -1000, 4, 105, 145, 145, -12,
	11, -13, -1000, -14, -1000, 0, 1, 1, 17, 16,
	111, -15, -1000, -7, -16, -24, -1000, 68, 30, -1000,
	88, 111, 51, 30, 108, -1000, -1000, 1, 101, -2,
	117, -1000, -1000, -1000, 130, 130, 34, 111, -1000, -1000,
	98, 96, 95, -1000, -1000, 30, 30, -1000, 108, 2,
	108, -1000, 111, 58, 33, 20, 92, 90, 1, -1000,
	111, -1000, 65, 14, 12, 65, 138, 32, 111, 111,
	111, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 111, -1000,
	-1000, 108, 1, 130, -2, -1000, 110, 121, -1000, -1000,
	57, 135, -1000, -1000, -1000, -1000, -1000, 28, 65, -1000,
	-1000, -10, 37, 56, 130, -1000, -1000, 1, 36, 65,
	84, 1, -1000, 1, 81, 26, 1, 130, 24, 65,
	130, -1000, 65, -1000,
}
var sqPgo = [...]int{

	0, 168, 11, 115, 10, 167, 85, 9, 64, 159,
	4, 160, 5, 1, 0, 2, 3, 158,
}
var sqR1 = [...]int{

	0, 17, 17, 17, 17, 17, 17, 17, 17, 6,
	6, 8, 7, 7, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 9, 9, 9, 9, 9, 9,
	10, 10, 11, 11, 11, 11, 12, 12, 13, 13,
	13, 13, 14, 14, 3, 2, 2, 2, 2, 2,
	2, 2, 2, 15, 16, 1, 1, 1, 1,
}
var sqR2 = [...]int{

	0, 4, 3, 4, 4, 3, 4, 4, 3, 1,
	3, 3, 1, 3, 3, 3, 3, 5, 5, 5,
	1, 1, 2, 1, 9, 7, 13, 14, 5, 5,
	1, 2, 2, 1, 1, 1, 2, 3, 0, 2,
	2, 4, 0, 2, 2, 3, 3, 3, 3, 2,
	3, 4, 3, 1, 1, 3, 3, 2, 1,
}
var sqChk = [...]int{

	-1000, -17, 4, 7, 6, -5, -9, -6, 23, 5,
	12, 9, 10, -16, 18, -4, -16, -6, -9, -3,
	11, -3, 38, -3, -16, 31, 13, 14, 33, 33,
	22, -3, 38, 20, -3, -3, 38, -1, 30, -2,
	-16, 29, -8, 33, 35, 38, 38, 33, -10, -11,
	37, -15, 17, 19, -10, -10, 37, 37, -6, 38,
	-15, 37, -8, 38, 38, 27, 28, -2, 25, 20,
	21, -16, 31, 30, -2, -7, -15, -10, 22, -12,
	37, 18, -13, 15, 16, -13, 34, -16, 22, 22,
	22, -2, -2, -15, -15, 37, -15, -16, 31, 34,
	36, 22, 22, -10, -16, -14, 26, 37, 37, -14,
	12, 34, -4, -4, -4, -16, -7, -10, -13, -12,
	18, 16, 31, 12, 34, -14, 37, 33, 31, -13,
	-10, 33, -14, 22, -10, -10, 22, 34, -10, -13,
	34, -14, -13, -14,
}
var sqDef = [...]int{

	0, -2, 0, 0, 0, 0, 0, 20, 21, 23,
	0, 0, 0, 9, 54, 0, 0, 0, 0, 0,
	0, 0, 2, 0, 22, 0, 0, 0, 0, 0,
	0, 0, 5, 0, 0, 0, 8, 44, 0, 58,
	0, 0, 0, 0, 0, 1, 3, 0, 0, 30,
	33, 34, 35, 53, 38, 38, 0, 0, 10, 4,
	14, 15, 16, 6, 7, 0, 0, 57, 0, 0,
	0, 49, 0, 0, 0, 0, 12, 0, 0, 31,
	0, 32, 42, 0, 0, 42, 0, 0, 0, 0,
	0, 55, 56, 45, 46, 47, 48, 50, 0, 52,
	11, 0, 0, 38, 36, 28, 0, 39, 40, 29,
	0, 0, 17, 18, 19, 51, 13, 0, 42, 37,
	43, 0, 0, 0, 38, 25, 41, 0, 0, 42,
	0, 0, 24, 0, 0, 0, 0, 38, 0, 42,
	38, 26, 42, 27,
}
var sqTok1 = [...]int{

	1,
}
var sqTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40,
}
var sqTok3 = [...]int{
	0,
}

var sqErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	sqDebug        = 0
	sqErrorVerbose = false
)

type sqLexer interface {
	Lex(lval *sqSymType) int
	Error(s string)
}

type sqParser interface {
	Parse(sqLexer) int
	Lookahead() int
}

type sqParserImpl struct {
	lval  sqSymType
	stack [sqInitialStackSize]sqSymType
	char  int
}

func (p *sqParserImpl) Lookahead() int {
	return p.char
}

func sqNewParser() sqParser {
	return &sqParserImpl{}
}

const sqFlag = -1000

func sqTokname(c int) string {
	if c >= 1 && c-1 < len(sqToknames) {
		if sqToknames[c-1] != "" {
			return sqToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func sqStatname(s int) string {
	if s >= 0 && s < len(sqStatenames) {
		if sqStatenames[s] != "" {
			return sqStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func sqErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !sqErrorVerbose {
		return "syntax error"
	}

	for _, e := range sqErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + sqTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := sqPact[state]
	for tok := TOKSTART; tok-1 < len(sqToknames); tok++ {
		if n := base + tok; n >= 0 && n < sqLast && sqChk[sqAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if sqDef[state] == -2 {
		i := 0
		for sqExca[i] != -1 || sqExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; sqExca[i] >= 0; i += 2 {
			tok := sqExca[i]
			if tok < TOKSTART || sqExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if sqExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += sqTokname(tok)
	}
	return res
}

func sqlex1(lex sqLexer, lval *sqSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = sqTok1[0]
		goto out
	}
	if char < len(sqTok1) {
		token = sqTok1[char]
		goto out
	}
	if char >= sqPrivate {
		if char < sqPrivate+len(sqTok2) {
			token = sqTok2[char-sqPrivate]
			goto out
		}
	}
	for i := 0; i < len(sqTok3); i += 2 {
		token = sqTok3[i+0]
		if token == char {
			token = sqTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = sqTok2[1] /* unknown char */
	}
	if sqDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", sqTokname(token), uint(char))
	}
	return char, token
}

func sqParse(sqlex sqLexer) int {
	return sqNewParser().Parse(sqlex)
}

func (sqrcvr *sqParserImpl) Parse(sqlex sqLexer) int {
	var sqn int
	var sqVAL sqSymType
	var sqDollar []sqSymType
	_ = sqDollar // silence set and not used
	sqS := sqrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	sqstate := 0
	sqrcvr.char = -1
	sqtoken := -1 // sqrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		sqstate = -1
		sqrcvr.char = -1
		sqtoken = -1
	}()
	sqp := -1
	goto sqstack

ret0:
	return 0

ret1:
	return 1

sqstack:
	/* put a state and value onto the stack */
	if sqDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", sqTokname(sqtoken), sqStatname(sqstate))
	}

	sqp++
	if sqp >= len(sqS) {
		nyys := make([]sqSymType, len(sqS)*2)
		copy(nyys, sqS)
		sqS = nyys
	}
	sqS[sqp] = sqVAL
	sqS[sqp].yys = sqstate

sqnewstate:
	sqn = sqPact[sqstate]
	if sqn <= sqFlag {
		goto sqdefault /* simple state */
	}
	if sqrcvr.char < 0 {
		sqrcvr.char, sqtoken = sqlex1(sqlex, &sqrcvr.lval)
	}
	sqn += sqtoken
	if sqn < 0 || sqn >= sqLast {
		goto sqdefault
	}
	sqn = sqAct[sqn]
	if sqChk[sqn] == sqtoken { /* valid shift */
		sqrcvr.char = -1
		sqtoken = -1
		sqVAL = sqrcvr.lval
		sqstate = sqn
		if Errflag > 0 {
			Errflag--
		}
		goto sqstack
	}

sqdefault:
	/* default state action */
	sqn = sqDef[sqstate]
	if sqn == -2 {
		if sqrcvr.char < 0 {
			sqrcvr.char, sqtoken = sqlex1(sqlex, &sqrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if sqExca[xi+0] == -1 && sqExca[xi+1] == sqstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			sqn = sqExca[xi+0]
			if sqn < 0 || sqn == sqtoken {
				break
			}
		}
		sqn = sqExca[xi+1]
		if sqn < 0 {
			goto ret0
		}
	}
	if sqn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			sqlex.Error(sqErrorMessage(sqstate, sqtoken))
			Nerrs++
			if sqDebug >= 1 {
				__yyfmt__.Printf("%s", sqStatname(sqstate))
				__yyfmt__.Printf(" saw %s\n", sqTokname(sqtoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for sqp >= 0 {
				sqn = sqPact[sqS[sqp].yys] + sqErrCode
				if sqn >= 0 && sqn < sqLast {
					sqstate = sqAct[sqn] /* simulate a shift of "error" */
					if sqChk[sqstate] == sqErrCode {
						goto sqstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if sqDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", sqS[sqp].yys)
				}
				sqp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if sqDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", sqTokname(sqtoken))
			}
			if sqtoken == sqEofCode {
				goto ret1
			}
			sqrcvr.char = -1
			sqtoken = -1
			goto sqnewstate /* try again in the same state */
		}
	}

	/* reduction by production sqn */
	if sqDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", sqn, sqStatname(sqstate))
	}

	sqnt := sqn
	sqpt := sqp
	_ = sqpt // guard against "declared and not used"

	sqp -= sqR2[sqn]
	// sqp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if sqp+1 >= len(sqS) {
		nyys := make([]sqSymType, len(sqS)*2)
		copy(nyys, sqS)
		sqS = nyys
	}
	sqVAL = sqS[sqp+1]

	/* consult goto table to find next state */
	sqn = sqR1[sqn]
	sqg := sqPgo[sqn]
	sqj := sqg + sqS[sqp].yys + 1

	if sqj >= sqLast {
		sqstate = sqAct[sqg]
	} else {
		sqstate = sqAct[sqj]
		if sqChk[sqstate] != -sqn {
			sqstate = sqAct[sqg]
		}
	}
	// dummy call; replaced with literal code
	switch sqnt {

	case 1:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:59
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.qtype = SELECT_TYPE
		}
	case 2:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:65
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.qtype = SELECT_TYPE
		}
	case 3:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:70
		{
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.data = sqDollar[2].data
			sqlex.(*sqLex).query.qtype = DATA_TYPE
		}
	case 4:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:76
		{
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.set = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = SET_TYPE
		}
	case 5:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:82
		{
			sqlex.(*sqLex).query.set = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = SET_TYPE
		}
	case 6:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:87
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.qtype = DELETE_TYPE
		}
	case 7:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:93
		{
			sqlex.(*sqLex).query.data = sqDollar[2].data
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.qtype = DELETE_TYPE
		}
	case 8:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:99
		{
			sqlex.(*sqLex).query.Contents = []string{}
			sqlex.(*sqLex).query.where = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = DELETE_TYPE
		}
	case 9:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:107
		{
			sqVAL.list = List{sqDollar[1].str}
		}
	case 10:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:111
		{
			sqVAL.list = append(List{sqDollar[1].str}, sqDollar[3].list...)
		}
	case 11:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:117
		{
			sqVAL.list = sqDollar[2].list
		}
	case 12:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:122
		{
			sqVAL.list = List{sqDollar[1].str}
		}
	case 13:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:126
		{
			sqVAL.list = append(List{sqDollar[1].str}, sqDollar[3].list...)
		}
	case 14:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:132
		{
			sqVAL.dict = common.Dict{sqDollar[1].str: sqDollar[3].str}
		}
	case 15:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:136
		{
			sqVAL.dict = common.Dict{sqDollar[1].str: sqDollar[3].str}
		}
	case 16:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:140
		{
			sqVAL.dict = common.Dict{sqDollar[1].str: sqDollar[3].list}
		}
	case 17:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:144
		{
			sqDollar[5].dict[sqDollar[1].str] = sqDollar[3].str
			sqVAL.dict = sqDollar[5].dict
		}
	case 18:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:149
		{
			sqDollar[5].dict[sqDollar[1].str] = sqDollar[3].str
			sqVAL.dict = sqDollar[5].dict
		}
	case 19:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:154
		{
			sqDollar[5].dict[sqDollar[1].str] = sqDollar[3].list
			sqVAL.dict = sqDollar[5].dict
		}
	case 20:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:161
		{
			sqlex.(*sqLex).query.Contents = sqDollar[1].list
			sqVAL.list = sqDollar[1].list
		}
	case 21:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:166
		{
			sqVAL.list = List{}
		}
	case 22:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:170
		{
			sqlex.(*sqLex).query.distinct = true
			sqVAL.list = List{sqDollar[2].str}
		}
	case 23:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:175
		{
			sqlex.(*sqLex).query.distinct = true
			sqVAL.list = List{}
		}
	case 24:
		sqDollar = sqS[sqpt-9 : sqpt+1]
		//line query.y:182
		{
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[4].time, End: sqDollar[6].time, Limit: sqDollar[8].limit, Timeconv: sqDollar[9].timeconv, IsStatistical: false, IsWindow: false}
		}
	case 25:
		sqDollar = sqS[sqpt-7 : sqpt+1]
		//line query.y:186
		{
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[3].time, End: sqDollar[5].time, Limit: sqDollar[6].limit, Timeconv: sqDollar[7].timeconv, IsStatistical: false, IsWindow: false}
		}
	case 26:
		sqDollar = sqS[sqpt-13 : sqpt+1]
		//line query.y:190
		{
			num, err := strconv.ParseInt(sqDollar[3].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[1].str, err.Error()))
			}
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[8].time, End: sqDollar[10].time, Limit: sqDollar[12].limit, Timeconv: sqDollar[13].timeconv, IsStatistical: true, IsWindow: false, PointWidth: uint64(num)}
		}
	case 27:
		sqDollar = sqS[sqpt-14 : sqpt+1]
		//line query.y:198
		{
			dur, err := common.ParseReltime(sqDollar[3].str, sqDollar[4].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[3].str, sqDollar[4].str, err.Error()))
			}
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[9].time, End: sqDollar[11].time, Limit: sqDollar[13].limit, Timeconv: sqDollar[14].timeconv, IsStatistical: false, IsWindow: true, Width: uint64(dur.Nanoseconds())}
		}
	case 28:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:206
		{
			sqVAL.data = &DataQuery{Dtype: BEFORE_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv, IsStatistical: false, IsWindow: false}
		}
	case 29:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:210
		{
			sqVAL.data = &DataQuery{Dtype: AFTER_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv, IsStatistical: false, IsWindow: false}
		}
	case 30:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:216
		{
			sqVAL.time = sqDollar[1].time
		}
	case 31:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:220
		{
			sqVAL.time = sqDollar[1].time.Add(sqDollar[2].timediff)
		}
	case 32:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:226
		{
			foundtime, err := common.ParseAbsTime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.time = foundtime
		}
	case 33:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:234
		{
			num, err := strconv.ParseInt(sqDollar[1].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[1].str, err.Error()))
			}
			sqVAL.time = _time.Unix(num, 0)
		}
	case 34:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:242
		{
			found := false
			for _, format := range supported_formats {
				t, err := _time.Parse(format, sqDollar[1].str)
				if err != nil {
					continue
				}
				sqVAL.time = t
				found = true
				break
			}
			if !found {
				sqlex.(*sqLex).Error(fmt.Sprintf("No time format matching \"%v\" found", sqDollar[1].str))
			}
		}
	case 35:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:258
		{
			sqVAL.time = _time.Now()
		}
	case 36:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:264
		{
			var err error
			sqVAL.timediff, err = common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
		}
	case 37:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:272
		{
			newDuration, err := common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.timediff = common.AddDurations(newDuration, sqDollar[3].timediff)
		}
	case 38:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:282
		{
			sqVAL.limit = Limit{Limit: -1, Streamlimit: -1}
		}
	case 39:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:286
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: num, Streamlimit: -1}
		}
	case 40:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:294
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: -1, Streamlimit: num}
		}
	case 41:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:302
		{
			limit_num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			slimit_num, err := strconv.ParseInt(sqDollar[4].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: limit_num, Streamlimit: slimit_num}
		}
	case 42:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:316
		{
			sqVAL.timeconv = common.UOT_MS
		}
	case 43:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:320
		{
			uot, err := common.ParseUOT(sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse unit of time %v (%v)", sqDollar[2].str, err))
			}
			sqVAL.timeconv = uot
		}
	case 44:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:332
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 45:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:339
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): common.Dict{"$regex": sqDollar[3].str}}
		}
	case 46:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:343
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 47:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:347
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 48:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:351
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): common.Dict{"$neq": sqDollar[3].str}}
		}
	case 49:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:355
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[2].str): common.Dict{"$exists": true}}
		}
	case 50:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:359
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[3].str): common.Dict{"$in": sqDollar[1].list}}
		}
	case 51:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:363
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[3].str): common.Dict{"$not": common.Dict{"$in": sqDollar[1].list}}}
		}
	case 52:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:367
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 53:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:373
		{
			sqVAL.str = sqDollar[1].str[1 : len(sqDollar[1].str)-1]
		}
	case 54:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:379
		{

			sqlex.(*sqLex)._keys[sqDollar[1].str] = struct{}{}
			sqVAL.str = cleantagstring(sqDollar[1].str)
		}
	case 55:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:387
		{
			sqVAL.dict = common.Dict{"$and": []common.Dict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 56:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:391
		{
			sqVAL.dict = common.Dict{"$or": []common.Dict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 57:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:395
		{
			tmp := make(common.Dict)
			for k, v := range sqDollar[2].dict {
				tmp[k] = common.Dict{"$ne": v}
			}
			sqVAL.dict = tmp
		}
	case 58:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:403
		{
			sqVAL.dict = sqDollar[1].dict
		}
	}
	goto sqstack /* stack new state and value */
}
