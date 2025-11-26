package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	"ms-gateway/models"
	"ms-gateway/protocol"
	ptl "ms-gateway/protocol"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

// swagger:route GET /serv/v01/version get version
// get version
// responses:
//
//	200: {object} protocol.OkResp "version: 0.9.1"
//
// @Summary get version
// @Description get version
func (p *AccountController) GetVersion(c *gin.Context) {
	p.ctl.SimpleRespOK(c, gin.H{"version": "0.9.1"})
}

// swagger:route GET /acc/v01/check/:id check user id
// check user id
// responses:
//
//	200:
//
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

// swagger:route GET /acc/v01/ckemail/:email check user email
// check user email
// responses:
//
//	200:
//
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

// swagger:route POST /acc/v01/regist register user
// register user
// responses: 200:
//
// @Summary Register a new user
// @Description Calls /acc/v01/regist to register a new user. Returns "success" message on success.
// @Tags user
// @Accept json
// @Produce json
// @Param request body protocol.RegistReq true "register request data {id : xxx, pw : hash, name : xxx, gender : 1(men)/0(women), age : xxx, birth : 1990-01-01, area : xxx, email : xxx}"
// @Success 200 {string} string "success"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 104 {object} protocol.RespHeader "Failed Register User"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Router /acc/v01/regist [post]
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
	// var err error
	req.Uid = utils.GenUuid()
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

// swagger:route POST /acc/v01/login login user
// login user
// responses:200:
// @Summary Login a user
// @Description Calls /acc/v01/login to login a user. Returns JWT token, user ID, and WebRTC config on success.
// @Tags user
// @Accept json
// @Produce json
// @Param data body protocol.LoginReq true "Login request data {id : xxx, pw : hash}"
// @Success 200 {object} protocol.LoginUserResp "Login success with token, uid, and webrtc config"
// @Failure 400 {object} protocol.RespHeader "Bad request"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 104 {object} protocol.RespHeader "Failed to login user"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /acc/v01/login [post]
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

	/* 	hsedPw, err := bcrypt.GenerateFromPassword([]byte(req.PW), 11)
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to hash password")
	   		return
	   	}
	*/
	// Register the user
	user, err := p.adb.LoginUser(req, []byte(strings.TrimSpace(req.PW)))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to login user"), http.StatusBadRequest, err)
		return
	}

	acTok, refTok, err := p.genLoginUserToken(user)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLoginFailed, "failed to generate login user token"), http.StatusBadRequest, err)
		return
	}

	mStr := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s/%s/%s",
		user.Uid, user.ID, user.Did, user.Nick, user.Gender, user.Age, user.Area, user.Email, user.MainPic, user.ThumbPic, user.SPIntro)
	// 	UID:             parts[0],
	// 	SID:		     parts[1],
	// 	DID:		     parts[2],
	// 	NICK: 		     parts[3],
	// 	GENDER: 	     parts[4],
	// 	AGE: 		     parts[5],
	// 	AREA: 		     parts[6],
	// 	EMAIL: 		     parts[7],
	// 	MAIN_PIC: 		 parts[8],
	// 	THMB_PIC: 		 parts[9],
	// 	SP_INTRO:        parts[10],

	// x-meta header에 저장
	c.Header("x-meta", mStr)

	// WebRTC 설정 정보 조회
	webrtcConfig, err := p.rdb.GetWebRTCConfig(user.Uid)
	if err != nil {
		log.Warn("Failed to get WebRTC config:", err)
		// WebRTC 설정 조회 실패는 로그인 실패로 처리하지 않음
		webrtcConfig = nil
	}

	// WebRTC 세션 초기화
	webrtcSession := models.UserWebRTCSession{
		UserID:          user.Uid,
		SessionID:       utils.GenUuid(),
		IsInCall:        false,
		ConnectionState: "connected",
		LastHeartbeat:   time.Now(),
		DeviceInfo:      c.GetHeader("User-Agent"),
		NetworkInfo:     c.ClientIP(),
	}

	err = p.rdb.SetUserWebRTCSession(webrtcSession)
	if err != nil {
		log.Warn("Failed to set WebRTC session:", err)
	}

	responseData := ptl.LoginUserResp{
		Message:      "success",
		AccessToken:  acTok,
		RefreshToken: refTok,
		UID:          user.Uid,
		WebRTCConfig: *webrtcConfig,
	}

	// p.ctl.SendDataResponse(c, http.StatusOK, responseData)
	p.ctl.RespSuccess(c, responseData)
}

