package controller

import (
	"database/sql"
	"encoding/json"
	"errors"
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
// @Summary      프로필 정보 조회
// @Description  프로필 정보 조회 API입니다.
// @Description
// @Description  **요청**: path tid: 조회 대상 UID.
// @Description
// @Description  **처리**: 대상 프로필과 활성 팔로워 수를 조회합니다.
// @Description
// @Description  **응답 data**: prf_Info, flw_cnt.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.PrfInfoResp "data: prf_Info, flw_cnt."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "프로필 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/info/{tid} [get]
// @Example Request: GET /story/v01/prf/info/4033287471439576593
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
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Profile not found")
		return
	}
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
// @Summary      본인 보유 자산 조회
// @Description  본인 보유 자산 조회 API입니다.
// @Description
// @Description  **요청**: JWT 인증, 별도 입력 없음.
// @Description
// @Description  **처리**: JWT 사용자의 포인트·캐시와 미읽음 메시지 수를 조회합니다.
// @Description
// @Description  **응답 data**: hold_point, hold_cash, msg_count.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      200 {object} protocol.PrfAssetsResp "data: hold_point, hold_cash, msg_count."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/assets [get]
// @Example Request: GET /story/v01/prf/assets
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
// @Summary      본인 프로필 사진 목록 조회
// @Description  본인 프로필 사진 목록 조회 API입니다.
// @Description
// @Description  **요청**: JWT 인증, 별도 입력 없음.
// @Description
// @Description  **처리**: 등록된 라우트에는 tid가 없으므로 JWT 본인의 사진을 조회합니다.
// @Description
// @Description  **응답 data**: PrfPicMap: 슬롯별 프로필 사진(url, stat).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      200 {object} protocol.RespDataHeader{data=protocol.PrfPicMap} "data: PrfPicMap: 슬롯별 프로필 사진(url, stat)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/pic/list [get]
// @Example Request: GET /story/v01/prf/pic/list
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
// @Summary      프로필 스토리 대표 미디어 조회
// @Description  프로필 스토리 대표 미디어 조회 API입니다.
// @Description
// @Description  **요청**: path tid: 대상 UID.
// @Description
// @Description  **처리**: JWT viewer와 대상 owner를 분리합니다. 최신순으로 텍스트 글과 잠긴 항목을 건너뛰며 열람 가능한 미디어가 있는 스토리 최대 4개를 채웁니다. 부족하면 이전 스토리까지 조회하며 스토리 idx를 키로 반환합니다.
// @Description
// @Description  **응답 data**: str_imgs, str_thmbls.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.PrfStoryListsResp "data: str_imgs, str_thmbls."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/story/{tid} [get]
// @Router       /story/v01/prf/story/lists/{tid} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"str_imgs":{},"str_thmbls":{}}}
// @Example Request: GET /story/v01/prf/story/4033287471439576593
// @x-response-example {"result":0,"resultString":"Success","data":{"str_imgs":{},"str_thmbls":{}}}
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

	strImgs, strthmbls, err := p.sdb.GetStoryFirstPicUrls(user.(*ptc.UserInfoResp).Uid, targetId64)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"str_imgs": strImgs, "str_thmbls": strthmbls})
}

