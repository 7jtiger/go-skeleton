package controller

import (
	"encoding/json"
	"mime/multipart"
	"ms-gateway/common/utils"
	ptc "ms-gateway/protocol"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ------------- profile -------------
// 프로필 + 팔로워 수(: 나를 팔로워, 내가 팔로우이)
// 화살 + 캐시 + 쪽지
// 프로필 사진 5개
// 최신 스토리 4개
/* func (p *StoryController) GetPrfHomeInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	homeInfo, err := p.sdb.GetPrfHomeInfo(uid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get profile home info", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, homeInfo)
}
*/

// GetPrfInfo godoc
// @Summary      스토리 프로필 정보 조회
// @Description  JWT 인증 후 대상 사용자 프로필(`prf_info`)과 팔로워 수를 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - path `tid`: 조회할 사용자 UID (본인 조회 시에도 본인 UID 전달)
// @Description  - 핸들러상 `tid`가 비어 있으면 JWT 본인 UID 사용
// @Description
// @Description  **처리**
// @Description  - `GetPrfInfo(tid)` → `prf_Info`
// @Description  - `GetFollowerCount(tid)` → `flw_cnt`
// @Description
// @Description  **응답 data**
// @Description  - `prf_Info`: `PrfInfo` (`uid`, `nick`, `gender`, `birth`, `age`, `area`, `intro`, `main_pic`)
// @Description  - `flw_cnt`: 팔로워 수
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/prf/info/8697414060736839837
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"flw_cnt":0,"prf_Info":{"uid":8697414060736839837,"nick":"건강한 꼬마 오렌지","gender":"1","birth":"1990-01-01T00:00:00Z","age":"-2","area":"1","intro":"반가워요 큐피톡에서 만나요!","main_pic":""}}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "조회 대상 UID (본인 조회 시에도 UID 전달)"
// @Success      200            {object} protocol.PrfInfoResp "prf_Info, flw_cnt"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "프로필/팔로워 조회 실패"
// @Router       /story/v01/prf/info/{tid} [get]
func (p *StoryController) GetPrfInfo(c *gin.Context) {
	// 프로필 정보
	// 개인 프로필, 타인 프로필 조회
	// tid 가 있으면 타인 없으면 본인
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	targetId64 := user.(*ptc.UserInfoResp).Uid
	tid := c.Param("tid")
	if tid != "" {
		var err error
		targetId64, err = strconv.ParseUint(tid, 10, 64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid", err)
			return
		}
	}

	prfInfo, err := p.sdb.GetPrfInfo(targetId64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get profile info", err)
		return
	}

	followerCount, err := p.sdb.GetFollowerCount(targetId64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get follower count", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"prf_Info": prfInfo,
		"flw_cnt":  followerCount,
	})
}

// GetPrfAssets godoc
// @Summary      스토리 프로필 자산 조회
// @Description  JWT 본인의 보유 화살(포인트), 캐시, DM 미읽음 쪽지 합계를 조회합니다.
// @Description
// @Description  **처리**
// @Description  - `GetUserAssetInfo(uid)` → `hold_point`, `hold_cash` (NULL이면 0)
// @Description  - `GetAllUnreadForUser(uid)` 합산 → `msg_count` (Redis)
// @Description
// @Description  **응답 data**
// @Description  - `hold_point`: 보유 화살/포인트
// @Description  - `hold_cash`: 보유 캐시
// @Description  - `msg_count`: 전체 DM 미읽음 쪽지 합계
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/prf/assets
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"hold_point":0,"hold_cash":0,"msg_count":3}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Success      200            {object} protocol.PrfAssetsResp "hold_point, hold_cash, msg_count"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "자산/쪽지 조회 실패"
// @Router       /story/v01/prf/assets [get]
func (p *StoryController) GetPrfAssets(c *gin.Context) {
	// 화살 + 캐시 + 쪽지 카운트
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	holdPoint, holdCash, err := p.adb.GetUserAssetInfo(uid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get profile assets", err)
		return
	}

	msgCount, err := p.rdb.GetAllUnreadForUser(uid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get message count", err)
		return
	}

	var totalMsgCount int64
	for _, count := range msgCount {
		totalMsgCount += count
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"hold_point": holdPoint, "hold_cash": holdCash, "msg_count": totalMsgCount})
}

