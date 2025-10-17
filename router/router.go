package router

import (
	"encoding/base32"

	"ms-gateway/common/logger"
	"ms-gateway/conf"
	"ms-gateway/models"

	ctl "ms-gateway/controller"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"

	"ms-gateway/docs"

	ginSwg "github.com/swaggo/gin-swagger"

	swgFiles "github.com/swaggo/files"
)

type Router struct {
	cfg *conf.Config
	wl  map[string]string
	ctl *ctl.Controller
	acc *ctl.AccountController
	pf  *ctl.ProfileController
	hm  *ctl.HomeController
	nt  *ctl.NotiController
	// chat *ctl.ChatController
	rdb *models.RedisDB
	// hHealth *ctl.Health
}

func NewRouter(cf *conf.Config, ct *ctl.Controller) (*Router, error) {
	r := &Router{
		cfg: cf,
		wl:  convertWhiteList(cf.WhiteList.Ips),
		ctl: ct,
		acc: ct.AccCtl,
		pf:  ct.PfCtl,
		hm:  ct.HomeCtl,
		nt:  ct.NotiCtl,
		// chat: ct.ChatCtl,
		rdb: ct.GetRedis(),
		// hHealth: ct.GetHealthHandler(),
	}

	return r, nil
}

func convertWhiteList(wl []string) map[string]string {
	converted := make(map[string]string)
	for _, v := range wl {
		converted[v] = v
	}

	return converted
}

/*
	 func CORS() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-Forwarded-For, Authorization, accept, origin, Cache-Control, X-Requested-With, OTP-Auth")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
		}
	}
*/
func validateOTP(otp string) bool {
	secret := "123456"
	if otp == "" {
		return false
	}

	encodedSecret := base32.StdEncoding.EncodeToString([]byte(secret))
	if !totp.Validate(otp, encodedSecret) {
		return false
	}

	return true
}

func liteAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil {
			c.Abort()
			return
		}

		auth := c.GetHeader("X-Totp")
		// if !validateOTP(auth) {
		// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		// 	return
		// }

		logger.Info("auth : ", auth)
		c.Next()
	}
}

func (p *Router) otpAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil {
			c.Abort()
			return
		}

		// Check if the request IP is in the allowed IP list from the config
		requestIP := c.ClientIP()
		if _, ok := p.wl[requestIP]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "IP not allowed"})
			return
		}

		auth := c.GetHeader("X-Otp")
		// if !validateOTP(auth) {
		// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		// 	return
		// }
		logger.Info("auth : ", auth)
		c.Next()
	}
}

