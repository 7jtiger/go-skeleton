package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	"ms-gateway/models"
	"ms-gateway/protocol"
	ptc "ms-gateway/protocol"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	ACCESS_TKN_EXPIRE  = 24 * time.Hour
	REFRESH_TKN_EXPIRE = 14 * 24 * time.Hour
)

type AccountController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	rdb *models.RedisDB
	adb *models.AccountDB
}

func NewAccountController(ctl *Controller, rep *models.Repositories) (*AccountController, error) {
	r := &AccountController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

// @Summary get version
// @Description get version
// @Tags Server
// @Produce json
// @Success 200 {object} map[string]interface{} "version info"
// @Router /serv/v01/version [get]
func (p *AccountController) GetVersion(c *gin.Context) {
	p.ctl.SimpleRespOK(c, gin.H{"version": "0.9.1"})
}

// @Summary check user id
// @Description When /acc/v01/check/:id is called, it returns "ok" on success and a message on failure.
// @Description id : user id
// @Tags user
// @Accept json
// @Produce json
// @Success 200 {object} protocol.OkResp "result: ok"
// @Failure 400 {object} protocol.RespHeader "result: request failed"
// @Failure 14 {object} protocol.RespHeader "result: ID is duplicate"
// @Router /acc/v01/check/:id [get]
func (p *AccountController) CheckID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "id is required")
		return
	}

	if p.adb.IsExistID(strings.TrimSpace(id)) {
		// p.ctl.SimpleError(c, http.StatusConflict, "id is duplicate")
		p.ctl.SimpleError(c, ptl.IDDuplicate, "ID is duplicate")
	} else {
		p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
	}
}

// @Summary check user email
// @Description When /acc/v01/ckemail/:email is called, it returns "ok" on success and a message on failure.
// @Description email : user email
// @Tags user
// @Accept json
// @Produce json
// @Success 200 {object} protocol.OkResp "result: ok"
// @Failure 400 {object} protocol.RespHeader "result: request failed"
// @Failure 15 {object} protocol.RespHeader "result: email is duplicate"
// @Router /acc/v01/ckemail/:email [get]
func (p *AccountController) CheckEmail(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "email is required")
		return
	}
	if p.adb.IsExistEmail(strings.TrimSpace(email)) {
		log.Warn("CheckEmail", email)
		p.ctl.SimpleError(c, ptl.EmailDuplicate, "Email is duplicate")
	} else {
		p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
	}
}

// @Summary Register a new user
// @Description Calls /acc/v01/regist to register a new user. Returns "success" message on success.
// @Description Required fields: id, pw (hashed), name, gender (1=male, 0=female), birth (YYYY-MM-DD format), area, email
// @Description Optional fields: did (device ID), dos (device OS)
// @Description System generates: uid (unique user ID), nick (default nickname), main_pic (default profile image), thmb_pic (default thumbnail), sp_intro (default introduction)
// @Tags user
// @Accept json
// @Produce json
// @Param request body protocol.RegistReq true "register request data"
// @Success 200 {object} protocol.OkResp "result: success"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 104 {object} protocol.RespHeader "Failed Register User"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 14 {object} protocol.RespHeader "ID is duplicate"
// @Failure 15 {object} protocol.RespHeader "Email is duplicate"
// @Router /acc/v01/regist [post]
// @Example request:
//
//	{
//	  "id": "testuser123",
//	  "pw": "hashedpassword123",
//	  "name": "홍길동",
//	  "email": "test@example.com",
//	  "gender": 1,
//	  "birth": "1990-01-01",
//	  "area": "서울",
//	  "did": "device_token_12345",
//	  "dos": "android"
//	}
//
// @Example response (success):
// {"msg": "success" }
//
// @Example response (ID duplicate):
// {"result": 14, "msg": "ID is already in use."}
//
// @Example response (Email duplicate):
// {"result": 15,"msg": "Email is already in use."}
//
// @Example response (missing parameter):
// {"result": 102, "msg": "all fields are required" }
func (p *AccountController) RegistUserInfo(c *gin.Context) {
	var req ptl.RegistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.ID == "" || req.PW == "" || req.Area == "" || req.Name == "" || req.Birth == "" || req.Email == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	if p.adb.IsExistID(strings.TrimSpace(req.ID)) {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.IDDuplicate, "ID is already in use."), http.StatusBadRequest, fmt.Errorf("ID is duplicate"))
		return
	}

	if p.adb.IsExistEmail(strings.TrimSpace(req.Email)) {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.EmailDuplicate, "Email is already in use."), http.StatusBadRequest, fmt.Errorf("Email is duplicate"))
		return
	}

	// gender := strconv.Itoa(req.Gender)
	// Generate additional fields
	req.Uid = p.getUid()

	req.Nick = utils.GenDefNick()
	req.ThumbIcon = GetRandDefIcon(req.Gender)
	req.MainPic = GetRandDefIntroImg(req.Gender)
	hsedPw, err := bcrypt.GenerateFromPassword([]byte(req.PW), 11)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	req.SPIntro = "반가워요 큐피톡에서 만나요!"

	// Register the user
	err = p.adb.RegistUser(req, hsedPw)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserRegistFailed, "failed to register user"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