func (p *AccountController) genLoginUserToken(user *ptl.UserInfoResp) (string, string, error) {
	claims := utils.JWTClaims{
		UserID: user.Uid,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cupitok.com",
			Subject:   "Authentication",
			Audience:  jwt.ClaimStrings{user.Uid},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(), // JTI - 토큰 고유 식별자
		},
	}
	// secret := "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"
	acTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, &claims, 24*time.Hour)
	if err != nil {
		log.Warn("Failed to create JWT token:", err)
		return "", "", err
	}

	refTok, err := utils.CreateJWTToken(p.cfg.Server.JWTSecret, &claims, 14*24*time.Hour)
	if err != nil {
		log.Warn("Failed to create JWT token:", err)
		return "", "", err
	}

	userInfo, err := json.Marshal(user)
	if err != nil {
		log.Warn("Failed to marshal user info:", err)
		return "", "", err
	}

	err = p.rdb.HSetJWTRefresh(refTok, string(userInfo))
	if err != nil {
		log.Warn("Failed to set JWT refresh token:", err)
		return "", "", err
	}

	err = p.rdb.HSetJWTAccess(acTok, string(userInfo))
	if err != nil {
		log.Warn("Failed to set JWT access token:", err)
		return "", "", err
	}

	return acTok, refTok, nil
}

//todo
/*
logout
1. delete jwt
2. last update
3. delete chat room
*/

// swagger:route POST /acc/v01/logout logout user
// @Summary Logout a user
// @Description Performs logout. Returns "success" message on success.
// @Tags user
// @Accept json
// @Produce json
// @Param id body string true "ID of the user to logout"
// @Success 200 {string} string "success"
// @Failure 400 {object} protocol.RespHeader
// @Router /acc/v01/logout [post]
func (p *AccountController) LogoutUser(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusInternalServerError, "%s", err.Error())
		return
	}

	if req.ID == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "id is required")
		return
	}

	// Authorization 헤더에서 토큰 추출
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		tokens := strings.Split(authHeader, " ")
		if len(tokens) == 2 {
			// 현재 토큰만 삭제
			err := p.rdb.DeleteJWTToken(tokens[1])
			if err != nil {
				log.Warn("Failed to delete JWT token:", err)
			}
		}
	}

	err := p.adb.LogoutUser(req.ID)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserLogoutFailed, "failed to logout user"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// swagger:route POST /acc/v01/leave leave user
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
// @Router /acc/v01/leave [post]
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

// 아이디 찾기, 여러개 일수 있음
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

// 비번 변경전 확인 절차, 이후 이메일 인증 번호 전송
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

// otp key 발급, 레디스 저장, 이메일 주소 전달, 이메일

// ReqAuthOTP godoc
// @Summary Request OTP authentication code
// @Description Generate and send OTP code to user's email for authentication
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "Encrypted data by aes256 {email : xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing or invalid"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 116 {object} protocol.RespHeader "Failed to set OTP"
// @Router /acc/v01/reqotp [post]
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

// email, otp 입력값을 받아 레디스에서 조회, 검증 확인
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

// ChangePW godoc
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

// ModifyUserInfo godoc
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
// @Router /acc/v01/modify [post]
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
	err := p.adb.ModifyUserInfo(req.ID, req.Uid, req.Email, req.Cate, req.Value)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserChangePWFailed, "failed to change pw"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// GetUserInfo godoc
