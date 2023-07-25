
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
    "errors"
)

// represents a select statement
type AST struct {
    Output_level string
    Select_level string
    Select_level_by_rule []string
    Rule_list_names []string  //  should be deprecated
    Rules []RuleSet // we need sets of rules for each series we describe
    CheckRules []RuleSet // we capture the special check rules here
    RulesTree []RuleTreeSet
}

var ast AST                 // our abstract syntax tree
var charpos int             // the char position in the string
var program string          // just a copy of the string to parse

var Rules2 RuleSetL
var currentRulesL []RuleSetL = nil  // we store the ruleSet information in this array so we can use an index
var currentRules []Rule = nil       // we store one rules information here
var currentCheckRules []Rule = nil  // for checks there is a separate list
var errorOnParse = false
var lastGroupTag []string           // a pair of group, tag in decimal format
var currentCheckTag1 []string       // a pair of named series '.' DICOM name
var currentCheckTag2 []string       // a pair of named series '.' DICOM name

// get index from $$
func getCurrentRuleIdx(entry string) (int64, error) {
    parts := strings.Split(entry, "currentRules:")
    if parts[0] == "" && len(parts) == 2 {
        b, err := strconv.ParseInt(strings.Trim(parts[1], " "), 10, 32)
        if err == nil {
            //fmt.Println("      getCurrentRuleIdx found:", b)
            return b, nil
        }
    }
    //fmt.Println(parts, " ", len(parts))
    return -1, errors.New("not a currentRules: part found");
}

// get index from $$
func getCurrentRulesLIdx(entry string) (int64, error) {
    parts := strings.Split(entry, "currentRulesL:")
    if parts[0] == "" && len(parts) == 2 {
        b, err := strconv.ParseInt(strings.Trim(parts[1], " "), 10, 32)
        if err == nil {
            //fmt.Println("      getCurrentRulesLIdx found:", b)
            return b, nil
        }
    }
    return -1, errors.New("not found currentRulesL");
}


%}

%union {
	num  float64
    word string
}

%type <word> top, command, select_stmt, base_select, level_types, rule_list, rule
%type <word> where_clause, where_clauses, level_types_with_name
%type <word> check_stmt base_check check_rule_list check_rule tag_string 
%type <word> group_tag_pair check_tag1 check_tag2 command_list

%token '+' '-' '*' '/' '"' '\''
%token SELECT FROM PATIENT STUDY SERIES IMAGE WHERE EQUALS HAS AND OR ALSO LBRACKET RBRACKET COMMA
%token CONTAINING SMALLER LARGER REGEXP NOT NAMED PROJECT CHECK AT SMALLEREQUAL LARGEREQUAL

%token	<num>	NUM
%token  <word>  STRING NOT

%start top

%%

top:
    command_list
    {
        //fmt.Printf("command \"%w\"\n", $1)
        //s, _ := json.MarshalIndent(ast, "", "  ")
        //fmt.Printf("internal ast is: \"%s\"\n",string(s))
        $$ = $1
    };

command_list:
    command
    {
        //fmt.Printf("IN COMMAND \"%s\"\n",$$)
        $$ = $1
    }
