%{

package querylang

import (
	"bufio"
	"fmt"
	"github.com/taylorchu/toki"
    "github.com/gtfierro/giles2/common"
	"strconv"
    _time "time"
)


/**
Notes here
**/
%}

%union{
	str string
	dict common.Dict
	data *DataQuery
	limit Limit
    timeconv common.UnitOfTime
	list List
	time _time.Time
    timediff _time.Duration
}

%token <str> SELECT DISTINCT DELETE SET APPLY
%token <str> WHERE
%token <str> DATA BEFORE AFTER LIMIT STREAMLIMIT NOW
%token <str> LVALUE QSTRING
%token <str> EQ NEQ COMMA ALL LEFTPIPE
%token <str> LIKE AS
%token <str> AND OR HAS NOT IN TO
%token <str> LPAREN RPAREN LBRACK RBRACK
%token NUMBER
%token SEMICOLON
%token NEWLINE
%token TIMEUNIT

%type <dict> whereList whereTerm whereClause setList
%type <list> selector tagList valueList valueListBrack
%type <data> dataClause
%type <time> timeref abstime
%type <timediff> reltime
%type <limit> limit
%type <timeconv> timeconv
%type <str> NUMBER qstring lvalue TIMEUNIT
%type <str> SEMICOLON NEWLINE

%right EQ

%%

query		: SELECT selector whereClause SEMICOLON
			{
				sqlex.(*sqLex).query.Contents = $2
				sqlex.(*sqLex).query.where = $3
				sqlex.(*sqLex).query.qtype = SELECT_TYPE
			}
			| SELECT selector SEMICOLON
			{
				sqlex.(*sqLex).query.Contents = $2
				sqlex.(*sqLex).query.qtype = SELECT_TYPE
			}
			| SELECT dataClause whereClause SEMICOLON
			{
				sqlex.(*sqLex).query.where = $3
				sqlex.(*sqLex).query.data = $2
				sqlex.(*sqLex).query.qtype = DATA_TYPE
			}
            | SET setList whereClause SEMICOLON
            {
				sqlex.(*sqLex).query.where = $3
				sqlex.(*sqLex).query.set = $2
                sqlex.(*sqLex).query.qtype = SET_TYPE
            }
            | SET setList SEMICOLON
            {
				sqlex.(*sqLex).query.set = $2
                sqlex.(*sqLex).query.qtype = SET_TYPE
            }
			| DELETE tagList whereClause SEMICOLON
			{
				sqlex.(*sqLex).query.Contents = $2
				sqlex.(*sqLex).query.where = $3
				sqlex.(*sqLex).query.qtype = DELETE_TYPE
			}
            | DELETE dataClause whereClause SEMICOLON
            {
				sqlex.(*sqLex).query.data = $2
				sqlex.(*sqLex).query.where = $3
				sqlex.(*sqLex).query.qtype = DELETE_TYPE
            }
			| DELETE whereClause SEMICOLON
			{
				sqlex.(*sqLex).query.Contents = []string{}
				sqlex.(*sqLex).query.where = $2
				sqlex.(*sqLex).query.qtype = DELETE_TYPE
			}
			;

tagList		: lvalue
			{
				$$ = List{$1}
			}
			| lvalue COMMA tagList
			{
				$$ = append(List{$1}, $3...)
			}
			;

valueListBrack : LBRACK valueList RBRACK
                 {
                  $$ = $2
                 }
               ;
valueList   : qstring
            {
                $$ = List{$1}
            }
            | qstring COMMA valueList
            {
                $$ = append(List{$1}, $3...)
            }
            ;

setList     : lvalue EQ qstring
            {
                $$ = common.Dict{$1: $3}
            }
            | lvalue EQ NUMBER
            {
                $$ = common.Dict{$1: $3}
            }
            | lvalue EQ valueListBrack
            {
                $$ = common.Dict{$1: $3}
            }
            | lvalue EQ qstring COMMA setList
            {
                $5[$1] = $3
                $$ = $5
            }
            | lvalue EQ NUMBER COMMA setList
            {
                $5[$1] = $3
                $$ = $5
            }
            | lvalue EQ valueListBrack COMMA setList
            {
                $5[$1] = $3
                $$ = $5
            }
            ;

selector	: tagList
			{
                sqlex.(*sqLex).query.Contents = $1
				$$ = $1
			}
			| ALL
			{
				$$ = List{};
			}
			| DISTINCT lvalue
			{
				sqlex.(*sqLex).query.distinct = true
				$$ = List{$2}
			}
			| DISTINCT
			{
				sqlex.(*sqLex).query.distinct = true
				$$ = List{}
			}
			;

dataClause : DATA IN LPAREN timeref COMMA timeref RPAREN limit timeconv
			{
				$$ = &DataQuery{Dtype: IN_TYPE, Start: $4, End: $6, Limit: $8, Timeconv: $9}
			}
		   | DATA IN timeref COMMA timeref limit timeconv
			{
				$$ = &DataQuery{Dtype: IN_TYPE, Start: $3, End: $5, Limit: $6, Timeconv: $7}
			}
		   | DATA BEFORE timeref limit timeconv
			{
				$$ = &DataQuery{Dtype: BEFORE_TYPE, Start: $3, Limit: $4, Timeconv: $5}
			}
		   | DATA AFTER timeref limit timeconv
			{
				$$ = &DataQuery{Dtype: AFTER_TYPE, Start: $3, Limit: $4, Timeconv: $5}
			}
		   ;

