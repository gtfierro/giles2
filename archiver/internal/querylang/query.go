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
const WHERE = 57351
const DATA = 57352
const BEFORE = 57353
const AFTER = 57354
const LIMIT = 57355
const STREAMLIMIT = 57356
const NOW = 57357
const LVALUE = 57358
const QSTRING = 57359
const EQ = 57360
const NEQ = 57361
const COMMA = 57362
const ALL = 57363
const LEFTPIPE = 57364
const LIKE = 57365
const AS = 57366
const AND = 57367
const OR = 57368
const HAS = 57369
const NOT = 57370
const IN = 57371
const TO = 57372
const LPAREN = 57373
const RPAREN = 57374
const LBRACK = 57375
const RBRACK = 57376
const NUMBER = 57377
const SEMICOLON = 57378
const NEWLINE = 57379
const TIMEUNIT = 57380

var sqToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"SELECT",
	"DISTINCT",
	"DELETE",
	"SET",
	"APPLY",
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

//line query.y:391

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

const sqNprod = 57
const sqPrivate = 57344

var sqTokenNames []string
var sqStates []string

const sqLast = 140

var sqAct = [...]int{

	97, 76, 36, 73, 69, 11, 14, 11, 47, 13,
	49, 48, 22, 49, 49, 35, 44, 48, 18, 49,
	18, 58, 57, 53, 42, 41, 40, 43, 55, 11,
	32, 46, 87, 114, 74, 100, 99, 46, 54, 92,
	65, 50, 51, 90, 112, 28, 12, 20, 91, 70,
	61, 12, 98, 79, 7, 68, 38, 37, 34, 15,
	71, 39, 37, 40, 24, 25, 39, 94, 40, 89,
	29, 85, 86, 88, 93, 83, 84, 96, 67, 66,
	101, 52, 23, 14, 14, 14, 56, 59, 60, 95,
	102, 103, 104, 105, 63, 64, 9, 108, 106, 62,
	109, 10, 70, 82, 81, 80, 72, 12, 26, 113,
	49, 107, 8, 17, 115, 110, 116, 18, 10, 19,
	21, 12, 75, 111, 12, 77, 78, 27, 18, 30,
	31, 2, 6, 4, 3, 1, 45, 16, 5, 33,
}
var sqPact = [...]int{

	127, -1000, 91, 105, 108, 11, 119, -1000, -1000, 105,
	53, 88, -1000, 9, 52, 119, 119, -6, 30, -11,
	-1000, -12, -1000, -4, 2, 2, 105, -13, -1000, -7,
	-14, -15, -1000, 62, 35, -1000, 76, 105, 50, 35,
	93, -1000, -1000, 2, 86, -1, 106, -1000, -1000, -1000,
	112, 112, -1000, -1000, 85, 84, 83, -1000, -1000, 35,
	35, -1000, 93, -3, 93, -1000, 105, 14, 16, 5,
	54, 47, 2, -1000, 105, -1000, 28, 1, 0, 28,
	105, 105, 105, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	105, -1000, -1000, 93, 2, 112, -1, -1000, 99, 109,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 12, 28, -1000,
	-1000, -2, 112, -1000, -1000, 28, -1000,
}
var sqPgo = [...]int{

	0, 139, 15, 113, 9, 138, 54, 4, 56, 132,
	16, 136, 3, 1, 0, 8, 2, 135,
}
var sqR1 = [...]int{

	0, 17, 17, 17, 17, 17, 17, 17, 17, 6,
	6, 8, 7, 7, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 9, 9, 9, 9, 10, 10,
	11, 11, 11, 11, 12, 12, 13, 13, 13, 13,
	14, 14, 3, 2, 2, 2, 2, 2, 2, 2,
	2, 15, 16, 1, 1, 1, 1,
}
var sqR2 = [...]int{

	0, 4, 3, 4, 4, 3, 4, 4, 3, 1,
	3, 3, 1, 3, 3, 3, 3, 5, 5, 5,
	1, 1, 2, 1, 9, 7, 5, 5, 1, 2,
	2, 1, 1, 1, 2, 3, 0, 2, 2, 4,
	0, 2, 2, 3, 3, 3, 3, 2, 3, 4,
	3, 1, 1, 3, 3, 2, 1,
}
var sqChk = [...]int{

	-1000, -17, 4, 7, 6, -5, -9, -6, 21, 5,
	10, -16, 16, -4, -16, -6, -9, -3, 9, -3,
	36, -3, -16, 29, 11, 12, 20, -3, 36, 18,
	-3, -3, 36, -1, 28, -2, -16, 27, -8, 31,
	33, 36, 36, 31, -10, -11, 35, -15, 15, 17,
	-10, -10, -6, 36, -15, 35, -8, 36, 36, 25,
	26, -2, 23, 18, 19, -16, 29, 28, -2, -7,
	-15, -10, 20, -12, 35, 16, -13, 13, 14, -13,
	20, 20, 20, -2, -2, -15, -15, 35, -15, -16,
	29, 32, 34, 20, 20, -10, -16, -14, 24, 35,
	35, -14, -4, -4, -4, -16, -7, -10, -13, -12,
	16, 14, 32, -14, 35, -13, -14,
}
var sqDef = [...]int{

	0, -2, 0, 0, 0, 0, 0, 20, 21, 23,
	0, 9, 52, 0, 0, 0, 0, 0, 0, 0,
	2, 0, 22, 0, 0, 0, 0, 0, 5, 0,
	0, 0, 8, 42, 0, 56, 0, 0, 0, 0,
	0, 1, 3, 0, 0, 28, 31, 32, 33, 51,
	36, 36, 10, 4, 14, 15, 16, 6, 7, 0,
	0, 55, 0, 0, 0, 47, 0, 0, 0, 0,
	12, 0, 0, 29, 0, 30, 40, 0, 0, 40,
	0, 0, 0, 53, 54, 43, 44, 45, 46, 48,
	0, 50, 11, 0, 0, 36, 34, 26, 0, 37,
	38, 27, 17, 18, 19, 49, 13, 0, 40, 35,
	41, 0, 36, 25, 39, 40, 24,
}
var sqTok1 = [...]int{

	1,
}
var sqTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38,
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
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[4].time, End: sqDollar[6].time, Limit: sqDollar[8].limit, Timeconv: sqDollar[9].timeconv}
		}
	case 25:
		sqDollar = sqS[sqpt-7 : sqpt+1]
		//line query.y:186
		{
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[3].time, End: sqDollar[5].time, Limit: sqDollar[6].limit, Timeconv: sqDollar[7].timeconv}
		}
	case 26:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:190
		{
			sqVAL.data = &DataQuery{Dtype: BEFORE_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv}
		}
	case 27:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:194
		{
			sqVAL.data = &DataQuery{Dtype: AFTER_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv}
		}
	case 28:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:200
		{
			sqVAL.time = sqDollar[1].time
		}
	case 29:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:204
		{
			sqVAL.time = sqDollar[1].time.Add(sqDollar[2].timediff)
		}
	case 30:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:210
		{
			foundtime, err := common.ParseAbsTime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.time = foundtime
		}
	case 31:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:218
		{
			num, err := strconv.ParseInt(sqDollar[1].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[1].str, err.Error()))
			}
			sqVAL.time = _time.Unix(num, 0)
		}
	case 32:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:226
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
	case 33:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:242
		{
			sqVAL.time = _time.Now()
		}
	case 34:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:248
		{
			var err error
			sqVAL.timediff, err = common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
		}
	case 35:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:256
		{
			newDuration, err := common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.timediff = common.AddDurations(newDuration, sqDollar[3].timediff)
		}
	case 36:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:266
		{
			sqVAL.limit = Limit{Limit: -1, Streamlimit: -1}
		}
	case 37:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:270
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: num, Streamlimit: -1}
		}
	case 38:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:278
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: -1, Streamlimit: num}
		}
	case 39:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:286
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
	case 40:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:300
		{
			sqVAL.timeconv = common.UOT_MS
		}
	case 41:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:304
		{
			uot, err := common.ParseUOT(sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse unit of time %v (%v)", sqDollar[2].str, err))
			}
			sqVAL.timeconv = uot
		}
	case 42:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:316
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 43:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:323
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): common.Dict{"$regex": sqDollar[3].str}}
		}
	case 44:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:327
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 45:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:331
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 46:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:335
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[1].str): common.Dict{"$neq": sqDollar[3].str}}
		}
	case 47:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:339
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[2].str): common.Dict{"$exists": true}}
		}
	case 48:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:343
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[3].str): common.Dict{"$in": sqDollar[1].list}}
		}
	case 49:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:347
		{
			sqVAL.dict = common.Dict{fixMongoKey(sqDollar[3].str): common.Dict{"$not": common.Dict{"$in": sqDollar[1].list}}}
		}
	case 50:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:351
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 51:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:357
		{
			sqVAL.str = sqDollar[1].str[1 : len(sqDollar[1].str)-1]
		}
	case 52:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:363
		{

			sqlex.(*sqLex)._keys[sqDollar[1].str] = struct{}{}
			sqVAL.str = cleantagstring(sqDollar[1].str)
		}
	case 53:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:371
		{
			sqVAL.dict = common.Dict{"$and": []common.Dict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 54:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:375
		{
			sqVAL.dict = common.Dict{"$or": []common.Dict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 55:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:379
		{
			tmp := make(common.Dict)
			for k, v := range sqDollar[2].dict {
				tmp[k] = common.Dict{"$ne": v}
			}
			sqVAL.dict = tmp
		}
	case 56:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:387
		{
			sqVAL.dict = sqDollar[1].dict
		}
	}
	goto sqstack /* stack new state and value */
}
