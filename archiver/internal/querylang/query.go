//line query.y:2
package querylang

import __yyfmt__ "fmt"

//line query.y:3
import (
	"bufio"
	"fmt"
	"github.com/gtfierro/giles2/common"
	"github.com/taylorchu/toki"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	_time "time"
)

/**
Notes here
**/

//line query.y:21
type sqSymType struct {
	yys      int
	str      string
	dict     qDict
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

//line query.y:386

const eof = 0

var supported_formats = []string{"1/2/2006",
	"1-2-2006",
	"1/2/2006 03:04:05 PM MST",
	"1-2-2006 03:04:05 PM MST",
	"1/2/2006 15:04:05 MST",
	"1-2-2006 15:04:05 MST",
	"2006-1-2 15:04:05 MST"}

type qDict map[string]interface{}
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
	set qDict
	// where clause for query
	where qDict
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

func (q *query) ContentsBson() bson.M {
	ret := bson.M{}
	for _, tag := range q.Contents {
		ret[tag] = 1
	}
	return ret
}

func (q *query) WhereBson() bson.M {
	return bson.M(q.where)
}

func (q *query) SetBson() bson.M {
	return bson.M(q.set)
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

const sqNprod = 56
const sqPrivate = 57344

var sqTokenNames []string
var sqStates []string

const sqLast = 134

var sqAct = [...]int{

	94, 73, 35, 70, 66, 11, 14, 11, 45, 13,
	55, 46, 21, 47, 47, 31, 42, 17, 46, 17,
	47, 51, 40, 39, 30, 111, 71, 41, 11, 97,
	38, 44, 53, 96, 47, 89, 109, 52, 44, 63,
	48, 49, 12, 88, 27, 95, 19, 67, 58, 59,
	37, 76, 85, 36, 32, 56, 57, 33, 68, 38,
	23, 24, 82, 65, 64, 56, 57, 87, 91, 83,
	84, 86, 80, 81, 93, 90, 79, 98, 22, 54,
	14, 14, 14, 78, 77, 69, 92, 99, 100, 101,
	25, 102, 28, 7, 105, 103, 16, 106, 15, 67,
	61, 62, 18, 20, 9, 60, 110, 47, 104, 10,
	26, 112, 29, 113, 17, 12, 107, 12, 72, 50,
	8, 12, 74, 75, 108, 17, 2, 1, 4, 3,
	43, 6, 5, 34,
}
var sqPact = [...]int{

	122, -1000, 99, 101, 105, 10, 116, -1000, -1000, 101,
	49, 70, -1000, 8, 74, 116, -12, 26, -13, -1000,
	-14, -1000, -4, 3, 3, 101, -15, -1000, -3, -26,
	-1000, 40, 26, 26, -1000, 82, 101, 35, 90, -1000,
	-1000, 3, 65, -9, 102, -1000, -1000, -1000, 109, 109,
	-1000, -1000, 64, 63, 56, -1000, 26, 26, 40, 30,
	90, 17, 90, -1000, 101, 14, 1, 55, 48, 3,
	-1000, 101, -1000, 21, -2, -6, 21, 101, 101, 101,
	40, 40, -1000, -1000, -1000, -1000, -1000, -1000, 101, -1000,
	90, 3, 109, -9, -1000, 100, 110, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 4, 21, -1000, -1000, -10, 109,
	-1000, -1000, 21, -1000,
}
var sqPgo = [...]int{

	0, 15, 133, 96, 9, 132, 93, 4, 50, 131,
	16, 130, 3, 1, 0, 8, 2, 127,
}
var sqR1 = [...]int{

	0, 17, 17, 17, 17, 17, 17, 17, 6, 6,
	8, 7, 7, 4, 4, 4, 4, 4, 4, 5,
	5, 5, 5, 9, 9, 9, 9, 10, 10, 11,
	11, 11, 11, 12, 12, 13, 13, 13, 13, 14,
	14, 3, 2, 2, 2, 2, 2, 2, 2, 15,
	16, 1, 1, 1, 1, 1,
}
var sqR2 = [...]int{

	0, 4, 3, 4, 4, 3, 4, 3, 1, 3,
	3, 1, 3, 3, 3, 3, 5, 5, 5, 1,
	1, 2, 1, 9, 7, 5, 5, 1, 2, 2,
	1, 1, 1, 2, 3, 0, 2, 2, 4, 0,
	2, 2, 3, 3, 3, 3, 2, 3, 4, 1,
	1, 3, 3, 2, 3, 1,
}
var sqChk = [...]int{

	-1000, -17, 4, 7, 6, -5, -9, -6, 21, 5,
	10, -16, 16, -4, -16, -6, -3, 9, -3, 36,
	-3, -16, 29, 11, 12, 20, -3, 36, 18, -3,
	36, -1, 28, 31, -2, -16, 27, -8, 33, 36,
	36, 31, -10, -11, 35, -15, 15, 17, -10, -10,
	-6, 36, -15, 35, -8, 36, 25, 26, -1, -1,
	23, 18, 19, -16, 29, 28, -7, -15, -10, 20,
	-12, 35, 16, -13, 13, 14, -13, 20, 20, 20,
	-1, -1, 32, -15, -15, 35, -15, -16, 29, 34,
	20, 20, -10, -16, -14, 24, 35, 35, -14, -4,
	-4, -4, -16, -7, -10, -13, -12, 16, 14, 32,
	-14, 35, -13, -14,
}
var sqDef = [...]int{

	0, -2, 0, 0, 0, 0, 0, 19, 20, 22,
	0, 8, 50, 0, 0, 0, 0, 0, 0, 2,
	0, 21, 0, 0, 0, 0, 0, 5, 0, 0,
	7, 41, 0, 0, 55, 0, 0, 0, 0, 1,
	3, 0, 0, 27, 30, 31, 32, 49, 35, 35,
	9, 4, 13, 14, 15, 6, 0, 0, 53, 0,
	0, 0, 0, 46, 0, 0, 0, 11, 0, 0,
	28, 0, 29, 39, 0, 0, 39, 0, 0, 0,
	51, 52, 54, 42, 43, 44, 45, 47, 0, 10,
	0, 0, 35, 33, 25, 0, 36, 37, 26, 16,
	17, 18, 48, 12, 0, 39, 34, 40, 0, 35,
	24, 38, 39, 23,
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
		//line query.y:60
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.qtype = SELECT_TYPE
		}
	case 2:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:66
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.qtype = SELECT_TYPE
		}
	case 3:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:71
		{
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.data = sqDollar[2].data
			sqlex.(*sqLex).query.qtype = DATA_TYPE
		}
	case 4:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:77
		{
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.set = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = SET_TYPE
		}
	case 5:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:83
		{
			sqlex.(*sqLex).query.set = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = SET_TYPE
		}
	case 6:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:88
		{
			sqlex.(*sqLex).query.Contents = sqDollar[2].list
			sqlex.(*sqLex).query.where = sqDollar[3].dict
			sqlex.(*sqLex).query.qtype = DELETE_TYPE
		}
	case 7:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:94
		{
			sqlex.(*sqLex).query.Contents = []string{}
			sqlex.(*sqLex).query.where = sqDollar[2].dict
			sqlex.(*sqLex).query.qtype = DELETE_TYPE
		}
	case 8:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:102
		{
			sqVAL.list = List{sqDollar[1].str}
		}
	case 9:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:106
		{
			sqVAL.list = append(List{sqDollar[1].str}, sqDollar[3].list...)
		}
	case 10:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:112
		{
			sqVAL.list = sqDollar[2].list
		}
	case 11:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:117
		{
			sqVAL.list = List{sqDollar[1].str}
		}
	case 12:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:121
		{
			sqVAL.list = append(List{sqDollar[1].str}, sqDollar[3].list...)
		}
	case 13:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:127
		{
			sqVAL.dict = qDict{sqDollar[1].str: sqDollar[3].str}
		}
	case 14:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:131
		{
			sqVAL.dict = qDict{sqDollar[1].str: sqDollar[3].str}
		}
	case 15:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:135
		{
			sqVAL.dict = qDict{sqDollar[1].str: sqDollar[3].list}
		}
	case 16:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:139
		{
			sqDollar[5].dict[sqDollar[1].str] = sqDollar[3].str
			sqVAL.dict = sqDollar[5].dict
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
			sqDollar[5].dict[sqDollar[1].str] = sqDollar[3].list
			sqVAL.dict = sqDollar[5].dict
		}
	case 19:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:156
		{
			sqlex.(*sqLex).query.Contents = sqDollar[1].list
			sqVAL.list = sqDollar[1].list
		}
	case 20:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:161
		{
			sqVAL.list = List{}
		}
	case 21:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:165
		{
			sqlex.(*sqLex).query.distinct = true
			sqVAL.list = List{sqDollar[2].str}
		}
	case 22:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:170
		{
			sqlex.(*sqLex).query.distinct = true
			sqVAL.list = List{}
		}
	case 23:
		sqDollar = sqS[sqpt-9 : sqpt+1]
		//line query.y:177
		{
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[4].time, End: sqDollar[6].time, Limit: sqDollar[8].limit, Timeconv: sqDollar[9].timeconv}
		}
	case 24:
		sqDollar = sqS[sqpt-7 : sqpt+1]
		//line query.y:181
		{
			sqVAL.data = &DataQuery{Dtype: IN_TYPE, Start: sqDollar[3].time, End: sqDollar[5].time, Limit: sqDollar[6].limit, Timeconv: sqDollar[7].timeconv}
		}
	case 25:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:185
		{
			sqVAL.data = &DataQuery{Dtype: BEFORE_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv}
		}
	case 26:
		sqDollar = sqS[sqpt-5 : sqpt+1]
		//line query.y:189
		{
			sqVAL.data = &DataQuery{Dtype: AFTER_TYPE, Start: sqDollar[3].time, Limit: sqDollar[4].limit, Timeconv: sqDollar[5].timeconv}
		}
	case 27:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:195
		{
			sqVAL.time = sqDollar[1].time
		}
	case 28:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:199
		{
			sqVAL.time = sqDollar[1].time.Add(sqDollar[2].timediff)
		}
	case 29:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:205
		{
			foundtime, err := common.ParseAbsTime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.time = foundtime
		}
	case 30:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:213
		{
			num, err := strconv.ParseInt(sqDollar[1].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[1].str, err.Error()))
			}
			sqVAL.time = _time.Unix(num, 0)
		}
	case 31:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:221
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
	case 32:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:237
		{
			sqVAL.time = _time.Now()
		}
	case 33:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:243
		{
			var err error
			sqVAL.timediff, err = common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
		}
	case 34:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:251
		{
			newDuration, err := common.ParseReltime(sqDollar[1].str, sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", sqDollar[1].str, sqDollar[2].str, err.Error()))
			}
			sqVAL.timediff = common.AddDurations(newDuration, sqDollar[3].timediff)
		}
	case 35:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:261
		{
			sqVAL.limit = Limit{Limit: -1, Streamlimit: -1}
		}
	case 36:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:265
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: num, Streamlimit: -1}
		}
	case 37:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:273
		{
			num, err := strconv.ParseInt(sqDollar[2].str, 10, 64)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", sqDollar[2].str, err.Error()))
			}
			sqVAL.limit = Limit{Limit: -1, Streamlimit: num}
		}
	case 38:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:281
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
	case 39:
		sqDollar = sqS[sqpt-0 : sqpt+1]
		//line query.y:295
		{
			sqVAL.timeconv = common.UOT_MS
		}
	case 40:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:299
		{
			uot, err := common.ParseUOT(sqDollar[2].str)
			if err != nil {
				sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse unit of time %v (%v)", sqDollar[2].str, err))
			}
			sqVAL.timeconv = uot
		}
	case 41:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:311
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 42:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:318
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[1].str): qDict{"$regex": sqDollar[3].str}}
		}
	case 43:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:322
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 44:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:326
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[1].str): sqDollar[3].str}
		}
	case 45:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:330
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[1].str): qDict{"$neq": sqDollar[3].str}}
		}
	case 46:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:334
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[2].str): qDict{"$exists": true}}
		}
	case 47:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:338
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[3].str): qDict{"$in": sqDollar[1].list}}
		}
	case 48:
		sqDollar = sqS[sqpt-4 : sqpt+1]
		//line query.y:342
		{
			sqVAL.dict = qDict{fixMongoKey(sqDollar[3].str): qDict{"$not": qDict{"$in": sqDollar[1].list}}}
		}
	case 49:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:348
		{
			sqVAL.str = sqDollar[1].str[1 : len(sqDollar[1].str)-1]
		}
	case 50:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:354
		{

			sqlex.(*sqLex)._keys[sqDollar[1].str] = struct{}{}
			sqVAL.str = cleantagstring(sqDollar[1].str)
		}
	case 51:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:362
		{
			sqVAL.dict = qDict{"$and": []qDict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 52:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:366
		{
			sqVAL.dict = qDict{"$or": []qDict{sqDollar[1].dict, sqDollar[3].dict}}
		}
	case 53:
		sqDollar = sqS[sqpt-2 : sqpt+1]
		//line query.y:370
		{
			tmp := make(qDict)
			for k, v := range sqDollar[2].dict {
				tmp[k] = qDict{"$ne": v}
			}
			sqVAL.dict = tmp
		}
	case 54:
		sqDollar = sqS[sqpt-3 : sqpt+1]
		//line query.y:378
		{
			sqVAL.dict = sqDollar[2].dict
		}
	case 55:
		sqDollar = sqS[sqpt-1 : sqpt+1]
		//line query.y:382
		{
			sqVAL.dict = sqDollar[1].dict
		}
	}
	goto sqstack /* stack new state and value */
}
