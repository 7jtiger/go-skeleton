package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
)

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

func StrToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func GetEndTime(day string) (*big.Int, error) {
	year, yearErr := StrToInt(GetFieldValueCount(day, "-", 0))
	mon, monErr := StrToInt(GetFieldValueCount(day, "-", 1))
	d, dErr := StrToInt(GetFieldValueCount(day, "-", 2))

	if yearErr != nil || monErr != nil || dErr != nil {
		return nil, fmt.Errorf("error parsing date components")
	}

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

// if day is nextday, return today
func GetDurationTime(day string) (time.Time, time.Time, error) {
	//var day string = "2020-12-08"
	year, yearErr := StrToInt(GetFieldValueCount(day, "-", 0))
	mon, monErr := StrToInt(GetFieldValueCount(day, "-", 1))
	d, dErr := StrToInt(GetFieldValueCount(day, "-", 2))

	if yearErr != nil || monErr != nil || dErr != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error parsing date components")
	}

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
	year, yearErr := StrToInt(GetFieldValueCount(day, "-", 0))
	mon, monErr := StrToInt(GetFieldValueCount(day, "-", 1))
	d, dErr := StrToInt(GetFieldValueCount(day, "-", 2))

	if yearErr != nil || monErr != nil || dErr != nil {
		return time.Time{}, fmt.Errorf("error parsing date components")
	}

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
	year, yearErr := StrToInt(GetFieldValueCount(day, "-", 0))
	mon, monErr := StrToInt(GetFieldValueCount(day, "-", 1))
	d, dErr := StrToInt(GetFieldValueCount(day, "-", 2))

	if yearErr != nil || monErr != nil || dErr != nil {
		return time.Now().Local()
	}

	t := time.Now().Local()
	if year == 0 || mon == 0 || d == 0 {
		return t
	}

	tday := time.Date(year, time.Month(mon), d, 0, 0, 0, 0, time.Local)

	return tday
}

func StrMonthToTime(month string) time.Time {
	//var day string = "2020-12-08"
	year, yearErr := StrToInt(GetFieldValueCount(month, "-", 0))
	mon, monErr := StrToInt(GetFieldValueCount(month, "-", 1))

	t := time.Now().Local()
	if yearErr != nil || monErr != nil || year == 0 || mon == 0 {
		return t
	}

	tday := time.Date(year, time.Month(mon), 0, 0, 0, 0, 0, time.Local)

	return tday
}
