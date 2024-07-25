package router

import (
	"encoding/base32"

	"basesk/common/logger"
	"basesk/conf"

	ctl "basesk/controller"

	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"

	"basesk/docs"

	ginSwg "github.com/swaggo/gin-swagger"

	swgFiles "github.com/swaggo/files"
)

type Router struct {
	cfg     *conf.Config
	wl map[string]string
	ct      *ctl.Controller
	hHealth *ctl.Health
}

func NewRouter(cf *conf.Config, ct *ctl.Controller) (*Router, error) {
	r := &Router{
		cfg:     cf,
		wl:      convertWhiteList(cf.WhiteList.Ips),
		ct:      ct,
		hHealth: ct.GetHealthHandler(),
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

		auth := c.GetHeader("OTP-Auth")
		if !validateOTP(auth) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
			return
		}

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

		auth := c.GetHeader("OTP-Auth")
		if !validateOTP(auth) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
			return
		}

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

	logger.Info("start server")

	e.GET("/swagger/:any", ginSwg.WrapHandler(swgFiles.Handler))
	docs.SwaggerInfo.Host = "localhost"

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
		wd.POST("/req", p.dt.AddDeposit)
		wd.GET("/myinfo/:id", p.pt.GetMyInfo)
	}

	return e
}
