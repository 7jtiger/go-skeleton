package controller

import (
	"ms-gateway/conf"
	"ms-gateway/models"
)

type ProfileController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories
}

func NewProfileController(ctl *Controller, rep *models.Repositories) (*ProfileController, error) {
	r := &ProfileController{
		ctl: ctl,
		rep: rep,
	}

	return r, nil
}
