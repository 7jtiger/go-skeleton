package controller

import (
	"errors"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HomeController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	rdb *models.RedisDB
	adb *models.AccountDB
	hdb *models.HistoryDB
	idb *models.ItemDB
}

func NewHomeController(ctl *Controller, rep *models.Repositories) (*HomeController, error) {
	r := &HomeController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.rdb, &r.hdb, &r.idb); err != nil {
		return nil, err
	}

	return r, nil
}

/*
상단메뉴 - 신규 알림 유무
메인메뉴 - 쪽지갯수
		- 출석체크 유무
배너 출력
영상통화 최신 7개 리스트 출력
음성통화 최신 7개 리스트 출력
스토리 최신 6개 리스트 출력
이용약관, 개인정보처리방침 링크
*/

type HomeData struct {
	NotiNew    int               `json:"notiNew"`
	MsgQty     int               `json:"msgQuantity"`
	CheckIn    bool              `json:"checkIn"`
	VdChatList *[]ptl.WTRoomUser `json:"videoChatList"`
	VoChatList *[]ptl.WTRoomUser `json:"voiceChatList"`
	StoryList  *[]ptl.Pre7Story  `json:"storyList"`
	TermsLink  string            `json:"termsLink"`
	PolicyLink string            `json:"policyLink"`
}

type ChatUser struct {
	UID     string `json:"uid"`     //user id
	MainPic string `json:"mainPic"` //profile picture
	Intro   string `json:"intro"`   //intro
	Gender  int    `json:"gender"`  //gender
	Nick    string `json:"nick"`    //nickname
	Area    string `json:"area"`    //area
	Age     int    `json:"age"`     //age
	NewStat bool   `json:"newStat"` //new status
}

type Story struct {
	UID      string `json:"uid"` //user id
	STID     string `json:"stid"`
	StoryPic string `json:"storyPic"`
}

// GetHomeMenInfo godoc
// @Summary Retrieves main information to be displayed on the home screen
// @Description Retrieves main information to be displayed on the home screen (new notifications, message count, check-in status, video/voice chat lists, story list, terms links)
// @Tags Home
// @Accept json
// @Produce json
// @Security JwtAuth
// @Success 200 {object} HomeData "Home screen information"
// @Failure 400 {object} ptl.RespHeader "Invalid request"
// @Failure 401 {object} ptl.RespHeader "Authentication failed"
// @Failure 500 {object} ptl.RespHeader "Server error"
// @Router /home/v01/mdata [get]
func (p *HomeController) GetHomeMenInfo(c *gin.Context) {
	//noti 확인
	userID, ok := c.Get("user")
	if !ok {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get user info"), http.StatusBadRequest, errors.New("user not found"))
		return
	}

	// 신규 알림 확인
	// msg_his
	msgQty, err := p.hdb.GetNewMsgCount(uint64(userID.(int)))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get new msg count"), http.StatusBadRequest, err)
		return
	}
	// 쪽지 갯수 확인
	// noti_his
	notiQty, err := p.hdb.GetNewNotiCount(uint64(userID.(int)))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get new noti count"), http.StatusBadRequest, err)
		return
	}
	// 출석체크 유무 확인
	// evt_chkin
	checkIn, err := p.idb.GetCheckIn(uint64(userID.(int)))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get check in"), http.StatusBadRequest, err)
		return
	}

	// 영상통화 최신 7개 리스트 출력
	// vd_his
	vdChatList, err := p.rdb.HGetJoinWTRoomPre7List()
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get new vd chat list"), http.StatusBadRequest, err)
		return
	}

	// 음성통화 최신 7개 리스트 출력
	// vo_his
	voChatList, err := p.rdb.HGetJoinWTRoomPre7List()
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get new vo chat list"), http.StatusBadRequest, err)
		return
	}

	// 스토리 최신 6개 리스트 출력
	// st_his
	storyList, err := p.adb.GetStory7List(uint64(userID.(int)))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get story list"), http.StatusBadRequest, err)
		return
	}

	// 이용약관, 개인정보처리방침 링크
	// anuc_his
	privacyUrl, termsUrl, err := p.idb.GetTermsInfo()
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get terms info"), http.StatusBadRequest, err)
		return
	}

	homeData := HomeData{
		MsgQty:     msgQty,
		NotiNew:    notiQty,
		CheckIn:    checkIn,
		VdChatList: vdChatList,
		VoChatList: voChatList,
		StoryList:  storyList,
		TermsLink:  termsUrl,
		PolicyLink: privacyUrl,
	}

	p.ctl.SimpleRespOK(c, homeData)
}
