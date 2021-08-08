
%{

package main

import (
	"bytes"
	"fmt"
	"log"
	//"math/big"
	"unicode/utf8"
    "unicode"
    // "encoding/json"
    "strings"
    "strconv"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// represents a select statement
type AST struct {
    Output_level string
    Select_level string
    Select_level_by_rule []string
    Rule_list_names []string
    Rules [][]Rule // we need sets of rules for each series we describe
    CheckRules [][]Rule // we capture the special check rules here
}

var ast AST                 // our abstract syntax tree
var charpos int             // the char position in the string
var program string          // just a copy of the string to parse

var currentRules []Rule = nil       // we store one rules information here
var currentCheckRules []Rule = nil  // for checks there is a separate list
var errorOnParse = false
var lastGroupTag []string           // a pair of group, tag in decimal format

%}

%union {
	num  float64
    word string
}

%type <word> command, select_stmt, base_select, level_types, rule_list, rule, where_clause, where_clauses, level_types_with_name
%type <word> check_stmt base_check check_rule_list check_rule tag_string group_tag_pair

%token '+' '-' '*' '/' '(' ')' '"' '\''
%token SELECT FROM PATIENT STUDY SERIES IMAGE WHERE EQUALS HAS AND ALSO
%token CONTAINING SMALLER LARGER REGEXP NOT NAMED PROJECT CHECK

%token	<num>	NUM
%token  <word>  STRING NOT

%start top

%%

top:
    command semicolon_opt
    {
        //fmt.Printf("command \"%w\"\n", $1)
        //s, _ := json.MarshalIndent(ast, "", "  ")
        //fmt.Printf("internal ast is: \"%s\"\n",string(s))
    };

semicolon_opt:
/*empty*/ {}
| ';' {};

command:
    select_stmt
    {
        $$ = $1
    }
|   check_stmt
    {
        $$ = $1
    };

select_stmt:
    base_select
    {
        $$ = $1
    };

check_stmt:
    base_check
    {
        $$ = $1
    };

base_select:
    SELECT level_types FROM level_types where_clauses 
    {
        // the FROM should be more complex. Something like:
        // FROM earliest study BY StudyDate AS DICOM

        ast.Output_level = string($2)
        ast.Select_level = string($4)
        currentRules = nil
        // get space in Rules now for rules
        if ast.Rules == nil {
            ast.Rules = make([][]Rule, 0)
        }

        $$ = fmt.Sprintf("\nlevel types: %s, from: %s", $2, $4)
    }
|   SELECT level_types where_clauses 
    {
        // the FROM should be more complex. Something like:
        // FROM earliest study BY StudyDate AS DICOM

        ast.Output_level = string($2)
        ast.Select_level = "series"
        if ast.Rule_list_names == nil {
            ast.Rule_list_names = make([]string,0)
        }
        currentRules = nil
        if ast.Rules == nil {
            ast.Rules = make([][]Rule, 0)
        }

        $$ = fmt.Sprintf("\nlevel types: %s, from: %s", $2, $3)
    };

where_clauses:
    where_clause
    {
        $$ = $1
    }
| where_clauses ALSO where_clause
    {
        //fmt.Println("We have an ALSO WHERE here")
        $$ = $1
    }

where_clause:
    /*empty*/
    {
        $$ = fmt.Sprintf("no where clause")
    }
|   WHERE level_types_with_name HAS rule_list
    {
        if len(currentRules) > 0 {
            // add the currentRules if they are not already in the list
            ast.Rules = append(ast.Rules, currentRules)
            ast.Select_level_by_rule = append(ast.Select_level_by_rule, $2)
            currentRules = nil
        }
        $$ = fmt.Sprintf("found a where clause with: %s and ruleset %s", $2, $4)
    }
|   WHERE rule_list
    {
        if len(currentRules) > 0 {
            // add the currentRules if they are not already in the list
            ast.Rules = append(ast.Rules, currentRules)
            ast.Select_level_by_rule = append(ast.Select_level_by_rule, "series")
            currentRules = nil
        }
        $$ = fmt.Sprintf("found a where clause with: series and ruleset %s", $2)
    }

level_types_with_name:
    level_types
    {
        ast.Rule_list_names = append(ast.Rule_list_names, "no-name")
        $$ = $1
    }
|   level_types NAMED STRING
    {
        $$ = $1
        ast.Rule_list_names = append(ast.Rule_list_names, $3)
    }

rule_list:
    rule 
    {
        //fmt.Printf("found a rule: \"%s\"\n", $1)
        // add the rule to the current list of rules

        $$ = $1
    }
|   rule_list AND rule
    {
        //fmt.Println("found AND rule")
        $$ = fmt.Sprintf("%s AND %s", $$, $3)
    }

rule:
    '(' rule_list ')'
    {
        $$ = fmt.Sprintf("%s, brackets %s", $$, $2)
    }
|   NOT rule
    {
        if currentRules[len(currentRules)-1].Negate == "" || currentRules[len(currentRules)-1].Negate == "no" {
            currentRules[len(currentRules)-1].Negate = "yes"
        } else {
            currentRules[len(currentRules)-1].Negate = "no"
        }
        $$ = fmt.Sprintf("%s NOT %s", $$, $1)
    }
|   tag_string EQUALS STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "==",
            Value: $3,
        }
        currentRules = append(currentRules, r)
        $$ = fmt.Sprintf("Variable %s = %s", $1, $3)
    }
