package router

import (
	"io/fs"
	"net/http"
	"strings"

	"livein-web/conf"
	"livein-web/controller"
	"livein-web/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	cfg  *conf.Config
	ctrl *controller.Controller
	dist fs.FS
}

func New(cfg *conf.Config, ctrl *controller.Controller, dist fs.FS) *Router {
	return &Router{cfg: cfg, ctrl: ctrl, dist: dist}
}

func (r *Router) Setup() *gin.Engine {
	if r.cfg.Server.Mode == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.Default()
	e.Use(middleware.CORS())
	e.Use(middleware.SecurityHeaders())

	api := e.Group("/api")
	{
		api.GET("/content", r.ctrl.GetContent)
		api.POST("/contact", r.ctrl.SubmitContact)
		api.GET("/health", r.ctrl.Health)
	}

	admin := api.Group("/admin")
	{
		admin.POST("/login", r.ctrl.Login)

		auth := admin.Group("", middleware.JWTAuth(r.cfg.Admin.JWTSecret))
		{
			auth.GET("/content", r.ctrl.GetContent)
			auth.PUT("/content", r.ctrl.UpdateContent)
			auth.GET("/inquiries", r.ctrl.ListInquiries)
		}
	}

	if r.dist != nil {
		r.serveEmbedded(e)
	} else {
		r.serveFilesystem(e)
	}

	return e
}

func (r *Router) serveFilesystem(e *gin.Engine) {
	staticDir := r.cfg.Server.StaticDir
	e.Static("/assets", staticDir+"/assets")
	e.StaticFile("/favicon.svg", staticDir+"/favicon.svg")

	e.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		c.File(staticDir + "/index.html")
	})
}

func (r *Router) serveEmbedded(e *gin.Engine) {
	assets, _ := fs.Sub(r.dist, "assets")
	e.StaticFS("/assets", http.FS(assets))

	e.GET("/favicon.svg", func(c *gin.Context) {
		data, err := fs.ReadFile(r.dist, "favicon.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", data)
	})

	e.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			return
		}
		data, err := fs.ReadFile(r.dist, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
