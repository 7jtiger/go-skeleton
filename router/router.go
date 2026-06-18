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
	cfg  *conf.Config
	wl   map[string]string
	ctl  *ctl.Controller
	acc  *ctl.AccountController
	set  *ctl.SetController
	pf   *ctl.ProfileController
	hm   *ctl.HomeController
	nt   *ctl.NotiController
	sig  *ctl.SignalingController
	st   *ctl.StoryController
	chat *ctl.ChatController
	rdb  *models.RedisDB
}

func NewRouter(cf *conf.Config, ct *ctl.Controller) (*Router, error) {
	r := &Router{
		cfg:  cf,
		wl:   convertWhiteList(cf.WhiteList.Ips),
		ctl:  ct,
		acc:  ct.AccCtl,
		pf:   ct.PfCtl,
		hm:   ct.HomeCtl,
		nt:   ct.NotiCtl,
		sig:  ct.Signaling,
		chat: ct.ChatCtl,
		st:   ct.StoryCtl,
		set:  ct.SetCtl,
		rdb:  ct.GetRedis(),
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
		auth := c.GetHeader("X-Totp")
		logger.Info("auth : ", auth)
		c.Next()
	}
}

func (p *Router) otpAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request IP is in the allowed IP list from the config
		requestIP := c.ClientIP()
		if _, ok := p.wl[requestIP]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "IP not allowed"})
			return
		}

		auth := c.GetHeader("X-Otp")
		logger.Info("auth : ", auth)
		c.Next()
	}
}