|   tag_string CONTAINING STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "contains",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string SMALLER NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "<",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("Variable %s contains %f", $1, $3)
    }
|   tag_string LARGER NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: ">",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("Variable %s contains %f", $1, $3)
    }
|   tag_string REGEXP STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "regexp",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("Variable %s contains %s", $1, $3)
    }

tag_string:
    STRING
    { 
        $$ = $1
        // we should also set the lastGroupTag here so wherever we use
        // tag_string we would have such a pair (mapping from string to tag pair)
        s, err := tag.FindByName($1)
        if err == nil {
            lastGroupTag = []string{fmt.Sprintf("%0x", s.Tag.Group), fmt.Sprintf("%0x", s.Tag.Element)}
        } else {
            lastGroupTag = []string{$1} // This could be classifyType, keep the value provided
        }
    }
|   '(' group_tag_pair ')'
    {
        $$ = $2
    }

group_tag_pair:
    STRING ',' STRING
    {
        // get the corresponding group and tag from hexadecimal
        group_str := strings.Replace($1,"0x","",-1)
        group_str = strings.Replace(group_str, "0X","", -1)
        group, err := strconv.ParseInt(group_str, 16, 64)
        if err != nil {
            exitGracefully(err)
        }
        tag_str := strings.Replace($3,"0x","",-1)
        tag_str = strings.Replace(tag_str, "0X","", -1)
        tag, err := strconv.ParseInt(tag_str, 16, 64)
        if err != nil {
            exitGracefully(err)
        }
        //lastGroupTag = []int{int(group), int(tag)}
        lastGroupTag = []string{$1, $3}
        $$ = fmt.Sprintf("(%d,%d)", group, tag)
    }

level_types:
    /*empty*/ 
    {
        $$ = fmt.Sprintf("if empty we can assume series level as default")
    }
|   PROJECT
    {
        $$ = fmt.Sprintf("project")
    }
|   PATIENT
    {
        $$ = fmt.Sprintf("patient")
    }
|   STUDY   
    {
        $$ = fmt.Sprintf("study")
    }
|   SERIES   
    {
        $$ = fmt.Sprintf("series")
    }
|   IMAGE   
    {
        $$ = fmt.Sprintf("image")
    }

base_check:
    CHECK check_rule_list
    {
        // do nothing for now...
    }

check_rule_list:
    check_rule 
    {
        //fmt.Printf("found a rule: \"%s\"\n", $1)
        // add the rule to the current list of rules

        $$ = $1
    }