func (p *AccountController) getUid() uint64 {
	// var err error
	for {
		uid, err := utils.GenRandomUID()
		if err != nil {
			continue
		}

		IsExistUid := p.adb.IsExistUid(uid)
		if IsExistUid { //true : 존재함, false : 존재하지 않음
			continue // 존재하면 다시 생성
		}

		return uid // 존재하지 않으면 반환
	}
}

// @Summary Login a user
// @Description Calls /acc/v01/login to login a user. Returns JWT token, user ID, and WebRTC config on success. Also sets x-meta header with user metadata.
// @Tags user
// @Accept json
// @Produce json
// @Param data body protocol.LoginReq true "Login request data {id : xxx, pw : hash}"
// @Success 200 {object} protocol.LoginUserResp "Login success with token, uid, and webrtc config"
// @Header 200 {string} x-meta "User metadata in format: uid/sid/did/nick/gender/age/area/email/main_pic/thmb_pic/sp_intro"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 104 {object} protocol.RespHeader "Failed to login user"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /acc/v01/login [post]
// @Example request:// {//   "id": "test123",//   "pw": "mypassword123"// }
//
// @Example response (success):
//
//	{
//	  "msg": "success",
//	  "acTok": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//	  "refTok": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
//	  "uid": "77665817",
//	  "wrtc": {
//	    "iceServers": [
//	      {
//	        "urls": ["stun:stun.l.google.com:19302"]
//	      }
//	    ]
//	  }
//	}
//
// @Example response (failure):
// {"result": 104,"msg": "failed to login user","data": null}
//
// @Note x-meta header format:
// uid/sid/did/nick/gender/age/area/email/main_pic/thmb_pic/sp_intro
// Example: "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
func (p *AccountController) LoginUser(c *gin.Context) {
	var req ptl.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields req.ID : login id, sid, service id
	if req.ID == "" || req.PW == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	user, err := p.adb.LoginUser(req, []byte(req.PW))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to login user"), http.StatusBadRequest, err)
		return
	}

	acTok, refTok, err := p.genLoginUserToken(user)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to generate login user token"), http.StatusBadRequest, err)
		return
	}

	// mStr := fmt.Sprintf("%d/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
	// 	user.Uid, user.ID, user.Did, user.Nick, user.Gender, user.Age, user.Area, user.Email, user.MainPic, user.ThumbPic, user.SPIntro)

	// // x-meta header에 저장
	// c.Header("x-meta", mStr)

	// WebRTC 설정 정보 조회
	webrtcConfig, err := p.rdb.GetWebRTCConfig(user.Uid)
	if err != nil {
		log.Warn("Failed to get WebRTC config:", err)
		// WebRTC 설정 조회 실패는 로그인 실패로 처리하지 않음
		webrtcConfig = nil
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to get WebRTC config"), http.StatusBadRequest, err)
		return
	}

	uidStr := strconv.FormatUint(user.Uid, 10)
	responseData := ptl.LoginUserResp{
		Message: "success",
		// MetaHeader:   "mStr",
		MetaHeader:   "",
		AccessToken:  acTok,
		RefreshToken: refTok,
		UID:          uidStr,
		WebRTCConfig: *webrtcConfig,
	}

	wtRoomUser := ptl.WTRoomUser{
		UID:      user.Uid,
		SID:      user.ID,
		DID:      user.Did,
		MainPic:  user.MainPic,
		ThumbPic: user.ThumbPic,
		Intro:    user.SPIntro,
		Gender:   user.Gender,
		Nick:     user.Nick,
		Area:     user.Area,
		Age:      user.Age,
		NewStat:  false,
	}

	err = p.rdb.HSetJoinWTRoom(user.Uid, &wtRoomUser)
	if err != nil {
		log.Warn("Failed to set join WebRTC room:", err)
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to set join WebRTC room"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, responseData)
}

