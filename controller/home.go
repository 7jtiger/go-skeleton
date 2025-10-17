package controller

import (
	"ms-gateway/conf"
	"ms-gateway/models"

	"github.com/gin-gonic/gin"
)

type HomeController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	rdb *models.RedisDB
	adb *models.AccountDB
}

func NewHomeController(ctl *Controller, rep *models.Repositories) (*HomeController, error) {
	r := &HomeController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

/*
상단메뉴 - 신규 알림 유무
메인메뉴 - 쪽지갯수
		- 출석체크 유무
배너 출력
영상통화 최신 7개 리스트 출력
음성통화 최신 7개 리스트 출력
스토리 최신 6개 리스트 출력
이용약관, 개인정보처리방침 링크
*/

func (p *HomeController) GetHomeMenInfo(c *gin.Context) {
	//noti 확인

}