|   command command_list
    {
        //fmt.Println("IN COMMAND SEMICOLON")
        $$ = fmt.Sprintf("%s ; %s", $1, $2)
    }

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
        currentRulesL = nil
        // get space in Rules now for rules
        if ast.Rules == nil {
            ast.Rules = make([]RuleSet, 0)
        }
        if ast.RulesTree == nil {
            ast.RulesTree = make([]RuleTreeSet,0)
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
        currentRulesL = nil
        if ast.Rules == nil {
            ast.Rules = make([]RuleSet, 0)
        }
        if ast.RulesTree == nil {
            ast.RulesTree = make([]RuleTreeSet,0)
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
        name_for_ruleset := ""
        if len(ast.Rule_list_names) > 0 {
            name_for_ruleset = ast.Rule_list_names[len(ast.Rule_list_names)-1]
        }

        // could be a simple rule or a ruleSetL in $4
        retRule := RuleSetL{
            Operator: "FIRST",
        }

        var cr1 Rule
        entry, err1 := getCurrentRuleIdx($4)
        if (err1 == nil) {
            cr1 = currentRules[entry]
            retRule.Operator = "FIRST"
            retRule.Rs1 = nil
            retRule.Rs2 = nil
            retRule.Leaf1 = &cr1
            retRule.Leaf2 = nil
        } else {
            // could be a more complex rule as well
            entry, err2 := getCurrentRulesLIdx($4)
            if err2 == nil {
                //fmt.Println("SHOULD BE HERE, so Rs1 ", $4, "should not be empty but is ", currentRulesL[entry], "with entry", entry)
                retRule.Operator = "FIRST"
                retRule.Rs1 = &currentRulesL[entry]
                retRule.Rs2 = nil
                retRule.Leaf1 = nil
                retRule.Leaf2 = nil
            } else {
                fmt.Println("SHOULD NEVER HAPPEN")
            }
        }

        // add the currentRules if they are not already in the list
        var rs RuleSet = RuleSet{
            Name: name_for_ruleset,
            Rs: currentRules,
        }
        ast.Rules = append(ast.Rules, rs)
        ast.Select_level_by_rule = append(ast.Select_level_by_rule, $2)
        var rts RuleTreeSet = RuleTreeSet {
            Name: name_for_ruleset,
            Rs: retRule,
        }
        ast.RulesTree = append(ast.RulesTree, rts)
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1)
    }
|   WHERE rule_list
    {
        // could be a simple rule or a ruleSetL in $2
        retRule := RuleSetL{
            Operator: "FIRST",
        }

        var cr1 Rule
        entry, err1 := getCurrentRuleIdx($2)
        if (err1 == nil) {
            cr1 = currentRules[entry]
            retRule.Operator = "FIRST"
            retRule.Rs1 = nil
            retRule.Rs2 = nil
            retRule.Leaf1 = &cr1
            retRule.Leaf2 = nil
        } else {
            // could be a more complex rule as well
            entry, err2 := getCurrentRulesLIdx($2)
            if err2 == nil {
                retRule.Operator = "FIRST"
                retRule.Rs1 = &currentRulesL[entry]
                retRule.Rs2 = nil
                retRule.Leaf1 = nil
                retRule.Leaf2 = nil
            } else {
                fmt.Println("Error: should never happen here")
            }
        }

        // add the currentRules if they are not already in the list
        var rs RuleSet = RuleSet{
            Name: "",
            Rs: currentRules,
        }
        ast.Rules = append(ast.Rules, rs)
        ast.Select_level_by_rule = append(ast.Select_level_by_rule, "series")
        var rts RuleTreeSet = RuleTreeSet {
            Name: "",
            Rs: retRule,
        }
        ast.RulesTree = append(ast.RulesTree, rts)
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1)
    }

level_types_with_name:
    level_types
    {
        // need to add the name to the current RuleSet, will use the last added name
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
        retRule := RuleSetL{
            Operator: "FIRST",
        }

        //fmt.Printf(" SINGLE RULE rule got $$ as \"%s\", right as \"%s\"\n", $$, $1)
        var cr1 Rule
        entry, err1 := getCurrentRuleIdx($1)
        if (err1 == nil) {
            cr1 = currentRules[entry]
            retRule.Operator = "FIRST" // only evaluate the first term, NO-OP
            retRule.Rs1 = nil
            retRule.Rs2 = nil
            retRule.Leaf1 = &cr1
            retRule.Leaf2 = nil
            //fmt.Println("IN single rule without anything else, found leaf1", cr1, "entry is", entry)
        } else {
            entry, err2 := getCurrentRulesLIdx($1)
            if err2 == nil {
                retRule.Operator = "FIRST" // only evaluate the first term, NO-OP
                retRule.Rs1 = &currentRulesL[entry]
                retRule.Rs2 = nil
                retRule.Leaf1 = nil
                retRule.Leaf2 = nil
                //fmt.Println("IN rule without anything else, found Rs1", currentRulesL[entry], "entry is", entry)
            } else {
               fmt.Println("Error: nether this or that, a single rule without complex stuff inside, should not happen")
            }
        }
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1)
    }
