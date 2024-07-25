package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/shopspring/decimal"
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

func BalanceToDecimal(balance *big.Int, price float32, quantity int) decimal.Decimal {
	// div = 10 ** quantity
	div := decimal.New(int64(1), int32(quantity))

	priceDecimal := decimal.NewFromFloat32(price)
	balanceDecimal := decimal.NewFromBigInt(balance, 0)
	mulValue := priceDecimal.Mul(balanceDecimal)

	return mulValue.Div(div)
}

func NewSha256(iValue []byte) []byte {
	hash := sha256.Sum256(iValue)
	return hash[:]
}

// 4126621CFCFF097CA991E97CA582D844D39D22101E -> TDUADNjywL9SXSEkQmMbUyWDG3LJsFNmzS
// plainData := (입력받은 데이터 + (sha256(sha256(ivalue))의 앞 4바이트 ))
// plainData 를 base58로 인코드
func Base58Encode(iValue string) (string, error) {
	if plainData, err := hex.DecodeString(iValue); err != nil {
		return "", err
	} else {
		hash0 := NewSha256(plainData)
		hash1 := NewSha256(hash0)
		plainData = append(plainData, hash1[:4]...)

		encodeData := base58.Encode(plainData)
		return encodeData, nil
	}
}

// TDUADNjywL9SXSEkQmMbUyWDG3LJsFNmzS -> 4126621CFCFF097CA991E97CA582D844D39D22101E
// base58로 디코드 후 마지막 4바이트 제거
func Base58Decode(iValue string) string {
	originDecode := base58.Decode(iValue)
	decodeData := originDecode[:len(originDecode)-4]
	return hex.EncodeToString(decodeData)
}
