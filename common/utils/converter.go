package utils

import (
	"strings"
)

func GetFieldValueCount(row, filter string, count int) string {
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
