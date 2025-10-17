package controller

import (
	"ms-gateway/conf"
	"ms-gateway/models"
)

type ItemController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories
}

func NewItemController(ctl *Controller, rep *models.Repositories) (*ItemController, error) {
	r := &ItemController{
		ctl: ctl,
		rep: rep,
	}

	return r, nil
}
