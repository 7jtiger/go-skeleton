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
// @Summary Get user notification settings
// @Description Retrieves notification settings for authenticated users via JWT token.
// @Description Notification settings are managed using bit flags:
// @Tags Setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} protocol.RespHeader{data=int} "Notification setting value (0-15)"
// @Failure 401 {object} protocol.RespHeader "Authentication failed - No JWT token"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /user/v01/set/{uid} [get]
// @Example request
// GET /user/v01/set/8697414060736839837
// @Example response 200
// {"code": 200, "message": "success", "data": 5}
// @Example response 401
// {"code": 401, "message": "No JWT"}
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
// @Summary Update user settings
// @Description Updates user settings based on category. Currently supports notification alert settings.
// @Description Alert settings use bit flags (0-15) with XOR operation to toggle:
// @Description - noti=1 (notification), call=2 (call), msg=4 (message), rvr=8 (reserve)
// @Description - 0 = all alerts enabled
// @Description - 15 = all alerts disabled
// @Description - Example: To toggle notification, send value "1"
// @Tags Setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}"
// @Param request body object{cate=string,value=string} true "Setting request"
// @Success 200 {object} protocol.RespHeader{data=object{msg=string}} "Setting updated successfully"
// @Failure 400 {object} protocol.RespHeader "Invalid request - Missing or invalid parameters"
// @Failure 401 {object} protocol.RespHeader "Authentication failed - No JWT token"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /user/v01/set [post]
// @Example request
// POST /user/v01/set
// {"cate": "alert", "value": "1"}
// @Example response 200
// {"code": 200, "message": "success", "data": {"msg": "success"}}
// @Example response 400
// {"code": 400, "message": "cate and value are required"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
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
