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

type Rule struct {
	Tag      []string    `json:"tag"`
	Tag2     []string    `json:"tag2"`  // only used for checkrules, contains the name of the series and a variable name
	Value    interface{} `json:"value"` // value can be a string or an array, we have to find out which is which first
	Operator string      `json:"operator"`
	Negate   string      `json:"negate"`
	Rule     string      `json:"rule"`
}

type RuleSet struct {
	Name string
	Rs   []Rule
}

// try to define a logical structure for the rules, we need to express AND and OR and NOT
type RuleSetL struct {
	Operator string
	Rs1      *RuleSetL // could be nil in case we are at a Leaf1
	Rs2      *RuleSetL // could be nil in case we have a Leaf2
	Leaf1    Rule
	Leaf2    Rule
}

type RuleTreeSet struct {
	Name string
	Rs   RuleSetL
}

// dataInfo.checkRules(rule, SeriesInstanceUID1, SeriesInstanceUID2)
func evalCheckRule(rule Rule, SeriesInstanceUID1 string, SeriesInstanceUID2 string, dataInfo map[string]map[string]SeriesInfo) bool {
	// we already have the series instance uids for both series in the rule
	var si1 SeriesInfo
	var si2 SeriesInfo
	for _, v := range dataInfo {
		for SeriesInstanceUID, vv := range v {
			if SeriesInstanceUID == SeriesInstanceUID1 {
				si1 = vv
			}
			if SeriesInstanceUID == SeriesInstanceUID2 {
				si2 = vv
			}
		}
	}
	if si1.All != nil && si2.All != nil {
		//fmt.Println("OK we have found the two SeriesInfos")
		ok, data1 := si1.getData(rule.Tag[1], rule.Tag[2])
		if !ok {
			return false
		}
		ok, data2 := si2.getData(rule.Tag2[1], rule.Tag2[2])
		if !ok {
			return false
		}
		// operator test, lets assume we have "=="
		if rule.Operator == "==" {
			for i, j := 0, 0; i < len(data1) && j < len(data2); i, j = i+1, j+1 {
				if data1[i] != data2[j] {
					return false
				}
			}
		} else {
			fmt.Println("WARNING: Operator not supported")
			return false
		}
		return true
	}

	return false
}

// getData returns the data from the data SeriesInfo for the DICOM group_str, tag_str
func (data SeriesInfo) getData(group_str string, tag_str string) (bool, []string) {
	dataData := []string{""}

	group_str = strings.Replace(group_str, "0x", "", -1)
	group_str = strings.Replace(group_str, "0X", "", -1)
	group, err := strconv.ParseInt(group_str, 16, 64)
	if err != nil {
		exitGracefully(err)
	}
	tag_str = strings.Replace(tag_str, "0x", "", -1)
	tag_str = strings.Replace(tag_str, "0X", "", -1)
	tag, err := strconv.ParseInt(tag_str, 16, 64)
	if err != nil {
		exitGracefully(err)
	}
	found := false
	for _, v := range data.All {
		if v.Tag.Group == uint16(group) && v.Tag.Element == uint16(tag) && v.Value != nil {
			found = true
			dataData = v.Value
			break
		}
	}
	return found, dataData
}

