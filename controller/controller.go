package controller

import (
	"encoding/json"
	"net/http"

	"go-skeleton/models"

	"github.com/gin-gonic/gin"

	log "go-skeleton/common/logger"
	"go-skeleton/conf"
	"go-skeleton/protocol"
)

// Controller
type Controller struct {
	cfg           *conf.Config
	healthHandler *Health
}

func NewCTL(cf *conf.Config, rep *models.Repositories) (*Controller, error) {
	r := &Controller{
		cfg: cf,
	}

	r.healthHandler = NewHeartbeat(r, rep)

	return r, nil
}

func (p *Controller) RespSuccess(c *gin.Context, resp interface{}) {
	c.JSON(http.StatusOK, resp)
}

func (p *Controller) GetConfig() *conf.Config {
	return p.cfg
}

func (p *Controller) RespError(c *gin.Context, body interface{}, status int, err ...interface{}) {
	bytes, _ := json.Marshal(body)

	log.Error("Request error", "path", c.FullPath(), "body", bytes, "status", status, "error", joinMsg(err))

	c.JSON(status, protocol.NewRespHeader(protocol.Failed, joinMsg(c.Request.URL, err)))
	c.Abort()
}

func (p *Controller) SimpleError(c *gin.Context, status int, err ...interface{}) {
	log.Error("Request error", "path", c.FullPath(), "status", status, "error", joinMsg(err))

	c.JSON(status, protocol.NewRespHeader(protocol.Failed, joinMsg(c.Request.URL, err)))
	c.Abort()
}

func (p *Controller) GetHealthHandler() *Health {
	return p.healthHandler
}
