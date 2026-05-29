package controller

import (
	"database/sql"
	"ms-gateway/conf"
	"ms-gateway/models"
	"net/http"
	"strconv"

	ptl "ms-gateway/protocol"

	log "ms-gateway/common/logger"

	"github.com/gin-gonic/gin"
)

type NotiController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	// rdb *models.RedisDB
	adb *models.AccountDB
	hdb *models.HistoryDB
}

func NewNotiController(ctl *Controller, rep *models.Repositories) (*NotiController, error) {
	r := &NotiController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.hdb); err != nil {
		return nil, err
	}

	return r, nil
}

// GetNotiList godoc
// @Summary      알림 목록 조회
// @Description  사용자의 알림 목록을 조회합니다 (JWT 인증 기반)
// @Tags         Noti
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  protocol.RespHeader
// @Failure      400  {object}  protocol.RespHeader "Bad request"
// @Failure      401  {object}  protocol.RespHeader "Unauthorized"
// @Router       /noti/v01/list [get]
func (p *NotiController) GetNotiList(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "id is required")
		return
	}

}

// 개인사용자 알림 조회, 페이징 처리,
func (p *NotiController) GetNoti(uid uint64) (*[]ptl.Noti, error) {
	noti, err := p.hdb.GetNotiAllList(uid)
	if err != nil {
		return nil, err
	}

	return noti, nil
	//return p.ctl.Rdb.Get(p.ctl.Rdb.Key(fmt.Sprintf("noti:%s", id)))
}

func (p *NotiController) GetNotiNewCount(uid uint64) (int, error) {
	count, err := p.hdb.GetNewNotiCount(uid)
	if err != nil {
		return 0, err
	}

	return count, nil
	//return p.ctl.Rdb.Get(p.ctl.Rdb.Key(fmt.Sprintf("noti:%s", id)))
}

// GetNotiDetail godoc
// @Summary      알림 상세 조회
// @Description  알림 상세 정보를 조회하고 읽음 처리합니다
// @Tags         Noti
// @Accept       json
// @Produce      json
// @Param        idx  path      string  true  "Noti index"
// @Success      200  {object}  protocol.Noti
// @Failure      400  {object}  protocol.RespHeader "idx is required"
// @Failure      500  {object}  protocol.RespHeader "Failed to get noti detail"
// @Router       /noti/v01/detail/{idx} [get]
func (p *NotiController) GetNotiDetail(c *gin.Context) {
	idx := c.Param("idx")
	if idx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "idx is required")
		return
	}

	notiIdx, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid idx")
		return
	}

	noti, err := p.hdb.GetNotiDetail(notiIdx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get noti detail")
		return
	}

	err = p.hdb.SetNotiRead(notiIdx)
	if err != nil {
		log.Warn("Failed to set noti read: ", err)
	}

	p.ctl.SendDataResponse(c, http.StatusOK, noti)
}

// GetAnnouncement godoc
// @Summary      공지사항 목록 조회
// @Description  공지사항 목록을 조회합니다 (일주일 이내 공지는 new 표시)
// @Tags         Noti
// @Accept       json
// @Produce      json
// @Success      200  {array}   protocol.Announcement
// @Failure      500  {object}  protocol.RespHeader "Failed to get announcement list"
// @Router       /noti/v01/anc/list [get]
func (p *NotiController) GetAnnouncement(c *gin.Context) {
	announcementList, err := p.hdb.GetAnnouncementList()
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get announcement list")
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, announcementList)
}

// admin 권한 확인, 공지사항 저장
func (p *NotiController) SetAnnouncement(c *gin.Context) {

	var req struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body" binding:"required"`
		Url   string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to parse JSON")
		return
	}

	announcement := &ptl.Announcement{
		Title: req.Title,
		Body:  req.Body,
		Url:   req.Url,
	}

	err := p.hdb.SaveAnnouncement(announcement)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to save announcement")
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "success"})
}

// GetAnnouncementDetail godoc
// @Summary      공지사항 상세 조회
// @Description  공지사항 상세 정보를 조회합니다
// @Tags         Noti
// @Accept       json
// @Produce      json
// @Param        idx  path      string  true  "Announcement index"
// @Success      200  {object}  protocol.Announcement
// @Failure      400  {object}  protocol.RespHeader "idx is required"
// @Failure      404  {object}  protocol.RespHeader "Announcement not found"
// @Failure      500  {object}  protocol.RespHeader "Failed to get announcement detail"
// @Router       /noti/v01/anc/detail/{idx} [get]
func (p *NotiController) GetAnnouncementDetail(c *gin.Context) {
	idx := c.Param("idx")
	if idx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "idx is required")
		return
	}

	announcementIdx, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid idx")
		return
	}

	announcement, err := p.hdb.GetAnnouncementDetail(announcementIdx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get announcement detail")
		return
	} else if err == sql.ErrNoRows {
		p.ctl.SimpleError(c, http.StatusNotFound, "Announcement not found")
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, announcement)
}

func (p *NotiController) SetNoti() {

	//return p.ctl.Rdb.Get(p.ctl.Rdb.Key(fmt.Sprintf("noti:%s", id)))
}