|   rule_list AND rule
    {
        retRule := RuleSetL{
            Operator: "AND",
        }
        //fmt.Printf(" AND rule got $$ as \"%s\", left as \"%s\", right as \"%s\"\n", $$, $1, $3)

        entry, err1 := getCurrentRuleIdx($1)
        if (err1 == nil) {
            retRule.Rs1 = nil
            retRule.Leaf1 = &currentRules[entry]
        } else {
            entry, err2 := getCurrentRulesLIdx($1)
            if err2 == nil {
                retRule.Leaf1 = nil
                retRule.Rs1 = &currentRulesL[entry]
            } else {
                fmt.Println("ERROR should not happen")
            }
        }
        entry, err2 := getCurrentRuleIdx($3)
        if (err2 == nil) {
            retRule.Rs2 = nil
            retRule.Leaf2 = &currentRules[entry]
        } else {
            entry, err2 := getCurrentRulesLIdx($3)
            if err2 == nil {
                retRule.Leaf2 = nil
                retRule.Rs2 = &currentRulesL[entry]
            }
        }
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1) // return an index to the last rule
    }
|   rule_list OR rule
    {
        retRule := RuleSetL{
            Operator: "OR",
        }

        //fmt.Printf(" OR rule got $$ as \"%s\", left as \"%s\", right as \"%s\"\n", $$, $1, $3)

        var cr1 Rule
        var cr2 Rule
        entry, err2 := getCurrentRuleIdx($3)
        if (err2 == nil) {
            cr2 = currentRules[entry]
        }
        entry, err1 := getCurrentRuleIdx($1)
        if (err1 == nil) {
            cr1 = currentRules[entry]
        }
        retRule.Operator = "OR"
        if err1 == nil {
            retRule.Rs1 = nil
            retRule.Leaf1 = &cr1
            fmt.Println("IN OR, found Leaf1 with", cr1)
        } else {
            entry, err2 := getCurrentRulesLIdx($1)
            if err2 == nil {
                retRule.Leaf1 = nil
                retRule.Rs1 = &currentRulesL[entry]
                //fmt.Println("IN OR, found Rs1 with", currentRulesL[entry], "entry is", entry)
            }
        }
        if err2 == nil {
            retRule.Rs2 = nil
            retRule.Leaf2 = &cr2
            fmt.Println("IN OR, found Leaf2 with", cr2)
        } else {
            entry, err2 := getCurrentRulesLIdx($3)
            if err2 == nil {
                retRule.Leaf2 = nil
                retRule.Rs2 = &currentRulesL[entry]
                //fmt.Println("IN OR, found Rs2 with", currentRulesL[entry], "entry is", entry)
            } else {
                fmt.Println("Error: should never happen because its either Leaf or Rs")
            }
        }
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1) // return an index to the last rule
    }
//|   NOT rule
//    {
//        cr2 := currentRules[len(currentRules)-1]
//        //fmt.Println("found NOT rule value: ", cr2.Value, " negate: ", cr2.Negate, " operator: ", cr2.Operator, " tag: ", cr2.Tag)
//        if (Rules2.Operator == "Initial") || (Rules2.Operator == "") {
//            // overwrite this one
//            Rules2.Operator = "NOT"
//            Rules2.Rs1 = nil
//            Rules2.Rs2 = nil
//            Rules2.Leaf1 = cr2
//            Rules2.Leaf2 = Rule{}
//        } else {
//            // make a new hierarchy and use the current node as Rs1
//            var copyRules RuleSetL = RuleSetL{
//                Operator: Rules2.Operator,
//                Rs1: Rules2.Rs1,
//                Rs2: Rules2.Rs2,
//                Leaf1: Rules2.Leaf1,
//                Leaf2: Rules2.Leaf2,
//            }
//            Rules2.Rs1 = &copyRules
//            Rules2.Operator = "NOT"
//            Rules2.Leaf2 = cr2
//            Rules2.Leaf1 = Rule{}
//            Rules2.Rs2 = nil 
//        }
//    }

