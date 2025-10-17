package common

import (
	"fmt"

	// "io/ioutil"

	"os"
	"strconv"
	"strings"
)

type StTest struct{}

type StConf struct {
	Home string
	Port int
}

func Hex2Int64(hexStr string) uint64 {
	cleaned := strings.Replace(hexStr, "0x", "", -1)
	result, _ := strconv.ParseUint(cleaned, 16, 64)
	return uint64(result)
}

func (st *StTest) Expose() {
	fmt.Println("inner expose")
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

func GetFilterCount(row, filter string) int {
	temp := strings.Split(row, filter)
	return len(temp)
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
