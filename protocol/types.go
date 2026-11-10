package protocol

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 모든 응답의 헤더
type RespHeader struct {
	Result       ResultCode `json:"result"`
	ResultString string     `json:"resultString"`
	Msg          string     `json:"msg"`
}

type OkResp struct {
	*RespHeader
}

// RespHeader : RespHeader 객체 생성 및 반환
func NewRespHeader(resultCode ResultCode, msg ...string) *RespHeader {
	return &RespHeader{
		Result:       resultCode,
		ResultString: resultCode.toString(),
		Msg:          strings.Join(msg, ","),
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

type PartnerInfo struct {
	PID      uint64 `json:"pid"`
	Nick     string `json:"nick"`
	ThumbPic string `json:"thumb_pic"`
	Gender   string `json:"gender"`
	Age      string `json:"age"`
	Area     string `json:"area"`
}

type DMRoomResp struct {
	RoomID      int64        `json:"rid"`
	Partner     *PartnerInfo `json:"partner"`
	LastMsg     string       `json:"last_msg"`
	PartnerLeft bool         `json:"partner_left"`
	Total       int          `json:"total"`
	Unread      int          `json:"unread"`
	AtCreate    string       `json:"at_crtchat"`
	AtUpdate    string       `json:"at_update"`
}

// ChatMessage 텍스트 채팅 메시지 구조체
type ChatMessage struct {
	Type      string       `json:"type"`               // text-message, typing, read-receipt, partner-left, system-message, call-*
	From      string       `json:"from"`               // 발신자 ID
	To        string       `json:"to"`                 // 수신자 ID
	RoomID    string       `json:"roomId"`             // 채팅방 ID
	Content   string       `json:"content,omitempty"`  // 메시지 내용
	Timestamp int64        `json:"timestamp"`          // 타임스탬프
	CallMode  string       `json:"callMode,omitempty"` // audio | text | gift | img
	MsgID     string       `json:"msgId,omitempty"`    // 클라이언트 메시지 식별자
	Partner   *PartnerInfo `json:"partner,omitempty"`  // 상대방 정보
	Unread    int          `json:"unread,omitempty"`   // 읽지 않은 메시지 수
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

type CutoutUserItem struct {
	Idx      int       `json:"idx" example:"12"`
	Tid      uint64    `json:"tid" example:"4033287471439576593"`
	Stat     int       `json:"stat" example:"1"`
	Tnick    string    `json:"tnick" example:"푸른 아름다운 양"`
	Tgen     int       `json:"tgen" example:"1"`
	Tbirth   string    `json:"tbirth" example:"1991-12-18"`
	Age      int       `json:"age" example:"34"`
	TspIntro string    `json:"tsp_intro" example:"반가워요"`
	Tarea    int       `json:"tarea" example:"1"`
	TthmbPic string    `json:"tthmb_pic" example:"https://i.ibb.co/99MhfMXt/icon-male-03.webp"`
	AtCreate time.Time `json:"at_crt" example:"2026-06-23T14:30:15Z"`
	AtUpdate time.Time `json:"at_upd" example:"2026-06-23T14:30:15Z"`
}

// CutoutActionData is set/cancel cutout response payload (data field).
type CutoutActionData struct {
	Msg string `json:"msg" example:"ok"`
	Res int64  `json:"res" example:"1"`
}

// CutoutActionResp is full API envelope for cutout set/cancel Swagger (@Success).
type CutoutActionResp struct {
	Result       int              `json:"result" example:"0"`
	ResultString string           `json:"resultString" example:"Success"`
	Data         CutoutActionData `json:"data"`
}

// CutoutCountData is GetCutoutCount response payload (data field).
type CutoutCountData struct {
	Msg   string `json:"msg" example:"ok"`
	Count int64  `json:"count" example:"3"`
}

// CutoutCountResp is full API envelope for GetCutoutCount Swagger (@Success).
type CutoutCountResp struct {
	Result       int             `json:"result" example:"0"`
	ResultString string          `json:"resultString" example:"Success"`
	Data         CutoutCountData `json:"data"`
}

// CutoutListData is GetCutoutList response payload (data field).
type CutoutListData struct {
	Msg        string           `json:"msg" example:"ok"`
	TotalCount int              `json:"total_count" example:"1"`
	List       []CutoutUserItem `json:"list"`
}

// CutoutListResp is full API envelope for GetCutoutList Swagger (@Success).
type CutoutListResp struct {
	Result       int            `json:"result" example:"0"`
	ResultString string         `json:"resultString" example:"Success"`
	Data         CutoutListData `json:"data"`
}

func CvtParamAtoi(param string, ty int) int {
	switch ty {
	case 1: // page default 1
		paramInt, err := strconv.Atoi(param)
		if err != nil || paramInt < 1 {
			return 1
		}
		return paramInt
	case 2: // limit default 20
		paramInt, err := strconv.Atoi(param)
		if err != nil {
			return 20
		}
		return paramInt
	case 3: // pgSize default 50
		paramInt, err := strconv.Atoi(param)
		if err != nil {
			return 50
		}
		return paramInt
	case 4: // limit default 10
		paramInt, err := strconv.Atoi(param)
		if err != nil {
			return 10
		}
		return paramInt
	}
	return 0
}