func (p *AccountController) genLoginUserToken(user *ptl.UserInfoResp) (string, string, error) {
	uidStr := strconv.FormatUint(user.Uid, 10)

	claims := utils.GetJWTClaims(uidStr, ACCESS_TKN_EXPIRE)
	acTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, claims)
	if err != nil {
		log.Warn("Failed to create JWT token:", err)
		return "", "", err
	}

	rfClaims := utils.GetJWTClaims(uidStr, REFRESH_TKN_EXPIRE)
	refTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, rfClaims)
	if err != nil {
		log.Warn("Failed to create JWT token:", err)
		return "", "", err
	}

	err = p.rdb.HSetJWTRefresh(refTok, user)
	if err != nil {
		log.Warn("Failed to set JWT refresh token:", err)
		return "", "", err
	}

	err = p.rdb.HSetJWTAccess(acTok, user)
	if err != nil {
		log.Warn("Failed to set JWT access token:", err)
		return "", "", err
	}

	return acTok, refTok, nil
}

// @Summary Refresh access token
// @Description Reissues access and refresh token using a valid refresh token.
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string false "Bearer refresh_token"
// @Param data body protocol.RefreshTokenReq false "Refresh token payload { refTok: xxx }"
// @Success 200 {object} protocol.RespDataHeader{data=protocol.RefreshTokenResp}
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Invalid refresh token"
// @Router /acc/v01/refresh [post]
// @xample 요청 예시
// POST /acc/v01/refresh
// Header:
//
//	Authorization: Bearer <refresh_token>
//
// Body:
//
//	{
//	    "refTok": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2ODg4ODg4ODgsInVzZXJpZCI6IjEyMzQ1In0.sM4wL5WOEV2TqtW06R1vGuFsxWXhYMrh6oZ7PWnykJc"
//	}
//
// @xample 응답 예시
// HTTP/1.1 200 OK
// Content-Type: application/json
//
//	{
//	    "msg": "success",
//	    "acTok": "new-access-token-here",
//	    "refTok": "new-refresh-token-here",
//	    "uid": "12345"
//	}
func (p *AccountController) RefreshToken(c *gin.Context) {
	refTok := extractRefreshToken(c)
	if refTok == "" {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "No refresh token")
		return
	}

	claims, err := utils.VerifyJWTToken(refTok, p.cfg.Server.JWTSecret)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	user, err := p.rdb.HGetJWTRefresh(refTok)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Refresh token not found")
		return
	}

	uidStr := strconv.FormatUint(user.Uid, 10)
	if claims.UserID != uidStr {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Token user mismatch")
		return
	}

	acClaims := utils.GetJWTClaims(uidStr, ACCESS_TKN_EXPIRE)
	acTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, acClaims)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create access token")
		return
	}

	rfClaims := utils.GetJWTClaims(uidStr, REFRESH_TKN_EXPIRE)
	newRefTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, rfClaims)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	if err := p.rdb.RotateJWTToken(refTok, acTok, newRefTok, user); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to rotate JWT token")
		return
	}

	// metaHeader := fmt.Sprintf("%d/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
	// 	user.Uid, user.ID, user.Did, user.Nick, user.Gender, user.Age, user.Area, user.Email, user.MainPic, user.ThumbPic, user.SPIntro)
	// c.Header("x-meta", metaHeader)

	resp := ptl.RefreshTokenResp{
		Message:      "success",
		AccessToken:  acTok,
		RefreshToken: newRefTok,
		UID:          uidStr,
	}
	p.ctl.SendDataResponse(c, http.StatusOK, resp)
}

func extractRefreshToken(c *gin.Context) string {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader != "" {
		parts := strings.Fields(authHeader)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token := strings.TrimSpace(parts[1])
			if token != "" {
				return token
			}
		}
	}

	if c == nil || c.Request == nil || c.Request.Body == nil {
		return ""
	}

	body, err := c.GetRawData()
	if err != nil {
		// Empty body on refresh endpoint is a valid case when token is provided via Authorization header.
		if errors.Is(err, io.EOF) {
			return ""
		}
		return ""
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}

	var req ptl.RefreshTokenReq
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	token := strings.TrimSpace(req.RefreshToken)
	if token != "" {
		return token
	}

	return ""
}

// @Summary Logout user
// @Description Logout user
// @Tags user
// @Accept json
// @Produce json
// @Param Authorization header string false "Bearer access_token"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /inserv/v01/logout [post]
func (p *AccountController) LogoutUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	log.Info("LogoutUser: ", user)
	userInfo, ok := user.(*ptl.UserInfoResp)
	if !ok {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Invalid user info")
		return
	}

	hd := c.GetHeader("Authorization")
	if hd == "" {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tokens := strings.Split(hd, " ")
	if len(tokens) != 2 {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "Invalid Bearer Type")
		return
	}

	err := p.rdb.DeleteJWTToken(tokens[1])
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLogoutFailed, "failed to delete JWT token"), http.StatusBadRequest, err)
		return
	}
	//
	// delete chat room
	err = p.rdb.HDeleteJoinWTRoom(userInfo.Uid)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLogoutFailed, "failed to delete chat room"), http.StatusBadRequest, err)
		return
	}

	err = p.adb.LogoutUser(userInfo.Uid)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLogoutFailed, "failed to logout user"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Leave a user
