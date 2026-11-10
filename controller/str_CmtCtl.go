package controller

import (
	"database/sql"
	"errors"
	"ms-gateway/common/utils"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

/* ------------- comment -------------*/

// CreateStrComment godoc
// @Summary      스토리 댓글 생성
// @Description  스토리 댓글 생성 API입니다.
// @Description
// @Description  **요청**: body StrCmtCreateReq: str_idx, stat, body 문자열이 필요합니다. wuid/nick는 호환용으로 허용하되 무시합니다.
// @Description
// @Description  **처리**: 작성자는 JWT UID, 닉네임은 서버 계정 정보로 강제합니다. 부모 글의 실제 열람 권한과 작성자와의 양방향 차단을 검사합니다.
// @Description
// @Description  **응답**: 최상위 msg, lastID.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.StrCmtCreateReq true "body StrCmtCreateReq: str_idx, stat, body 문자열이 필요합니다. wuid/nick는 호환용으로 허용하되 무시합니다."
// @Success      200 {object} object{msg=string,lastID=int} "최상위 msg, lastID."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "스토리 또는 본인 댓글 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/comment/create [post]
// @Example Response: {"msg":"ok","lastID":1}
// @x-response-example {"msg":"ok","lastID":1}
// @Example Request: POST /story/v01/comment/create
// @Example Request Body: {"str_idx":"15","wuid":"4033287471439576593","nick":"닉네임","stat":"0","body":"댓글"}
// @x-request-example {"str_idx":"15","wuid":"4033287471439576593","nick":"닉네임","stat":"0","body":"댓글"}
func (p *StoryController) CreateStrComment(c *gin.Context) {
	actor, ok := p.storyActorUID(c)
	if !ok {
		return
	}
	var req ptl.StrCmtCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, 400, "Invalid JSON", err)
		return
	}
	idx, err := strconv.Atoi(req.StrIdx)
	if err != nil || idx <= 0 || req.Body == "" {
		p.ctl.SimpleError(c, 400, "str_idx and body are required")
		return
	}
	stat, err := strconv.Atoi(req.Stat)
	if err != nil || stat < 0 || stat > 4 {
		p.ctl.SimpleError(c, 400, "stat must be 0..4")
		return
	}
	if !p.checkCommentAccess(c, actor, idx) {
		return
	}
	account, err := p.adb.GetUserInfoByUID(actor)
	if err != nil {
		p.commentError(c, err)
		return
	}
	cmt := &ptl.StrComment{StrIdx: idx, Wuid: actor, Nick: account.Nick, ThumbUrl: account.ThumbPic, WGender: account.Gender, WAge: strconv.Itoa(utils.CalcBirth2Age(account.Birth)), WArea: account.Area, Stat: stat, Body: req.Body}
	lastID, err := p.sdb.SetStrComment(cmt)
	if err != nil {
		p.commentError(c, err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "lastID": lastID})
}

// GetStrCmtDetail godoc
// @Summary      스토리 댓글 목록 조회
// @Description  스토리 댓글 목록 조회 API입니다.
// @Description
// @Description  **요청**: path idx: 스토리 인덱스, page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: 부모 스토리의 실제 열람 권한을 확인한 뒤 댓글을 조회합니다. 잠긴 글은 403, 비공개 타인 글·삭제 글·없는 글은 404입니다.
// @Description
// @Description  **응답 data**: comment, total_cnt, page, limit.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.StrCmtDetailResp "data: comment, total_cnt, page, limit."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "부모 글 잠금 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "스토리 또는 본인 댓글 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/comment/{idx}/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"comment":[],"total_cnt":0,"page":1,"limit":10}}
// @x-response-example {"result":0,"resultString":"Success","data":{"comment":[],"total_cnt":0,"page":1,"limit":10}}
// @Example Request: GET /story/v01/comment/15/1/10
func (p *StoryController) GetStrCmtDetail(c *gin.Context) {
	actor, ok := p.storyActorUID(c)
	if !ok {
		return
	}
	idx, err := strconv.Atoi(c.Param("idx"))
	if err != nil || idx <= 0 {
		p.ctl.SimpleError(c, 400, "idx is invalid")
		return
	}
	page, limit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, 400, "page or limit is invalid", err)
		return
	}
	comments, total, err := p.sdb.GetStrCmtDetail(actor, idx, page, limit)
	if err != nil {
		p.commentError(c, err)
		return
	}

	p.ctl.SendDataResponse(c, 200, gin.H{"comment": comments, "total_cnt": total, "page": page, "limit": limit})
}

