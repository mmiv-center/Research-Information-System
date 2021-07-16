package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

//go:embed templates/classifyRules.json
var classifyRules string

type Classes []Class

type Class struct {
	Type        string `json:"type"`
	Id          string `json:"id"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules"`
}

var jsonObj interface{}

type Rule struct {
	Tag      []string    `json:"tag"`
	Value    interface{} `json:"value"` // value can be a string or an array, we have to find out which is which first
	Operator string      `json:"operator"`
	Negate   string      `json:"negate"`
	Rule     string      `json:"rule"`
}

func evalRules(dataset dicom.Dataset, ruleList []Rule, classifications Classes, typesList []string) bool {
	// foreach of the rules we need a truth value for the dataset
	var theseRulesWork bool = true // even if one if false we cancel here with false
	for _, r := range ruleList {
		//fmt.Println("rule is now:", r)

		var t tag.Tag
		var foundTag bool = false
		var tagValue string
		// if there is a tag get its value
		if len(r.Tag) == 1 { // its a name
			//fmt.Println("We have a single tag value in this rule")
			// we need to find out what tag this string is from tagDict
			Info, err := tag.FindByName(r.Tag[0])
			if err == nil {
				// is the name the right one? We found the tag
				t = Info.Tag
				foundTag = true
			}
		} else if len(r.Tag) == 2 { // its a tag pair
			//fmt.Println("We have two tag values in this rule")
			var t1, t2 uint16
			var ok int = 0
			t1_val, err := strconv.ParseInt(r.Tag[0], 0, 16)
			if err == nil {
				t1 = uint16(t1_val)
				ok++
			}
			t2_val, err := strconv.ParseInt(r.Tag[1], 0, 16)
			if err == nil {
				t2 = uint16(t2_val)
				ok++
			}
			if ok == 2 {
				t = tag.Tag{t1, t2}
				foundTag = true
			}
		} else {
			// no tag value specified, could be another rule here that is referenced
			//fmt.Println("No tag, we reference another rule here?", r.Rule)
			// recurse with that rule only to get a single truth value for it
			foundRule := false
			var rs []Rule
			for _, v := range classifications {
				//fmt.Println("Check if this is the rule", v.Id, "for", r.Rule)
				if v.Id == r.Rule {
					rs = v.Rules
					foundRule = true
					break
				}
			}
			if foundRule {
				//fmt.Println("We need to check rule ", r.Rule)
				o := evalRules(dataset, rs, classifications, typesList)
				// if all of the outs are true we are true for this rule
				if !o {
					// at least one of the rules did not work return false for all
					//fmt.Println(" RULE fails")
					theseRulesWork = false
					return theseRulesWork
				}
			} else {
				fmt.Println("We did not find the referenced rule for", r.Rule)
			}
		}
		if foundTag {
			dataElement, err := dataset.FindElementByTag(t)
			if err == nil {
				if dataElement.Value.ValueType() == dicom.Strings {
					tagValue = strings.Join(dataElement.Value.GetValue().([]string), ", ")
				} else if dataElement.Value.ValueType() == dicom.Ints {
					tagValue = strings.Trim(strings.Join(strings.Split(fmt.Sprint(dataElement.Value.GetValue().([]int)), " "), ", "), "[]")
				} else {
					tagValue = fmt.Sprintf("tag value is not string but: %d", dataElement.Value.ValueType())
				}
				//tagValue = dataElement.Value
				//fmt.Println("tag value is:", tagValue, "does it match with", r.Value, "?")

				// call applyOperator
				var thisCheck bool = applyOperator(r, tagValue)
				if !thisCheck {
					theseRulesWork = false
					return theseRulesWork
				}
			} //else {
			//	fmt.Println("Tag", t, "does not exist in dataset ", t)
			// we assume everything is still fine
			//}
		} else {
			// even if there is no tag we might still have something like "ClassifyType" in the first argument
			if len(r.Tag) == 1 && r.Tag[0] == "ClassifyType" {
				// check the operator with that typesList
				var tagValue = strings.Join(typesList, ", ")
				//fmt.Println(" CLASSIFYTYPE OPERATOR: ", tagValue)
				var thisCheck bool = applyOperator(r, tagValue)
				if !thisCheck {
					theseRulesWork = false
					return theseRulesWork
				}
				//fmt.Println("WORKS")
			}
		}

	}
	return theseRulesWork
}