rule:
    LBRACKET rule_list RBRACKET
    {
        // where to put Rules2?? 
        // Lets ignore Rule2 and just take what comes back as $$ for now
        retRule := RuleSetL{
            Operator: "FIRST",
        }
        var cr2 Rule
        entry, err2 := getCurrentRuleIdx($2)
        if (err2 == nil) {
            //fmt.Println("Found bracket current rule as :", entry)
            cr2 = currentRules[entry]

            retRule.Operator = "FIRST" // only evaluate the first term, NO-OP
            retRule.Rs1 = nil
            retRule.Rs2 = nil
            retRule.Leaf1 = &cr2
            retRule.Leaf2 = nil
        } else {
            // A more complex rule here that cannot be checked from currentRules
            entry, err2 = getCurrentRulesLIdx($2)
            if err2 == nil {
                retRule.Operator = "FIRST"
                retRule.Rs1 = &currentRulesL[entry] // maybe that is empty right now?
                retRule.Leaf2 = nil
                retRule.Leaf1 = nil
                retRule.Rs2 = nil
                //fmt.Println("FOUND bracket rule with complex entry for Rs1", currentRulesL[entry], "entry is", entry)
            } else {
                fmt.Println("ERROR: should never happen, we need at least an index for something here")
            }
        }
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1) // return an index to the last rule
    }
|   NOT rule
    {
        retRule := RuleSetL{
            Operator: "NOT",
        }
        //fmt.Printf(" NOT rule got $$ as \"%s\", right as \"%s\"\n", $$, $2)
        var cr2 Rule
        entry, err2 := getCurrentRuleIdx($2)
        if (err2 == nil) {
            cr2 = currentRules[entry]
            retRule.Operator = "NOT"
            retRule.Rs1 = nil
            retRule.Rs2 = nil
            retRule.Leaf1 = &cr2
            retRule.Leaf2 = nil
        } else {
            entry, err2 = getCurrentRulesLIdx($2)
            if err2 == nil {
                retRule.Operator = "NOT"
                retRule.Rs1 = &currentRulesL[entry] // maybe that is empty right now?
                retRule.Leaf2 = nil
                retRule.Leaf1 = nil
                retRule.Rs2 = nil 
                //fmt.Println("in NOT rule found", currentRulesL[entry])
            } else {
                fmt.Println("Error: Should never happen in NOT rule")
            }
        }
        currentRulesL = append(currentRulesL, retRule)
        $$ = fmt.Sprintf("currentRulesL:%d", len(currentRulesL)-1) // return an index to the last rule
    }
//|   NOT rule
//    {
//        if currentRules[len(currentRules)-1].Negate == "" || currentRules[len(currentRules)-1].Negate == "no" {
//            currentRules[len(currentRules)-1].Negate = "yes"
//        } else {
//            currentRules[len(currentRules)-1].Negate = "no"
//        }
//        $$ = fmt.Sprintf("%s NOT %s", $$, $1)
//    }
|   tag_string EQUALS STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "==",
            Value: $3,
        }
        currentRules = append(currentRules, r)
        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string EQUALS NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "==",
            Value: $3,
        }
        currentRules = append(currentRules, r)
        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string CONTAINING STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "contains",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string SMALLER NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "<",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string LARGER NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: ">",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string SMALLEREQUAL NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "<=",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string LARGEREQUAL NUM
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: ">=",
            Value: $3,
        }
        currentRules = append(currentRules, r)

        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
    }
