package controller

import (

	// "strconv"

	"encoding/json"
	"fmt"
	"ms-gateway/hachecker"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"

	"ms-gateway/conf"

	log "ms-gateway/common/logger"
	// "ms-gateway/hachecker"

	"github.com/gin-gonic/gin"
)

// Controller
type Controller struct {
	cfg       *conf.Config
	hchecker  *hachecker.HAChecker
	AccCtl    *AccountController
	PfCtl     *ProfileController
	HomeCtl   *HomeController
	NotiCtl   *NotiController
	FCMPusher *FCMPusher
	Signaling *SignalingController
	ChatCtl   *ChatController
	StoryCtl  *StoryController
	Rdb       *models.RedisDB
}

func NewCTL(cf *conf.Config, hch *hachecker.HAChecker, rep *models.Repositories) (*Controller, error) {
	r := &Controller{
		cfg:      cf,
		hchecker: hch,
	}

	var err error
	if err = rep.Get(&r.Rdb); err != nil {
		return nil, err
	}

	if r.AccCtl, err = NewAccountController(r, rep); err != nil {
		return nil, err
	}

	if r.PfCtl, err = NewProfileController(r, rep); err != nil {
		return nil, err
	}

	if r.NotiCtl, err = NewNotiController(r, rep); err != nil {
		return nil, err
	}

	if r.HomeCtl, err = NewHomeController(r, rep); err != nil {
		return nil, err
	}

	/* 	if r.FCMPusher, err = NewFCMPusher(r, hch, rep); err != nil {
	   		return nil, err
	   	}
	*/
	// Signaling 컨트롤러 생성
	if r.Signaling, err = NewSignalingController(r, rep); err != nil {
		return nil, err
	}

	if r.StoryCtl, err = NewStoryController(r, rep); err != nil {
		return nil, err
	}

	// Chat 컨트롤러 생성
	if r.ChatCtl, err = NewChatController(r, rep); err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Controller) SimpleRespOK(c *gin.Context, resp interface{}) {
	c.JSON(http.StatusOK, resp)
}

func (p *Controller) RespError(c *gin.Context, body interface{}, status int, err ...interface{}) {
	bytes, _ := json.Marshal(body)

	// 로그 레벨 개선
	errorMsg := joinMsg(err)
	if status >= 500 {
		log.Error("Server error", " path ", c.FullPath(), " body ", string(bytes), " status ", status, " error ", errorMsg)
	} else {
		log.Warn("Client error", " path ", c.FullPath(), " status ", status, " error ", errorMsg)
	}

	c.JSON(status, body) // 기존 NewRespHeader 제거로 단순화
	c.Abort()
}

func (p *Controller) SimpleError(c *gin.Context, status int, err ...interface{}) {
	errorMsg := joinMsg(err)

	if status >= 500 {
		log.Error("Server error", " path ", c.FullPath(), " status ", status, " error ", errorMsg)
	} else {
		log.Warn("Client error", " path ", c.FullPath(), " status ", status, " error ", errorMsg)
	}

	c.JSON(status, ptl.NewRespHeader(ptl.Failed, errorMsg))
	c.Abort()
}

func (p *Controller) RespSuccess(c *gin.Context, resp interface{}) {
	c.JSON(http.StatusOK, resp)
}

func (p *Controller) SendResponse(c *gin.Context, status int, data interface{}) {
	dataStr, ok := data.(string)
	if !ok {
		log.Error("Invalid data type, expected string")
		c.JSON(http.StatusInternalServerError, ptl.NewRespHeader(ptl.Failed, "Internal Server Error"))
		return
	}
	c.JSON(status, ptl.NewRespHeader(ptl.Success, dataStr))
}

// 정상 처리할 때 리턴 Data가 있을 경우 사용
func (p *Controller) SendDataResponse(c *gin.Context, status int, data interface{}) {
	c.JSON(status, ptl.NewRespDataHeader(ptl.Success, data))
}