timeref		: abstime
			{
				$$ = $1
			}
			| abstime reltime
			{
                $$ = $1.Add($2)
			}
			;

abstime		: NUMBER LVALUE
            {
                foundtime, err := common.ParseAbsTime($1, $2)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse time \"%v %v\" (%v)", $1, $2, err.Error()))
                }
                $$ = foundtime
            }
            | NUMBER
            {
                num, err := strconv.ParseInt($1, 10, 64)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", $1, err.Error()))
                }
                $$ = _time.Unix(num, 0)
            }
			| qstring
            {
                found := false
                for _, format := range supported_formats {
                    t, err := _time.Parse(format, $1)
                    if err != nil {
                        continue
                    }
                    $$ = t
                    found = true
                    break
                }
                if !found {
				    sqlex.(*sqLex).Error(fmt.Sprintf("No time format matching \"%v\" found", $1))
                }
            }
			| NOW
            {
                $$ = _time.Now()
            }
			;

reltime		: NUMBER lvalue
            {
                var err error
                $$, err = common.ParseReltime($1, $2)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", $1, $2, err.Error()))
                }
            }
			| NUMBER lvalue reltime
            {
                newDuration, err := common.ParseReltime($1, $2)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Error parsing relative time \"%v %v\" (%v)", $1, $2, err.Error()))
                }
                $$ = common.AddDurations(newDuration, $3)
            }
			;

limit		: /* empty */
			{
				$$ = Limit{Limit: -1, Streamlimit: -1}
			}
			| LIMIT NUMBER
			{
				num, err := strconv.ParseInt($2, 10, 64)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", $2, err.Error()))
                }
				$$ = Limit{Limit: num, Streamlimit: -1}
			}
			| STREAMLIMIT NUMBER
			{
				num, err := strconv.ParseInt($2, 10, 64)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", $2, err.Error()))
                }
				$$ = Limit{Limit: -1, Streamlimit: num}
			}
			| LIMIT NUMBER STREAMLIMIT NUMBER
			{
				limit_num, err := strconv.ParseInt($2, 10, 64)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", $2, err.Error()))
                }
				slimit_num, err := strconv.ParseInt($4, 10, 64)
                if err != nil {
				    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse integer \"%v\" (%v)", $2, err.Error()))
                }
				$$ = Limit{Limit: limit_num, Streamlimit: slimit_num}
			}
			;

timeconv    : /* empty */
            {
                $$ = common.UOT_MS
            }
            | AS LVALUE
            {
                uot, err := common.ParseUOT($2)
                if err != nil {
                    sqlex.(*sqLex).Error(fmt.Sprintf("Could not parse unit of time %v (%v)", $2, err))
                }
                $$ = uot
            }
            ;



whereClause : WHERE whereList
			{
			  $$ = $2
			}
			;


whereTerm : lvalue LIKE qstring
			{
				$$ = common.Dict{fixMongoKey($1): common.Dict{"$regex": $3}}
			}
		  | lvalue EQ qstring
			{
				$$ = common.Dict{fixMongoKey($1): $3}
			}
          | lvalue EQ NUMBER
            {
				$$ = common.Dict{fixMongoKey($1): $3}
            }
		  | lvalue NEQ qstring
			{
				$$ = common.Dict{fixMongoKey($1): common.Dict{"$neq": $3}}
			}
		  | HAS lvalue
			{
				$$ = common.Dict{fixMongoKey($2): common.Dict{"$exists": true}}
			}
          | valueListBrack IN lvalue
            {
                $$ = common.Dict{fixMongoKey($3): common.Dict{"$in": $1}}
            }
          | valueListBrack NOT IN lvalue
            {
                $$ = common.Dict{fixMongoKey($3): common.Dict{"$not": common.Dict{"$in": $1}}}
            }
          | LPAREN whereTerm RPAREN
            {
                $$ = $2
            }
		  ;

qstring   : QSTRING
          {
            $$ = $1[1:len($1)-1]
          }
          ;

lvalue    : LVALUE
          {

		    sqlex.(*sqLex)._keys[$1] = struct{}{}
            $$ = cleantagstring($1)
          }
          ;

whereList : whereList AND whereTerm
			{
				$$ = common.Dict{"$and": []common.Dict{$1, $3}}
			}
		  | whereList OR whereTerm
			{
				$$ = common.Dict{"$or": []common.Dict{$1, $3}}
			}
		  | NOT whereTerm
			{
                tmp := make(common.Dict)
                for k,v := range $2 {
                    tmp[k] = common.Dict{"$ne": v}
                }
				$$ = tmp
			}
		  | whereTerm
			{
				$$ = $1
			}
		  ;
%%

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
	qtype	   QueryType
	// information about a data query if we are one
	data	   *DataQuery
    // key-value pairs to add
    set         common.Dict
	// where clause for query
	where	  common.Dict
	// are we querying distinct values?
	distinct  bool
	// list of tags to target for deletion, selection
	Contents  []string
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
	query	*query
	scanner *toki.Scanner
    lasttoken string
    tokens  []string
    error   error
    // all keys that we encounter. Used for republish concerns
    _keys    map[string]struct{}
    keys    []string
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