// @Description Performs user leave. Returns "success" message on success.
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 GCM mode {id : xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 113 {object} protocol.RespHeader "Failed to leave user"
// @Router /inserv/v01/leave [post]
func (p *AccountController) LeaveUser(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.ID == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	// Register the user
	err := p.adb.LeaveUser(req.ID)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLeaveFailed, "failed to leave user"), http.StatusBadRequest, err)
		return
	}

	//todo
	// add param email?
	// delete jwt
	// delete chat room

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Find user ID
// @Description Find user ID by name and birth date
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 GCM mode {name : xxx, birth : 1990-01-01, gener : 1}"
// @Success 200 {object} protocol.RespDataHeader{data=[]string}
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 111 {object} protocol.RespHeader "Failed to find id"
// @Router /acc/v01/fnid [post]
func (p *AccountController) FindID(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Birth  string `json:"birth" binding:"required"`
		Gender string `json:"gender" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.Name == "" || req.Birth == "" || req.Gender == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	ids := p.adb.FindID(req.Name, req.Birth, req.Gender)
	if ids == nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserFindIDFailed, "failed to find id"), http.StatusBadRequest, fmt.Errorf("failed to find id"))
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, ids)
}

// @Summary Find user PW
// @Description Find user password by id and birth date
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 {sid : xxx, name: xxx, birth 1990-01-018, gender : 1}"
// @Success 200 {object} protocol.OkResp
// @Success 200 {object} protocol.RespDataHeader{data=object} "success with {msg : ok, email : xxx}"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 104 {object} protocol.RespHeader "Failed Decrypt"
// @Failure 106 {object} protocol.RespHeader "Failed Parse JSON"
// @Failure 102 {object} protocol.RespHeader "All fields are required"
// @Failure 112 {object} protocol.RespHeader "Failed to find pw"
// @Router /acc/v01/fnpw [post]
func (p *AccountController) FindPW(c *gin.Context) {
	// Email string `json:"email" binding:"required"`
	var req struct {
		SID    string `json:"sid" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Birth  string `json:"birth" binding:"required"`
		Gender string `json:"gender" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.SID == "" || req.Name == "" || req.Birth == "" || req.Gender == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	res, email := p.adb.CheckNameBirth(req.SID, req.Name, req.Birth, req.Gender)
	switch res {
	case 0: // db error
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserFindPWFailed, "db connect err"), http.StatusBadRequest, fmt.Errorf("db connect error"))
		return
	case 1, 3: // no rows
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserFindPWFailed, "not found user"), http.StatusBadRequest, fmt.Errorf("not found user"))
		return
	case 4: // email decrypt error
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserFindPWFailed, "data error"), http.StatusBadRequest, fmt.Errorf("data error"))
		return
	case 2: // success
		p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "email": email})
		return
	}
}

func (p *AccountController) sendEmailOtpCode(email, nick, otp string) bool {
	tmpl, err := template.New("email").Parse(protocol.EmailOTPCode)
	if err != nil {
		log.Error("Error parsing template:", err)
		return false
	}

	// Prepare data for template
	data := protocol.EmailData{
		OTPCode: otp,
	}

	// Execute template with data
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Error("Error executing template:", err)
		return false
	}

	// Output the rendered HTML
	// fmt.Println(buf.String())

	err = utils.SendGoMail(email, nick, "[CupiTok] OTP 코드", buf.String())
	if err != nil {
		log.Error("Failed to send email: %v", err)
		return false
	}

	return true
}

// ReqAuthOTP - Request OTP authentication code (NOT ROUTED - internal use)
func (p *AccountController) ReqAuthOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	if req.Email == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}
	var userID string
	atIndex := strings.Index(req.Email, "@")
	if atIndex != -1 {
		userID = req.Email[:atIndex]
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	// Generate OTP
	otp := utils.GenerateOTP()
	go func() {
		if !p.sendEmailOtpCode(req.Email, userID, otp) {
			log.Error("Failed to send OTP email to:", req.Email)
		}
	}()

	err := p.rdb.HSetOTP(req.Email, otp)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserAuthOTPFailed, "failed to set OTP"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Verify OTP authentication code
// @Description Verify OTP code for authentication
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 {email : xxx, otp : xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 116 {object} protocol.RespHeader "Failed to verify OTP"
// @Router /acc/v01/verifyotp [post]
func (p *AccountController) VerifyOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
		OTP   string `json:"otp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	if req.Email == "" || req.OTP == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	otp, err := p.rdb.HGetOTP(req.Email)
	if err != nil {
		log.Error("failed to get OTP:", err)
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserAuthOTPFailed, "failed to get OTP"), http.StatusBadRequest, err)
		return
	}

	if otp != req.OTP {
		log.Error("invalid OTP:", otp, req.OTP)
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserAuthOTPFailed, "invalid OTP"), http.StatusBadRequest, fmt.Errorf("invalid OTP"))
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Change user password
// @Description Change the password of a user
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 {id : xxx, uid : xxx, email : xxx, pw : hash, newpw : hash}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 109 {object} protocol.RespHeader "Failed to change pw"
// @Router /acc/v01/cngpw [post]
func (p *AccountController) ChangePW(c *gin.Context) {
	var req struct {
		ID    string `json:"id" binding:"required"`
		Uid   string `json:"uid" binding:"required"`
		Email string `json:"email" binding:"required"`
		PW    string `json:"pw" binding:"required"`
		NewPW string `json:"newpw" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.ID == "" || req.Uid == "" || req.Email == "" || req.PW == "" || req.NewPW == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}
	//todo 비번 확인 프로세스
	// Register the user
	err := p.adb.ChangePW(req.ID, req.Uid, req.Email, req.PW, req.NewPW)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserChangePWFailed, "failed to change pw"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Modify user information
// @Description Modify user information. The 'cate' field should be one of "pw, area, nick, email" indicating the target to be changed, and the 'value' field should contain the new value for the target.
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 GCM mode {id : xxx, uid : xxx, email : xxx, cate : xxx, value : xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 104 {object} protocol.RespHeader "Failed to change pw"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Router /inserv/v01/modify [post]
func (p *AccountController) ModifyUserInfo(c *gin.Context) {
	var req struct {
		ID    string `json:"id" binding:"required"`
		Uid   string `json:"uid" binding:"required"`
		Email string `json:"email" binding:"required"`
		Cate  string `json:"cate" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// Validate the required fields
	if req.Cate == "" || req.Value == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	// Register the user
	//TODO : 변경시 레디스도 업데이트 필요요
	err := p.adb.ModifyUserInfo(req.ID, req.Uid, req.Email, req.Cate, req.Value)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserChangePWFailed, "failed to change pw"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Get user information
// @Description Get user from sid information
// @Tags user
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} protocol.UserInfoResp "User information"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 114 {object} protocol.RespHeader "Failed to get user info"
// @Router /inserv/v01/info/{id} [get]
func (p *AccountController) GetSidUserInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "id is required")
		return
	}

	user, err := p.adb.GetUserInfo(id)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserInfoFailed, "failed to get user info"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, user)
}

