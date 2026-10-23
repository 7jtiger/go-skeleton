package controller

import (
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

/* ------------- comment -------------*/

// CreateStrComment godoc
// @Summary      스토리 댓글 작성
// @Description  JWT 인증 후 스토리 댓글을 생성합니다. 작성자 메타(`thumb_url`, `wgender`, `wage`, `warea`)는 `wuid`로 계정에서 조회해 저장합니다.
// @Description
// @Description  **요청 body** (`StrCmtCreateReq`) — 모두 필수(빈 문자열 불가)
// @Description  - `str_idx`: 스토리 idx (숫자 문자열)
// @Description  - `wuid`: 작성자 UID (숫자 문자열)
// @Description  - `nick`: 닉네임
// @Description  - `stat`: 0=default, 1=private, 2=reserved, 3=reserved, 4=deleted
// @Description  - `body`: 댓글 본문 (max 256)
// @Description
// @Description  **응답** (`SimpleRespOK` — result envelope 없음)
// @Description  - `msg`, `lastID` (str_cmt INSERT LastInsertId)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/comment/create
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - Body: {"str_idx":"3","wuid":"77645423236541","nick":"test","stat":"1","body":"111111 test comment body"}
// @Description
// @Description  **Response example**
// @Description  {"msg":"ok","lastID":1}
// @Tags         Story_Comment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string                     true  "Bearer access token"
// @Param        request        body    protocol.StrCmtCreateReq   true  "댓글 생성 데이터"
// @Success      200            {object} protocol.StrCmtCreateResp  "msg, lastID"
// @Failure      400            {object} map[string]interface{}     "JSON/필수값/숫자 파싱 오류"
// @Failure      401            {object} map[string]interface{}     "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{}     "계정 조회/댓글 저장 실패"
// @Router       /story/v01/comment/create [post]
func (p *StoryController) CreateStrComment(c *gin.Context) {
	var req ptl.StrCmtCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.StrIdx == "" || req.Wuid == "" || req.Nick == "" || req.Stat == "" || req.Body == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Cid is required")
		return
	}

	strIdxInt, err := strconv.ParseInt(req.StrIdx, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StrIdx is required")
		return
	}

	wuidInt, err := strconv.ParseInt(req.Wuid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Wuid is required")
	}

	nstat, err := strconv.Atoi(req.Stat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Stat is required")
		return
	}

	account, err := p.adb.GetUserInfoByUID(uint64(wuidInt))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get account", err)
		return
	}
	/* 	ownerUid, err := p.sdb.GetStoryOwnerUID(int(strIdxInt))
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story owner", err)
	   		return
	   	}
	   	isBlocked, err := p.sdb.IsBlockedPair(uint64(wuidInt), ownerUid)
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check block relation", err)
	   		return
	   	}
	   	if isBlocked {
	   		p.ctl.SimpleError(c, http.StatusForbidden, "blocked relation")
	   		return
	   	}
	*/
	cmt := &ptl.StrComment{
		StrIdx:   int(strIdxInt),
		Wuid:     uint64(wuidInt),
		Nick:     req.Nick,
		ThumbUrl: account.ThumbPic,
		WGender:  account.Gender,
		WAge:     strconv.Itoa(utils.CalcBirth2Age(account.Birth)),
		WArea:    account.Area,
		Stat:     nstat,
		Body:     req.Body,
	}

	lastID, err := p.sdb.SetStrComment(cmt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set str comment", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "lastID": lastID})
}