// @Summary Get user information
// @Description Get user information
// @Tags user
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} protocol.UserInfoResp "User information"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 114 {object} protocol.RespHeader "Failed to get user info"
// @Router /acc/v01/info/{id} [get]
func (p *AccountController) GetUserInfo(c *gin.Context) {
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

// DeleteUser godoc
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
// @Router /acc/v01/delete/{id} [post]
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

// ModifyMainPic godoc
// @Summary Modify main picture
// @Description Modify main picture
// @Tags user
// @Accept json
// @Produce json
// @Param data body object true "User ID {uid: xxx}"
// @Success 200 {object} protocol.OkResp "msg: success"
// @Failure 400 {object} protocol.RespHeader "error: error message"
// @Failure 102 {object} protocol.RespHeader "Parameter is missing"
// @Failure 106 {object} protocol.RespHeader "Failed to parse JSON"
// @Failure 115 {object} protocol.RespHeader "Failed to delete user"
func (p *AccountController) ModifyMainPic(c *gin.Context) {
	var req struct {
		UID string `json:"uid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	if req.UID == "" {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "all fields are required"), http.StatusBadRequest, fmt.Errorf("all fields are required"))
		return
	}

	fileInfo, ok := c.Get("uploadedFiles")
	if !ok {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "invalid request body"), http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	files := fileInfo.([]*multipart.FileHeader)
	putInfo := make(map[string]map[string]string)

	fmt.Println(files)
	fmt.Println(putInfo)

	// Upload the file to S3

	p.ctl.SendResponse(c, http.StatusOK, "success")
	return
	/* 	eaddr := c.MustGet("user").(string)

	   	fileInfo, ok := c.Get("uploadedFiles")
	   	if !ok {
	   		p.ctl.RespError(c, ptl.NewRespHeader(ptl.InvalidParam, "invalid request body"), http.StatusBadRequest, fmt.Errorf("invalid request body"))
	   		return
	   	}

	   	files := fileInfo.([]*multipart.FileHeader)
	   	putInfo := make(map[string]map[string]string)
	   	for _, file := range files {
	   		putInfo[file.Filename] = map[string]string{
	   			"bucket": p.cfg.AWS.S3.BucketName,
	   			"key":    fmt.Sprintf("c2c/user/%s/%s", "profile", file.Filename),
	   		}
	   	}

	   	// Upload the file to S3
	   	uploadResp, err := utils.FilesS3Uploader(files, putInfo, p.s3)
	   	if err != nil {
	   		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to upload file"), http.StatusInternalServerError, err)
	   		return
	   	}

	   	// Update the profile image in the database
	   	err = p.accountDB.UpdateUserProfileImage(eaddr, uploadResp[0])
	   	if err != nil {
	   		p.ctl.RespError(c, http.StatusInternalServerError, ptl.Wrap(err))
	   		return
	   	}
	*/
}

// @Summary Get WebRTC configuration
// @Description Get WebRTC configuration for the authenticated user
// @Tags webrtc
// @Accept json
// @Produce json
// @Success 200 {object} protocol.WebRTCConfig "WebRTC configuration"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Router /webrtc/v01/config [get]
func (p *AccountController) GetWebRTCConfig(c *gin.Context) {
	// JWT에서 사용자 ID 추출
	userID, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	config, err := p.rdb.GetWebRTCConfig(userID.(string))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get WebRTC config"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, config)
}

// @Summary Get available users for call
// @Description Get list of users available for video call
// @Tags webrtc
// @Accept json
// @Produce json
// @Success 200 {object} []string "Available user IDs"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Router /webrtc/v01/available-users [get]
func (p *AccountController) GetAvailableUsers(c *gin.Context) {
	// JWT에서 사용자 ID 추출
	userID, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	users, err := p.rdb.GetAvailableUsersForCall(userID.(string))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get available users"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, users)
}

// @Summary Update call status
// @Description Update user's call status
// @Tags webrtc
// @Accept json
// @Produce json
// @Param data body object true "Call status update {isInCall: bool, callWith: string}"
// @Success 200 {object} protocol.OkResp "Status updated"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Router /webrtc/v01/call-status [post]
func (p *AccountController) UpdateCallStatus(c *gin.Context) {
	// JWT에서 사용자 ID 추출
	userID, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	var req struct {
		IsInCall bool   `json:"isInCall" binding:"required"`
		CallWith string `json:"callWith"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	err := p.rdb.UpdateUserCallStatus(userID.(string), req.IsInCall, req.CallWith)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to update call status"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "call status updated"})
}

// @Summary Update WebRTC session heartbeat
// @Description Update WebRTC session heartbeat to keep session alive
// @Tags webrtc
// @Accept json
// @Produce json
// @Param data body object true "Session update {deviceInfo: string, networkInfo: string}"
// @Success 200 {object} protocol.OkResp "Heartbeat updated"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 401 {object} protocol.RespHeader "Unauthorized"
// @Router /webrtc/v01/heartbeat [post]
func (p *AccountController) UpdateWebRTCHeartbeat(c *gin.Context) {
	// JWT에서 사용자 ID 추출
	userID, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	var req struct {
		DeviceInfo  string `json:"deviceInfo"`
		NetworkInfo string `json:"networkInfo"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.JsonParseFailed, "failed to parse JSON"), http.StatusBadRequest, err)
		return
	}

	// 기존 세션 조회 후 업데이트
	session, err := p.rdb.GetUserWebRTCSession(userID.(string))
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "session not found"), http.StatusNotFound, err)
		return
	}

	// 하트비트 업데이트
	session.LastHeartbeat = time.Now()
	if req.DeviceInfo != "" {
		session.DeviceInfo = req.DeviceInfo
	}
	if req.NetworkInfo != "" {
		session.NetworkInfo = req.NetworkInfo
	}

	err = p.rdb.SetUserWebRTCSession(*session)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to update session"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "heartbeat updated"})
}

// @Summary Get WebRTC statistics
// @Description Get WebRTC system statistics
// @Tags webrtc
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "WebRTC statistics"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /webrtc/v01/stats [get]
func (p *AccountController) GetWebRTCStats(c *gin.Context) {
	stats, err := p.rdb.GetWebRTCStats()
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to get WebRTC stats"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, stats)
}

// @Summary Test STUN server connectivity
// @Description Test STUN server connectivity and get public IP
// @Tags webrtc
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "STUN test results"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /webrtc/v01/test-stun [get]
func (p *AccountController) TestStunServers(c *gin.Context) {
	// 기본 STUN 서버들 가져오기
	stunServers := GetRecommendedStunServers()

	// 테스트 보고서 생성
	report, err := CreateStunTestReport(c.Request.Context(), stunServers)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.Failed, "failed to test STUN servers"), http.StatusInternalServerError, err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, report)
}

// @Summary Test specific STUN server
// @Description Test connectivity to a specific STUN server
// @Tags webrtc
// @Accept json
// @Produce json
// @Param data body object true "STUN server URL {stunUrl: string}"
// @Success 200 {object} controller.STUNTestResult "STUN test result"
// @Failure 400 {object} protocol.RespHeader "Invalid request"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /webrtc/v01/test-stun-server [post]
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