// UpdatePrfInfo godoc
// @Summary      본인 프로필 수정
// @Description  본인 프로필 수정 API입니다.
// @Description
// @Description  **요청**: body PrfInfo: 수정할 프로필 필드.
// @Description
// @Description  **처리**: body uid를 JWT UID로 덮어쓰고 프로필 정보를 갱신합니다.
// @Description
// @Description  **응답 data**: msg.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.PrfInfo true "body PrfInfo: 수정할 프로필 필드."
// @Success      200 {object} protocol.RespDataHeader "data.msg."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/upd [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @Example Request: POST /story/v01/prf/upd
// @Example Request Body: {"nick":"새 닉네임"}
// @x-request-example {"nick":"새 닉네임"}
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
// @Description  프로필 사진 업로드 API입니다.
// @Description
// @Description  **요청**: multipart files: 신규 이미지, 최대 5개·각 5MiB. urls: 유지할 기존 URL의 JSON 배열 문자열(기본 []).
// @Description
// @Description  **처리**: 신규 파일 업로드 후 기존 URL(stat=3)과 신규 URL(stat=1)을 합쳐 최대 5개 슬롯으로 저장합니다. 파일이 하나 이상 필요합니다.
// @Description
// @Description  **응답 data**: msg, pics.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Description  업로드 미들웨어 오류는 JSON 문자열 또는 error 객체일 수 있으며 핸들러의 RespHeader 오류와 형식이 다릅니다.
// @Description  **업로드 공통 제한**: 요청 전체 크기·동시 업로드 제한을 초과하면 413/503입니다. 기본값은 530MiB/8건이며 서버 설정에 따라 달라집니다.
// @Tags         Profile
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        files formData file true "업로드 파일 (동일 필드명으로 여러 파일 전송 가능)"
// @Param        urls formData string false "유지할 기존 URL의 JSON 배열 문자열" default([])
// @Success      200 {object} protocol.PrfPicUploadResp "data: msg, pics."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      413 {object} map[string]interface{} "업로드 요청 크기 제한 초과"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Failure      503 {object} map[string]interface{} "서버 동시 처리·연결 한도 초과"
// @Router       /story/v01/prf/pic/upload [post]
// @Example Request: POST /story/v01/prf/pic/upload
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
// @Summary      본인 프로필 사진 삭제
// @Description  본인 프로필 사진 삭제 API입니다.
// @Description
// @Description  **요청**: path idx: 삭제할 사진 슬롯.
// @Description
// @Description  **처리**: JWT 사용자의 사진 슬롯을 삭제합니다.
// @Description
// @Description  **응답**: 성공 헤더: result, resultString, desc. data는 없습니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Success      200 {object} protocol.RespHeader "성공 헤더: result, resultString, desc. data는 없습니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/pic/delete/{idx} [post]
// @Example Response: {"result":0,"resultString":"Success","msg":"success"}
// @x-response-example {"result":0,"resultString":"Success","msg":"success"}
// @Example Request: POST /story/v01/prf/pic/delete/15
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
// @Summary      관리자 프로필 사진 승인 대기 조회
// @Description  관리자 프로필 사진 승인 대기 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: JWT UID가 서버 ProfileAdminUIDs 허용 목록에 있어야 합니다. 비관리자는 403입니다.
// @Description
// @Description  **응답 data**: msg, total_count, list.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.PrfPicWaitingListResp "data: msg, total_count, list."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/pic/waiting/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":0,"list":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":0,"list":[]}}
// @Example Request: GET /story/v01/prf/pic/waiting/1/10
func (p *StoryController) GetPrfPicWaitingList(c *gin.Context) {
	if !p.requireProfileAdmin(c) {
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
// @Summary      관리자 프로필 사진 승인
// @Description  관리자 프로필 사진 승인 API입니다.
// @Description
// @Description  **요청**: body SetPrfPicStatReq: uid는 양의 정수(uint64), slot은 1~5.
// @Description
// @Description  **처리**: JWT UID가 서버 ProfileAdminUIDs 허용 목록에 있어야 합니다. 요청·대기 상태(1/2) 사진만 승인(3)합니다.
// @Description
// @Description  **응답 data**: msg, res.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.SetPrfPicStatReq true "body SetPrfPicStatReq: uid는 양의 정수(uint64), slot은 1~5."
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,res=int}} "data: msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/pic/stat [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @Example Request: POST /story/v01/prf/pic/stat
// @Example Request Body: {"uid":4033287471439576593,"slot":1}
// @x-request-example {"uid":4033287471439576593,"slot":1}
func (p *StoryController) SetPrfPicStat(c *gin.Context) {
	if !p.requireProfileAdmin(c) {
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
		"msg": "ok",
		"res": affected,
	})
}

// SetMainPic godoc
// @Summary      본인 대표 사진 설정
// @Description  본인 대표 사진 설정 API입니다.
// @Description
// @Description  **요청**: path url: URL 인코딩한 대표 사진 URL.
// @Description
// @Description  **처리**: JWT 사용자의 대표 사진을 변경합니다. 현재 빈 URL을 받는 별도 경로는 등록되어 있지 않습니다.
// @Description
// @Description  **응답 data**: msg.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        url path string true "URL 인코딩한 대표 사진 URL"
// @Success      200 {object} protocol.RespDataHeader "data.msg."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/set/mainpic/{url} [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @Example Request: POST /story/v01/prf/set/mainpic/https%3A%2F%2Fexample.com%2Fphoto.jpg
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

// GetPrfStoryList godoc
// @Summary      프로필 스토리 요약 목록 조회
// @Description  프로필 스토리 요약 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100. owner는 tid, 생략 시 JWT UID이며 viewer는 항상 JWT UID입니다.
// @Description
// @Description  **처리**: owner 조건·가시성 조건과 공통 열람 판정을 적용합니다. 잠긴 항목은 is_locked=1이며 cnt_url/cnt_thumb/cnt_type은 빈 문자열입니다. 승인된 미리보기 자산이 없어 원본 URL을 대체 반환하지 않습니다.
// @Description
// @Description  **응답 data**: StoryListResp 배열. total_count 래퍼는 없습니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=[]protocol.StoryListResp} "data: StoryListResp 배열. total_count 래퍼는 없습니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/prf/story/list/{page}/{limit} [get]
// @Router       /story/v01/prf/story/list/{page}/{limit}/{tid} [get]
// @Example Response: {"result":0,"resultString":"Success","data":[]}
// @x-response-example {"result":0,"resultString":"Success","data":[]}
// @Example Request: GET /story/v01/prf/story/list/1/10
func (p *StoryController) GetPrfStoryList(c *gin.Context) {
	viewer, ok := p.storyActorUID(c)
	if !ok {
		return
	}
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

	storyList, err := p.sdb.GetDefStoryList(viewer, tuid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, storyList)
}