func (p *Controller) GetPaging(num, limit string, tot int) (ptl.Pagination, error) {
	pagination := ptl.Pagination{}
	curPage, err := strconv.Atoi(num)
	if err != nil {
		return pagination, err
	}

	lim, err := strconv.Atoi(limit)
	if err != nil {
		return pagination, err
	}

	totPages := (tot + lim - 1) / lim

	pagination.Page = curPage
	pagination.Limit = lim
	pagination.Total = totPages

	return pagination, nil
}

// GetController 특정 컨트롤러 인스턴스 반환
func (p *Controller) GetController(target interface{}) error {
	switch t := target.(type) {
	case **AccountController:
		*t = p.AccCtl
	case **ProfileController:
		*t = p.PfCtl
	case **ChatController:
		*t = p.ChatCtl
	default:
		return fmt.Errorf("unknown controller type")
	}
	return nil
}

func (p *Controller) GetRedis() *models.RedisDB {
	return p.Rdb
}

func (p *Controller) GetHAChecker() *hachecker.HAChecker {
	return p.hchecker
}

/*
1. 홈 화면 조회
	1-1 상단 메뉴
		a. 알림 아이콘 - 별도 페이지 - noti.go
			: 신규 있는지 표기

		b. 프로필 조회 - 별도 페이지 - profile.go
			: 프로필 이미지, 닉네임, 나이, 지역, 성별

		c. 상점 버튼 - 별도 페이지 - shop.go
			: 상점 화면 이동

		d. 햄버거바 메뉴 - 별도 페이지 - setting.go
			: 메뉴 화면 이동
2. 남자 홈
	2-1 메인 메뉴
		a. 로고 이미지 - 인앱 처리
		b. 영상통화 - 별도 페이지 - videoChat.go
			- 영상통화 화면 이동 : 리스트 출력

		c. 음성통하 - 별도 페이지 - voiceChat.go
			- 음성통화 화면 이동 : 리스트 출력

		d. 스토리 - 별도 페이지 - storyList.go
			- 스토리 화면 이동 : 리스트 출력

		e. 쪽지함 - 별도 페이지 - inbox.go
			- 쪽지함 화면 이동 : 리스트 출력

		f. 결혼 재혼 - 별도 페이지 - marriage.go
			- 결혼 재혼 화면 이동 : 리스트 출력

		g. 출석체크 - 별도 페이지 - checkin.go
			- 출석체크 화면 이동 : 리스트 출력
			: 24시 기준 리셋
	2-2 배너 - 링크
		a. 구글 애드센스

	2-3 영상통화 리스트 - 7개 리스트
		전체보기
		a. 최신 7개 리스트 - 레디스 조회 및 업데이트

	2-4 음성통화 - 7개 리스트
		전체보기
		a. 최신 7개 리스트 - 레디스 조회 및 업데이트
	2-5 스토리 - 6개 리스트
		전체보기
		a. 최신 6개 리스트 - 레디스 조회 및 업데이트
3. 여자 홈
	3-1 보유 금액
	3-2 상단 메뉴
		a. 로고 이미지 - 인앱 처리

		b. 영상 통화 - 별도 페이지 - videoChat.go
			- 영상통화 화면 이동 : 리스트 출력

		c. 음성 통화 - 별도 페이지 - voiceChat.go
			- 음성통화 화면 이동 : 리스트 출력

		d. 스토리 - 별도 페이지 - storyList.go
			- 스토리 화면 이동 : 리스트 출력

		e. 쪽지함 - 별도 페이지 - inbox.go
			- 쪽지함 화면 이동 : 리스트 출력

	3-3 배너 - 링크
		a. 구글 애드센스
	3-4 미션
		a. 미션 총 상금
			- 레디스 조회, DB 업데이트
		b. 미션 카드
			- 레디스 조회, DB 업데이트
	3-5 스토리 - 6개 리스트
		전체보기 - 라우터 처리
		a. 최신 6개 리스트 - 레디스 조회 및 업데이트
4. 푸터
	a. 이용약관 - 캐시 처리
	b. 개인정보처리방침 - 캐시 처리
	c. 고객센터	- 라우터 처리
*/
