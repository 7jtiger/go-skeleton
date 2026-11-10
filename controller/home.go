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
// @Summary      남성 홈 데이터 조회 (현재 인증 컨텍스트 제한)
// @Description  남성 홈 데이터 조회 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음.
// @Description
// @Description  **처리**: 현재 user를 int로 단언하지만 JWT 미들웨어는 *UserInfoResp를 제공합니다. 정상 JWT 요청도 panic/Recovery 500이 발생하여 홈 데이터 응답이 완료되지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Home
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /home/v01/mdata [get]
// @Example Request: GET /home/v01/mdata
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
