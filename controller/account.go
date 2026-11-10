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
	"ms-gateway/common/version"
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
	sdb *models.StoryDB
}

func NewAccountController(ctl *Controller, rep *models.Repositories) (*AccountController, error) {
	r := &AccountController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.sdb, &r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

// GetVersion godoc
// @Summary      서버 버전 조회
// @Description  서버 버전 조회 API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음.
// @Description
// @Description  **처리**: 빌드 및 실행 중인 Go 버전 정보를 조회합니다.
// @Description
// @Description  **응답**: 최상위 version, go_version, revision, modified 및 선택적 빌드 정보.
// @Tags         Server
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{} "최상위 version, go_version, revision, modified 및 선택적 빌드 정보."
// @Router       /serv/v01/version [get]
// @Example Request: GET /serv/v01/version
func (p *AccountController) GetVersion(c *gin.Context) {
	p.ctl.SimpleRespOK(c, version.Info())
}

// CheckID godoc
// @Summary      아이디 중복 확인
// @Description  아이디 중복 확인 API입니다.
// @Description
// @Description  **요청**: path id: 확인할 값.
// @Description
// @Description  **처리**: 사용 가능하면 msg=ok입니다. **현재 제한**: 중복 시 비표준 HTTP 상태 14를 전달하여 net/http 응답 처리에 실패할 수 있습니다. 409 응답으로 구현되어 있지 않습니다.
// @Description
// @Description  **응답**: 사용 가능할 때 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        id path string true "계정 아이디"
// @Success      200 {object} map[string]interface{} "사용 가능할 때 최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      409 {object} protocol.RespHeader "이미 사용 중인 ID"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /acc/v01/check/{id} [get]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: GET /acc/v01/check/example
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

// CheckEmail godoc
// @Summary      이메일 중복 확인
// @Description  이메일 중복 확인 API입니다.
// @Description
// @Description  **요청**: path email: 확인할 값.
// @Description
// @Description  **처리**: 사용 가능하면 msg=ok입니다. **현재 제한**: 중복 시 비표준 HTTP 상태 15를 전달하여 net/http 응답 처리에 실패할 수 있습니다. 409 응답으로 구현되어 있지 않습니다.
// @Description
// @Description  **응답**: 사용 가능할 때 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        email path string true "이메일"
// @Success      200 {object} map[string]interface{} "사용 가능할 때 최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /acc/v01/ckemail/{email} [get]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: GET /acc/v01/ckemail/user@example.com
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

// RegistUserInfo godoc
// @Summary      회원 가입
// @Description  회원 가입 API입니다.
// @Description
// @Description  **요청**: 실제 body는 AesDataForm: {data: AES-GCM 암호문}. 복호화 후 RegistReq의 id/pw/area/name/birth/email 등 필드를 검증합니다.
// @Description
// @Description  **처리**: AesDecrypt 미들웨어가 JSON을 복호화합니다. 서버 UID·닉네임 생성, 비밀번호 해시와 초기 정보를 저장합니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body protocol.AesDataForm true "실제 body는 AesDataForm: {data: AES-GCM 암호문}. 복호화 후 RegistReq의 id/pw/area/name/birth/email 등 필드를 검증합니다."
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /acc/v01/regist [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /acc/v01/regist
// @Example Request Body: {"data":"<AES-GCM encrypted RegistReq>"}
// @x-request-example {"data":"<AES-GCM encrypted RegistReq>"}
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

	err = p.sdb.SaveInitPrfInfo(req)
	if err != nil {
		p.ctl.RespError(c, ptl.NewRespHeader(ptl.UserRegistFailed, "failed to save profile info"), http.StatusBadRequest, err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
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

// LoginUser godoc
// @Summary      로그인
// @Description  로그인 API입니다.
// @Description
// @Description  **요청**: 실제 body는 AesDataForm: {data: AES-GCM 암호문}. 복호화된 LoginReq는 id, pw가 필요합니다.
// @Description
// @Description  **처리**: 로그인 검증 후 JWT 발급·Redis 저장, WebRTC 설정과 대기실 정보를 구성합니다.
// @Description
// @Description  **응답 data**: LoginUserResp (msg, acTok, refTok, uid, meta, wrtc).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body protocol.AesDataForm true "실제 body는 AesDataForm: {data: AES-GCM 암호문}. 복호화된 LoginReq는 id, pw가 필요합니다."
// @Success      200 {object} protocol.RespDataHeader{data=protocol.LoginUserResp} "data: LoginUserResp (msg, acTok, refTok, uid, meta, wrtc)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Router       /acc/v01/login [post]
// @Example Request: POST /acc/v01/login
// @Example Request Body: {"data":"<AES-GCM encrypted LoginReq>"}
// @x-request-example {"data":"<AES-GCM encrypted LoginReq>"}
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
		Message: "ok",
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

// RefreshToken godoc
// @Summary      인증 토큰 재발급
// @Description  인증 토큰 재발급 API입니다.
// @Description
// @Description  **요청**: Authorization: Bearer refresh_token 또는 body RefreshTokenReq의 refTok. 유효한 형식의 헤더 토큰이 우선합니다.
// @Description
// @Description  **처리**: refresh JWT와 Redis 저장값 검증 후 access/refresh 토큰을 재발급합니다.
// @Description
// @Description  **응답 data**: RefreshTokenResp (msg, acTok, refTok, uid).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        Authorization header string false "Bearer refresh_token (body보다 우선)"
// @Param        request body protocol.RefreshTokenReq false "Authorization: Bearer refresh_token 또는 body RefreshTokenReq의 refTok. 유효한 형식의 헤더 토큰이 우선합니다."
// @Success      200 {object} protocol.RespDataHeader{data=protocol.RefreshTokenResp} "data: RefreshTokenResp (msg, acTok, refTok, uid)."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /acc/v01/refresh [post]
// @Example Request: POST /acc/v01/refresh
// @Example Request Body: {"refTok":"<refresh_token>"}
// @x-request-example {"refTok":"<refresh_token>"}
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
		Message:      "ok",
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

// LogoutUser godoc
// @Summary      로그아웃
// @Description  로그아웃 API입니다.
// @Description
// @Description  **요청**: JWT Authorization 헤더, body 없음.
// @Description
// @Description  **처리**: JWT 저장값과 대기실 정보를 삭제하고 로그아웃 상태를 갱신합니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /inserv/v01/logout [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /inserv/v01/logout
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// LeaveUser godoc
// @Summary      회원 탈퇴
// @Description  회원 탈퇴 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: id (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: body id로 탈퇴를 처리합니다. 토큰 삭제는 현재 구현되어 있지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body object{id=string} true "필수 문자열 필드: id"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Router       /inserv/v01/leave [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /inserv/v01/leave
// @Example Request Body: {"id":"example"}
// @x-request-example {"id":"example"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// FindID godoc
// @Summary      아이디 찾기
// @Description  아이디 찾기 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: name, birth, gender (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: 입력 정보로 계정을 조회합니다.
// @Description
// @Description  **응답 data**: ids.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body object{name=string,birth=string,gender=string} true "필수 문자열 필드: name, birth, gender"
// @Success      200 {object} protocol.RespDataHeader{data=[]string} "아이디 문자열 배열."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Router       /acc/v01/fnid [post]
// @Example Response: {"result":0,"resultString":"Success","data":[]}
// @x-response-example {"result":0,"resultString":"Success","data":[]}
// @Example Request: POST /acc/v01/fnid
// @Example Request Body: {"name":"홍길동","birth":"1990-01-01","gender":"1"}
// @x-request-example {"name":"홍길동","birth":"1990-01-01","gender":"1"}
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

// FindPW godoc
// @Summary      비밀번호 찾기
// @Description  비밀번호 찾기 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: sid, name, birth, gender (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: 입력 정보로 이메일을 조회합니다.
// @Description
// @Description  **응답**: 최상위 msg, email.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body object{sid=string,name=string,birth=string,gender=string} true "필수 문자열 필드: sid, name, birth, gender"
// @Success      200 {object} object{msg=string,email=string} "최상위 msg, email."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Router       /acc/v01/fnpw [post]
// @Example Response: {"msg":"ok","email":"user@example.com"}
// @x-response-example {"msg":"ok","email":"user@example.com"}
// @Example Request: POST /acc/v01/fnpw
// @Example Request Body: {"sid":"example","name":"홍길동","birth":"1990-01-01","gender":"1"}
// @x-request-example {"sid":"example","name":"홍길동","birth":"1990-01-01","gender":"1"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// VerifyOTP godoc
// @Summary      OTP 확인
// @Description  OTP 확인 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: email, otp (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: 이메일 OTP를 검증합니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body object{email=string,otp=string} true "필수 문자열 필드: email, otp"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Router       /acc/v01/verifyotp [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /acc/v01/verifyotp
// @Example Request Body: {"email":"user@example.com","otp":"<otp>"}
// @x-request-example {"email":"user@example.com","otp":"<otp>"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// ChangePW godoc
// @Summary      비밀번호 변경
// @Description  비밀번호 변경 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: id, uid, email, pw, newpw (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: 전달된 계정 정보와 기존 비밀번호로 변경을 처리합니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        request body object{id=string,uid=string,email=string,pw=string,newpw=string} true "필수 문자열 필드: id, uid, email, pw, newpw"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Router       /acc/v01/cngpw [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /acc/v01/cngpw
// @Example Request Body: {"id":"example","uid":"4033287471439576593","email":"user@example.com","pw":"<current-password>","newpw":"<new-password>"}
// @x-request-example {"id":"example","uid":"4033287471439576593","email":"user@example.com","pw":"<current-password>","newpw":"<new-password>"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// ModifyUserInfo godoc
// @Summary      회원 정보 변경
// @Description  회원 정보 변경 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: id, uid, email, cate, value (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: JWT 인증 후 body의 uid와 수정 필드를 전달합니다. 이 핸들러는 uid를 JWT 값으로 덮어쓰지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body object{id=string,uid=string,email=string,cate=string,value=string} true "필수 문자열 필드: id, uid, email, cate, value"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Router       /inserv/v01/modify [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /inserv/v01/modify
// @Example Request Body: {"id":"example","uid":"4033287471439576593","email":"user@example.com","cate":"nick","value":"새 닉네임"}
// @x-request-example {"id":"example","uid":"4033287471439576593","email":"user@example.com","cate":"nick","value":"새 닉네임"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// GetSidUserInfo godoc
// @Summary      아이디로 회원 조회 (현재 응답 제한)
// @Description  아이디로 회원 조회 (현재 응답 제한) API입니다.
// @Description
// @Description  **요청**: path id: 계정 아이디.
// @Description
// @Description  **처리**: 조회 결과 객체를 문자열 전용 SendResponse에 전달하므로 조회 성공 후에도 500을 반환합니다.
// @Description
// @Description  **응답**: 현재 정상 회원 응답 없음. 오류 헤더.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        id path string true "계정 아이디"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /inserv/v01/info/{id} [get]
// @Example Request: GET /inserv/v01/info/example
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

// GetUidFromInfo godoc
// @Summary      UID로 회원 정보 조회
// @Description  UID로 회원 정보 조회 API입니다.
// @Description
// @Description  **요청**: path tid: 대상 UID.
// @Description
// @Description  **처리**: 대상 회원 정보를 조회합니다.
// @Description
// @Description  **응답 data**: UserInfoResp.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=protocol.UserInfoResp} "data: UserInfoResp."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Router       /inserv/v01/uinfo/{tid} [get]
// @Example Request: GET /inserv/v01/uinfo/4033287471439576593
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

	p.ctl.SendDataResponse(c, http.StatusOK, user)
}

// DeleteUser godoc
// @Summary      회원 삭제
// @Description  회원 삭제 API입니다.
// @Description
// @Description  **요청**: 일반 JSON body: id (모두 필수 문자열). AES 래퍼를 사용하지 않습니다.
// @Description
// @Description  **처리**: 실제 삭제 대상은 body id입니다. path id는 사용하지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        id path string true "경로 호환용이며 무시합니다. body id가 삭제 대상입니다."
// @Param        request body object{id=string} true "필수 문자열 필드: id"
// @Success      200 {object} map[string]interface{} "최상위 msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Router       /inserv/v01/delete/{id} [post]
// @Example Response: {"msg":"ok"}
// @x-response-example {"msg":"ok"}
// @Example Request: POST /inserv/v01/delete/example
// @Example Request Body: {"id":"example"}
// @x-request-example {"id":"example"}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok"})
}

// ModifyMainPic godoc
// @Summary      계정 대표 사진 업로드 (현재 연결 제한)
// @Description  계정 대표 사진 업로드 (현재 연결 제한) API입니다.
// @Description
// @Description  **요청**: multipart files: 1개·5MiB 이하.
// @Description
// @Description  **처리**: 현재 미들웨어는 upFiles를 설정하지만 핸들러는 uploadedFiles와 sinfo를 기대합니다. 일반 요청은 No files uploaded(400)로 실패하며 정상 업로드 경로가 연결되지 않았습니다.
// @Description
// @Description  **응답**: 현재 정상 업로드 응답 없음. 오류 헤더.
// @Description  업로드 미들웨어 오류는 JSON 문자열 또는 error 객체일 수 있으며 핸들러의 RespHeader 오류와 형식이 다릅니다.
// @Description  **업로드 공통 제한**: 요청 전체 크기·동시 업로드 제한을 초과하면 413/503입니다. 기본값은 530MiB/8건이며 서버 설정에 따라 달라집니다.
// @Tags         Account
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        files formData file true "업로드 파일 (동일 필드명으로 여러 파일 전송 가능)"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      413 {object} map[string]interface{} "업로드 요청 크기 제한 초과"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Failure      503 {object} map[string]interface{} "서버 동시 처리·연결 한도 초과"
// @Router       /inserv/v01/upd/mpic [post]
// @Example Request: POST /inserv/v01/upd/mpic
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

// GetWebRTCConfig godoc
// @Summary      WebRTC 설정 조회 (현재 라우트 제한)
// @Description  WebRTC 설정 조회 (현재 라우트 제한) API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음. /serv/v01/wsconf와 /webrtc/v01/config는 같은 핸들러입니다.
// @Description
// @Description  **처리**: 현재 두 라우트 모두 JWT 미들웨어가 없어 user 컨텍스트가 설정되지 않으며 401을 반환합니다. 핸들러의 SendResponse도 객체를 허용하지 않아 설정 객체 정상 응답은 구현과 일치하지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 설정 응답 없음. 오류 헤더(result, resultString, desc).
// @Tags         WebRTC
// @Accept       json
// @Produce      json
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /serv/v01/wsconf [get]
// @Router       /webrtc/v01/config [get]
// @Example Request: GET /serv/v01/wsconf
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
// @Summary      즐겨찾기 등록 (현재 인증 컨텍스트 제한)
// @Description  즐겨찾기 등록 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path tid, body 없음.
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /fav/v01/set/{tid} [post]
// @Example Request: POST /fav/v01/set/4033287471439576593
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
		"msg":    "ok",
	})
}