|   check_rule_list AND check_rule
    {
        //fmt.Println("found AND rule")
        $$ = fmt.Sprintf("%s AND %s", $$, $3)
    }

check_rule:
    '(' check_rule_list ')'
    {
        $$ = fmt.Sprintf("%s, brackets %s", $$, $2)
    }
|   NOT check_rule
    {
        if currentCheckRules[len(currentCheckRules)-1].Negate == "" || currentCheckRules[len(currentCheckRules)-1].Negate == "no" {
            currentCheckRules[len(currentCheckRules)-1].Negate = "yes"
        } else {
            currentCheckRules[len(currentCheckRules)-1].Negate = "no"
        }
        $$ = fmt.Sprintf("%s NOT %s", $$, $1)
    }
|   STRING EQUALS STRING
    {
        r := Rule{
            Tag: []string{$1},
            Operator: "==",
            Value: $3,
        }
        currentCheckRules = append(currentCheckRules, r)
        $$ = fmt.Sprintf("Variable %s = %s", $1, $3)
    }
|   STRING CONTAINING STRING
    {
        r := Rule{
            Tag: []string{$1},
            Operator: "contains",
            Value: $3,
        }
        currentCheckRules = append(currentCheckRules, r)

        $$ = fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   STRING SMALLER NUM
    {
        r := Rule{
            Tag: []string{$1},
            Operator: "<",
            Value: $3,
        }
        currentCheckRules = append(currentCheckRules, r)

        $$ = fmt.Sprintf("Variable %s contains %f", $1, $3)
    }
|   STRING LARGER NUM
    {
        r := Rule{
            Tag: []string{$1},
            Operator: ">",
            Value: $3,
        }
        currentCheckRules = append(currentCheckRules, r)

        $$ = fmt.Sprintf("Variable %s contains %f", $1, $3)
    }
|   STRING REGEXP STRING
    {
        r := Rule{
            Tag: []string{$1},
            Operator: "regexp",
            Value: $3,
        }
        currentCheckRules = append(currentCheckRules, r)

        $$ = fmt.Sprintf("Variable %s contains %s", $1, $3)
    }



%%

func InitParser() {
    currentRules = nil
    ast.Rules = nil
    ast.Output_level = ""
    ast.Select_level = ""
    errorOnParse = false
    charpos = 0
    program = ""
}

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const eof = 0

// The parser uses the type <prefix>Lex as a lexer. It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type exprLex struct {
	line []byte
	peek rune
}

// The parser calls this method to get each new token. This
// implementation returns operators and NUM.
func (x *exprLex) Lex(yylval *yySymType) int {
	for {
		c := x.next()
		switch c {
		case eof:
			return eof
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return x.num(c, yylval)
		case '+', '-', '*', '/', '(', ')':
            charpos = charpos + 1
			return int(c)

		// Recognize Unicode multiplication and division
		// symbols, returning what the parser expects.
		case '×':
            charpos = charpos + 1
			return '*'
		case '÷':
            charpos = charpos + 1
			return '/'
        case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
            return x.word(c, yylval, rune(0))
        case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
            return x.word(c, yylval, rune(0))
        case '^', '$', '[', ']', '.':
            return x.word(c, yylval, rune(0))
        case '<':
            return SMALLER
        case '>':
            return LARGER
        case '=':
            return EQUALS
        case '"':
            // read until the next delimiter (eat up spaces as well)
            return x.word(c, yylval, rune('"'))
		case ' ', '\t', '\n', '\r':
            charpos = charpos + 1
		default:
			log.Printf("unrecognized character %q", c)
		}
	}
}

