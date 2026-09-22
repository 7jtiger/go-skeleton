package controller

import (
	"encoding/json"
	"mime/multipart"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptc "ms-gateway/protocol"
	"net/http"
	"strconv"
	"strings"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	ptl "ms-gateway/protocol"

	"github.com/gin-gonic/gin"
)

type CldFlrInfo struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type StoryController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	adb *models.AccountDB
	hdb *models.HistoryDB
	sdb *models.StoryDB
	rdb *models.RedisDB
}

func NewStoryController(ctl *Controller, rep *models.Repositories) (*StoryController, error) {
	r := &StoryController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.hdb, &r.sdb, &r.rdb); err != nil {
		return nil, err
	}

	return r, nil
}

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
// @Description  JWT 인증 후 본인 또는 타인의 프로필 정보와 팔로워 수를 조회합니다.
// @Description
// @Description  **tid**: 조회 대상 UID. 비어 있으면 JWT 본인 UID 사용. 값이 있으면 해당 사용자 프로필 조회.
// @Description  **응답 data**: `prf_Info`(protocol.PrfInfo), `flw_cnt`(팔로워 수)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        tid            path    string  true  "조회 대상 UID (본인 조회 시에도 UID 전달)"
// @Success      200            {object} map[string]interface{} "data.prf_Info, data.flw_cnt"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "프로필/팔로워 조회 실패"
// @Example Request: GET /story/v01/prf/info
// @Example Request: GET /story/v01/prf/info/8697414060736839837
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"flw_cnt":0,"prf_Info":{"uid":8697414060736839837,"nick":"건강한 꼬마 오렌지","gender":"1","birth":"1990-01-01T00:00:00Z","age":"-2","area":"1","intro":"반가워요 큐피톡에서 만나요!","main_pic":""}}}
// @Router       /story/v01/prf/info [get] // 본인 프로필 조회
// @Router       /story/v01/prf/info/{tid} [get] // 타인 프로필 조회
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
// @Description  JWT 인증 사용자 본인의 보유 화살(포인트), 캐시, DM 미읽음 쪽지 합계를 조회합니다.
// @Description
// @Description  **응답 data**
// @Description  - `hold_point`: 보유 화살/포인트 (`user_info.hold_point`, NULL이면 0)
// @Description  - `hold_cash`: 보유 캐시 (`user_info.hold_cash`, NULL이면 0)
// @Description  - `msg_count`: 전체 DM 미읽음 쪽지 합계 (Redis)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Success      200            {object} map[string]interface{} "data.hold_point, data.hold_cash, data.msg_count"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "자산/쪽지 조회 실패"
// @Example Request: GET /story/v01/prf/assets
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"hold_point":0,"hold_cash":0,"msg_count":3}}
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
// @Description  **조회 대상**: 현재 라우트는 `/prf/pic/list` (본인 JWT UID). 코드상 `tid` path가 있으면 해당 UID 조회.
// @Description  **응답 data**: 비어 있지 않은 슬롯만 포함한 맵
// @Description  `{"1":{"url":"...","stat":3},"2":{"url":"...","stat":3}}` (키=`sub_pic` 번호)
// @Description
// @Description  **PrfPicInfo.stat**: 1=요청중, 2=승인대기, 3=승인, 4=거절/삭제
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Success      200            {object} map[string]interface{} "data: PrfPicMap"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "프로필 사진 조회 실패"
// @Example Request: GET /story/v01/prf/pic/list
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"1":{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":3},"2":{"url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","stat":3}}}
// @Router       /story/v01/prf/pic/list [get]
// @Router       /story/v01/prf/pic/list/{tid} [get]
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
// @Description  JWT 인증 후 본인 또는 타인의 최신 스토리 최대 4개의 대표 미디어 URL을 조회합니다.
// @Description
// @Description  **tid**: 조회 대상 UID. 비어 있으면 JWT 본인 UID 사용.
// @Description  **응답 data**
// @Description  - `str_imgs`: 이미지 스토리 대표 URL 맵 (key=story idx, value=URL)
// @Description  - `str_thmbls`: 동영상 스토리 썸네일 URL 맵 (key=story idx, value=thumb URL)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        tid            path    string  true  "조회 대상 UID (본인 조회 시에도 UID 전달)"
// @Success      200            {object} map[string]interface{} "data.str_imgs, data.str_thmbls"
// @Failure      400            {object} map[string]interface{} "tid 형식 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 목록 조회 실패"
// @Example Request: GET /story/v01/prf/story/lists
// @Example Request: GET /story/v01/prf/story/lists/8697414060736839837
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"str_imgs":{"12":"https://imagedelivery.net/xxx/public"},"str_thmbls":{"15":"https://imagedelivery.net/xxx/thumb/public"}}}
// @Router       /story/v01/prf/story/lists [get]
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
// @Description  - `nick`, `gender`, `age`, `area`, `intro`
// @Description  - `uid` / `birth` / `main_pic` 는 본 API에서 사용하지 않음 (대표사진은 `/prf/set/mainpic`)
// @Description  - `area`는 지역명 문자열 → 서버에서 area code로 변환
// @Description
// @Description  **응답 data**: `{"msg":"success"}`
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string            true  "Bearer access token. Example: Bearer {access_token}"
// @Param        request        body    protocol.PrfInfo  true  "수정할 프로필 필드 (부분 업데이트)"
// @Success      200            {object} map[string]interface{} "data.msg=success"
// @Failure      400            {object} map[string]interface{} "JSON 바인딩 실패"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Example Request: POST /story/v01/prf/upd
// @Example Request Header: Authorization: Bearer ...
// @Example Request Body: {"nick":"prf_test_nick","intro":"프로필 소개","area":"서울"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success"}}
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "success"})
}