// UnfavoriteUser godoc
// @Summary      즐겨찾기 해제 (현재 인증 컨텍스트 제한)
// @Description  즐겨찾기 해제 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path tid, body 없음.
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /fav/v01/cancel/{tid} [post]
// @Example Request: POST /fav/v01/cancel/4033287471439576593
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
		"msg":    "ok",
	})
}

/*
// 나를(현재 유저) - 팔로우 하는 사용자 목록 조회
// 내 uid 로 나를 팔로우하는 개별 사용자 목록 조회 (페이지 단위), 사용자 정보(닉네임, 나이, 젠더, 프로필사진, 소개글, 지역, 등록시간, 영상통화 가능여부)
// 내가 즐겨찾기 한 사용자 목록 조회 (페이지 단위), 사용자 정보(닉네임, 나이, 젠더, 프로필사진, 소개글, 지역, 등록시간, 영상통화 가능여부)
*/

// GetFavoriteList godoc
// @Summary      즐겨찾기 목록 (현재 인증 컨텍스트 제한)
// @Description  즐겨찾기 목록 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path page/limit은 사용하지 않으며 query page/limit을 사용합니다(기본 1/4).
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다.
// @Description
// @Description  **응답 data**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "경로 호환용이며 무시합니다. query 값을 사용하세요."
// @Param        limit path int true "경로 호환용이며 무시합니다. query 값을 사용하세요."
// @Param        page query int false "실제 조회 페이지" default(1)
// @Param        limit query int false "실제 페이지 크기" default(4)
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /fav/v01/list/{page}/{limit} [get]
// @Example Request: GET /fav/v01/list/1/10
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
		"msg":    "ok",
		"count":  count,
		"list":   list,
	})
}