|   tag_string REGEXP STRING
    {
        r := Rule{
            Tag: lastGroupTag,
            Operator: "regexp",
            Value: $3,
        }
        currentRules = append(currentRules, r)
        //fmt.Printf("store a currentRule as regexp %s \"%s\" (id: %d)\n", lastGroupTag, $3, len(currentRules)-1)
        $$ = fmt.Sprintf("currentRules:%d", len(currentRules)-1) // fmt.Sprintf("Variable %s contains %s", $1, $3)
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
|   LBRACKET group_tag_pair RBRACKET
    {
        //fmt.Println("We are in the group tag pair now")
        $$ = $2
    }

group_tag_pair:
    STRING COMMA STRING
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
        $$ = fmt.Sprintf("(%x,%x)", group, tag)
    }
/*|   NUM COMMA NUM
    {
        // interpret the numbers as strings (todo we want hex immediately)
        g1 := fmt.Sprintf("%d", $1)
        g2 := fmt.Sprintf("%d", $3)
        // get the corresponding group and tag from hexadecimal
        group_str := strings.Replace(g1,"0x","",-1)
        group_str = strings.Replace(group_str, "0X","", -1)
        group, err := strconv.ParseInt(group_str, 16, 64)
        if err != nil {
            exitGracefully(err)
        }
        tag_str := strings.Replace(g2,"0x","",-1)
        tag_str = strings.Replace(tag_str, "0X","", -1)
        tag, err := strconv.ParseInt(tag_str, 16, 64)
        if err != nil {
            exitGracefully(err)
        }
        //lastGroupTag = []int{int(group), int(tag)}
        lastGroupTag = []string{g1, g2}
        $$ = fmt.Sprintf("(%d,%d)", group, tag)
    } */

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
        if ast.CheckRules == nil {
           ast.CheckRules = make([]RuleSet,0)
        }
        if len(currentCheckRules)  > 0 {
            var rs RuleSet = RuleSet{
                Name: "",
                Rs: currentCheckRules,
            }
            ast.CheckRules  = append(ast.CheckRules, rs)
        }
        currentCheckRules = nil
        $$ = $2
    }

check_rule_list:
    check_rule 
    {
        //fmt.Printf("found a rule: \"%s\"\n", $1)
        // add the rule to the current list of rules
        if len(currentCheckRules) > 0 {
            var rs RuleSet = RuleSet{
                Name: "",
                Rs: currentCheckRules,
            }
            ast.CheckRules  = append(ast.CheckRules, rs)
            currentCheckRules = nil 
        }
        $$ = $1
    }
|   check_rule_list AND check_rule
    {
        // fmt.Println("found AND rule")
        $$ = $1 + " AND " + $3
        // res := fmt.Sprintf("%s AND %s", $$, $3)
        // $$ = res
    }