// UploadPrfPic godoc
// @Summary      프로필 사진 업로드
// @Description  JWT 인증 사용자의 프로필 사진을 Cloudflare Images에 업로드하고 `prf_info.sub_pic1~5`에 저장합니다.
// @Description
// @Description  **요청 (multipart/form-data)**
// @Description  - `files`: 신규 이미지 (JPG/PNG/GIF/WEBP, 각 ≤5MB, 최대 5개). **thbnl 금지**(이미지 전용).
// @Description  - `urls`: 유지할 기존 사진 URL JSON 배열 문자열. 신규만이면 `[]`.
// @Description
// @Description  **처리 순서**
// @Description  1. `urls` 기존 URL을 앞에서부터 채움 (stat=3 승인)
// @Description  2. `files` 신규 업로드 URL을 이어 붙임 (stat=1 요청중)
// @Description  3. 합계 최대 5장. 순차적으로 `sub_pic1`~`sub_picN`에 `{"url","stat"}` JSON 저장 (idx 없음)
// @Description
// @Description  **PrfPicInfo.stat**: 1=요청중, 2=승인대기, 3=승인, 4=거절/삭제
// @Description  "stat" Enums(1(요청중),2(승인대기),3(승인),4(거절/삭제)) default(1)
// @Description
// @Description  **응답 data**: `msg`, `pics`([]PrfPicInfo — url/stat만, 배열 순서=슬롯)
// @Tags         Profile
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header    string  true   "Bearer access token. Example: Bearer {access_token}"
// @Param        files          formData  file    true   "신규 프로필 이미지 (최대 5개, 이미지 전용)"
// @Param        urls           formData  string  false  "유지할 기존 URL JSON 배열. 예: [] 또는 [\"https://...\"]"
// @Success      200            {object}  map[string]interface{} "data.msg, data.pics"
// @Failure      400            {object}  map[string]interface{} "파일 없음, urls JSON 오류, 저장할 사진 없음"
// @Failure      401            {object}  map[string]interface{} "인증 실패"
// @Failure      500            {object}  map[string]interface{} "Cloudflare/DB 저장 실패"
// @Example Request: POST /story/v01/prf/pic/upload
// @Example Request Header: Authorization: Bearer ...
// @Example curl: curl -X POST http://localhost:8080/story/v01/prf/pic/upload -H "Authorization: Bearer {jwt}" -F "files=@a.jpg" -F "files=@b.jpg" -F "urls=[\"https://imary.net/.../public\", \"https://imagy.net/.../public\"]"
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","pics":[{"url":"https://imagedelivery.net/.../public","stat":3},{"url":"https://imagedelivery.net/.../public","stat":1}]}}
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "success", "pics": prfPicInfos})
}

// DeletePrfPic godoc
// @Summary      프로필 사진 삭제
// @Description  JWT 인증 사용자의 `prf_info.sub_pic{slot}`를 삭제하고, 뒤 슬롯을 앞으로 당깁니다.
// @Description
// @Description  **idx**(path): 삭제할 슬롯 번호 1~5. 예: 2 → sub_pic2 삭제 후 sub_pic3→2, sub_pic4→3, sub_pic5→4, sub_pic5=NULL
// @Description  슬롯 JSON은 `{"url","stat"}`만 저장(idx 없음). 순번은 컬럼 위치.
// @Description  **응답**: `SendResponse` — `desc=success` (data 필드 없음)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        idx            path    int     true  "삭제할 프로필 사진 슬롯 (1~5)"
// @Success      200            {object} protocol.RespHeader "result=0, desc=success"
// @Failure      400            {object} map[string]interface{} "idx 누락/형식 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 삭제 실패"
// @Example Request: POST /story/v01/prf/pic/delete/2
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","desc":"success"}
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
// @Description  **응답 data**
// @Description  - `msg`: success
// @Description  - `total_count`: 대기 항목 전체 개수
// @Description  - `list`: []PrfPicWaitingItem (uid, nick, slot, url, stat) — slot=sub_pic 컬럼 번호
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        page           path    int     true  "페이지 번호(1부터)"
// @Param        limit          path    int     true  "페이지 크기(1~100)"
// @Success      200            {object} map[string]interface{} "msg, total_count, list"
// @Failure      400            {object} map[string]interface{} "page/limit 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "조회 실패"
// @Example Request: GET /story/v01/prf/pic/waiting/1/20
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","total_count":2,"list":[{"uid":8697414060736839837,"nick":"건강한 꼬마 오렌지","slot":1,"url":"https://imagedelivery.net/.../public","stat":1}]}}
// @Router       /story/v01/prf/pic/waiting/{page}/{limit} [get]
func (p *StoryController) GetPrfPicWaitingList(c *gin.Context) {
	// admin only — 승인 대기(stat=1|2) 프로필 사진 페이징
	if _, exists := c.Get("user"); !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	page, err := strconv.Atoi(c.Param("page"))
	if err != nil || page < 1 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page is invalid")
		return
	}
	limit, err := strconv.Atoi(c.Param("limit"))
	if err != nil || limit < 1 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "limit is invalid")
		return
	}

	list, total, err := p.sdb.GetPrfPicWaitingList(page, limit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get waiting profile pictures", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":         "success",
		"total_count": total,
		"list":        list,
	})
}

// SetPrfPicStat godoc
// @Summary      프로필 사진 승인 (관리자)
// @Description  승인 대기 목록(`GetPrfPicWaitingList`)의 특정 항목을 승인합니다. `prf_info.sub_pic{slot}` JSON의 `stat`을 **3(승인)** 으로 변경합니다.
// @Description
// @Description  **요청 body**: `uid`(대상 사용자), `slot`(1~5, sub_pic 슬롯)
// @Description  **조건**: 해당 슬롯의 현재 stat이 1 또는 2일 때만 갱신
// @Description  **응답 data**: `msg`, `affected` (0이면 대상 없음/이미 처리됨)
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string                 true  "Bearer access token. Example: Bearer {access_token}"
// @Param        request        body    protocol.SetPrfPicStatReq  true  "승인 대상 uid + slot"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "JSON/파라미터 오류"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Example Request: POST /story/v01/prf/pic/stat
// @Example Request Header: Authorization: Bearer ...
// @Example Request Body: {"uid":8697414060736839837,"slot":1}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","affected":1}}
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
		"msg":      "success",
		"affected": affected,
	})
}

// SetMainPic godoc
// @Summary      프로필 대표사진 설정/삭제
// @Description  JWT 인증 사용자의 `prf_info.main_pic`을 설정하거나 삭제합니다.
// @Description
// @Description  **url path**
// @Description  - 값이 있으면 해당 URL을 대표사진으로 저장
// @Description  - 비어 있으면 대표사진 삭제(`main_pic` 빈 문자열)
// @Description  - URL에 `/`가 포함되면 path-escape 필요 (예: `url.PathEscape`)
// @Description
// @Description  **응답 data**: `{"msg":"success"}`
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        url            path    string  true  "대표사진 URL (삭제 시 빈 값 또는 placeholder)"
// @Success      200            {object} map[string]interface{} "data.msg=success"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 업데이트 실패"
// @Example Request: POST /story/v01/prf/set/mainpic/https%3A%2F%2Fimagedelivery.net%2Fxxx%2Fpublic
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success"}}
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "success"})
}

// ------------- profile -------------

// 현재 접속중인 이성 기준, 리스트 수집
// 해당 리스트별 스토리 최신 1개 리스트 출력
func (p *StoryController) GetStoryHomeList(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is required")
	}

	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is required")
		return
	}

	storyList, err := p.sdb.GetDefStoryList(uidInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, storyList)
}