// GetPrfPicLists godoc
// @Summary      프로필 사진 목록 조회
// @Description  JWT 인증 후 프로필 사진(`prf_info.sub_pic1`~`sub_pic5`) 목록을 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - 등록 라우트: `GET /prf/pic/list` → JWT 본인 UID
// @Description  - 핸들러상 path `tid`가 있으면 해당 UID 조회 (라우트 미등록 시 본인만)
// @Description
// @Description  **응답 data**: 비어 있지 않은 슬롯만 포함한 `PrfPicMap`
// @Description  - 키: `"1"`~`"5"` (`sub_pic` 컬럼 번호)
// @Description  - 값: `PrfPicInfo` (`url`, `stat`)
// @Description
// @Description  **PrfPicInfo.stat**: 1=요청중, 2=승인대기, 3=승인, 4=거절/삭제
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/prf/pic/list
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"1":{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":3},"2":{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":3}}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Success      200            {object} protocol.PrfPicListResp "data: PrfPicMap"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "프로필 사진 조회 실패"
// @Router       /story/v01/prf/pic/list [get]
func (p *StoryController) GetPrfPicLists(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid
	tid := c.Param("tid")
	if tid != "" {
		var err error
		uid64, err = strconv.ParseUint(tid, 10, 64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid", err)
			return
		}
	}

	prfPicInfos, err := p.sdb.GetPrfPicList(uid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get profile picture list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, prfPicInfos)
}

// GetPrfStoryLists godoc
// @Summary      프로필 최근 스토리 대표 이미지 목록
// @Description  JWT 인증 후 대상 UID의 최신 스토리 최대 4개의 대표 미디어 URL을 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - 등록 라우트: `GET /prf/story/lists/:tid` → path `tid` UID
// @Description  - 핸들러상 `tid`가 비어 있으면 JWT 본인 UID (라우트는 `:tid` 필수)
// @Description
// @Description  **응답 data** (`PrfStoryListsData`)
// @Description  - `str_imgs`: 이미지 스토리 대표 URL 맵 (key=story idx, value=URL)
// @Description  - `str_thmbls`: 동영상 스토리 썸네일 URL 맵 (key=story idx, value=thumb URL)
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/prf/story/lists/8697414060736839837
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"str_imgs":{"4":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public","5":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e7b2b58b-a7ec-49a1-5e6d-175e42609b00/public"},"str_thmbls":{"11":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "조회 대상 UID (본인 조회 시에도 UID 전달)"
// @Success      200            {object} protocol.PrfStoryListsResp "data: str_imgs, str_thmbls"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 목록 조회 실패"
// @Router       /story/v01/prf/story/lists/{tid} [get]
func (p *StoryController) GetPrfStoryLists(c *gin.Context) {
	// 리밋 4개
	// tid 가 있으면 타인, 없으면 본인
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	targetId64 := user.(*ptc.UserInfoResp).Uid
	tid := c.Param("tid")
	if tid != "" {
		var err error
		targetId64, err = strconv.ParseUint(tid, 10, 64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid", err)
			return
		}
	}

	strImgs, strthmbls, err := p.sdb.GetStoryFirstPicUrls(targetId64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"str_imgs": strImgs, "str_thmbls": strthmbls})
}

