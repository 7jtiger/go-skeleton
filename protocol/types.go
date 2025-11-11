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

type NotiItem struct {
	Idx   string
	Title string
	Msg   string
	Cate  int
}

// ChatMessage 텍스트 채팅 메시지 구조체
type ChatMessage struct {
	Type      string `json:"type"`      // text-message, typing, read-receipt
	From      string `json:"from"`      // 발신자 ID
	To        string `json:"to"`        // 수신자 ID
	RoomID    string `json:"roomId"`    // 채팅방 ID
	Content   string `json:"content"`   // 메시지 내용
	Timestamp int64  `json:"timestamp"` // 타임스탬프
}

// CallRequest 통화 요청 구조체
type CallRequest struct {
	From     string `json:"from"`     // 발신자 ID
	To       string `json:"to"`       // 수신자 ID
	CallType string `json:"callType"` // video, audio, text
	RoomID   string `json:"roomId"`   // 채팅방 ID (생성된 경우)
}

// CallResponse 통화 응답 구조체
type CallResponse struct {
	From     string `json:"from"`     // 응답자 ID
	To       string `json:"to"`       // 요청자 ID
	RoomID   string `json:"roomId"`   // 채팅방 ID
	Accepted bool   `json:"accepted"` // 수락 여부
}