// GetStoryList godoc
// @Summary Get story list by user ID
// @Description 디폴트, 최신순 스토리 리스트 출력
// @Tags Story
// @Accept json
// @Produce json
// @Param uid path string true "User ID"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved story list - str_img is unified media slot map"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid or missing uid"
// @Failure 500 {object} protocol.RespHeader "Internal server error - failed to get story list"
// @Router /story/v01/list/{uid} [get]
// @Example Request: GET /story/v01/list/5817
// @Example Response: {"result":0,"resultString":"Success","data":[{"idx":2,"nick":"testnick","str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"vdo","url":"https://.../video.m3u8","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2026-02-02T07:27:44Z"}]}
func (p *StoryController) GetStoryDefaultList(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is required")
		return
	}

	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is required")
		return
	}

	storyList, err := p.sdb.GetDefStoryList(uidInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, storyList)
}

// @Param type query string false "타입(img=이미지, vdo=비디오, all=이미지+비디오, rsv=예약). 기본값: img"

// GetStoryConditionList godoc
// @Summary 스토리(Story) 조건부 리스트 조회
// @Description 다양한 조건(area/stat/type/order/gen/page/limit)으로 스토리 리스트를 조회합니다.
// @Tags Story
// @Accept  json
// @Produce json
// @Param area query string false "지역" Enums(all,seoul,gyeonggi,incheon,busan,daejeon/sejong/chungnam,chungbuk/cheonju/chungju,daegu/gyeongbuk,gyeongnam/ulsan,gwangju/jeonnam,jeonbuk/jeonju,gangwon/chuncheon,jeju) default(all)
// @Param stat query string false "상태" Enums(pub(전체공개),flw(팔로워전용),pay(유료),prv(비공개),del(삭제),rsv(예약)) default(pub)
// @Param type query string false "타입" Enums(img(이미지),vdo(비디오),all(이미지+비디오),rsv(예약)) default(img)
// @Param order query string false "정렬" Enums(new(최신순),cmt(댓글많음),god(좋아요많음),view(조회수많음),rsv(예약)) default(new)
// @Param gen query string false "성별" Enums(0(여성),1(남성),2(예약)) default(1)
// @Param page query int false "페이지 번호(1부터 시작)" default(1)
// @Param limit query int false "페이지 당 데이터 개수" default(10)
// @Success 200 {object} protocol.RespDataHeader "성공적으로 스토리 리스트를 반환합니다. story_home_list 필드는 인덱스별 이미지 URL 정보를 포함합니다."
// @Failure 400 {object} protocol.RespHeader "잘못된 파라미터 입력 시 반환"
// @Failure 500 {object} protocol.RespHeader "서버 에러 시 반환"
// @Router /story/v01/condition/list/{area}/{stat}/{type}/{order}/{gen}/{page}/{limit} [get]
// @Example Request: GET /story/v01/condition/list/seoul/pub/img/new/0/1/10
// @Example Response: {"result":0,"resultString":"Success","data":{"story_home_list":{"{story idx}":"{img_url}","{story idx}":"image url","{story index}":"image url", ....}}}
// @Example Response: {"result":0,"resultString":"Success","data":{"story_home_list":{"1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","2":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","3":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5512db27-6340-43cd-b5ab-85d7b9bd4800/public"}}}
func (p *StoryController) GetStoryConditionList(c *gin.Context) {
	area := c.DefaultQuery("area", "all")
	stat := c.DefaultQuery("stat", "pub")
	mtype := c.DefaultQuery("type", "img")
	order := c.DefaultQuery("order", "new")
	gen := c.DefaultQuery("gen", "1")
	nPage := ptc.CvtParamAtoi(c.Query("page"), 1)   //defaultQuery "1"
	nLimit := ptc.CvtParamAtoi(c.Query("limit"), 4) //defaultQuery "10"

	args := []interface{}{}
	conds := []string{}

	nArea := ptl.GetAreaCode(area)
	if nArea > 0 {
		conds = append(conds, "area = ?")
		args = append(args, nArea)
	}

	conds = append(conds, "gender = ?")
	nGen, err := strconv.Atoi(gen)
	if err != nil {
		nGen = 1
	}
	args = append(args, nGen)

	nMType := ptl.GetTypeCode(mtype)
	if nMType != 2 {
		conds = append(conds, "type = ?")
		args = append(args, nMType)
	}

	conds = append(conds, "stat = ?")
	nStat := ptl.GetStatCode(stat)
	args = append(args, nStat)

	// 요청 사용자 cutout 목록은 story feed에서 제외한다.
	user, exists := c.Get("user")
	if exists {
		uid64 := user.(*ptc.UserInfoResp).Uid
		cutoutTids, err := p.sdb.GetActiveCutoutTIDs(uid64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get cutout list", err)
			return
		}
		if len(cutoutTids) > 0 {
			placeholders := make([]string, len(cutoutTids))
			for i, tid := range cutoutTids {
				placeholders[i] = "?"
				args = append(args, tid)
			}
			conds = append(conds, "uid NOT IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	orderQuery := ptl.GetOrderQuery(order)
	offset := (nPage - 1) * nLimit
	args = append(args, nLimit, offset)
	storyList, err := p.sdb.GetCondStoryList(conds, orderQuery, args)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"story_home_list": storyList})
}

// -------------------- user cut out ---------------------------------

// SetCutoutUser godoc
// @Summary      스토리 cutout 설정
// @Description  JWT 인증 사용자가 특정 사용자(tid)를 스토리 feed에서 숨김 처리합니다. 서비스 전체 차단(user_block)과 별개이며, 대상 프로필 스냅샷(tnick/tgen/tbirth/tsp_intro/tarea/tthmb_pic)을 str_cutout에 저장합니다. 자기 자신(tid=본인 uid) cutout은 불가합니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        tid            path    string  true  "cutout 대상 사용자 UID"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid invalid / self cutout / target user not found"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "서버 에러"
// @Example Request: POST /story/v01/cutout/set/{tid}
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","affected":1}}
// @Router       /story/v01/cutout/set/{tid} [post]
func (p *StoryController) SetCutoutUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	tid64, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil || tid64 == 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
		return
	}
	if uid64 == tid64 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "self cutout is not allowed")
		return
	}

	tuser, err := p.adb.GetUserInfoByUID(tid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "target user not found", err)
		return
	}
	nGen, err := strconv.Atoi(tuser.Gender)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "target user gender is invalid", err)
		return
	}
	cutUser := &ptc.CutoutUserItem{
		Tid:      tid64,
		Stat:     1,
		Tnick:    tuser.Nick,
		Tgen:     nGen,
		Tbirth:   utils.Time2StrDay(tuser.Birth),
		TspIntro: tuser.SPIntro,
		Tarea:    ptc.GetAreaCode(tuser.Area),
		TthmbPic: tuser.ThumbPic,
	}

	affected, err := p.sdb.SetCutoutUser(uid64, cutUser)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set cutout user", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":      "success",
		"affected": affected,
	})
}