// Lex a word.
func (x *exprLex) word(c rune, yylval *yySymType, delimiter rune) int {
	add := func(b *bytes.Buffer, c rune) {
		if _, err := b.WriteRune(c); err != nil {
			log.Fatalf("WriteRune: %s", err)
		}
	}
	var b bytes.Buffer
    if delimiter == rune(0) {
	    add(&b, c)
    }
	L: for {
		c = x.next()
        if unicode.IsSpace(c) {
            if delimiter == rune(0) {
                charpos = charpos + 1
                break L
            } else {
                add(&b, c)
                charpos = charpos + 1
                continue L
            }
        }
		switch c {
		case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
			add(&b, c)
            charpos = charpos + 1
        case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
			add(&b, c)
            charpos = charpos + 1
        case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '^', '$', '.', '*', '[', ']':
			add(&b, c)
            charpos = charpos + 1
        case delimiter:
            c = x.next() // make sure we will not look at it if it stays in peek
            charpos = charpos + 1
            break L
		default:
			break L
		}
	}
	if c != eof {
		x.peek = c
	}
	yylval.word = ""
    if strings.ToLower(b.String()) == "select" {
        return SELECT
    } else if strings.ToLower(b.String()) == "from" {
        return FROM
    } else if b.String() == "project" {
        return PROJECT
    } else if b.String() == "patient" {
        return PATIENT
    } else if b.String() == "study" {
        return STUDY
    } else if b.String() == "series" {
        return SERIES
    } else if b.String() == "image" {
        return IMAGE
    } else if strings.ToLower(b.String()) == "has" {
        return HAS
    } else if strings.ToLower(b.String()) == "and" {
        return AND
    } else if strings.ToLower(b.String()) == "containing" {
        return CONTAINING
    } else if strings.ToLower(b.String()) == "where" {
        return WHERE
    } else if strings.ToLower(b.String()) == "also" {
        return ALSO
    } else if strings.ToLower(b.String()) == "regexp" {
        return REGEXP
    } else if strings.ToLower(b.String()) == "not" {
        return NOT
    } else if strings.ToLower(b.String()) == "named" {
        return NAMED
    } else if strings.ToLower(b.String()) == "check" {
        return CHECK
    } else {
		log.Printf("unknown word %s", b.String())
        yylval.word = b.String()
        return STRING
    }
    // this code is unreachable
	// return STRING
}

// Lex a number.
func (x *exprLex) num(c rune, yylval *yySymType) int {
	add := func(b *bytes.Buffer, c rune) {
		if _, err := b.WriteRune(c); err != nil {
			log.Fatalf("WriteRune: %s", err)
		}
	}
	var b bytes.Buffer
	add(&b, c)
	L: for {
		c = x.next()
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.', 'e', 'E':
			add(&b, c)
            charpos = charpos + 1
		default:
			break L
		}
	}
	if c != eof {
		x.peek = c
	}
	yylval.num = 0 // &big.Rat{}
    t_val, err := strconv.ParseFloat(b.String(), 32)
    if err != nil {
        yylval.num = 0.0
    } else {
        yylval.num = t_val
    }
	/*_, ok := yylval.num.SetString(b.String())
	if !ok {
		log.Printf("bad number %q", b.String())
		return eof
	}*/
	return NUM
}

// Return the next rune for the lexer.
func (x *exprLex) next() rune {
    if program == "" {
        program = string(x.line[0:])
        //fmt.Println("SETTING OF program to ", program)
    }
	if x.peek != eof {
		r := x.peek
		x.peek = eof
		return r
	}
	if len(x.line) == 0 {
		return eof
	}
	c, size := utf8.DecodeRune(x.line)
	x.line = x.line[size:]
	if c == utf8.RuneError && size == 1 {
		log.Print("invalid utf8")
        fmt.Printf("invalid utf8 found %c", utf8.RuneError)
		return x.next()
	}
	return c
}

// The parser calls this method on a parse error.
func (x *exprLex) Error(s string) {
    errorOnParse = true
    if charpos < len(program) {
    	fmt.Printf("parse error (before pos %d): \"%s\" program: %s\n", charpos, s, program)
    } else {
    	fmt.Printf("parse error (before pos %d): \"%s\" program: %s\n", charpos, s, program)
    }
}
