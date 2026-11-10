package controller

import (
	"ms-gateway/conf"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

//noti=1, call=2, msg=4, rerv=8
//0 = all alert
//15 1111 = no alert
//1 0001 = no noti
//3 0011 = no noti call
//70111 = no noti, call, msg
//15 1111 = no alert
//5 0101 = no noti, msg
//6 0110 = no call msg
//0111 = no noti, call, msg

type SetController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	rdb *models.RedisDB
	adb *models.AccountDB
	hdb *models.HistoryDB
}

func NewSetController(ctl *Controller, rep *models.Repositories) (*SetController, error) {
	r := &SetController{
		ctl: ctl,
		cfg: ctl.cfg,
		rep: rep,
	}

	if err := rep.Get(&r.rdb, &r.adb, &r.hdb); err != nil {
		return nil, err
	}

	return r, nil
}

// GetSetting godoc
// @Summary      본인 알림 설정 조회
// @Description  본인 알림 설정 조회 API입니다.
// @Description
// @Description  **요청**: path uid는 호환용이며 무시합니다.
// @Description
// @Description  **처리**: JWT 본인의 알림 비트 값을 조회합니다.
// @Description
// @Description  **응답 data**: 정수 알림 설정 값.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        uid path string true "호환용 값. 실제 대상은 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=int} "data: 정수 알림 설정 값."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /user/v01/getset/{uid} [get]
// @Example Response: {"result":0,"resultString":"Success","data":0}
// @x-response-example {"result":0,"resultString":"Success","data":0}
// @Example Request: GET /user/v01/getset/4033287471439576593
func (p *SetController) GetSetting(c *gin.Context) {
	// {test243 8697414060736839837  test243@test.com 홍두께 건강한 꼬마 오렌지 1 24 1990-01-01T00:00:00Z 서울 0 https://i.ibb.co/QF37KRST/male-ai-02.webp https://i.ibb.co/C5c51dYg/icon-male-04.webp 반가워요 큐피톡에서 만나요!}
	user, exists := c.Get("user")
	if !exists {
		return
	}

	userInfo, ok := user.(*ptl.UserInfoResp)
	if !ok {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Invalid user info")
		return
	}

	setAlert, err := p.adb.GetSetAlert(userInfo.Uid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get setting")
		return
	}

	/* // todo
	차후 설정 정보 조회
	*/

	p.ctl.SendDataResponse(c, http.StatusOK, setAlert)
}

// SetSetting godoc
// @Summary      본인 알림 설정 토글
// @Description  본인 알림 설정 토글 API입니다.
// @Description
// @Description  **요청**: 실제 입력은 JSON body cate, value(필수 문자열). path cate/value는 무시합니다.
// @Description
// @Description  **처리**: cate는 alert만 허용합니다. value를 정수로 변환하여 기존 알림 값과 XOR하고 저장합니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        cate path string true "호환용 값. 실제 입력은 body cate"
// @Param        value path string true "호환용 값. 실제 입력은 body value"
// @Param        request body object{cate=string,value=string} true "필수 문자열 필드: cate, value"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /user/v01/set/{cate}/{value} [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /user/v01/set/alert/1
// @Example Request Body: {"cate":"alert","value":"1"}
// @x-request-example {"cate":"alert","value":"1"}
func (p *SetController) SetSetting(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		return
	}

	var req struct {
		Cate  string `json:"cate" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to parse JSON")
		return
	}
	if req.Cate == "" || req.Value == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "cate and value are required")
		return
	}

	userInfo, ok := user.(*ptl.UserInfoResp)
	if !ok {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Invalid user info")
		return
	}

	switch req.Cate {
	case "alert":
		err := p.setAlert(userInfo.Uid, req.Value)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update alert setting")
			return
		}
	default:
		p.ctl.SimpleError(c, http.StatusBadRequest, "invalid cate")
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

func (p *SetController) setAlert(uid uint64, value string) error {
	setAlert, err := strconv.Atoi(value)
	if err != nil {
		return err
	}

	oldSetAlert, err := p.adb.GetSetAlert(uid)
	if err != nil {
		return err
	}

	newSetAlert := oldSetAlert ^ setAlert

	err = p.adb.SetAlert(uid, newSetAlert)
	if err != nil {
		return err
	}

	return nil
}