// @Summary Get user information by UID
// @Description 주어진 UID로 사용자의 정보를 조회합니다.
// @Tags user
// @Accept json
// @Produce json
// @Param tid path string true "Target User UID"
// @Success 200 {object} protocol.UserInfoResp "User information"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 114 {object} protocol.RespHeader "Failed to get user info"
// @Router /inserv/v01/uinfo/{tid} [get]
func (p *AccountController) GetUidFromInfo(c *gin.Context) {
	tid := c.Param("tid")
	if tid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "uid is required")
		return
	}

	uidUint, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserInfoFailed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	user, err := p.adb.GetUserInfoByUID(uidUint)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserInfoFailed, "failed to get user info"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, user)
}

// @Summary Delete user
// @Description Delete user, but just update user status to 4
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 GCM mode {id : xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 115 {object} protocol.RespHeader "Failed to delete user"
// @Router /inserv/v01/delete/{id} [post]
func (p *AccountController) DeleteUser(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	if req.ID == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	err := p.adb.DeleteUser(req.ID)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserDeleteFailed, "failed to delete user"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// @Summary Modify main picture
// @Description Upload and modify user's main profile picture to Cloudflare Images
// @Tags user
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param files formData file true "Profile image file (max 5MB)"
// @Param data formData string true "Encrypted user data {uid: xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message - No files uploaded, No sinfo uploaded, Failed to decrypt sinfo, Failed to parse sinfo data, UID is required in sinfo"
// @Failure 500 {object} protocol.RespHeader "error: error message - Failed to upload main picture, Failed to modify main picture on db"
// @Router /inserv/v01/upd/mpic [post]
//
//	@Example curl -X POST http://localhost:8080/inserv/v01/upd/mpic \
//	  -F "files=@/home/tmp/bc1.jpg" \
//	  -F "data=wdXTrpgAu+c8HhkKx6fNEzLIdrtGCsVYSLcDT6rgBm0+aEEGPSjnQVqL2nc=" {"uid": "8697414060736839837"} \
//	  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
//	  -H "x-meta: 8697414060736839837/test243//건강한 꼬마 오렌지/1/24/서울/test243@test.com/https://i.ibb.co/QF37KRST/male-ai-02.webp/https://i.ibb.co/C5c51dYg/icon-male-04.webp/반가워요 큐피톡에서 만나요!" \
//	  -v
func (p *AccountController) ModifyMainPic(c *gin.Context) {
	fileInfo, ok := c.Get("uploadedFiles")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No files uploaded")
		return
	}
	//from.Value :map[data:[wdXTrpgAu+c8HhkKx6fNEzLIdrtGCsVYSLcDT6rgBm0+aEEGPSjnQVqL2nc=]]
	//from.Value["data"][0] : 6XMm5LGE6ETe4HRsQbEJWxypTmyYiFEnZaKlyOOhbxaCzGNwTScIFxJGnLQ=
	sinfo, ok := c.Get("sinfo")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No sinfo uploaded")
		return
	}

	sinfoBytes, err := utils.DecryptGCM(sinfo.(string), []byte(p.cfg.Server.BaseKey))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to decrypt sinfo", err)
		return
	}

	var sinfoData struct {
		UID string `json:"uid"`
	}

	if err := json.Unmarshal(sinfoBytes, &sinfoData); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to parse sinfo data", err)
		return
	}

	if sinfoData.UID == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "UID is required in sinfo")
		return
	}

	files := fileInfo.([]*multipart.FileHeader)
	cldFlrInfos, err := utils.UploadCldFlrImg(files, p.cfg.Server.CfId, p.cfg.Server.CfToken)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload main picture", err)
		return
	}

	// Extract URLs from cldFlrInfos map
	urls := make([]string, 0, len(*cldFlrInfos))
	for _, url := range *cldFlrInfos {
		urls = append(urls, url)
	}

	if err := p.adb.ModifyUserInfo("", sinfoData.UID, "", "main_pic", urls[0]); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to modify main picture on db", err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, "success")
}