check_rule:
    LBRACKET check_rule_list RBRACKET
    {
        $$ = $$ + ", brackets " + $2
        //$$ = fmt.Sprintf("%s, brackets %s", $$, $2) // should be $2
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
|   check_tag1 EQUALS check_tag2
    {
        r := Rule{
            Tag: currentCheckTag1,
            Tag2: currentCheckTag2,
            Operator: "==",
            Value: $1 + " == " + $1, // fmt.Sprintf("%s == %s", $1, $3),
        }
        currentCheckRules = append(currentCheckRules, r)
        $$ = $1 + " == " + $3 // fmt.Sprintf("Variable %s == %s", $1, $3)
    }

check_tag1:
    STRING AT tag_string
    {
        currentCheckTag1 = []string{$1, lastGroupTag[0]}
        if len(lastGroupTag) > 1 {
            currentCheckTag1 =  append(currentCheckTag1, lastGroupTag[1])
        }
        $$ = $1 + " @ " + $3 // fmt.Sprintf("%s @ %s", $1, $3)
    }

check_tag2:
    STRING AT tag_string
    {
        currentCheckTag2 = []string{$1, lastGroupTag[0]}
        if len(lastGroupTag) > 1 {
            currentCheckTag2 =  append(currentCheckTag2, lastGroupTag[1])
        }
        $$ = $1 + " @ " + $3 // fmt.Sprintf("%s @ %s", $1, $3)
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
            // read in hexadecimal as well
            if c == '0' && (x.peek == 'x' || x.peek == 'X') {
                // read as hexadecimal
                return x.hex(c, yylval)
            }
			return x.num(c, yylval)
		case '+', '-', '*', '/':
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
        case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'ø', 'å', 'æ':
            return x.word(c, yylval, rune(0))
        case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'Ø', 'Å', 'Æ':
            return x.word(c, yylval, rune(0))
        case '^', '$', '[', ']', '.', '_', '|':
            return x.word(c, yylval, rune(0))
        case '<':
            // how about <= ?
            peek := x.nextButKeep()
            if peek == '=' {
                _ = x.next()
                charpos = charpos + 2
                return SMALLEREQUAL
            }
            charpos = charpos + 1
            return SMALLER
        case '>':
            // how about >= ?
            peek := x.nextButKeep()
            if peek == '=' {
                _ = x.next()
                charpos = charpos + 2
                return LARGEREQUAL
            }
            charpos = charpos + 1
            return LARGER
        case '=':
            // how about == ?
            peek := x.nextButKeep()
            if peek == '=' {
                _ = x.next()
                charpos = charpos + 2
                return EQUALS
            }
            charpos = charpos + 1
            return EQUALS
        case '(':
            charpos = charpos + 1
            return LBRACKET
        case ')':
            charpos = charpos + 1
            return RBRACKET
        case ',':
            charpos = charpos + 1
            return COMMA
        case '@':
            charpos = charpos + 1
            return AT
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
		case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'ø', 'å', 'æ':
			add(&b, c)
            charpos = charpos + 1
        case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'Ø', 'Å', 'Æ':
			add(&b, c)
            charpos = charpos + 1
        case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '^', '$', '.', '*', '[', ']', '/', '\\', '_', '|', '(', ')':
			add(&b, c)
            charpos = charpos + 1
        case delimiter:
            c = x.next() // make sure we will not look at it if it stays in peek
            charpos = charpos + 1
            break L
		default:
            if delimiter != rune(0) && (c == '-' || c == '_') {
    			add(&b, c)
                charpos = charpos + 1
            } else {
			    break L
            }
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
    } else if b.String() == "participant" {
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
    } else if strings.ToLower(b.String()) == "or" {
        return OR
    } else if strings.ToLower(b.String()) == "not" {
        return NOT
    } else if strings.ToLower(b.String()) == "containing" {
        return CONTAINING
    } else if strings.ToLower(b.String()) == "contains" {
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
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.', 'e', 'E', '+':
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

// Lex a hex-number.
func (x *exprLex) hex(c rune, yylval *yySymType) int {
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
        case 'A':
            add(&b, 'a')
            charpos = charpos + 1
        case 'B':
            add(&b, 'b')
            charpos = charpos + 1
        case 'C':
            add(&b, 'c')
            charpos = charpos + 1
        case 'E':
            add(&b, 'e')
            charpos = charpos + 1
        case 'F':
            add(&b, 'f')
            charpos = charpos + 1
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f':
			add(&b, c)
            charpos = charpos + 1
		default:
			break L
		}
	}
	if c != eof {
		x.peek = c
	}
    yylval.word = b.String()
    return STRING
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

// Return the next rune but don't advance the lexer.
func (x *exprLex) nextButKeep() rune {
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
	//x.line = x.line[size:]
	if c == utf8.RuneError && size == 1 {
		log.Print("invalid utf8")
        fmt.Printf("invalid utf8 found %c", utf8.RuneError)
		return x.nextButKeep()
	}
	return c
}


// The parser calls this method on a parse error.
func (x *exprLex) Error(s string) {
    errorOnParse = true
    if charpos < len(program) {
    	fmt.Printf("parse error (before pos %d): \"%s\" program: %s\n", charpos, s, program)
    } else {
    	fmt.Printf("parse error (before pos %d): \"%s\" program: %s\nline: \"%v\"\n", charpos, s, program, x.line)
    }
}