// UnsetCutoutUser godoc
// @Summary      스토리 cutout 해제
// @Description  JWT 인증 사용자가 특정 사용자(tid)에 대한 스토리 feed 숨김을 해제합니다(str_cutout.stat=0).
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        tid            path    string  true  "cutout 해제 대상 사용자 UID"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid invalid"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "서버 에러"
// @Example Request: POST /story/v01/cutout/cancel/{tid}
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","affected":1}}
// @Router       /story/v01/cutout/cancel/{tid} [post]
func (p *StoryController) UnsetCutoutUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	tid64, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil || tid64 == 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
		return
	}

	affected, err := p.sdb.UnsetCutoutUser(uid64, tid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to unset cutout user", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":      "success",
		"affected": affected,
	})
}

// GetCutoutCount godoc
// @Summary      스토리 cutout 개수 조회
// @Description  JWT 인증 사용자의 활성 cutout(str_cutout.stat=1) 개수를 반환합니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Success      200            {object} map[string]interface{} "msg, count"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "서버 에러"
// @Example Request: POST /story/v01/cutout/count
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","count":3}}
// @Router       /story/v01/cutout/count [post]
func (p *StoryController) GetCutoutCount(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	count, err := p.sdb.GetCutoutCount(uid64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get cutout count", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":   "success",
		"count": count,
	})
}

// GetCutoutList godoc
// @Summary      스토리 cutout 목록 조회
// @Description  JWT 인증 사용자의 활성 cutout 목록을 페이지네이션으로 조회합니다. list 항목은 CutoutUserItem(idx, tid, stat, tnick, tgen, tbirth, tsp_intro, tarea, tthmb_pic, at_crt, at_upd) 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token. Example: Bearer {access_token}"
// @Param        page           path    int     true  "페이지 번호(1부터)"
// @Param        limit          path    int     true  "페이지 크기(기본 4, 최대 100)"
// @Success      200            {object} map[string]interface{} "msg, total_count, list"
// @Failure      400            {object} map[string]interface{} "잘못된 파라미터"
// @Failure      401            {object} map[string]interface{} "인증 실패"
// @Failure      500            {object} map[string]interface{} "서버 에러"
// @Example Request: GET /story/v01/cutout/list/1/10
// @Example Request Header: Authorization: Bearer ...
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"success","total_count":1,"list":[{"idx":12,"tid":4033287471439576593,"stat":1,"tnick":"qqqq11111","tgen":1,"tbirth":"1990","tsp_intro":"hello","tarea":1,"tthmb_pic":"https://example.com/thumb.jpg","at_crt":"2026-06-23T10:00:00Z","at_upd":"2026-06-23T10:00:00Z"}]}}
// @Example Response: {"result":0,"resultString":"Success","data":{"list":[{"idx":1,"tid":4033287471439576593,"stat":1,"tnick":"푸른 아름다운 양","tgen":1,"tbirth":"1991-12-18T00:00:00Z","age":34,"tsp_intro":"반가워요 큐피톡에서 만나요!","tarea":1,"tthmb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","at_crt":"2026-06-23T14:30:15Z","at_upd":"2026-06-23T14:30:15Z"}],"msg":"success","total_count":1}}
// @Router       /story/v01/cutout/list/{page}/{limit} [get]
func (p *StoryController) GetCutoutList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	pageInt := ptc.CvtParamAtoi(c.Param("page"), 1)
	limitInt := ptc.CvtParamAtoi(c.Param("limit"), 4)
	if limitInt > 100 {
		limitInt = 100
	}

	list, totalCount, err := p.sdb.GetCutoutList(uid64, pageInt, limitInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get cutout list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"msg":         "success",
		"total_count": totalCount,
		"list":        list,
	})
}

// -------------------- user cut out ---------------------------------

// GetStoryDetail godoc
// @Summary Get story detail by story index
// @Description Retrieve detailed information of a specific story including comments
// @Tags Story
// @Accept json
// @Produce json
// @Param idx path string true "Story Index"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved story detail - str_img is unified media slot map"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid or missing idx"
// @Failure 500 {object} protocol.RespHeader "Internal server error - failed to get story detail or comment list"
// @Router /story/v01/detail/{idx} [get]
// @Example Request: GET /story/v01/detail/3
// @Example Response: {"result":0,"resultString":"Success","data":{"story":{"nick":"testnick","body":"test story body","str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"vdo","url":"https://.../video.m3u8","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2024-01-01T12:00:00Z"}}}
func (p *StoryController) GetStoryDetail(c *gin.Context) {
	idx := c.Param("idx")
	if idx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	story, err := p.sdb.GetStory(idxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story detail", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"story": story})
}

// GetStrCmtDetail godoc
// @Summary Get story comment details (list)
// @Description Retrieve a paginated list of comments for a specific story. Each comment includes idx, wuid, nick, body, at_create, etc.
// @Tags Story
// @Accept json
// @Produce json
// @Param idx path string true "Story Index"
// @Param page path string true "Page number (starting from 1)"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved story comments - returns {comment_list, total_count}"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid or missing idx/page"
// @Failure 500 {object} protocol.RespHeader "Internal server error - failed to get story comments"
// @Router /story/v01/comment/{idx}/{page} [get]
// @Example Request: GET /story/v01/comment/3/1
// @Example Response: {"code":200,"message":"success","data":{"comment":[{"idx":2,"str_idx":3,"wuid":77645423236541,"nick":"test","thumb_url":"assdd","wgender":"","wage":"","warea":"","body":"111111 test comment body","stat":1,"at_create":"2026-02-02T13:50:06Z","at_update":"2026-02-02T13:50:06Z"}],"total_count":52}}
func (p *StoryController) GetStrCmtDetail(c *gin.Context) {
	idx := c.Param("idx")
	page := c.Param("page")

	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	nPage, err := strconv.Atoi(page)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Page is required")
		return
	}

	commentList, totalCount, err := p.sdb.GetStrCmtDetail(idxInt, nPage)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get str comment list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"comment":     commentList,
		"total_count": totalCount,
	})
}

