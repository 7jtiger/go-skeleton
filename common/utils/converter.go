package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type StTest struct{}

type StConf struct {
	Home string
	Port int
}

type Emt struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

func (st *StTest) Expose() {
	fmt.Println("inner expose")
}

func (el Emt) GetIfValue(ifs interface{}, filter string) string {
	var sRes string = ""
	return sRes
}

/*
//slice copy - with delete memory
func CopyDigits(filename string) []byte {
	b, _ := os.ReadFile(filename)
	b = digitRegexp.Find(b)
	c := make([]byte, len(b))
	copy(c,b)
	return c
}
*/

func GetValue(mItem map[string]interface{}, strFiled string) string {
	bys, err := json.Marshal(mItem)
	if err != nil {
		log.Fatal(err)
	}

	var dat map[string]interface{}
	json.Unmarshal(bys, &dat)

	srt := fmt.Sprintf("%v", dat[strFiled])
	return srt
}

func GetJsonStr2Value(str, filed string) string {
	var dat map[string]interface{}
	json.Unmarshal([]byte(str), &dat)

	srt := fmt.Sprintf("%v", dat[filed])
	return srt
}

func GetAto64(sTarget string) int64 {
	nConverted, err := strconv.ParseInt(sTarget, 10, 64)
	if err != nil {
		return 0
	}
	return nConverted
}

func GetFieldValue(row, filter string) string {
	//nIN := strings.Index(row, "name")
	nIN := strings.Index(row, filter)
	if nIN < 0 {
		return ""
	}
	strLsNm := row[nIN+len(filter)+1:]

	nIC := strings.Index(strLsNm, ",")
	if nIC < 0 {
		return ""
	}

	return strLsNm[:nIC]
}

func GetFieldValueCount(row, filter string, count int) string {
	//nIN := strings.Index(row, "name")
	temp := strings.Split(row, filter)
	var i int = 0
	var el string = ""
	for i, el = range temp {
		if i == count {
			return el
		}
	}

	return ""
}

func GetFilterCount(row, filter string) int {
	temp := strings.Split(row, filter)
	return len(temp)
}

func GetJsonValue(row string, filter string) string {
	nIN := strings.Index(row, filter)
	if nIN < 0 {
		return ""
	}
	strLsNm := row[nIN+len(filter)+3:]

	nIC := strings.Index(strLsNm, "\"")
	if nIC < 0 {
		return ""
	}

	return strLsNm[:nIC]
}
