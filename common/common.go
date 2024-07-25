package common

import (
	"encoding/json"
	"fmt"

	// "io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
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

func ByteToStr(c []byte) string {
	n := -1
	for i, b := range c {
		if b == 0 {
			break
		}
		n = i
	}
	return string(c[:n+1])
}

func StrToUint(str string) uint {
	u64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		fmt.Println("common", "StrToUint", err.Error())
	}
	return uint(u64)
}

func StrToUint64(str string) uint64 {
	u64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		fmt.Println("common", "StrToUint", err.Error())
	}
	return uint64(u64)
}

func IntToInt64(n int) int64 {
	return int64(n)
}

func StrToInt(str string) int {
	i32, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("common", "StrToInt", err.Error())
		return -1
	}
	return int(i32)
}

func StrToHex(str string) {
	//res := hex.EncodeToString([]byte(str))
	//return res
}

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

func GetAto64(sTarget string) int64 {
	nConverted, err := strconv.ParseInt(sTarget, 10, 64)
	if err != nil {
		return 0
	}
	return nConverted
}

func ReadFileLastNum(path string) int64 {
	_, err := os.ReadFile(path)
	if err != nil {
		return 0
	} else {
		return 1
	}
}

func WriteFileLastNum(path string) bool {
	_, err := os.ReadFile(path)
	if err != nil {
		return true
	}

	return true
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

/*
//slice copy - with delete memory
func CopyDigits(filename string) []byte {
	b, _ := ioutil.ReadFile(filename)
	b = digitRegexp.Find(b)
	c := make([]byte, len(b))
	copy(c,b)
	return c
}
*/

func UnixToTime(i64 int64) time.Time {
	//r := "1572428388"
	//q, err := strconv.ParseInt(r, 10, 64)
	t := time.Unix(i64, 0)
	return t
}

func UnixToTimeStamp(i64 uint64) string {
	t := time.Unix(int64(i64), 0)
	res := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	return res
}

func StrUnixToTime(sTime string) time.Time {
	var t time.Time
	q, err := strconv.ParseInt(sTime, 10, 64)
	if err != nil {
		return t
	}
	t = time.Unix(q, 0)
	return t
}

func GetEndTime(day string) (*big.Int, error) {
	year := StrToInt(GetFieldValueCount(day, "-", 0))
	mon := StrToInt(GetFieldValueCount(day, "-", 1))
	d := StrToInt(GetFieldValueCount(day, "-", 2))

	t := time.Now()
	toDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix()
	if year == 0 || mon == 0 || d == 0 {
		return big.NewInt(toDay), fmt.Errorf("error, empty parameter")
	}

	tagetDay := time.Date(year, time.Month(mon), d, 0, 0, 0, 0, time.Local).Unix()

	if toDay <= tagetDay {
		return big.NewInt(toDay), nil
	} else {
		return big.NewInt(tagetDay), nil
	}
}

func GetMiddlePath(path string) string {
	//strG := strings.Index(path, "/")
	if strings.Contains(path, "admnoti") {
		return "noti/"
	} else if strings.Contains(path, "admfaq") {
		return "faq/"
	} else {
		return ""
	}
}

// if day is nextday, return today
func GetDurationTime(day string) (time.Time, time.Time, error) {
	//var day string = "2020-12-08"
	year := StrToInt(GetFieldValueCount(day, "-", 0))
	mon := StrToInt(GetFieldValueCount(day, "-", 1))
	d := StrToInt(GetFieldValueCount(day, "-", 2))

	t := time.Now()
	if year == 0 || mon == 0 || d == 0 {
		return t, t, fmt.Errorf("error, empty parameter")
	}

	tday := time.Date(year, time.Month(mon), d, 0, 0, 0, 0, time.Local)
	ts := t.Sub(tday)

	oneDay := 24 * time.Hour
	if ts < 0 { //future next time, return today
		return tday, t, nil
	} else if ts < oneDay { //today
		return tday, t, nil
	} else {
		yday := tday.AddDate(0, 0, 1)
		return tday, yday, nil
	}
}

func ConvertStrToTime(day string) (time.Time, error) {
	//var day string = "2020-12-08"
	year := StrToInt(GetFieldValueCount(day, "-", 0))
	mon := StrToInt(GetFieldValueCount(day, "-", 1))
	d := StrToInt(GetFieldValueCount(day, "-", 2))

	t := time.Now().Local()
	if year == 0 || mon == 0 || d == 0 {
		return t, fmt.Errorf("error, empty parameter")
	}

	tday := time.Date(year, time.Month(mon), d, 0, 0, 0, 0, time.Local) // this day+1 00:00:00
	// check future date
	toDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)

	if toDay.Unix() <= tday.Unix() {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), fmt.Errorf("convert time to today")
	}

	return tday, nil
}

func StrDayToTime(day string) time.Time {
	//var day string = "2020-12-08"
	year := StrToInt(GetFieldValueCount(day, "-", 0))
	mon := StrToInt(GetFieldValueCount(day, "-", 1))
	d := StrToInt(GetFieldValueCount(day, "-", 2))

	t := time.Now().Local()
	if year == 0 || mon == 0 || d == 0 {
		return t
	}

	tday := time.Date(year, time.Month(mon), d, 0, 0, 0, 0, time.Local)

	return tday
}

func StrMonthToTime(month string) time.Time {
	//var day string = "2020-12-08"
	year := StrToInt(GetFieldValueCount(month, "-", 0))
	mon := StrToInt(GetFieldValueCount(month, "-", 1))

	t := time.Now().Local()
	if year == 0 || mon == 0 {
		return t
	}

	tday := time.Date(year, time.Month(mon), 0, 0, 0, 0, 0, time.Local)

	return tday
}

/*
func Unmarshal(b []byte) error {
	var stuff map[string]string
	err := json.Unmarshal(b, &stuff)
	if err != nil {
		return err
	}
	for key, value := range stuff {
		numericKey, err := strconv.Atoi(key)
		if err != nil {
			return err
		}

		Stuff[key] = value
	}
	return nil
}
*/