// UploadStoryContent godoc
// @Summary 스토리 미디어 업로드 (Cloudflare Images / Stream)
// @Description JWT 인증 후 스토리용 미디어만 Cloudflare에 업로드합니다. DB 저장 없이 **통합 media** URL만 반환합니다.
// @Description
// @Description **용량**: 이미지·썸네일 각 5MB 이하, 동영상(MP4) 각 100MB 이하, files 최대 5개.
// @Description
// @Description **응답 data.media**: 업로드 순서 슬롯 맵
// @Description `{"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...m3u8","thumb":"..."}}`
// @Description - `type`: `img` | `vdo`
// @Description - `url`: 본편 URL
// @Description - `thumb`: 동영상 썸네일 (vdo만)
// @Description
// @Description **동영상 포함**: `files`의 MP4마다 `thbnl` 썸네일 1장 필수(순서 1:1).
// @Tags Story
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param files formData file true "미디어 (이미지 JPG/PNG/GIF/WEBP ≤5MB 또는 MP4 ≤100MB, 최대 5개)"
// @Param thbnl formData file false "동영상 썸네일 (이미지 ≤5MB). 동영상 있을 때만 필수, 개수·순서 1:1"
// @Success 200 {object} map[string]interface{} "{\"msg\":\"ok\",\"media\":{...}}"
// @Failure 400 {object} map[string]interface{} "파일 없음, 용량 초과, thbnl 불일치 등"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Failure 500 {object} map[string]interface{} "Cloudflare 업로드 실패"
// @Router /story/v01/upload [post]
// @Example 이미지 curl -X POST http://localhost:8080/story/v01/upload -H "Authorization: Bearer {jwt}" -F "files=@a.jpg" -F "files=@b.jpg"
// @Example 동영상 curl -X POST http://localhost:8080/story/v01/upload -H "Authorization: Bearer {jwt}" -F "files=@v1.mp4" -F "thbnl=@t1.jpg"
// @Example Response {"result":0,"resultString":"Success","data":{"msg":"ok","media":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"vdo","url":"https://.../manifest/video.m3u8","thumb":"https://imagedelivery.net/.../public"}}}}
func (p *StoryController) UploadStoryContent(c *gin.Context) {
	fileInfo, ok := c.Get("upFiles")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No files uploaded")
		return
	}

	files := fileInfo.([]*multipart.FileHeader)

	var thumbnails []*multipart.FileHeader
	if thumbInfo, exists := c.Get("upThumb"); exists {
		thumbnails, _ = thumbInfo.([]*multipart.FileHeader)
	}

	videoCount := 0
	for _, f := range files {
		if utils.GetFileType(f.Filename) == 1 {
			videoCount++
		}
	}
	if videoCount > len(thumbnails) {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Thumbnail is required for each video"+" videoCount: "+strconv.Itoa(videoCount)+" thumbnailsCount: "+strconv.Itoa(len(thumbnails)))
		return
	}

	var (
		cldFlrInfos *map[string]string
		thumbInfos  *map[string]string
		err         error
	)

	if videoCount > 0 {
		cldFlrInfos, err = utils.UpTotalCldFlr(files, p.cfg.Server.CfId, p.cfg.Server.CfToken)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload story media", err)
			return
		}
		if len(thumbnails) > 0 {
			thumbInfos, err = utils.UploadCldFlrImg(thumbnails, p.cfg.Server.CfId, p.cfg.Server.CfToken)
			if err != nil {
				p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload thumbnail", err)
				return
			}
		}
	} else {
		cldFlrInfos, err = utils.UploadCldFlrImg(files, p.cfg.Server.CfId, p.cfg.Server.CfToken)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload story media", err)
			return
		}
	}

	media := make(ptl.StoryStrImg, len(files))
	thumbIdx := 0
	for i, f := range files {
		slot := strconv.Itoa(i + 1)
		url := (*cldFlrInfos)[f.Filename]
		switch utils.GetFileType(f.Filename) {
		case 1: // video
			item := ptl.StoryMediaItem{Type: ptl.StoryMediaTypeVdo, URL: url}
			if thumbInfos != nil && thumbIdx < len(thumbnails) {
				thumbFile := thumbnails[thumbIdx]
				item.Thumb = (*thumbInfos)[thumbFile.Filename]
				thumbIdx++
			}
			media[slot] = item
		default: // image
			media[slot] = ptl.StoryMediaItem{Type: ptl.StoryMediaTypeImg, URL: url}
		}
	}

	raw, _, err := models.BuildStrImgForDB(media)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := models.ParseStoryMedia(string(raw))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build media response", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "media": out})
}

// CreateStory godoc
// @Summary 스토리 생성 (미디어 URL + 본문 DB 저장)
// @Description JWT 인증 사용자의 스토리를 DB에 저장합니다. 미디어 파일은 직접 받지 않고, **업로드 API의 data.media**를 전달합니다.
// @Description
// @Description **권장 플로우**
// @Description 1. `POST /story/v01/upload` — Cloudflare 업로드 → `data.media` 수신
// @Description 2. `POST /story/v01/create` — JSON body에 `media`(upload 응답 객체) 전달
// @Description
// @Description **media 슬롯**: 최대 5개. `{"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...","thumb":"..."}}`
// @Description **stat**: 0=del, 1=pub, 2=private(팔로우공개), 3=limit(유료공개), 4=resv
// @Description
// @Tags Story
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body protocol.CreateStoryReq true "스토리 생성 요청"
// @Success 200 {object} map[string]interface{} "성공: {\"msg\":\"ok\",\"idx\":5}"
// @Failure 400 {object} map[string]interface{} "JSON 파싱 실패, media 누락, 프로필 필드 부족"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Failure 500 {object} map[string]interface{} "DB 저장 실패"
// @Router /story/v01/create [post]
// @Example Request {"stat":1,"sbody":"오늘의 스토리","media":{"1":{"type":"img","url":"https://imagedelivery.net/xxx/public"},"2":{"type":"vdo","url":"https://.../video.m3u8","thumb":"https://imagedelivery.net/xxx/public"}}}
// @Example Response {"result":0,"resultString":"Success","data":{"msg":"ok","idx":5}}
func (p *StoryController) CreateStory(c *gin.Context) {
	userInfo, ok := c.Get("user")
	if !ok {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	u := userInfo.(*ptl.UserInfoResp)

	var sinfo ptl.CreateStoryReq
	if err := c.ShouldBindJSON(&sinfo); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if err := models.ValidateStoryMedia(sinfo.Media); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
		return
	}

	birth := utils.Time2StrDay(u.Birth)
	if birth == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, birth is required")
		return
	}
	if u.Nick == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, nick is required")
		return
	}

	nGen, err := strconv.Atoi(u.Gender)
	if err != nil || nGen < 0 || nGen > 2 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, gender is required", err)
		return
	}

	area := ptl.GetAreaCode(u.Area)
	if area <= 0 {
		area = 0
	}

	simg := &ptl.StoryImage{
		Uid:    u.Uid,
		Nick:   u.Nick,
		Birth:  birth,
		Area:   area,
		Gender: nGen,
		Body:   sinfo.Sbody,
		Stat:   sinfo.Stat,
	}

	lastID, err := p.sdb.SaveStory(simg, sinfo.Media)
	if err != nil {
		log.Error("Failed to set story: %v", err)
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set story", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "idx": lastID})
}