// UpdatePrfInfo godoc
// @Summary      스토리 프로필 정보 수정
// @Description  JWT 인증 사용자의 `prf_info`를 부분 수정합니다. UID는 JWT에서 강제 설정합니다.
// @Description
// @Description  **요청 body** (`protocol.PrfInfo`): 값이 있는 필드만 UPDATE (빈 문자열은 무시)
// @Description  - 반영 필드: `nick`, `gender`, `age`, `area`, `intro`
// @Description  - 미사용: `uid`(JWT 강제), `birth`, `main_pic` (대표사진은 `/prf/set/mainpic`)
// @Description  - `area`: 지역명 문자열 → 서버에서 area code로 변환 (`GetAreaCode`)
// @Description  - 모든 반영 필드가 비어 있으면 DB UPDATE 없이 성공 응답
// @Description
// @Description  **응답 data**: `{"msg":"ok"}`
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/prf/upd
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - Body: {"nick":"prf_test_nick","intro":"프로필 소개 — automated prf test","area":"서울"}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string            true  "Bearer access token"
// @Param        request        body    protocol.PrfInfo  true  "수정할 프로필 필드 (부분 업데이트)"
// @Success      200            {object} protocol.PrfUpdResp "data.msg=ok"
// @Failure      400            {object} map[string]interface{} "JSON 바인딩 실패"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Router       /story/v01/prf/upd [post]
func (p *StoryController) UpdatePrfInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	var req ptc.PrfInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	req.Uid = uid64
	err := p.sdb.UpdatePrfInfo(req)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update profile info", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok"})
}

// UploadPrfPic godoc
// @Summary      프로필 사진 업로드
// @Description  JWT 인증 사용자의 프로필 사진을 Cloudflare Images에 업로드하고 `prf_info.sub_pic1~5`에 저장합니다.
// @Description
// @Description  **미들웨어**: `ValidateFileUpload(5, 5)` — `files` 최대 5개, 이미지 ≤5MB. 이미지 전용(`thbnl` 첨부 시 400)
// @Description
// @Description  **요청 (multipart/form-data)**
// @Description  - `files`: 신규 이미지 (JPG/PNG/GIF/WEBP). 업로드 순서가 슬롯에 반영
// @Description  - `urls`: 유지할 기존 사진 URL JSON 배열 문자열. 신규만이면 `[]` (생략 시 `[]`)
// @Description
// @Description  **처리 순서**
// @Description  1. `urls` 기존 URL을 앞에서부터 채움 (`stat=3` 승인)
// @Description  2. `files` 신규 업로드 URL을 이어 붙임 (`stat=1` 요청중)
// @Description  3. 합계 최대 5장 → `sub_pic1`~`sub_picN`에 `{"url","stat"}` JSON 저장 (idx 없음, 나머지 슬롯 NULL)
// @Description
// @Description  **PrfPicInfo.stat**: 1=요청중, 2=승인대기, 3=승인, 4=거절/삭제
// @Description
// @Description  **응답 data**
// @Description  - `msg`: `"ok"`
// @Description  - `pics`: `[]PrfPicInfo` (url/stat만, 배열 순서=슬롯 1..N)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/prf/pic/upload
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - multipart: files=@a.jpg, files=@b.jpg, urls=["https://imagedelivery.net/.../public"]
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","pics":[{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public","stat":3},{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":1}]}}
// @Tags         Profile
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header    string  true   "Bearer access token"
// @Param        files          formData  file    true   "신규 프로필 이미지 (최대 5개, 이미지 전용)"
// @Param        urls           formData  string  false  "유지할 기존 URL JSON 배열. 예: [] 또는 [\"https://...\"]"
// @Success      200            {object}  protocol.PrfPicUploadResp "data.msg, data.pics"
// @Failure      400            {object}  map[string]interface{} "파일 없음, urls JSON 오류, 저장할 사진 없음, thbnl 첨부"
// @Failure      401            {object}  map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object}  map[string]interface{} "Cloudflare/DB 저장 실패"
// @Router       /story/v01/prf/pic/upload [post]
func (p *StoryController) UploadPrfPic(c *gin.Context) {
	// 프로필 사진 최대 5장. 기존 urls 유지 + 신규 files 업로드 → sub_pic1부터 순차 저장.
	// PrfPicInfo: {"url","stat"} — idx 없음(슬롯=컬럼 번호)
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	fileInfo, ok := c.Get("upFiles")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No files uploaded")
		return
	}
	files := fileInfo.([]*multipart.FileHeader)

	cldFlrInfos, err := utils.UploadCldFlrImg(files, p.cfg.Server.CfId, p.cfg.Server.CfToken)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload story media", err)
		return
	}

	urlsRaw := c.PostForm("urls")
	if urlsRaw == "" {
		urlsRaw = "[]"
	}
	var urls []string
	if err := json.Unmarshal([]byte(urlsRaw), &urls); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to unmarshal URLs", err)
		return
	}

	const maxSlots = 5
	prfPicInfos := make([]ptc.PrfPicInfo, 0, maxSlots)

	// 1) 클라이언트가 유지하는 기존 URL (앞에서부터 sub_pic1..)
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			continue
		}
		if len(prfPicInfos) >= maxSlots {
			break
		}
		prfPicInfos = append(prfPicInfos, ptc.PrfPicInfo{
			Url:  u,
			Stat: 3, // 기존 = 승인
		})
	}

	// 2) 신규 업로드 — files 순서 유지
	for _, f := range files {
		if len(prfPicInfos) >= maxSlots {
			break
		}
		mediaURL, found := (*cldFlrInfos)[f.Filename]
		if !found || mediaURL == "" {
			continue
		}
		prfPicInfos = append(prfPicInfos, ptc.PrfPicInfo{
			Url:  mediaURL,
			Stat: 1, // 신규 = 요청중
		})
	}

	if len(prfPicInfos) == 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No profile pictures to save")
		return
	}

	if err := p.sdb.UploadPrfPic(&prfPicInfos, uid64); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload profile picture", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "pics": prfPicInfos})
}