// UpdateStrStatComment godoc
// @Summary      댓글 상태 변경
// @Description  댓글 상태 변경 API입니다.
// @Description
// @Description  **요청**: body: idx, stat 문자열이 필요합니다.
// @Description
// @Description  **처리**: JWT 댓글 작성자 또는 서버 CommentAdminUIDs의 관리자만 수정할 수 있습니다. 부모 글 열람 권한·차단 관계를 확인하고 SQL에 댓글 idx와 작성자 또는 서버 승인 관리자 조건을 적용합니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.StrCmtUpdStatReq true "body: idx, stat 문자열이 필요합니다."
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "부모 글 잠금 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "스토리 또는 본인 댓글 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/comment/updstat [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/comment/updstat
// @Example Request Body: {"idx":"3","stat":"4"}
// @x-request-example {"idx":"3","stat":"4"}
func (p *StoryController) UpdateStrStatComment(c *gin.Context) {
	actor, ok := p.storyActorUID(c)
	if !ok {
		return
	}
	var req ptl.StrCmtUpdStatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, 400, "Invalid JSON", err)
		return
	}
	idx, err := strconv.Atoi(req.Idx)
	if err != nil || idx <= 0 {
		p.ctl.SimpleError(c, 400, "idx is invalid")
		return
	}
	stat, err := strconv.Atoi(req.Stat)
	if err != nil || stat < 0 || stat > 4 {
		p.ctl.SimpleError(c, 400, "stat must be 0..4")
		return
	}
	parent, err := p.sdb.GetCommentStoryID(idx, actor, p.isCommentAdmin(actor))
	if err != nil {
		p.commentError(c, err)
		return
	}
	if !p.checkCommentAccess(c, actor, parent) {
		return
	}
	affected, err := p.sdb.UpdateStrStatComment(idx, actor, p.isCommentAdmin(actor), stat)
	if err != nil {
		p.commentError(c, err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// UpdateStrBodyComment godoc
// @Summary      댓글 본문 변경
// @Description  댓글 본문 변경 API입니다.
// @Description
// @Description  **요청**: body: idx, body 문자열이 필요합니다.
// @Description
// @Description  **처리**: JWT 댓글 작성자 또는 서버 CommentAdminUIDs의 관리자만 수정할 수 있습니다. 부모 글 열람 권한·차단 관계를 확인하고 SQL에 댓글 idx와 작성자 또는 서버 승인 관리자 조건을 적용합니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.StrCmtUpdBodyReq true "body: idx, body 문자열이 필요합니다."
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "부모 글 잠금 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "스토리 또는 본인 댓글 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/comment/updbody [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/comment/updbody
// @Example Request Body: {"idx":"3","body":"수정한 댓글"}
// @x-request-example {"idx":"3","body":"수정한 댓글"}
func (p *StoryController) UpdateStrBodyComment(c *gin.Context) {
	actor, ok := p.storyActorUID(c)
	if !ok {
		return
	}
	var req ptl.StrCmtUpdBodyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, 400, "Invalid JSON", err)
		return
	}
	idx, err := strconv.Atoi(req.Idx)
	if err != nil || idx <= 0 {
		p.ctl.SimpleError(c, 400, "idx is invalid")
		return
	}
	if req.Body == "" {
		p.ctl.SimpleError(c, 400, "body is required")
		return
	}
	parent, err := p.sdb.GetCommentStoryID(idx, actor, p.isCommentAdmin(actor))
	if err != nil {
		p.commentError(c, err)
		return
	}
	if !p.checkCommentAccess(c, actor, parent) {
		return
	}
	affected, err := p.sdb.UpdateStrBodyComment(idx, actor, p.isCommentAdmin(actor), req.Body)
	if err != nil {
		p.commentError(c, err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// Comment mutation requires content access and no block in either direction.
func (p *StoryController) checkCommentAccess(c *gin.Context, actor uint64, idx int) bool {
	owner, err := p.sdb.RequireStoryRead(actor, idx)
	if err != nil {
		p.commentError(c, err)
		return false
	}
	blocked, err := p.adb.IsBlockedPair(actor, owner)
	if err != nil {
		p.commentError(c, err)
		return false
	}
	if blocked {
		p.ctl.SimpleError(c, http.StatusForbidden, "Blocked relation")
		return false
	}
	return true
}
func (p *StoryController) commentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		p.ctl.SimpleError(c, http.StatusNotFound, "Story or comment not found")
	case errors.Is(err, models.ErrStoryLocked):
		p.ctl.SimpleError(c, http.StatusForbidden, "Story content is locked")
	default:
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Comment operation failed", err)
	}
}

func (p *StoryController) isCommentAdmin(actor uint64) bool {
	if actor == 0 || p.cfg == nil {
		return false
	}
	for _, uid := range p.cfg.Server.CommentAdminUIDs {
		if uid == actor {
			return true
		}
	}
	return false
}