func applyOperator(r Rule, tagValue string) bool {
	// what is r.Value?
	var value_string string
	var value_array []float32
	//fmt.Printf("TYPE: %T %s\n", r.Value, reflect.TypeOf(r.Value))
	switch obj := r.Value.(type) {
	case string:
		tmp := []string([]string{obj})
		value_string = strings.Join(tmp, "")
		//fmt.Println("Found a string", value_string)
	case float32:
		//fmt.Println("Found an array of float32")
		tmp := []float32([]float32{obj})
		value_array = tmp
		//fmt.Println("Found an array of float32", value_array)
	case float64:
		//fmt.Println("Found an array of float32")
		tmp := []float64([]float64{obj})
		for _, v := range tmp {
			value_array = append(value_array, float32(v))
			//fmt.Println("FOund an INT: ", v)
		}
		//fmt.Println("Found an array of float32", value_array)
	case int32:
		tmp := []int32([]int32{obj})
		for _, v := range tmp {
			value_array = append(value_array, float32(v))
			//fmt.Println("FOund an INT: ", v)
		}
		//tmp := []float32([]float64{obj})
		//value_array = tmp
		//fmt.Println("Found an array of float32", value_array)
	case []interface{}:
		//fmt.Println("Found an array of interfaces of type:", reflect.TypeOf(r.Value).Elem())
		tmp2 := []interface{}([]interface{}{obj})
		s := fmt.Sprintf("%f", tmp2)
		s = strings.Replace(s, "[[", "", -1)
		s = strings.Replace(s, "]]", "", -1)
		s_array := strings.Split(s, " ")
		for _, v := range s_array {
			vv_val, err := strconv.ParseFloat(v, 32)
			if err == nil {
				value_array = append(value_array, float32(vv_val))
			}
		}
		//fmt.Println("s is now: ", value_array)
		//fmt.Println("Found array as :", s, " with parsed values: ", value_array)
	default:
		fmt.Println("Error, unknown value type for ", obj)
		fmt.Printf("type: %T\n", r.Value)
	}

	operator := r.Operator
	var thisCheck bool = true // the rule applies (we will find all the ways the rule does not apply)
	if operator == "contains" {
		// create a regexp
		if !strings.Contains(tagValue, value_string) {
			thisCheck = false
		}
	} else if operator == "==" {
		if tagValue != r.Value {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == "" {
		// if operator is empty string we assume regexp?
		//fmt.Println("operator is empty, assume we have a regular expression")
		var rRegex = regexp.MustCompile(value_string)
		if !rRegex.MatchString(tagValue) {
			thisCheck = false
		} //else {
		//	fmt.Println("YES MATCHES, test next")
		//}
	} else if operator == "<" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		if err1 != nil && err2 != nil && var1 >= var2 {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == ">" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		if err1 != nil && err2 != nil && var1 <= var2 {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == "approx" {
		// tagValue, r.Value
		// split tagValue into array of floats
		tmp := strings.Split(tagValue, ", ")
		var tag_array []float32
		for _, v := range tmp {
			v, err := strconv.ParseFloat(v, 32)
			if err == nil {
				tag_array = append(tag_array, float32(v))
			} else {
				fmt.Println("Could not read as float32!")
			}
		}
		// now check each pair, if one pair has a larger value count the whole list as false
		var e float32 = 1e-3
		//var ok = true
		//fmt.Println(len(tag_array), " and ", len(value_array))
		for i, j := 0, 0; i < len(tag_array) && j < len(value_array); i, j = i+1, j+1 {
			//fmt.Println("CHECK ", tag_array[i], value_array[j], "value is:", math.Abs(float64(tag_array[i]-value_array[j])), ">", e)
			if math.Abs(float64(tag_array[i]-value_array[j])) > float64(e) {
				thisCheck = false
				break
			}
		}
		//fmt.Println("APPROX for: ", tag_array, "and", value_array, "rule:", r, "check is: ", thisCheck)
	} else {
		fmt.Println("ERROR UNKNOWN OPERATOR: ", r.Operator)
	}

	// now we check if we match with value
	if r.Negate == "yes" {
		thisCheck = !thisCheck
	}
	return thisCheck
}

func ClassifyDICOM(dataset dicom.Dataset) []string {

	// parse the classifyRules using its structure
	var classifications Classes

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal([]byte(classifyRules), &classifications)

	var classes []string

	// we need to match all classes one after another to the data, whenever one fits we
	// can add it to the output array
	//dcmMeta, err := json.Marshal(dataset)
	//if err != nil {
	//	return classes
	//}
	for _, v := range classifications {
		//fmt.Println("check for type:", v.Type)
		// walk through all rules, if one fails cancel
		// rules can reference other rules, in that case we need to recurse here
		//fmt.Println("rule is now:", v.Rules)
		if evalRules(dataset, v.Rules, classifications, classes) {
			classes = append(classes, v.Type)
		}
	}

	return classes
}