// @Summary Get WebRTC configuration
// @Description Get WebRTC configuration for the authenticated user
// @Tags WebRTC
// @Accept json
// @Produce json
// @Success 200 {object} protocol.WebRTCConfig "WebRTC configuration"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Router /serv/v01/wsconf [get]
// @Router /webrtc/v01/config [get]
func (p *AccountController) GetWebRTCConfig(c *gin.Context) {
	// JWT에서 사용자 ID 추출
	userID, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid, err := strconv.ParseUint(userID.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse user ID"), http.StatusBadRequest, err)
		return
	}

	config, err := p.rdb.GetWebRTCConfig(uid)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get WebRTC config"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, config)
}

// TestSpecificStunServer - Test specific STUN server (NOT ROUTED - internal use)
func (p *AccountController) TestSpecificStunServer(c *gin.Context) {
	var req struct {
		StunURL string `json:"stunUrl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// STUN 서버 테스트
	result := TestStunServer(req.StunURL, 5*time.Second)

	p.ctl.SendResponse(c, http.StatusOK, result)
}

/*
// Followser 나를 팔로우 하는 사용자 목록 조회
// Followee 내가 팔로우 하는 사용자 목록 조회
// Blocker 나를 차단한 사용자 목록 조회
// Blockee 내가 차단한 사용자 목록 조회

// FollowUser 나를 팔로우 하는 사용자 목록 조회
// UnfollowUser 내가 팔로우 하는 사용자 목록 조회
// BlockUser 나를 차단한 사용자 차단
// UnblockUser 내가 차단한 사용자 차단 해제

*/ // 내가(현재 유저) - 해당 사용자를 즐겨찾기 등록

// SetFavoriteUser godoc
// @Summary      즐겨찾기 등록
// @Description  현재 로그인한 사용자가 특정 사용자를 즐겨찾기 목록에 추가합니다.
// @Description  request : POST /account/v01/favorite/set/{tid}
// @Description  response : {"result":0,"msg":"Success"}
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        tid   path      string  true  "즐겨찾기 등록 대상 사용자 uid"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /account/v01/favorite/set/{tid} [post]
func (p *AccountController) SetFavoriteUser(c *gin.Context) {
	tid := c.Param("tid")
	if tid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is required")
		return
	}

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	tid64, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse tid"), http.StatusBadRequest, err)
		return
	}

	err = p.adb.SetFavoriteUser(uid64, tid64, true)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to set favorite user"), http.StatusInternalServerError, err)
		return
	}
	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
	})
}

// UnfavoriteUser godoc
// @Summary      즐겨찾기 해제
// @Description  현재 로그인한 사용자가 특정 사용자를 즐겨찾기 목록에서 제거합니다.
// @Description  request : POST /account/v01/favorite/cancel/{tid}
// @Description  response : {"result":0,"msg":"Success"}
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        tid   path      string  true  "즐겨찾기 해제 대상 사용자 uid"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /account/v01/favorite/cancel/{tid} [post]
func (p *AccountController) UnfavoriteUser(c *gin.Context) {
	tid := c.Param("tid")
	if tid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is required")
		return
	}

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	tid64, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse tid"), http.StatusBadRequest, err)
		return
	}

	// 즐겨찾기 해제 처리 (enable=false)
	err = p.adb.SetFavoriteUser(uid64, tid64, false)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to unfavorite user"), http.StatusInternalServerError, err)
		return
	}
	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
	})
}

/*
// 나를(현재 유저) - 팔로우 하는 사용자 목록 조회
// 내 uid 로 나를 팔로우하는 개별 사용자 목록 조회 (페이지 단위), 사용자 정보(닉네임, 나이, 젠더, 프로필사진, 소개글, 지역, 등록시간, 영상통화 가능여부)
// 내가 즐겨찾기 한 사용자 목록 조회 (페이지 단위), 사용자 정보(닉네임, 나이, 젠더, 프로필사진, 소개글, 지역, 등록시간, 영상통화 가능여부)
*/

// GetFavoriteList
// @Summary 즐겨찾기한 사용자 목록 조회
// @Description 내가 즐겨찾기한 사용자 목록을 페이지 단위로 조회합니다. (페이지네이션 지원)
//
// @Tags Favorite
// @Security BearerAuth
// @Produce json
//
// @Param page query int false "페이지 번호 (기본값 1)"
// @Param limit query int false "페이지당 항목 수 (기본값 10)"
// @Success 200 {object} map[string]interface{} "result, msg, count, list"
// @Description 요청 예시: GET /fav/v01/list?page=1&limit=10 (Authorization: Bearer)
// @Description 응답 예시: {\"result\":0,\"msg\":\"Success\",\"count\":35,\"list\":[...]}
// @Failure 400 {object} object "잘못된 요청(파라미터 에러) 또는 사용자 정보 없음 예시: {\"result\":400, \"msg\":\"bad request\"}"
// @Failure 401 {object} object "인증 실패 예시: {\"result\":401, \"msg\":\"unauthorized\"}"
// @Failure 500 {object} object "서버 오류 예시: {\"result\":500, \"msg\":\"internal server error\"}"
//
// @Router /fav/v01/list [get]
func (p *AccountController) GetFavoriteList(c *gin.Context) {
	pageInt := ptc.CvtParamAtoi(c.Query("page"), 1)   //defaultQuery "1"
	limitInt := ptc.CvtParamAtoi(c.Query("limit"), 4) //defaultQuery "10"

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	list, count, err := p.adb.GetFavoriteUsers(uid64, pageInt, limitInt)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get favorite list"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
		"list":   list,
	})
}

// 해당 사용자를 팔로우 하는 사용자 수

// GetFavoriteCount
// @Summary 즐겨찾기 한 사용자 수 조회
// @Description 해당 사용자를 즐겨찾기 한 사용자의 총 개수를 반환합니다.
// @Tags Favorite
// @Produce json
// @Success 200 {object} object "성공 예시: {\"result\":0, \"msg\":\"Success\", \"count\":5}"
// @Failure 400 {object} object "잘못된 요청(파라미터 에러) 예시: {\"result\":400, \"msg\":\"bad request\"}"
// @Failure 401 {object} object "인증 실패 예시: {\"result\":401, \"msg\":\"unauthorized\"}"
// @Failure 500 {object} object "서버 오류 예시: {\"result\":500, \"msg\":\"internal server error\"}"
// @Router /fav/v01/count [get]
// @Description 요청 예시: GET /fav/v01/count (Authorization: Bearer)
// @Description 응답 예시: {\"result\":0,\"msg\":\"Success\",\"count\":5}
func (p *AccountController) GetFavoriteCount(c *gin.Context) {
	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	count, err := p.adb.CountFavoriteUsers(uid64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get favorite count"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
	})
}

// --------------------- user block ---------------------
// 내가(현재 유저) - 해당 사용자를 차단

// SetBlockUser godoc
// @Summary 사용자 차단하기 (Block a User)
// @Description 현재 로그인한 사용자가 대상 사용자를 차단합니다. reason 값이 없으면 "etc"로 처리됩니다.
// @Tags Account
// @Accept json
// @Produce json
// @Param tid path string true "차단할 대상 사용자 ID (target user id)"
// @Param reason body object true "차단 사유 JSON {\"reason\":\"욕설\"}"
// @Success 200 {object} object "성공 예시: {\"result\":0, \"msg\":\"Success\", \"count\":5}"
// @Failure 400 {object} object "잘못된 요청(파라미터/바디 오류) 예시: {\"result\":400, \"msg\":\"bad request\"}"
// @Failure 401 {object} object "인증 실패 예시: {\"result\":401, \"msg\":\"unauthorized\"}"
// @Failure 500 {object} object "서버 오류 예시: {\"result\":500, \"msg\":\"internal server error\"}"
// @Router /block/v01/add/{tid} [post]
// @Description 요청 예시: POST /block/v01/add/125 Body {\"reason\":\"욕설\"}
// @Description 응답 예시: {\"result\":0,\"msg\":\"Success\",\"count\":5}
func (p *AccountController) SetBlockUser(c *gin.Context) {
	tid := c.Param("tid")
	if tid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "uid is required")
		return
	}

	tid64, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	var req struct {
		Rsn string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	reason := req.Rsn
	// Validate the required fields
	if req.Rsn == "" {
		reason = "etc"
	}

	/* 	reason := c.DefaultQuery("reason", "")
	   	if reason == "" {
	   		reason = "etc"
	   	}
	*/
	_, err = p.adb.SetBlockUser(uid64, tid64, reason)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to set block user"), http.StatusInternalServerError, err)
		return
	}

	count, err := p.adb.CountBlockUsers(uid64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get block count"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
	})
}

// 해당 사용자를 차단 해제

// UnblockUser godoc
// @Summary 차단 해제 (Unblock User)
// @Description 지정된 상대방(tid)에 대한 차단을 해제합니다. 성공 시 unblock 후 현재 차단 유저수(count)를 반환합니다.
// @Tags Account
// @Accept  json
// @Produce  json
// @Param tid path string true "차단 해제 대상 유저 ID (Target User ID)"
// @Success 200 {object} map[string]interface{} "차단 해제 성공 - result=0, msg=Success, count=차단 유저수"
// @Failure 400 {object} protocol.RespHeader "잘못된 요청 (tid/uid 변환 오류)"
// @Failure 401 {object} protocol.RespHeader "인증 실패 (user not found in context)"
// @Failure 500 {object} protocol.RespHeader "서버 에러 (차단 해제 실패 또는 블록 카운트 조회 실패)"
// @Router /account/v01/unblock/{tid} [post]
// @Security Bearer
// @Description 요청 예시: POST /account/v01/unblock/7
// @Description 응답 예시: {"result":0,"msg":"Success","count":2}
func (p *AccountController) UnblockUser(c *gin.Context) {
	tid := c.Param("tid")
	if tid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "uid is required")
		return
	}

	tid64, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	_, err = p.adb.SetUnblock(uid64, tid64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to unblock user"), http.StatusInternalServerError, err)
		return
	}

	count, err := p.adb.CountBlockUsers(uid64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get block count"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
	})
}

// @Summary Get blocked user list
// @Description Retrieves the list of users blocked by the authenticated user, with paging.
// @Tags user
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of results per page" default(10)
// @Success 200 {object} map[string]interface{} "list: blocked user list, count: 총 차단 인원"
// @Failure 400 {object} protocol.RespHeader "파라미터 오류 혹은 파싱 에러"
// @Failure 401 {object} protocol.RespHeader "인증 실패 (user not found in context)"
// @Failure 500 {object} protocol.RespHeader "서버 에러 (차단 리스트 조회 실패)"
// @Router /block/v01/list/{page}/{limit} [get]
// @Security Bearer
// @Description 요청 예시: GET /block/v01/list/1/10
// @Description 응답 예시: {\"result\":0,\"msg\":\"Success\",\"count\":2,\"list\":[...]}
func (p *AccountController) GetBlockUser(c *gin.Context) {
	pageInt := ptc.CvtParamAtoi(c.Query("page"), 1)   //defaultQuery "1"
	limitInt := ptc.CvtParamAtoi(c.Query("limit"), 4) //defaultQuery "10"

	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}
	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	list, count, err := p.adb.GetBlockList(uid64, pageInt, limitInt)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get block list"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
		"list":   list,
	})
}

// GetBlockCount godoc
// @Summary      Get blocked user count
// @Description  Returns the total number of users blocked by the authenticated user.
// @Tags         account
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{} "Blocked user count"
// @Failure      401  {object}  map[string]interface{} "user not found in context"
// @Failure      400  {object}  map[string]interface{} "failed to parse uid"
// @Failure      500  {object}  map[string]interface{} "failed to get block count"
// @Router       /account/block/count [get]
// @Param        Authorization header string true "Bearer Token"
// @Description 요청 예시: GET /account/block/count (Authorization: Bearer)
// @Description 응답 예시: {\"result\":0,\"msg\":\"Success\",\"count\":5}
func (p *AccountController) GetBlockCount(c *gin.Context) {
	uid, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}
	uid64, err := strconv.ParseUint(uid.(string), 10, 64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to parse uid"), http.StatusBadRequest, err)
		return
	}

	count, err := p.adb.CountBlockUsers(uid64)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get block count"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, gin.H{
		"result": 0,
		"msg":    "Success",
		"count":  count,
	})
}