/*
// func (p *StoryController) uploadCldFlr(files []*multipart.FileHeader, putInfo map[string]map[string]string) (*[]CldFlrInfo, error) {
func (p *StoryController) uploadCldFlr(files []*multipart.FileHeader) (*map[string]string, error) {
	accountID := "3a5160a653bf6175ea5c5b24ee01e344"
	apiToken := "lhmeGe8y_2zCunsLcr8g4eUxJdPTkPGNMI_pNEWU"

	// cldFlrInfos := make([]CldFlrInfo, 4)
	cldFlrInfos := make(map[string]string, 4)
	for _, file := range files {
		// Open the file
		src, err := file.Open()
		if err != nil {
			log.Error("Failed to open file: %v", err)
			return nil, err
		}
		defer src.Close()

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file to form
		part, err := writer.CreateFormFile("file", file.Filename)
		if err != nil {
			log.Error("Failed to create form file: %v", err)
			return nil, err
		}

		if _, err := io.Copy(part, src); err != nil {
			log.Error("Failed to copy file: %v", err)
			return nil, err
		}

		writer.Close()

		// Create request
		url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", accountID)
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			log.Error("Failed to create request: %v", err)
			return nil, err
		}

		// Set headers
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Error("Failed to upload to Cloudflare: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		// Read response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("Failed to read response body: %v", err)
			return nil, err
		}

		// cldFlrInfo := CldFlrInfo{}
		// Parse response to extract success, filename, and variants
		var tempResponse struct {
			Success bool `json:"success"`
			Result  struct {
				Filename string   `json:"filename"`
				Variants []string `json:"variants"`
			} `json:"result"`
		}

		if err := json.Unmarshal(bodyBytes, &tempResponse); err != nil {
			log.Error("Failed to parse response for extraction: %v", err)
			return nil, err
		}

		cldFlrInfos[file.Filename] = tempResponse.Result.Variants[0]

		//{true {bc1.jpg [https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/6e8b29d4-4378-499a-7334-bd019dc25900/public]}}
		// 2026-01-26T22:33:00.075+0900	INFO	info	{"Info": "Response body: %s{\n  \"result\": {\n    \"id\": \"100320e3-fb16-490a-7b21-a931feec6400\",\n    \"filename\": \"bc1.jpg\",\n    \"uploaded\": \"2026-01-26T13:30:51.197Z\",\n    \"requireSignedURLs\": false,\n    \"variants\": [\n      \"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/100320e3-fb16-490a-7b21-a931feec6400/public\"\n    ]\n  },\n  \"success\": true,\n  \"errors\": [],\n  \"messages\": []\n}"}
		log.Info("Response body: %s", string(bodyBytes))

		if resp.StatusCode != http.StatusOK {
			// bodyBytes, _ := io.ReadAll(resp.Body)
			log.Error("Cloudflare upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("cloudflare upload failed with status %d", resp.StatusCode)
		}
	}

	return &cldFlrInfos, nil
}
*/

// UpdateStoryStat godoc
// @Summary Update story status
// @Description Update the status of a story
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.UpdateStrStatReq true "Update story status request stat=0 : del, stat=1 : pub, stat=2 : private, stat=3 : limit, stat=4 :resv"
// @Success 200 {object} map[string]interface{} "Successfully updated story stat"
// @Failure 400 {object} map[string]interface{} "Bad request - Invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/updstat [post]
// @Example Request: POST /story/v01/updstat
// @Example Request Body: {"idx":"3","stat":"2"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully updated story stat","affected":1}}
func (p *StoryController) UpdateStoryStat(c *gin.Context) {
	var req ptl.UpdateStrStatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Idx == "" || req.Stat == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx and Stat are required")
		return
	}

	strIdx, err := strconv.Atoi(req.Idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	nstat, err := strconv.Atoi(req.Stat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Stat is required")
		return
	}

	affected, err := p.sdb.UpdateStoryStat(strIdx, nstat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story stat", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully updated story stat", "affected": affected})
}

// UpdateStrBody godoc
// @Summary Update story body
// @Description Update the body content of a story
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.UpdateStrBodyReq true "Update story body request"
// @Success 200 {object} map[string]interface{} "Successfully updated story body"
// @Failure 400 {object} map[string]interface{} "Bad request - Invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/updbody [post]
// @Example Request: POST /story/v01/updbody
// @Example Request Body: {"idx":"3","body":"updated story body content"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully updated story body","affected":1}}
func (p *StoryController) UpdateStrBody(c *gin.Context) {
	var req ptl.UpdateStrBodyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Idx == "" || req.Body == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx and Body are required")
		return
	}

	strIdx, err := strconv.Atoi(req.Idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StrIdx is required")
		return
	}

	affected, err := p.sdb.UpdateStrBody(strIdx, req.Body)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update str body comment", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully updated str body comment", "affected": affected})
}

// DeleteStrPic godoc
// @Summary Delete a story picture
// @Description Delete a media slot from story str_img and move it to backup. Unified format reindexes remaining slots 1..N.
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.DeleteStrPicReq true "Delete story picture request"
// @Success 200 {object} map[string]interface{} "Successfully deleted story picture"
// @Failure 400 {object} map[string]interface{} "Bad request - Invalid parameters or empty picture list"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/delpic [post]
// @Example Request: POST /story/v01/delpic
// @Example Request Body: {"idx":"3","pic_idx":"1"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully deleted story picture","affected":1}}
func (p *StoryController) DeleteStrPic(c *gin.Context) {

	var req ptl.DeleteStrPicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Idx == "" || req.Pic == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx and Pic are required")
		return
	}

	strIdxInt, err := strconv.Atoi(req.Idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StrIdx is required")
		return
	}

	sPic := req.Pic
	strImg, strImgBak, err := p.sdb.GetStrPicList(strIdxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get str pic list", err)
		return
	}
	if strImg == nil && strImgBak == nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Str pic list is empty")
		return
	}

	media, err := models.ParseStoryMedia(string(strImg))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "invalid str_img media format", err)
		return
	}

	item, ok := media[sPic]
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "picture slot not found")
		return
	}

	bakMedia := make(ptl.StoryStrImg)
	if len(strImgBak) > 0 {
		if parsedBak, bakErr := models.LoadStoryStrImg(strImgBak); bakErr == nil {
			bakMedia = parsedBak
		}
	}
	bakMedia[sPic] = item
	media = models.ReindexStoryMediaAfterDelete(media, sPic)

	var mImgBytes json.RawMessage
	if len(media) == 0 {
		mImgBytes = json.RawMessage("{}")
	} else {
		mImgBytes, _, err = models.BuildStrImgForDB(media)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build str_img", err)
			return
		}
	}
	mImgBakBytes, err := json.Marshal(bakMedia)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to marshal mImgBak", err)
		return
	}
	affected, err := p.sdb.DeleteStrPic(strIdxInt, mImgBytes, mImgBakBytes)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update str pic", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully deleted story picture", "affected": affected})
}

