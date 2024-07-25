package types

import (
	"time"
)

type Elem struct {
	Name    string `json:"name"`
	Desc    string `json:"descript"`
	Url     string `json:"url"`
	Status  string `json:"status"`
	Reserve string `json:"reserv"`
}

// //////////////////////////////////////////////////////////////////////////////
// //////////////////////////////////////////////////////////////////////////////
type Notice struct {
	Index    string    `json:"_id,omitempty"`
	View     int       `json:"view,omitempty"`
	Category string    `json:"cate,omitempty"`
	Writer   string    `json:"wrt,omitempty"`
	TBegin   time.Time `json:"begin,omitempty"`
	TEnd     time.Time `json:"end,omitempty"`
	TUpdate  time.Time `json:"update,omitempty"`
	TWrited  time.Time `json:"writed,omitempty"`
	Bodys    []Body    `json:"bodys,omitempty"`
}

type Body struct {
	Title    string `json:"title,omitempty"`
	TCreated string `json:"created,omitempty"`
	Content  string `json:"body,omitempty"`
	Location string `json:"loc,omitempty"`
}

type ResultCode int

const (
	Success            ResultCode = 0
	Failed                        = 1   // 요청이 실패하였습니다.
	UserIDNotFound                = 13  // 유저 아이디가 존재하지 않습니다
	AccessTokenInvalid            = 101 // 접속 토큰이 유효하지 않음
	UserNotFound                  = 102 // 유저 정보를 찾을 수 없음
)

type RespHeader struct {
	Result       ResultCode `json:"Result"`
	ResultString string     `json:"ResultString"`
	Desc         string     `json:"Desc"`
}
