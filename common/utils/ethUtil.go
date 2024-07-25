package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Hex2Int64(hexStr string) uint64 {
	cleaned := strings.Replace(hexStr, "0x", "", -1)
	result, _ := strconv.ParseUint(cleaned, 16, 64)
	return uint64(result)
}

var ether, _ = new(big.Float).SetString(new(big.Int).SetUint64(1e18).String())

func MustConvertToEther(n *big.Int) float64 {
	r, _ := new(big.Float).Quo(new(big.Float).SetInt(n), ether).Float64()
	return r
}

// IsValidAddress validate hex address
func IsValidAddress(iaddress interface{}) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	switch v := iaddress.(type) {
	case string:
		return re.MatchString(v)
	case common.Address:
		return re.MatchString(v.Hex())
	default:
		return false
	}
}

// IsZeroAddress validate if it's a 0 address
func IsZeroAddress(iaddress interface{}) bool {
	var address common.Address
	switch v := iaddress.(type) {
	case string:
		address = common.HexToAddress(v)
	case common.Address:
		address = v
	default:
		return false
	}

	zeroAddressBytes := common.FromHex("0x0000000000000000000000000000000000000000")
	addressBytes := address.Bytes()
	return reflect.DeepEqual(addressBytes, zeroAddressBytes)
}

// ToDecimal wei to decimals
func ToDecimal(ivalue interface{}, decimals int) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

// ToWei decimals to wei
func ToWei(iamount interface{}) *big.Int {
	amount := decimal.NewFromFloat(0)
	switch v := iamount.(type) {
	case string:
		amount, _ = decimal.NewFromString(v)
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	default:
		return nil
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei
}

// +0 18개 = e18
func StrToWei(val string) *big.Int {
	amount := decimal.NewFromFloat(0)
	amount, _ = decimal.NewFromString(val)

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	result := amount.Mul(mul)

	res := new(big.Int)
	res.SetString(result.String(), 10)

	return res
}

func ToEther(ivalue interface{}) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	case decimal.Decimal:
		value = v.BigInt()
	case *decimal.Decimal:
		value = v.BigInt()
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

func ToEtherWithDecimal(ivalue interface{}, d uint64) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	case decimal.Decimal:
		value = v.BigInt()
	case *decimal.Decimal:
		value = v.BigInt()
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(d)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

func AtomiToUint64(av atomic.Value) uint64 {
	return av.Load().(uint64)
}

func AtomiToBigInt(av atomic.Value) *big.Int {
	return av.Load().(*big.Int)
}

func IfToBigInt(iamount interface{}) *big.Int {
	amount := decimal.NewFromFloat(0)

	switch v := iamount.(type) {
	case string:
		amount, _ = decimal.NewFromString(v)
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	default:
		return nil
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei
}

func IfToUint64(iamount interface{}) uint64 {
	// amount := decimal.NewFromFloat(0)
	var nRes uint64
	switch v := iamount.(type) {
	case uint64:
		nRes = uint64(v)
	default:
		return 0
	}

	return nRes
}

func StrToEther(ivalue string) decimal.Decimal {
	value := new(big.Int)
	v := ivalue
	value.SetString(v, 10)

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

func DecimalToStr(ivalue decimal.Decimal) string {
	return ivalue.String()
}

func DecimalToBigInt(ivalue primitive.Decimal128) (*big.Int, error) {
	if x, y, err := ivalue.BigInt(); err != nil {
		return nil, err
	} else if y > 0 {
		multmp := big.NewInt(int64(math.Pow(10, float64(y))))
		return x.Mul(x, multmp), nil
	} else {
		return x, nil
	}
}

func StrToBigInt(ivalue string) *big.Int {
	res := new(big.Int)
	if result, b := res.SetString(ivalue, 10); !b {
		return nil
	} else {
		return result
	}
}

func StrToBigFloat(ivalue string) *big.Float {
	res := new(big.Float)
	if result, b := res.SetString(ivalue); !b {
		return nil
	} else {
		return result
	}
}

func Uint64ToBigInt(u64 uint64) *big.Int {
	converted := new(big.Int).SetUint64(u64)
	return converted
}

func Int64ToBigInt(i64 int64) *big.Int {
	converted := new(big.Int).SetInt64(i64)
	return converted
}

// CalcGasCost calculate gas cost given gas limit (units) and gas price (wei)
func CalcGasCost(gasLimit uint64, gasPrice *big.Int) *big.Int {
	gasLimitBig := big.NewInt(int64(gasLimit))
	return gasLimitBig.Mul(gasLimitBig, gasPrice)
}

func BigInt2Uint64(bi *big.Int) uint64 {
	converted := bi.Uint64()
	return converted
}

// SigRSV signatures R S V returned as arrays
func SigRSV(isig interface{}) ([32]byte, [32]byte, uint8) {
	var sig []byte
	switch v := isig.(type) {
	case []byte:
		sig = v
	case string:
		sig, _ = hexutil.Decode(v)
	}

	sigstr := common.Bytes2Hex(sig)
	rS := sigstr[0:64]
	sS := sigstr[64:128]
	R := [32]byte{}
	S := [32]byte{}
	copy(R[:], common.FromHex(rS))
	copy(S[:], common.FromHex(sS))
	vStr := sigstr[128:130]
	vI, _ := strconv.Atoi(vStr)
	V := uint8(vI + 27)

	return R, S, V
}

func ToHex(src []byte) string {
	return "0x" + hex.EncodeToString(src)
}

func Str2Float64(sTarget string) float64 {
	if fres, err := strconv.ParseFloat(sTarget, 64); err != nil {
		return 0
	} else {
		return fres
	}

}

func Float64ToStr(fTarget float64) string {
	return fmt.Sprintf("%v", fTarget)
}

func Float64ToBigInt(iValue float64, digit int) *big.Int {
	iValueBig := big.NewFloat(iValue)
	iValueBig.Mul(iValueBig, big.NewFloat(math.Pow(float64(10), float64(digit))))
	result, _ := iValueBig.Int(big.NewInt(1))
	return result
}

func Int64ToString(iVal int64) string {
	return strconv.FormatInt(iVal, 10)
}

func StrToInt64(sVal string) int64 {
	n, err := strconv.ParseInt(sVal, 10, 64)
	if err != nil {
		return -1
	}
	return n
}

//8fffffff(string) -> (uint64)2415919103
func StrHexToUint64(src string) uint64 {
	hex, err := strconv.ParseUint(src, 16, 64)
	if err != nil {
		return 0
	}
	return hex
}

//(uint64)2415919103 -> 8fffffff(string)
func Uint64ToHexStr(u64 uint64) string {
	//fmt.Sprintf("%02x", u64)  // 0x + u64
	return fmt.Sprintf("%x", u64)
}

func StrToBytes32(str string) [32]byte {
	dest := [32]byte{}
	copy(dest[:], []byte(str))
	return dest
}

func StrToBytes(str string) []byte {
	return []byte(str)
}

func BytesToStr(bystr []byte) string {
	return string(bystr)
}

func Bytes32ToString(bytes32 [32]byte) string {
	return string(bytes32[:bytes.Index(bytes32[:], []byte{0})])
}

//////////////////////////////////////////////////////////////////////
func isArray(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Array ||
		reflect.TypeOf(value).Kind() == reflect.Slice
}

// BigPow returns a ** b as a big integer.
func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}

var (
	tt255     = BigPow(2, 255)
	tt256     = BigPow(2, 256)
	tt256m1   = new(big.Int).Sub(tt256, common.Big1)
	MaxBig256 = new(big.Int).Set(tt256m1)
	tt63      = BigPow(2, 63)
	MaxBig63  = new(big.Int).Sub(tt63, common.Big1)
)

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8
)

// U256 encodes as a 256 bit two's complement number. This operation is destructive.
func U256(x *big.Int) *big.Int {
	return x.And(x, tt256m1)
}

func ReadBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}

func PaddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	ReadBits(bigint, ret)
	return ret
}

func U256Bytes(n *big.Int) []byte {
	return PaddedBigBytes(U256(n), 32)
}

func Uint256(input interface{}) []byte {
	switch v := input.(type) {
	case *big.Int:
		return U256Bytes(v)
	case string:
		bn := new(big.Int)
		bn.SetString(v, 10)
		return U256Bytes(bn)
	}

	if isArray(input) {
		return Uint256Array(input)
	}

	return common.RightPadBytes([]byte(""), 32)
}

// Uint256Array uint256 array
func Uint256Array(input interface{}) []byte {
	var values []byte
	s := reflect.ValueOf(input)
	for i := 0; i < s.Len(); i++ {
		val := s.Index(i).Interface()
		result := common.LeftPadBytes(Uint256(val), 32)
		values = append(values, result...)
	}
	return values
}

// Encode encodes b as a hex string with 0x prefix.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

func Uint64ToString(n64 uint64) string {
	return strconv.FormatUint(n64, 10)
}

func GenGxAddr(addr string, code string) string {
	err := types.VerifyAddressFormat([]byte(addr))
	if err != nil {
		return ""
	}

	pref := addr[:2]
	if pref == "0x" {
		strTmp := addr[2:]
		dcStr, _ := hex.DecodeString(strTmp)
		bc32Addr := sdk.MustBech32ifyAddressBytes(code, dcStr)

		return bc32Addr
		// log.Info("convert eth to cosmos : ", bc32Addr)
	} else {
		return ""
	}
}