/* ------------- comment -------------*/

// CreateStrComment godoc
// @Summary Create a story comment
// @Description Create a new comment for a story
// @Tags Story
// @Accept json
// @Produce json
// @Param str_idx query string true "Story index"
// @Param wuid query string true "Writer user ID"
// @Param nick query string true "Nickname"
// @Param stat query string true "Status" Enums(0(default),1(private),2(reserved),3(reserved),4(deleted))
// @Param body query string true "Comment body (max 256 characters)"
// @Param data body protocol.StrCmtCreateReq true "Comment data"
// @Success 200 {object} map[string]interface{} "Successfully created str comment"
// @Failure 400 {object} map[string]interface{} "Bad request - missing or invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/comment/create [post]
// @Example Request: POST /story/v01/comment/create
// @Example Request Body: {"str_idx":"3","wuid":"77645423236541","nick":"test","stat":"1","body":"111111 test comment body"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully created str comment","lastID":1}}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully created str comment", "lastID": lastID})
}

// UpdateStrStatComment godoc
// @Summary Update story comment status
// @Description Update the status of a story comment "Status" Enums(0(default),1(private),2(reserved),3(reserved),4(deleted))
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.StrCmtUpdStatReq true "Comment status update request"
// @Success 200 {object} map[string]interface{} "Successfully updated str stat comment"
// @Failure 400 {object} map[string]interface{} "Bad request - Invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/comment/updstat [post]
// @Example Request: POST /story/v01/comment/updstat
// @Example Request Body: {"idx":"3","stat":"2"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully updated str stat comment","affected":1}}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully updated str stat comment", "affected": affected})
}

// UpdateStrBodyComment godoc
// @Summary Update story comment body
// @Description Update the body content of a story comment
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.StrCmtUpdBodyReq true "Comment body update request"
// @Success 200 {object} map[string]interface{} "Successfully updated str body comment"
// @Failure 400 {object} map[string]interface{} "Bad request - Invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/comment/updbody [post]
// @Example Request: POST /story/v01/comment/updbody
// @Example Request Body: {"idx":"3","body":"updated comment body content"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"Successfully updated str body comment","affected":1}}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully updated str body comment", "affected": affected})
}

// ToggleStoryLike godoc
// @Summary Toggle story like
// @Description Toggle like/unlike for a story and return latest like count
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.StoryLikeToggleReq true "Story like toggle request"
// @Success 200 {object} map[string]interface{} "Successfully toggled story like"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid parameters"
// @Failure 403 {object} map[string]interface{} "Forbidden - blocked relation"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/like/toggle [post]
func (p *StoryController) ToggleStoryLike(c *gin.Context) {
	var req ptl.StoryLikeToggleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if req.StoryIdx == "" || req.Uid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StoryIdx and Uid are required")
		return
	}

	storyIdx, err := strconv.Atoi(req.StoryIdx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StoryIdx is invalid")
		return
	}
	uid, err := strconv.ParseUint(req.Uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is invalid")
		return
	}
	/*
	   	ownerUid, err := p.sdb.GetStoryOwnerUID(storyIdx)
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story owner", err)
	   		return
	   	}
	    	isBlocked, err := p.sdb.IsBlockedPair(uid, ownerUid)
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check block relation", err)
	   		return
	   	}
	   	if isBlocked {
	   		p.ctl.SimpleError(c, http.StatusForbidden, "blocked relation")
	   		return
	   	}
	*/
	liked, likeCount, err := p.sdb.ToggleStoryLikeTx(storyIdx, uid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to toggle story like", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully toggled story like", "liked": liked, "like_count": likeCount})
}

// FollowUser godoc
// @Summary      팔로우(팔로잉) 설정
// @Description  JWT 인증된 사용자가 지정 대상(tid)의 팔로우를 생성하거나 다시 활성화합니다. 자기 자신은 팔로우할 수 없습니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Param        tid     path    string true  "팔로우할 대상 유저의 uid (followeeUid)"
// @Success      200     {object} map[string]interface{} "msg: 성공 메시지, affected: 실제 반영된 row count"
// @Failure      400     {object} map[string]interface{} "잘못된 요청 파라미터 또는 자기자신 팔로우 시도"
// @Failure      401     {object} map[string]interface{} "인증되지 않은 사용자"
// @Failure      500     {object} map[string]interface{} "내부 서버 에러"
// @Router       /story/v01/follow/set/{tid} [post]
func (p *StoryController) FollowUser(c *gin.Context) {
	// followerUid = jwt uid, followeeUid = path param tid
	// follower 내가 팔로우하는 사람
	// followee 내가 팔로우 받는 사람
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	followerUid := user.(*ptc.UserInfoResp).Uid

	followeeUid, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
		return
	}
	if followerUid == followeeUid {
		p.ctl.SimpleError(c, http.StatusBadRequest, "self follow is not allowed")
		return
	}
	// 블락 상태 체크 기능이 필요하면 아래 주석을 해제하면 됩니다.
	// isBlocked, err := p.sdb.IsBlockedPair(followerUid, followeeUid)
	// if err != nil {
	// 	p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check block relation", err)
	// 	return
	// }
	// if isBlocked {
	// 	p.ctl.SimpleError(c, http.StatusForbidden, "blocked relation")
	// 	return
	// }

	affected, err := p.sdb.SetFollow(followerUid, followeeUid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to follow user", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully followed user", "affected": affected})
}