// 해당 사용자를 팔로우 하는 사용자 수

// GetFavoriteCount godoc
// @Summary      즐겨찾기 수 (현재 인증 컨텍스트 제한)
// @Description  즐겨찾기 수 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음.
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /fav/v01/count [get]
// @Example Request: GET /fav/v01/count
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
		"msg":    "ok",
		"count":  count,
	})
}

// --------------------- user block ---------------------
// 내가(현재 유저) - 해당 사용자를 차단

// SetBlockUser godoc
// @Summary      사용자 차단 (현재 인증 컨텍스트 제한)
// @Description  사용자 차단 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path tid, body reason(필수 문자열).
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Param        request body object{reason=string} true "필수 문자열 필드: reason"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /block/v01/set/{tid} [post]
// @Example Request: POST /block/v01/set/4033287471439576593
// @Example Request Body: {"reason":"욕설"}
// @x-request-example {"reason":"욕설"}
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
		"msg":    "ok",
		"count":  count,
	})
}

// 해당 사용자를 차단 해제

// UnblockUser godoc
// @Summary      사용자 차단 해제 (현재 인증 컨텍스트 제한)
// @Description  사용자 차단 해제 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path tid, body 없음.
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /block/v01/cancel/{tid} [post]
// @Example Request: POST /block/v01/cancel/4033287471439576593
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
		"msg":    "ok",
		"count":  count,
	})
}

