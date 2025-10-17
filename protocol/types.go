package protocol

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
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

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type DefaultPaginationQuery struct {
	Page  int `form:"page,default=1" binding:"min=1"`
	Limit int `form:"limit,default=10" binding:"min=1"`
}

type AesDataForm struct {
	Data string `json:"data"`
}

type BodyWriter struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

type RespDataHeader struct {
	Result       ResultCode  `json:"result"`
	ResultString string      `json:"resultString"`
	Data         interface{} `json:"data"`
}

func NewRespDataHeader(resultCode ResultCode, data interface{}) *RespDataHeader {
	return &RespDataHeader{
		Result:       resultCode,
		ResultString: resultCode.toString(),
		Data:         data,
	}
}