// DeletePrfPic godoc
// @Summary      프로필 사진 삭제
// @Description  JWT 인증 사용자의 `prf_info.sub_pic{slot}`를 삭제하고, 뒤 슬롯을 앞으로 당깁니다.
// @Description
// @Description  **idx**(path): 삭제할 슬롯 번호 1~5
// @Description  - 예: `2` → sub_pic2 삭제 후 sub_pic3→2, sub_pic4→3, sub_pic5→4, sub_pic5=NULL
// @Description  - 슬롯 JSON은 `{"url","stat"}`만 저장(idx 없음). 순번은 컬럼 위치
// @Description
// @Description  **응답**: `SendResponse` → `RespHeader` (`desc=success`, data 필드 없음)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/prf/pic/delete/2
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","desc":"success"}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "삭제할 프로필 사진 슬롯 (1~5)"
// @Success      200            {object} protocol.RespHeader "result=0, desc=success"
// @Failure      400            {object} map[string]interface{} "idx 누락/형식 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 삭제 실패 (슬롯 범위 포함)"
// @Router       /story/v01/prf/pic/delete/{idx} [post]
func (p *StoryController) DeletePrfPic(c *gin.Context) {
	// 프로필 사진 슬롯 삭제 후 뒤 슬롯을 앞으로 당김
	// PrfPicInfo.stat: 1=요청중, 2=승인대기, 3=승인, 4=거절/삭제
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	idx := c.Param("idx")
	if idx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "idx is required")
		return
	}

	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "idx is invalid", err)
		return
	}

	if err := p.sdb.DeletePrfPic(uid64, idxInt); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to delete profile picture", err)
		return
	}

	p.ctl.SendResponse(c, http.StatusOK, "success")
}