func (p *Router) Idx() *gin.Engine {
	e := gin.Default()

	// MultipartForm 메모리 제한 설정 (32MB - 파일 업로드용)
	e.MaxMultipartMemory = 32 << 20 // 32MB

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

	// 클라이언트 테스트 페이지 서빙
	e.Static("/client", "./client")

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
		account.POST("/regist", p.AesDecrypt(), p.acc.RegistUserInfo)

		// 로그인
		account.POST("/login", p.AesDecrypt(), p.acc.LoginUser)

		// 토큰 재발급
		account.POST("/refresh", p.acc.RefreshToken)

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

		pfset.POST("/upd/mpic", p.EncParamFileUpload(1, 5), p.acc.ModifyMainPic)

		// 로그아웃
		pfset.POST("/logout", p.acc.LogoutUser)

		// 회원 탈퇴
		pfset.POST("/leave", p.acc.LeaveUser)
		// 회원 정보 조회
		pfset.GET("/info/:id", p.acc.GetSidUserInfo)

		pfset.GET("/uinfo/:tid", p.acc.GetUidFromInfo)

		// 회원 정보 삭제
		pfset.POST("/delete/:id", p.acc.DeleteUser)
	}

	binfo := e.Group("block/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// block user
		// TODO : reason 추가 post test
		binfo.POST("/set/:tid", p.acc.SetBlockUser)
		binfo.POST("/cancel/:tid", p.acc.UnblockUser)

		// get block list
		binfo.GET("/list/:page/:limit", p.acc.GetBlockUser)
		binfo.GET("/count", p.acc.GetBlockCount)
	}

	fwbinfo := e.Group("fav/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// follow user
		fwbinfo.POST("/set/:tid", p.acc.SetFavoriteUser)
		fwbinfo.POST("/cancel/:tid", p.acc.UnfavoriteUser)

		// get follower list
		fwbinfo.GET("/list/:page/:limit", p.acc.GetFavoriteList)
		fwbinfo.GET("/count", p.acc.GetFavoriteCount)
	}

	user := e.Group("user/v01", p.SecurityHeaders(), liteAuth())
	{
		//보유금액, 즐겨찾기, 통화내역
		user.GET("/myinfo")
		user.GET("/wyinfo")
		user.GET("/getset/:uid", p.JwtAuth(), p.set.GetSetting)
		user.POST("/set/:cate/:value", p.JwtAuth(), p.set.SetSetting)
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
	/*
		inbox := e.Group("inbox/v01", p.SecurityHeaders(), p.JwtAuth())
		{
			inbox.GET("/list", p.chat.GetChatRooms)
			inbox.GET("/unread", p.chat.GetTotalUnread)
		}
	*/
	present := e.Group("present/v01", p.SecurityHeaders(), liteAuth())
	{
		present.PUT("/target")
	}

	mission := e.Group("mission/v01", p.SecurityHeaders(), liteAuth())
	{
		mission.GET("/list")
		mission.GET("/detail/:id")
	}

	story := e.Group("story/v01", p.SecurityHeaders())
	{
		// 좋아요 카운트, 팔로워, 조회수, 팔로잉?, 신고카운트
		// story.GET("/home/:id")
		story.POST("/create", p.JwtAuth(), p.ValidateFileUpload(5, 5), p.st.CreateStory)
		story.GET("/condition/list/:area/:stat/:type/:order/:gen/:page/:limit", p.st.GetStoryConditionList)

		story.GET("/list", p.st.GetStoryDefaultList)
		story.GET("/detail/:idx", p.st.GetStoryDetail)

		// ------------- story -------------
		story.POST("/updstat", p.st.UpdateStoryStat)
		story.POST("/updbody", p.st.UpdateStrBody)
		story.POST("/delpic", p.st.DeleteStrPic)

		// ------------- comment -------------
		story.GET("/comment/:idx/:page", p.st.GetStrCmtDetail)
		story.POST("/comment/create", p.st.CreateStrComment)
		story.POST("/comment/updstat", p.st.UpdateStrStatComment)
		story.POST("/comment/updbody", p.st.UpdateStrBodyComment)

		// ------------- like/follow/block -------------
		story.POST("/like/toggle", p.st.ToggleStoryLike)
		story.POST("/follow/set/:tid", p.JwtAuth(), p.st.FollowUser)
		story.POST("/follow/cancel/:tid", p.JwtAuth(), p.st.UnfollowUser)
		story.GET("/follower/list/:page/:limit", p.JwtAuth(), p.st.GetFollowerList)
		story.GET("/following/list/:page/:limit", p.JwtAuth(), p.st.GetFollowingList)
		/* story.POST("/block/create", p.st.BlockUser)
		story.POST("/block/cancel", p.st.UnblockUser)
		story.GET("/block/list/:uid/:page/:limit", p.st.GetBlockList) */

	}

	//누드, 음모 확인 기능
	upload := e.Group("upload/v01", p.SecurityHeaders(), liteAuth())
	{
		upload.POST("/story/img", p.ValidateFileUpload(5, 5))
		upload.POST("/present")
	}

	setting := e.Group("setting/v01", p.SecurityHeaders(), liteAuth())
	{
		setting.GET("/alert")
	}

	// WebRTC 시그널링 인터페이스 (화상 + 음성 공통)
	webrtc := e.Group("webrtc/v01", p.SecurityHeaders())
	{
		// WebRTC 설정 정보 조회
		webrtc.GET("/config", p.acc.GetWebRTCConfig)

		// WebSocket 시그널링 엔드포인트
		webrtc.GET("/ws", p.sig.HandleConnection)
	}

	// 텍스트 채팅 인터페이스
	// chat := e.Group("chat/v01", p.SecurityHeaders(), p.JwtAuth())
	chat := e.Group("dm/v01", p.SecurityHeaders(), p.JwtAuth())
	{
		// WebSocket 메시징 엔드포인트
		chat.GET("/ws", p.chat.HandleWebSocket)

		// 채팅방 생성
		chat.POST("/mkroom/:pid", p.chat.CreateChatRoom)

		// 채팅방 삭제
		chat.POST("/rmroom/:room_id", p.chat.DeleteChatRoom)

		// 사용자의 채팅방 목록 조회
		chat.GET("/list/:page", p.chat.GetChatRooms)

		//unread total count
		chat.GET("/total/unread", p.chat.GetTotalUnread)

		chat.GET("/history/:room_id", p.chat.GetChatList)

		chat.POST("/upload/img", p.simpleImgUploadCF(5, 5), p.chat.UploadImage)
		/*
			// 메시지 전송 (REST API)
			// chat.POST("/message", p.chat.SendMessage)
			// chat.GET("/rooms", p.JwtAuth(), p.chat.GetChatRooms)

			// 채팅 기록 조회
			// chat.GET("/list/:roomId", p.chat.GetChatList)
		*/
	}

	return e
}
