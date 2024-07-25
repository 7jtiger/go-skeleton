package controller

import (
	"go-skeleton/conf"
	"go-skeleton/models"
	"go-skeleton/protocol"

	"github.com/gin-gonic/gin"
)

type Health struct {
	ctl *Controller
	cfg *conf.Config
}

func NewHeartbeat(h *Controller, rep *models.Repositories) *Health {
	return &Health{
		ctl: h,
	}
}

// @Summary Check
// @Description 서버 살아있는지 확인
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} protocol.RespHeader
// @Router /health/check [get]
func (p *Health) Check(c *gin.Context) {
	p.ctl.RespSuccess(c, protocol.NewRespHeader(protocol.Success))
}
