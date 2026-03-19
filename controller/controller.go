package controller

import (
	"encoding/json"
	"ms-gateway/hachecker"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"

	"ms-gateway/conf"

	log "ms-gateway/common/logger"

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
	SetCtl    *SetController

	Rdb *models.RedisDB
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

	if r.SetCtl, err = NewSetController(r, rep); err != nil {
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

func (p *Controller) GetRedis() *models.RedisDB {
	return p.Rdb
}

func (p *Controller) GetHAChecker() *hachecker.HAChecker {
	return p.hchecker
}