// GetPrfPicWaitingList godoc
// @Summary      프로필 사진 승인 대기 목록 (관리자)
// @Description  `prf_info.sub_pic1`~`sub_pic5` JSON 중 `stat`이 1(요청중) 또는 2(승인대기)인 항목을 전체 사용자 대상으로 페이징 조회합니다.
// @Description
// @Description  **path**
// @Description  - `page`: 페이지 번호 (1부터, `getPageAndLimit`)
// @Description  - `limit`: 페이지 크기 (1~100)
// @Description
// @Description  **응답 data** (`PrfPicWaitingListData`)
// @Description  - `msg`: `"ok"`
// @Description  - `total_count`: 대기 항목 전체 개수
// @Description  - `list`: `[]PrfPicWaitingItem` (`uid`, `nick`, `slot`, `url`, `stat`) — `slot`=sub_pic 컬럼 번호
// @Description
// @Description  **PrfPicWaitingItem.stat**: 1=요청중, 2=승인대기
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/prf/pic/waiting/1/20
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":2,"list":[{"uid":8697414060736839837,"nick":"건강한 꼬마 오렌지","slot":1,"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":1}]}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        page           path    int     true  "페이지 번호(1부터)"
// @Param        limit          path    int     true  "페이지 크기(1~100)"
// @Success      200            {object} protocol.PrfPicWaitingListResp "data.msg, total_count, list"
// @Failure      400            {object} map[string]interface{} "page/limit 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "조회 실패"
// @Router       /story/v01/prf/pic/waiting/{page}/{limit} [get]
func (p *StoryController) GetPrfPicWaitingList(c *gin.Context) {
	// admin only — 승인 대기(stat=1|2) 프로필 사진 페이징
	if _, exists := c.Get("user"); !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	list, total, err := p.sdb.GetPrfPicWaitingList(nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get waiting profile pictures", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":         "ok",
		"total_count": total,
		"list":        list,
	})
}

// SetPrfPicStat godoc
// @Summary      프로필 사진 승인 (관리자)
// @Description  승인 대기 목록(`GetPrfPicWaitingList`)의 특정 항목을 승인합니다.
// @Description  `prf_info.sub_pic{slot}` JSON의 `stat`을 **3(승인)** 으로 변경합니다.
// @Description
// @Description  **요청 body** (`SetPrfPicStatReq`)
// @Description  - `uid`: 대상 사용자 UID (필수, 0 불가)
// @Description  - `slot`: sub_pic 슬롯 1~5
// @Description
// @Description  **조건**: 해당 슬롯의 현재 `stat`이 1 또는 2일 때만 갱신
// @Description
// @Description  **응답 data** (`SetPrfPicStatData`)
// @Description  - `msg`: `"ok"`
// @Description  - `affected`: 갱신 행 수 (0이면 대상 없음/이미 처리됨)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/prf/pic/stat
// @Description  - Header: Authorization: Bearer {access_token}
// @Description  - Body: {"uid":8697414060736839837,"slot":1}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string                     true  "Bearer access token"
// @Param        request        body    protocol.SetPrfPicStatReq  true  "승인 대상 uid + slot"
// @Success      200            {object} protocol.SetPrfPicStatResp "data.msg, data.affected"
// @Failure      400            {object} map[string]interface{} "JSON/uid/slot 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Router       /story/v01/prf/pic/stat [post]
func (p *StoryController) SetPrfPicStat(c *gin.Context) {
	// admin only — 대기(stat=1|2) 프로필 사진을 승인(stat=3)
	if _, exists := c.Get("user"); !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req ptc.SetPrfPicStatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if req.Uid == 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "uid is required")
		return
	}
	if req.Slot < 1 || req.Slot > 5 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "slot must be 1-5")
		return
	}

	const approvedStat = 3
	affected, err := p.sdb.SetPrfPicStat(req.Uid, req.Slot, approvedStat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set profile picture stat", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":      "ok",
		"affected": affected,
	})
}