func (p *Router) Idx() *gin.Engine {
	e := gin.Default()
	e.Use(logger.GinLogger())
	e.Use(logger.GinRecovery(true))
	e.Use(CORS())

	if p.cfg.Server.Mode == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info("start server : ", p.cfg.Server.Port)

	e.GET("/swagger/:any", ginSwg.WrapHandler(swgFiles.Handler))
	docs.SwaggerInfo.Host = "localhost"

	e.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	//metadata : uid/성별/지역/나이/did/로그인타입(01234)/

	server := e.Group("serv/v01", p.SecurityHeaders())
	{
		server.GET("/version", p.acc.GetVersion)
		server.GET("/wsconf", p.acc.GetWebRTCConfig)
	}
	// 회원가입
	account := e.Group("acc/v01", p.SecurityHeaders())
	{
		// 아이디 중복 체크
		account.GET("/check/:id", p.acc.CheckID)

		// 이메일 중복 체크
		account.GET("/ckemail/:email", p.acc.CheckEmail)

		// 회원가입
		// account.POST("/regist/:id/:pw/:area/:name/:age/:birth/:gender")
		account.POST("/regist", p.AesDecrypt(), p.acc.RegistUserInfo)

		// 로그인
		account.POST("/login", p.AesDecrypt(), p.acc.LoginUser)

		// 아이디 찾기 - pass 연동 필요 /:name/:birth
		account.POST("/fnid", p.acc.FindID)

		// 비밀번호 찾기 - pass 연동 필요 /:id/:birth
		account.POST("/fnpw", p.acc.FindPW)

		// 비밀번호 변경
		account.POST("/cngpw", p.acc.ChangePW)

		account.POST("/verifyotp", p.acc.VerifyOTP)

	}

	// login 후 요청
	pfset := e.Group("inserv/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// 회원 정보 수정 :id/:pw/:area/:name/:birth/:gender/:terms
		//cate : pw / area / nick / email
		pfset.POST("/modify", p.acc.ModifyUserInfo)

		pfset.POST("/upd/mpic", p.AesDecrypt(), p.acc.ModifyMainPic)

		// 로그아웃
		pfset.POST("/logout", p.acc.LogoutUser)

		// 회원 탈퇴
		pfset.POST("/leave", p.acc.LeaveUser)
		// 회원 정보 조회
		//todo : 토큰 확인 필요
		pfset.GET("/info/:id", p.acc.GetUserInfo)

		// 회원 정보 삭제
		//todo : 토큰 확인 필요
		pfset.POST("/delete/:id", p.acc.DeleteUser)
	}

	user := e.Group("user/v01", p.SecurityHeaders(), liteAuth())
	{
		//보유금액, 즐겨찾기, 통화내역
		user.GET("/myinfo")
		user.GET("/wyinfo")
	}

	home := e.Group("home/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// 홈 data
		home.GET("/mdata", p.hm.GetHomeMenInfo)
		home.GET("/wdata")
	}

	noti := e.Group("noti/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// noti 알림
		noti.GET("/list", p.nt.GetNotiList)
		noti.GET("/detail/:idx", p.nt.GetNotiDetail)

		// 공지사항
		noti.GET("/anc/list", p.nt.GetAnnouncement)
		noti.GET("/anc/detail/:idx", p.nt.GetAnnouncementDetail)
	}

	chat := e.Group("chat/v01", p.SecurityHeaders())
	{
		chat.GET("/mlistvd")
		home.GET("/wlistvd")

		chat.GET("/mlistvo")
		home.GET("/wlistvo")
	}

	inbox := e.Group("inbox/v01", p.SecurityHeaders())
	{
		inbox.GET("/list")
	}

	present := e.Group("present/v01", p.SecurityHeaders(), liteAuth())
	{
		present.PUT("/target")
		// present.GET("/wavlist", )
	}

	mission := e.Group("mission/v01", p.SecurityHeaders(), liteAuth())
	{
		mission.GET("/list")
		mission.GET("/detail/:id")
	}

	story := e.Group("story/v01", p.SecurityHeaders(), liteAuth())
	{
		// 좋아요 카운트, 팔로워, 조회수, 팔로잉?, 신고카운트
		story.GET("/home/:id")
		story.GET("/list/:stat/:id")
		story.GET("/detail/:idx")
	}

	//누드, 음모 확인 기능
	upload := e.Group("upload/v01", p.SecurityHeaders(), liteAuth())
	{
		upload.POST("/story/img")
		upload.POST("/present")
	}

	setting := e.Group("setting/v01", p.SecurityHeaders(), liteAuth())
	{
		setting.GET("/alert")
	}

	// WebRTC 관련 엔드포인트 추가
	webrtc := e.Group("webrtc/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// WebRTC 설정 정보 조회
		webrtc.GET("/config", p.acc.GetWebRTCConfig)

		// 통화 가능한 사용자 목록 조회
		webrtc.GET("/available-users", p.acc.GetAvailableUsers)

		// 통화 상태 업데이트
		webrtc.POST("/call-status", p.acc.UpdateCallStatus)

		// WebRTC 세션 하트비트 업데이트
		webrtc.POST("/heartbeat", p.acc.UpdateWebRTCHeartbeat)

		// WebRTC 통계 조회 (선택적 - 관리자용)
		webrtc.GET("/stats", p.acc.GetWebRTCStats)

		// STUN 서버 연결성 테스트 (인증 없이 접근 가능)
		webrtc.GET("/test-stun", p.acc.TestStunServers)

		// 특정 STUN 서버 테스트
		webrtc.POST("/test-stun-server", p.acc.TestSpecificStunServer)
	}

	/*
		// 채팅 관련 엔드포인트 추가
		chat := e.Group("chat/v01", p.SecurityHeaders())
		{
			// 채팅방 생성
			chat.POST("/room", p.chat.CreateChatRoomHandler)

			// 사용자의 채팅방 목록 조회
			chat.GET("/rooms", p.chat.GetChatRooms)

			// 채팅 기록 조회
			chat.GET("/history/:roomId", p.chat.GetChatHistory)

			// WebSocket 연결 엔드포인트
			chat.GET("/ws", p.chat.HandleWebsocket)

			// P2P 연결 상태 조회
			chat.GET("/p2p/:userId", p.chat.GetActivePeerConnections)
		}

		// WebRTC 시그널링 서버 엔드포인트
		rtc := e.Group("rtc/v01", p.SecurityHeaders())
		{
			// WebSocket 연결 엔드포인트 (시그널링 서버)
			rtc.GET("/signal", p.chat.HandleWebsocket)
		}
	*/
	/*
		e.GET("/health", p.hHealth.Check)

		// e.GET("/swagger/:any", ginSwg.WrapHandler())
		// ginSwagger.WrapHandler(swaggerFiles.Handler,
		// 	ginSwagger.URL("http://localhost:8080/swagger/doc.json"),
		// 	ginSwagger.DefaultModelsExpandDepth(-1))

		account := e.Group("acc/v01", liteAuth())
		{
			account.GET("/ok", p.hHealth.Check)
		}

		wd := e.Group("wd/v01", p.otpAuth())
		{
			wd.POST("/req", p.hHealth.Check)
			wd.GET("/myinfo/:id", p.hHealth.Check)
		}
	*/
	return e
}