// GetStrCmtDetail godoc
// @Summary      스토리 댓글 목록 조회
// @Description  JWT 인증 후 특정 스토리의 댓글을 페이징 조회합니다.
// @Description
// @Description  **path**
// @Description  - `idx`: 스토리 idx (`str_cmt.str_idx`)
// @Description  - `page`: 페이지 번호 (1부터, `getPageAndLimit`)
// @Description  - `limit`: 페이지 크기 (1~100, 0이면 DB 기본 20)
// @Description
// @Description  **응답 data** (`StrCmtDetailData`)
// @Description  - `comment`: `[]StrComment` (idx, str_idx, wuid, nick, thumb_url, wgender, wage, warea, body, stat, at_create, at_update)
// @Description  - `total_count`: 전체 댓글 수
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/comment/3/1/20
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"comment":[{"idx":2,"str_idx":3,"wuid":77645423236541,"nick":"test","thumb_url":"https://imagedelivery.net/.../public","wgender":"1","wage":"30","warea":"1","body":"111111 test comment body","stat":1,"at_create":"2026-02-02T13:50:06Z","at_update":"2026-02-02T13:50:06Z"}],"total_count":52}}
// @Tags         Story_Comment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "스토리 idx"
// @Param        page           path    int     true  "페이지 번호(1부터)"
// @Param        limit          path    int     true  "페이지 크기(1~100)"
// @Success      200            {object} protocol.StrCmtDetailResp "data.comment, data.total_count"
// @Failure      400            {object} map[string]interface{} "idx/page/limit 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "댓글 목록 조회 실패"
// @Router       /story/v01/comment/{idx}/{page}/{limit} [get]
func (p *StoryController) GetStrCmtDetail(c *gin.Context) {
	idx := c.Param("idx")

	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	commentList, totalCount, err := p.sdb.GetStrCmtDetail(idxInt, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get str comment list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"comment": commentList, "total_count": totalCount})
}

// UpdateStrStatComment godoc
// @Summary      스토리 댓글 상태 변경
// @Description  JWT 인증 후 `str_cmt.stat`을 변경합니다.
// @Description
// @Description  **요청 body** (`StrCmtUpdStatReq`) — 모두 필수
// @Description  - `idx`: 댓글 idx (숫자 문자열)
// @Description  - `stat`: 0=default, 1=private, 2=reserved, 3=reserved, 4=deleted
// @Description
// @Description  **응답** (`SimpleRespOK` — result envelope 없음)
// @Description  - `msg`, `affected` (RowsAffected)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/comment/updstat
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - Body: {"idx":"3","stat":"2"}
// @Description
// @Description  **Response example**
// @Description  {"msg":"ok","affected":1}
// @Tags         Story_Comment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string                      true  "Bearer access token"
// @Param        request        body    protocol.StrCmtUpdStatReq   true  "댓글 상태 변경"
// @Success      200            {object} protocol.StrCmtUpdStatResp  "msg, affected"
// @Failure      400            {object} map[string]interface{}      "JSON/idx/stat 오류"
// @Failure      401            {object} map[string]interface{}      "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{}      "DB 업데이트 실패"
// @Router       /story/v01/comment/updstat [post]
func (p *StoryController) UpdateStrStatComment(c *gin.Context) {
	//"Status (0:default, 1:private, 2:reserved, 3:reserved, 4:deleted)"
	var req ptl.StrCmtUpdStatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Idx == "" || req.Stat == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx and Stat are required")
		return
	}

	cmtIdx, err := strconv.Atoi(req.Idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	nstat, err := strconv.Atoi(req.Stat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Stat is required")
		return
	}

	affected, err := p.sdb.UpdateStrStatComment(cmtIdx, nstat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update str stat comment", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// UpdateStrBodyComment godoc
// @Summary      스토리 댓글 본문 수정
// @Description  JWT 인증 후 `str_cmt.body`를 수정합니다.
// @Description
// @Description  **요청 body** (`StrCmtUpdBodyReq`) — 모두 필수
// @Description  - `idx`: 댓글 idx (숫자 문자열)
// @Description  - `body`: 수정할 본문 (빈 문자열 불가)
// @Description
// @Description  **응답** (`SimpleRespOK` — result envelope 없음)
// @Description  - `msg`, `affected` (RowsAffected)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/comment/updbody
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - Body: {"idx":"3","body":"updated comment body content"}
// @Description
// @Description  **Response example**
// @Description  {"msg":"ok","affected":1}
// @Tags         Story_Comment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string                      true  "Bearer access token"
// @Param        request        body    protocol.StrCmtUpdBodyReq   true  "댓글 본문 수정"
// @Success      200            {object} protocol.StrCmtUpdBodyResp  "msg, affected"
// @Failure      400            {object} map[string]interface{}      "JSON/idx/body 오류"
// @Failure      401            {object} map[string]interface{}      "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{}      "DB 업데이트 실패"
// @Router       /story/v01/comment/updbody [post]
func (p *StoryController) UpdateStrBodyComment(c *gin.Context) {
	var req ptl.StrCmtUpdBodyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Body == "" || req.Idx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Body and Idx are required")
		return
	}

	cmtIdx, err := strconv.Atoi(req.Idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	affected, err := p.sdb.UpdateStrBodyComment(cmtIdx, req.Body)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update str body comment", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}