// GetBlockUser godoc
// @Summary      차단 사용자 목록 (현재 인증 컨텍스트 제한)
// @Description  차단 사용자 목록 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: path page/limit은 사용하지 않으며 query page/limit을 사용합니다(기본 1/4).
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다.
// @Description
// @Description  **응답 data**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "경로 호환용이며 무시합니다. query 값을 사용하세요."
// @Param        limit path int true "경로 호환용이며 무시합니다. query 값을 사용하세요."
// @Param        page query int false "실제 조회 페이지" default(1)
// @Param        limit query int false "실제 페이지 크기" default(4)
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /block/v01/list/{page}/{limit} [get]
// @Example Request: GET /block/v01/list/1/10
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
		"msg":    "ok",
		"count":  count,
		"list":   list,
	})
}

// GetBlockCount godoc
// @Summary      차단 사용자 수 (현재 인증 컨텍스트 제한)
// @Description  차단 사용자 수 (현재 인증 컨텍스트 제한) API입니다.
// @Description
// @Description  **요청**: 별도 입력 없음.
// @Description
// @Description  **처리**: JWT 미들웨어는 *UserInfoResp를 저장하지만 현재 핸들러는 user를 string으로 단언하여 정상 요청에서도 panic/Recovery 500이 발생합니다. 이후 SendResponse에 객체를 전달하는 응답 경로도 문자열 전용 도우미와 맞지 않습니다.
// @Description
// @Description  **응답**: 현재 정상 응답 없음. Recovery 오류는 JSON 헤더 형식이 보장되지 않습니다.
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {string} string "내부 처리 실패"
// @Router       /block/v01/count [get]
// @Example Request: GET /block/v01/count
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
		"msg":    "ok",
		"count":  count,
	})
}
