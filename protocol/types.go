package protocol

import (
	"strings"
)

// 모든 응답의 헤더
type RespHeader struct {
	Result       ResultCode `json:"result"`
	ResultString string     `json:"resultString"`
	Desc         string     `json:"desc"`
}

type OkResp struct {
	*RespHeader
}

// RespHeader : RespHeader 객체 생성 및 반환
func NewRespHeader(resultCode ResultCode, desc ...string) *RespHeader {
	return &RespHeader{
		Result:       resultCode,
		ResultString: resultCode.toString(),
		Desc:         strings.Join(desc, ","),
	}
}

var LangCode = map[string]bool{
	"ko":  true,
	"en":  true,
	"cn":  true,
	"tss": true,
	"es":  true,
	"ja":  true,
}