// SetMainPic godoc
// @Summary      프로필 대표사진 설정/삭제
// @Description  JWT 인증 사용자의 `prf_info.main_pic`을 설정하거나 삭제합니다.
// @Description
// @Description  **url**(path)
// @Description  - 값이 있으면 해당 URL을 대표사진으로 저장 (`IFNULL` → 빈 문자열 허용)
// @Description  - 비어 있으면 대표사진 삭제(`main_pic` 빈 문자열)
// @Description  - URL에 `/`가 포함되면 path-escape 필수 (예: `url.PathEscape`)
// @Description
// @Description  **응답 data**: `{"msg":"ok"}`
// @Description
// @Description  **Request example (설정)**
// @Description  - POST /story/v01/prf/set/mainpic/https%3A%2F%2Fimagedelivery.net%2FbhnuJ7hC7hq1zO__1yxVLg%2F5bca857f-75b1-4548-af93-ffa9fdd05600%2Fpublic
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        url            path    string  true  "대표사진 URL (PathEscape). 삭제 시 빈 값"
// @Success      200            {object} protocol.PrfUpdResp "data.msg=ok"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Router       /story/v01/prf/set/mainpic/{url} [post]
func (p *StoryController) SetMainPic(c *gin.Context) {
	// url이 null이면 대표사진 삭제, 있으면 해당 url로 대표사진 업데이트
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	url := c.Param("url")
	if err := p.sdb.SetMainPic(url, uid64); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set main picture", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok"})
}

// ------------- profile -------------

// ------------- profile list presentation-------------

// GetStoryDefaultList godoc
// @Summary      스토리 기본 목록 조회 (본인/타인)
// @Description  페이지네이션으로 스토리 리스트를 최신순으로 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - `tid` 없음: JWT 본인 UID로 조회 (`/list/{page}/{limit}`)
// @Description  - `tid` 있음: 해당 사용자 UID로 조회 (`/list/{page}/{limit}/{tid}`)
// @Description
// @Description  **응답 data**: `StoryListResp` 배열
// @Description  - `idx`, `uid`, `nick`, `stat`, `cnt_count`, `cnt_url`, `cnt_type`, `cnt_thumb`, `at_create`
// @Description  - 대표 미디어: `cnt_url`/`cnt_type`/`cnt_thumb` (전체 `str_img` 슬롯 맵은 포함하지 않음)
// @Description
// @Description  **stat**: 0=PUB, 1=FLW, 2=PAY, 3=PRV, 4=DEL, 5=RSV
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":[{"idx":15,"uid":403,"nick":"푸양","stat":1,"cnt_count":2,"cnt_url":"https://imag.net/.../public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-02T13:35:24Z"}]}
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true   "Bearer access token"
// @Param        page           path    int     true   "페이지 번호(1부터)"
// @Param        limit          path    int     true   "페이지 크기(기본 4, 최대 100)"
// @Param        tid            path    string  false  "조회 대상 UID (없으면 JWT 본인)"
// @Success      200            {object} map[string]interface{} "data: StoryListResp 배열"
// @Failure      400            {object} map[string]interface{} "page/limit/tid 잘못된 값"
// @Failure      401            {object} map[string]interface{} "tid 없을 때 JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "서버 에러"
// @Router       /story/v01/list/{page}/{limit} [get]
// @Router       /story/v01/list/{page}/{limit}/{tid} [get]
// @Example Request: GET /story/v01/list/1/10
// @Example Request: GET /story/v01/list/1/10/403
// @Example Response: {"result":0,"resultString":"Success","data":[{"idx":15,"uid":403,"nick":"푸양","stat":1,"cnt_count":2,"cnt_url":"https://imag.net/.../public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-02T13:35:24Z"}]}
func (p *StoryController) GetPrfStoryList(c *gin.Context) {
	// 홈 디폴트 리스트, 최신 등록순으로 전체 고객 스토리 리스트 출력, 해당 리스트 클릭시, 대상 사용자의 스토리로 이동해야함
	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	tid := c.Param("tid")
	var tuid64 uint64
	if tid != "" {
		// other user's story list
		tuid64, err = strconv.ParseUint(tid, 10, 64)
		if err != nil || tuid64 == 0 {
			p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
			return
		}
	} else {
		// my own story list
		user, exists := c.Get("user")
		if !exists {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}
		tuid64 = user.(*ptc.UserInfoResp).Uid
	}

	storyList, err := p.sdb.GetDefStoryList(tuid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, storyList)
}