// return the index of the rule that matched together with the error
func (data SeriesInfo) evalRules(ruleList []Rule) bool {
	// we assume that in the rulelist we reference only values from the data as type SeriesInfo
	var matches bool = true
	// lets convert the struct to a map
	// all rules have to match!
	for _, val := range ruleList {
		// in a rule list all rules have to fit
		foundValue := false
		t := val.Tag
		o := val.Operator
		v := val.Value
		dataData := []string{""}
		// if we have two fields, one for group, one for tag we need to look into the all fields to find it
		if len(t) == 2 {
			// lookup the value by group and tag
			// values should be read by hexadecimal number
			var group_str = t[0]
			var tag_str = t[1]
			// it is not sufficient to find the entry and set foundValue to true.
			// if we cannot find the entry we shhould use an empty string but that might
			// match later, so best if we can cancel here - or we explicitly have to set the
			// following test false
			foundValue, dataData = data.getData(group_str, tag_str)
			if !foundValue { // nothing can make this correct again
				matches = false
			}
			foundValue = false // set this to false again so we can do a test of the value as well
		} else { // we have a single entry (really?) and treat it as the name of a variable
			if t[0] == "ClassifyType" {
				dataData = data.ClassifyTypes
			} else if t[0] == "ClassifyTypes" {
				dataData = data.ClassifyTypes
			} else if t[0] == "SeriesDescription" {
				dataData = []string{data.SeriesDescription}
			} else if t[0] == "NumImages" {
				dataData = []string{fmt.Sprintf("%d", data.NumImages)}
			} else if t[0] == "NumSlices" {
				dataData = []string{fmt.Sprintf("%d", data.NumImages)}
			} else if t[0] == "SeriesNumber" {
				dataData = []string{fmt.Sprintf("%d", data.SeriesNumber)}
			} else if t[0] == "SequenceName" {
				dataData = []string{data.SequenceName}
			} else if t[0] == "Modality" {
				dataData = []string{data.Modality}
			} else if t[0] == "StudyDescription" {
				dataData = []string{data.StudyDescription}
			} else if t[0] == "Manufacturer" {
				dataData = []string{data.Manufacturer}
			} else if t[0] == "ManufacturerModelName" {
				dataData = []string{data.ManufacturerModelName}
			} else if t[0] == "Path" {
				dataData = []string{data.Path}
			} else if t[0] == "PatientID" {
				dataData = []string{data.PatientID}
			} else if t[0] == "PatientName" {
				dataData = []string{data.PatientName}
			} else {
				// We need to look for named entities here as well. So names that appear in all as groups.

				fmt.Println("Warning: unknown value selected")
			}
		}
		if o == "contains" {
			for _, vv := range dataData {
				if vv == v {
					foundValue = true
				}
			}
		} else if o == "<" {
			for _, vv := range dataData {
				if vv == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue, err := strconv.ParseFloat(vv, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\"\n rule: %v", vv, val)
					exitGracefully(err)
				}
				tmp_str := fmt.Sprintf("%v", v)
				if tmp_str == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue2, err := strconv.ParseFloat(tmp_str, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\" rule: %v\n", v, val)
					exitGracefully(err)
				}
				if numValue < numValue2 {
					foundValue = true
				}
			}
		} else if o == ">=" {
			for _, vv := range dataData {
				if vv == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue, err := strconv.ParseFloat(vv, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\"\n rule: %v", vv, val)
					exitGracefully(err)
				}
				tmp_str := fmt.Sprintf("%v", v)
				if tmp_str == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue2, err := strconv.ParseFloat(tmp_str, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\" rule: %v\n", v, val)
					exitGracefully(err)
				}
				if numValue >= numValue2 {
					foundValue = true
				}
			}
		} else if o == "<=" {
			for _, vv := range dataData {
				if vv == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue, err := strconv.ParseFloat(vv, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\"\n rule: %v", vv, val)
					exitGracefully(err)
				}
				tmp_str := fmt.Sprintf("%v", v)
				if tmp_str == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue2, err := strconv.ParseFloat(tmp_str, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\" rule: %v\n", v, val)
					exitGracefully(err)
				}
				if numValue <= numValue2 {
					foundValue = true
				}
			}
		} else if o == ">" {
			for _, vv := range dataData {
				if vv == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue, err := strconv.ParseFloat(vv, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\"\n rule: %v", vv, val)
					exitGracefully(err)
				}
				tmp_str := fmt.Sprintf("%v", v)
				if tmp_str == "" { // no value matches nothing
					foundValue = false
					continue
				}
				numValue2, err := strconv.ParseFloat(tmp_str, 32)
				if err != nil {
					fmt.Printf("Error: could not convert value to numeric: \"%s\"\n rule: %v", v, val)
					exitGracefully(err)
				}
				if numValue > numValue2 {
					foundValue = true
				}
			}
		} else if o == "regexp" { // on every single item
			for _, vv := range dataData {
				var rRegex = regexp.MustCompile(fmt.Sprintf("%v", v))
				if rRegex.MatchString(vv) {
					foundValue = true
				}
			}
		} else if o == "==" { // this is a non-numeric operator for us we need to be able to work with strings
			allTrue := true
			//fmt.Println("We are testing now ==")
			for _, vv := range dataData {
				if vv == "" { // no value matches nothing
					allTrue = false
					continue
				}
				// if we are non-numeric we should just compare the strings
				if vv != fmt.Sprintf("%v", v) {
					allTrue = false // no further tests are needed
				}
			}
			foundValue = allTrue
		} else {
			fmt.Printf("Error: unknown operator: %s\n", o)
		}
		if !foundValue { // any one rule that does not match will result in false
			matches = false
		}
	}

	return matches
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
				t = tag.Tag{Group: t1, Element: t2}
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
			} else {
				//fmt.Println("Tag", t, "does not exist in dataset ", t, "SHOULD WE PANIC HERE?")
				return false
				// we assume everything is still fine
			}
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
		if tagValue != value_string {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == "" {
		// if operator is empty string we assume regexp?
		//fmt.Printf("regular expression \"%s\" compare with \"%s\"\n", value_string, tagValue)
		var rRegex = regexp.MustCompile(value_string)
		if !rRegex.MatchString(tagValue) {
			thisCheck = false
		} //else {
		//	fmt.Println("YES MATCHES, test next")
		//}
	} else if operator == "<" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		//fmt.Println("WE are checking < for", var1, var2)
		if err1 != nil && err2 != nil && var1 < var2 {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == "<=" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		if err1 != nil && err2 != nil && var1 <= var2 {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == ">" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		if err1 != nil && err2 != nil && var1 > var2 {
			//fmt.Println("== sign operator false for", tagValue, r.Value)
			thisCheck = false
		}
	} else if operator == ">=" {
		var1, err1 := strconv.ParseFloat(tagValue, 32)
		var2, err2 := strconv.ParseFloat(value_string, 32)
		if err1 != nil && err2 != nil && var1 >= var2 {
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

	var classes []string = make([]string, 0)

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
