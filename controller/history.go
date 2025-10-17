package controller

import (
	"ms-gateway/conf"
	"ms-gateway/models"
)

type HistoryController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories
}

func NewHistoryController(ctl *Controller, rep *models.Repositories) (*HistoryController, error) {
	r := &HistoryController{
		ctl: ctl,
		rep: rep,
	}

	return r, nil
}