// UnfollowUser godoc
// @Summary      언팔로우 처리
// @Description  JWT 인증된 사용자가 지정 대상(tid) 유저에 대한 팔로우 관계를 비활성화(언팔로우)합니다. 자기 자신은 언팔로우할 수 없습니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Param        tid     path    string true  "언팔로우할 대상 유저의 uid (followeeUid)"
// @Success      200     {object} map[string]interface{} "msg: 성공 메시지, affected: 실제로 언팔로우 반영된 row 수"
// @Failure      400     {object} map[string]interface{} "잘못된 요청 파라미터 또는 자기자신 언팔로우 시도"
// @Failure      401     {object} map[string]interface{} "인증되지 않은 사용자"
// @Failure      500     {object} map[string]interface{} "내부 서버 에러"
// @Router       /story/v01/follow/cancel/{tid} [post]
func (p *StoryController) UnfollowUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	followerUid := user.(*ptc.UserInfoResp).Uid

	followeeUid, err := strconv.ParseUint(c.Param("tid"), 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
		return
	}
	if followerUid == followeeUid {
		p.ctl.SimpleError(c, http.StatusBadRequest, "self unfollow is not allowed")
		return
	}
	affected, err := p.sdb.SetUnfollow(followerUid, followeeUid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to unfollow user", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully unfollowed user", "affected": affected})
}

// GetFollowerList godoc
// @Summary      팔로워 목록 조회
// @Description  JWT 인증된 사용자의 팔로워 목록을 페이지네이션과 함께 조회합니다. 결과는 전체 팔로워 수와 팔로워 사용자 개별 정보(UID, Nick, ThumbPic, AtUpdate) 리스트로 반환됩니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page  path  int    false "페이지 번호 (기본값: 1)"
// @Param        limit path  int    false "페이지당 항목 개수 (기본값: 20)"
// @Success      200 {object} map[string]interface{} "data: total_count(전체 팔로워 수), followers(팔로워 목록 배열). followers 예시: [{Uid, Nick, ThumbPic, AtUpdate}]"
// @Failure      400 {object} map[string]interface{} "잘못된 요청 파라미터"
// @Failure      401 {object} map[string]interface{} "인증되지 않은 사용자"
// @Failure      500 {object} map[string]interface{} "내부 서버 에러"
// @Router       /story/v01/follower/list [get]
// @Description  요청 예시: GET /story/v01/follower/list/1/20 (Authorization: Bearer)
// @Description  응답 예시: data.total_count, data.followers[]
func (p *StoryController) GetFollowerList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	uid64 := user.(*ptc.UserInfoResp).Uid
	pageInt := ptc.CvtParamAtoi(c.Query("page"), 1)   //defaultQuery "1"
	limitInt := ptc.CvtParamAtoi(c.Query("limit"), 2) //defaultQuery "20"

	list, totalCount, err := p.sdb.GetFollowerList(uid64, pageInt, limitInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get follower list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"total_count": totalCount,
		"list":        *list,
	})
}

// GetFollowingList godoc
// @Summary      팔로잉 목록 조회
// @Description Retrieve following list by uid with pagination
// @Tags Story
// @Accept json
// @Produce json
// @Param uid path string true "User ID"
// @Param page path string true "Page number"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved following list"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid parameters"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /story/v01/following/list/{page}/{limit} [get]
func (p *StoryController) GetFollowingList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	pageInt := ptc.CvtParamAtoi(c.Query("page"), 1)   //defaultQuery "1"
	limitInt := ptc.CvtParamAtoi(c.Query("limit"), 2) //defaultQuery "20"

	list, totalCount, err := p.sdb.GetFollowingList(uid64, pageInt, limitInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get following list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"total_count": totalCount,
		"list":        *list,
	})
}

/*
// BlockUser godoc
// @Summary Block user
// @Description Create or reactivate block relation between users
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.BlockReq true "Block request"
// @Success 200 {object} map[string]interface{} "Successfully blocked user"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/block/create [post]
func (p *StoryController) BlockUser(c *gin.Context) {
	var req ptl.BlockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if req.Uid == "" || req.Bid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockerUid and BlockedUid are required")
		return
	}
	uid, err := strconv.ParseUint(req.Uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockerUid is invalid")
		return
	}
	bid, err := strconv.ParseUint(req.Bid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockedUid is invalid")
		return
	}
	if uid == bid {
		p.ctl.SimpleError(c, http.StatusBadRequest, "self block is not allowed")
		return
	}

	affected, err := p.adb.SetBlockUser(uid, bid, req.Reason)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to block user", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully blocked user", "affected": affected})
}

// UnblockUser godoc
// @Summary Unblock user
// @Description Deactivate block relation between users
// @Tags Story
// @Accept json
// @Produce json
// @Param request body protocol.BlockReq true "Unblock request"
// @Success 200 {object} map[string]interface{} "Successfully unblocked user"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /story/v01/block/cancel [post]
func (p *StoryController) UnblockUser(c *gin.Context) {
	var req ptl.BlockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if req.Uid == "" || req.Bid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockerUid and BlockedUid are required")
		return
	}
	uid, err := strconv.ParseUint(req.Uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockerUid is invalid")
		return
	}
	bid, err := strconv.ParseUint(req.Bid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "BlockedUid is invalid")
		return
	}

	affected, err := p.adb.SetUnblock(uid, bid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to unblock user", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully unblocked user", "affected": affected})
}

// GetBlockList godoc
// @Summary Get block list
// @Description Retrieve blocked user list by uid with pagination
// @Tags Story
// @Accept json
// @Produce json
// @Param uid path string true "User ID"
// @Param page path string true "Page number (start at 1)"
// @Param limit path string true "Page size (1~100)"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved block list"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid parameters"
// @Failure 500 {object} protocol.RespHeader "Internal server error"
// @Router /story/v01/block/list/{uid}/{page}/{limit} [get]
func (p *StoryController) GetBlockList(c *gin.Context) {
	uid := c.Param("uid")
	page := c.Param("page")
	limit := c.Param("limit")

	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is invalid")
		return
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Page is invalid")
		return
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt <= 0 || limitInt > 100 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Limit is invalid")
		return
	}

	list, err := p.adb.GetBlockList(uidInt, pageInt, limitInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get block list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, list)
}
*/
/*
func (p *StoryController) GetStrCommentList(c *gin.Context) {
	strIdx := c.Query("str_idx")
	if strIdx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StrIdx is required")
		return
	}

	strIdxInt, err := strconv.ParseInt(strIdx, 10, 64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StrIdx is required")
		return
	}

	comments, err := p.sdb.GetStrCommentList(strIdxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get str comment list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, comments)
}
*/
/* func (p *StoryController) GetStoryList(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Uid is required")
		return
	}

	storyList, err := p.sdb.GetStoryList(uint64(uid))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}
} */
