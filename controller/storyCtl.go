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

// GetStoryDetail godoc
// @Summary      스토리 상세 목록 조회 (본인/타인)
// @Description  특정 스토리 idx를 기준으로 해당 작성자(tid)의 스토리를 페이징 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - `tid` 없음: JWT 본인 UID (`/detail/{idx}/{page}/{limit}`)
// @Description  - `tid` 있음: 해당 사용자 UID (`/detail/{idx}/{page}/{limit}/{tid}`)
// @Description
// @Description  **조회 규칙**
// @Description  - `uid = tid` 이고 `story.idx <= idx` 인 행을 `idx DESC`로 페이징
// @Description  - 각 항목에 최신 댓글 최대 2개(`comment_list`)와 `cmt_count` 포함
// @Description  - 조회 성공 시 요청 idx 스토리의 `qt_view`를 +1 (`UpdateStoryView`)
// @Description
// @Description  **응답 data**
// @Description  - `total_count`: 해당 tid의 전체 스토리 수
// @Description  - `story`: `StoryDetailResp` 배열
// @Description    (`idx`, `uid`, `nick`, `body`, `stat`, `qt_good`, `qt_view`, `content_total`,
// @Description     `str_img` 슬롯 맵, `at_create`, `cmt_count`, `comment_list`)
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"total_count":14,"story":[{"idx":15,"uid":4033287471439576593,"nick":"푸른 아름다운 양","body":"E 둘다","stat":3,"qt_good":16,"qt_view":7,"content_total":3,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"img","url":"https://imagedelivery.net/.../public"},"3":{"type":"vdo","url":"https://.../video.mp4","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"comment_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","body":"comment","stat":1,"at_create":"2026-02-02T13:50:06Z"}]}]}}
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true   "Bearer access token"
// @Param        idx            path    int     true   "기준 스토리 인덱스 (이 idx 이하부터 조회)"
// @Param        page           path    int     true   "페이지 번호(1부터)"
// @Param        limit          path    int     true   "페이지 크기(기본 4, 최대 100)"
// @Param        tid            path    string  false  "스토리 작성자 UID (없으면 JWT 본인)"
// @Success      200            {object} map[string]interface{} "total_count, story(StoryDetailResp[])"
// @Failure      400            {object} map[string]interface{} "idx/page/limit/tid 잘못된 값"
// @Failure      401            {object} map[string]interface{} "tid 없을 때 JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "상세 조회 또는 조회수 갱신 실패"
// @Router       /story/v01/detail/{idx}/{page}/{limit} [get]
// @Router       /story/v01/detail/{idx}/{page}/{limit}/{tid} [get]
// @Example Request: GET /story/v01/detail/15/1/10
// @Example Request: GET /story/v01/detail/15/1/10/4033287471439576593
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":14,"story":[{"idx":15,"uid":4033287471439576593,"nick":"푸른 아름다운 양","body":"E 둘다","stat":3,"qt_good":16,"qt_view":7,"content_total":3,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"img","url":"https://imagedelivery.net/.../public"},"3":{"type":"vdo","url":"https://.../video.mp4","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"comment_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","body":"comment","stat":1,"at_create":"2026-02-02T13:50:06Z"}]}]}}
func (p *StoryController) GetStoryList(c *gin.Context) {
	/* 	idx := c.Param("idx")
	   	if idx == "" {
	   		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
	   		return
	   	}

	   	idxInt, err := strconv.Atoi(idx)
	   	if err != nil {
	   		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
	   		return
	   	}
	*/
	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	tidParam := c.Param("tid")
	var tid64 uint64
	if tidParam == "" {
		user, exists := c.Get("user")
		if !exists {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}
		tid64 = user.(*ptc.UserInfoResp).Uid
	} else {
		tid64, err = strconv.ParseUint(tidParam, 10, 64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "Tid is invalid")
			return
		}
	}

	story, cnt, err := p.sdb.GetDetailList(tid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story detail", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": cnt, "story": story})
}

// GetStoryConditionList godoc
// @Summary      스토리 조건부 리스트 조회
// @Description  path 파라미터(area/stat/type/order/gen/page/limit)로 스토리 피드를 조회합니다.
// @Description
// @Description  **필터**
// @Description  - `area`: all이면 지역 조건 생략, 그 외 `story.area` 매칭
// @Description  - `stat`: pub/flw/pay/prv/del/rsv → story.stat (0~5)
// @Description  - `type`: img/vdo/all/rsv — all이면 type 조건 생략
// @Description  - `order`: new/old/cmt/god/view/rsv
// @Description  - `gen`: 0=여성, 1=남성 (기본 1)
// @Description
// @Description  **cutout**: JWT가 있으면 해당 사용자의 활성 cutout UID를 피드에서 제외
// @Description
// @Description  **응답 data**: `StoryListResp` 배열
// @Description  (`idx`, `uid`, `nick`, `stat`, `cnt_count`, `cnt_url`, `cnt_type`, `cnt_thumb`, `at_create`)
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  false  "Bearer access token (있으면 cutout 제외 적용)"
// @Param        area           path    string  true   "지역" Enums(all,seoul,gyeonggi,incheon,busan,daejeon,chungbuk,daegu,gyeongnam,gwangju,jeonbuk,gangwon,jeju) default(all)
// @Param        stat           path    string  true   "상태" Enums(pub,flw,pay,prv,del,rsv) default(pub)
// @Param        type           path    string  true   "미디어 타입" Enums(img,vdo,all,rsv) default(img)
// @Param        order          path    string  true   "정렬" Enums(new,old,cmt,god,view,rsv) default(new)
// @Param        gen            path    string  true   "성별 0=여성 1=남성" Enums(0,1) default(1)
// @Param        page           path    int     true   "페이지 번호(1부터)"
// @Param        limit          path    int     true   "페이지 크기(기본 4, 최대 100)"
// @Success      200            {object} map[string]interface{} "data: StoryListResp 배열"
// @Failure      400            {object} map[string]interface{} "page/limit 잘못된 값"
// @Failure      500            {object} map[string]interface{} "cutout 조회 실패 또는 리스트 조회 실패"
// @Router       /story/v01/condition/list/{area}/{stat}/{type}/{order}/{gen}/{page}/{limit} [get]
// @Example Request: GET /story/v01/condition/list/seoul/pub/img/new/1/1/10
// @Example Request: GET /story/v01/condition/list/all/pub/all/new/1/1/10
// @Example Response: {"result":0,"resultString":"Success","data":[{"idx":15,"uid":4033287471439576593,"nick":"푸른 아름다운 양","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/.../public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-02T13:35:24Z"}]}
func (p *StoryController) GetStoryConditionList(c *gin.Context) {
	// 홈 디폴트 리스트, 최신 등록순으로 전체 고객 스토리 리스트 출력, 해당 리스트 출력시 대상 사용자의 스토리로 이동해야함
	area := switchDefParam(c.Param("area"), "all")
	stat := switchDefParam(c.Param("stat"), "pub")
	mtype := switchDefParam(c.Param("type"), "img")
	order := switchDefParam(c.Param("order"), "new")
	gen := switchDefParam(c.Param("gen"), "1")

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

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
	// p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"story_home_list": storyList})
	p.ctl.SendDataResponse(c, http.StatusOK, storyList)
}

// GetStoryDetail godoc
// @Summary      스토리 상세 목록 조회 (본인/타인)
// @Description  특정 스토리 idx를 기준으로 해당 작성자(tid)의 스토리를 페이징 조회합니다.
// @Description
// @Description  **조회 대상**
// @Description  - `tid` 없음: JWT 본인 UID (`/detail/{idx}/{page}/{limit}`)
// @Description  - `tid` 있음: 해당 사용자 UID (`/detail/{idx}/{page}/{limit}/{tid}`)
// @Description
// @Description  **조회 규칙**
// @Description  - `uid = tid` 이고 `story.idx <= idx` 인 행을 `idx DESC`로 페이징
// @Description  - 각 항목에 최신 댓글 최대 2개(`comment_list`)와 `cmt_count` 포함
// @Description  - 조회 성공 시 요청 idx 스토리의 `qt_view`를 +1 (`UpdateStoryView`)
// @Description
// @Description  **응답 data**
// @Description  - `total_count`: 해당 tid의 전체 스토리 수
// @Description  - `story`: `StoryDetailResp` 배열
// @Description    (`idx`, `uid`, `nick`, `body`, `stat`, `qt_good`, `qt_view`, `content_total`,
// @Description     `str_img` 슬롯 맵, `at_create`, `cmt_count`, `comment_list`)
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"total_count":14,"story":[{"idx":15,"uid":4033287471439576593,"nick":"푸른 아름다운 양","body":"E 둘다","stat":3,"qt_good":16,"qt_view":7,"content_total":3,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"img","url":"https://imagedelivery.net/.../public"},"3":{"type":"vdo","url":"https://.../video.mp4","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"comment_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","body":"comment","stat":1,"at_create":"2026-02-02T13:50:06Z"}]}]}}
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true   "Bearer access token"
// @Param        idx            path    int     true   "기준 스토리 인덱스 (이 idx 이하부터 조회)"
// @Param        page           path    int     true   "페이지 번호(1부터)"
// @Param        limit          path    int     true   "페이지 크기(기본 4, 최대 100)"
// @Param        tid            path    string  false  "스토리 작성자 UID (없으면 JWT 본인)"
// @Success      200            {object} map[string]interface{} "total_count, story(StoryDetailResp[])"
// @Failure      400            {object} map[string]interface{} "idx/page/limit/tid 잘못된 값"
// @Failure      401            {object} map[string]interface{} "tid 없을 때 JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "상세 조회 또는 조회수 갱신 실패"
// @Router       /story/v01/detail/{idx}/{page}/{limit} [get]
// @Router       /story/v01/detail/{idx}/{page}/{limit}/{tid} [get]
// @Example Request: GET /story/v01/detail/15/1/10
// @Example Request: GET /story/v01/detail/15/1/10/4033287471439576593
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":14,"story":[{"idx":15,"uid":4033287471439576593,"nick":"푸른 아름다운 양","body":"E 둘다","stat":3,"qt_good":16,"qt_view":7,"content_total":3,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"img","url":"https://imagedelivery.net/.../public"},"3":{"type":"vdo","url":"https://.../video.mp4","thumb":"https://imagedelivery.net/.../public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"comment_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","body":"comment","stat":1,"at_create":"2026-02-02T13:50:06Z"}]}]}}
func (p *StoryController) GetStoryDetail(c *gin.Context) {
	// 스토리 리스트 클릭후 -> 해당 사용자의 스토리 모음으로 이동 -> 해당 스토리 인덱스가 최상위, 그 밑으로 최신순으로 스토리 리스트 출력
	// 출력 내역 : 해당대상자 프로필아이콘, 닉네임, 나이, 지역, 팔로잉 여부, 스토리컨텐츠 리스트, 전체 컨텐츠수, 본문, 좋아요, 댓글, 조회수, 생성일자, 댓글 최신 2개,
	// 댓글 내역 : 아이콘프로필, 성별, 닉네임, 지역, 나이, 댓글내역 - ptc.StrComment

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

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	tidParam := c.Param("tid")
	var tid64 uint64
	var isMyStory bool = false
	if tidParam == "" {
		user, exists := c.Get("user")
		if !exists {
			p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}
		tid64 = user.(*ptc.UserInfoResp).Uid
		isMyStory = true
	} else {
		tid64, err = strconv.ParseUint(tidParam, 10, 64)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusBadRequest, "Tid is invalid")
			return
		}
		isMyStory = false
	}

	story, cnt, err := p.sdb.GetDetailList(tid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story detail", err)
		return
	}

	if !isMyStory {
		//TODO: cache 필요 - 당일 조회는 중복 카운트 없이 구성 필요
		err = p.sdb.UpdateStoryView(idxInt)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story view", err)
			return
		}
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": cnt, "story": story})
}

// GetPurchaseList godoc
// @Summary      구매한 스토리 목록 조회
// @Description  JWT 본인이 구매한 스토리 목록을 페이징 조회합니다.
// @Description
// @Description  **조회 규칙**
// @Description  - `str_pcs.pcs_uid = JWT uid` 인 구매 기록과 `story`를 INNER JOIN
// @Description  - `story.at_create DESC` 최신순
// @Description  - 대표 미디어만 반환 (`cnt_url`/`cnt_type`/`cnt_thumb`, 전체 `str_img` 슬롯 맵 미포함)
// @Description
// @Description  **응답 data**
// @Description  - `msg`: `"ok"`
// @Description  - `total_count`: 구매·조인된 전체 건수
// @Description  - `list`: `StoryListResp` 배열
// @Description    (`idx`, `uid`, `nick`, `stat`, `cnt_count`, `cnt_url`, `cnt_type`, `cnt_thumb`, `at_create`)
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        page           path    int     true  "페이지 번호(1부터)"
// @Param        limit          path    int     true  "페이지 크기(기본 4, 최대 100)"
// @Success      200            {object} map[string]interface{} "msg, total_count, list(StoryListResp[])"
// @Failure      400            {object} map[string]interface{} "page/limit 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "구매 목록 조회 실패"
// @Router       /story/v01/pcs/list/{page}/{limit} [get]
// @Example Request: GET /story/v01/pcs/list/1/10
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":1,"list":[{"idx":15,"uid":403,"nick":"푸양","stat":2,"cnt_count":1,"cnt_url":"https://imagedelivery.net/.../public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-02T13:35:24Z"}]}}
func (p *StoryController) GetPurchaseList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	list, totalCount, err := p.sdb.GetPurchaseList(uid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get purchase list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"total_count": totalCount, "list": list})
}

// BuyStory godoc
// @Summary      스토리 구매
// @Description  JWT 본인이 지정한 스토리(`idx`)를 구매 기록합니다.
// @Description
// @Description  **처리**
// @Description  - `story.idx`로 판매자 UID(`tid`) 조회
// @Description  - `str_pcs`에 INSERT: `str_idx`, `tid`(판매자), `pcs_uid`(JWT 구매자), `qt_payed=0`
// @Description  - 결제 프로세스(잔액 차감 등)는 아직 미구현(TODO)
// @Description
// @Description  **응답 data**: `{"msg":"ok"}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "구매할 스토리 인덱스"
// @Success      200            {object} map[string]interface{} "msg"
// @Failure      400            {object} map[string]interface{} "idx 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 없음 또는 str_pcs 저장 실패"
// @Router       /story/v01/buy/{idx} [post]
// @Example Request: POST /story/v01/buy/15
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok"}}
func (p *StoryController) BuyStory(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	idx := c.Param("idx")
	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	//TODO : 결제 프로세스 필요
	if err = p.sdb.BuyStory(uid64, idxInt); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to buy story", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok"})
}

// UpdateStoryViewCnt godoc
// @Summary      스토리 조회수 증가
// @Description  지정한 스토리(`idx`)의 `qt_view`를 +1 합니다.
// @Description
// @Description  **처리**
// @Description  - 대상 행이 없으면 500 (`story not found`)
// @Description  - 응답에는 증가 후 조회수 값을 포함하지 않음 (`msg`만 반환)
// @Description
// @Description  **응답 data**: `{"msg":"ok"}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "조회수를 올릴 스토리 인덱스"
// @Success      200            {object} map[string]interface{} "msg"
// @Failure      400            {object} map[string]interface{} "idx 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 없음 또는 조회수 갱신 실패"
// @Router       /story/v01/view/update/{idx} [post]
// @Example Request: POST /story/v01/view/update/15
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok"}}
func (p *StoryController) UpdateStoryViewCnt(c *gin.Context) {
	idx := c.Param("idx")
	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	err = p.sdb.UpdateStoryView(idxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story view", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok"})
}

// GetStoryViewCount godoc
// @Summary      스토리 조회수 조회
// @Description  지정한 스토리(`idx`)의 현재 조회수(`story.qt_view`)를 반환합니다.
// @Description
// @Description  **응답 data**: `{"msg":"ok","count":<qt_view>}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "조회할 스토리 인덱스"
// @Success      200            {object} map[string]interface{} "msg, count"
// @Failure      400            {object} map[string]interface{} "idx 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 없음 또는 조회 실패"
// @Router       /story/v01/view/count/{idx} [get]
// @Example Request: GET /story/v01/view/count/15
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","count":10}}
func (p *StoryController) GetStoryViewCount(c *gin.Context) {
	idx := c.Param("idx")
	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	count, err := p.sdb.GetStoryViewCount(idxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story view count", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "count": count})
}

// GetStoryLikeCount godoc
// @Summary      스토리 좋아요 수 조회
// @Description  지정한 스토리(`idx`)의 현재 좋아요 수(`story.qt_good`)를 반환합니다.
// @Description
// @Description  **응답 data**: `{"msg":"ok","count":<qt_good>}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        idx            path    int     true  "조회할 스토리 인덱스"
// @Success      200            {object} map[string]interface{} "msg, count"
// @Failure      400            {object} map[string]interface{} "idx 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "스토리 없음 또는 조회 실패"
// @Router       /story/v01/like/count/{idx} [get]
// @Example Request: GET /story/v01/like/count/15
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","count":10}}
func (p *StoryController) GetStoryLikeCount(c *gin.Context) {
	idx := c.Param("idx")
	idxInt, err := strconv.Atoi(idx)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	count, err := p.sdb.GetStoryLikeCount(idxInt)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story like count", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "count": count})
}

// UpdateStoryLike godoc
// @Summary      스토리 좋아요 토글
// @Description  스토리 좋아요/취소를 토글하고 최신 좋아요 수를 반환합니다.
// @Description
// @Description  **요청 body** (`StoryLikeToggleReq`)
// @Description  - `story_idx`: 스토리 인덱스 (문자열 숫자)
// @Description  - `uid`: 좋아요를 누른 사용자 UID (문자열 숫자)
// @Description
// @Description  **처리** (`ToggleStoryLikeTx`)
// @Description  - 차단(block) 검사는 현재 코드에서 비활성(주석 처리)
// @Description
// @Description  **응답 data**: `{"msg":"...","liked":true|false,"like_count":N}`
// @Description  - `liked`: 토글 후 좋아요 여부
// @Description  - `like_count`: 갱신된 `story.qt_good`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        request        body    protocol.StoryLikeToggleReq  true  "story_idx, uid"
// @Success      200            {object} map[string]interface{} "msg, liked, like_count"
// @Failure      400            {object} map[string]interface{} "JSON/story_idx/uid 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "토글/카운트 갱신 실패"
// @Router       /story/v01/like/toggle [post]
// @Example Request: POST /story/v01/like/toggle
// @Example Request Body: {"story_idx":"15","uid":"4033287471439576593"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","liked":true,"like_count":17}}
func (p *StoryController) UpdateStoryLike(c *gin.Context) {
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
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "liked": liked, "like_count": likeCount})
}

// UploadStoryContent godoc
// @Summary      스토리 미디어 업로드 (Cloudflare)
// @Description  JWT 인증 후 스토리용 미디어를 Cloudflare에 업로드하고, DB 저장 없이 통합 `media` 슬롯 맵만 반환합니다.
// @Description
// @Description  **미들웨어**: `ValidateFileUpload(5, 5)` — `files` 최대 5개, 이미지·썸네일 ≤5MB, MP4 ≤100MB
// @Description
// @Description  **요청 (multipart/form-data)**
// @Description  - `files`: 이미지(JPG/PNG/GIF/WEBP) 또는 MP4, 업로드 순서가 슬롯 `1..N`
// @Description  - `thbnl`: 동영상 썸네일 이미지. MP4마다 1장 필수(개수·순서 1:1)
// @Description
// @Description  **처리**
// @Description  - 동영상 포함: Stream/Images 혼합 업로드 + `thbnl` 업로드
// @Description  - 이미지만: Images 업로드
// @Description  - `BuildStrImgForDB` → `LoadStoryStrImg`로 정규화된 media 응답 (최대 5슬롯)
// @Description
// @Description  **응답 data**
// @Description  - `msg`: `"ok"`
// @Description  - `media`: `StoryStrImg` 슬롯 맵
// @Description    `{"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...m3u8","thumb":"..."}}`
// @Description  - `type`: `img` | `vdo` / `url`: 본편 / `thumb`: vdo만
// @Description
// @Description  **다음 단계**: 응답 `media`를 `POST /story/v01/create` body에 넣어 스토리 생성
// @Tags         Story
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header    string  true   "Bearer access token"
// @Param        files          formData  file    true   "미디어 (이미지 ≤5MB 또는 MP4 ≤100MB, 최대 5개)"
// @Param        thbnl          formData  file    false  "동영상 썸네일 (이미지 ≤5MB). 동영상마다 1장 필수"
// @Success      200            {object}  map[string]interface{} "msg, media(StoryStrImg)"
// @Failure      400            {object}  map[string]interface{} "파일 없음, thbnl 부족, media 검증 실패"
// @Failure      401            {object}  map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object}  map[string]interface{} "Cloudflare 업로드 또는 media 구성 실패"
// @Router       /story/v01/upload [post]
// @Example Request (image): curl -X POST /story/v01/upload -H "Authorization: Bearer {jwt}" -F "files=@a.jpg" -F "files=@b.jpg"
// @Example Request (video): curl -X POST /story/v01/upload -H "Authorization: Bearer {jwt}" -F "files=@v1.mp4" -F "thbnl=@t1.jpg"
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","media":{"1":{"type":"img","url":"https://imagedelivery.net/.../public"},"2":{"type":"vdo","url":"https://.../manifest/video.m3u8","thumb":"https://imagedelivery.net/.../public"}}}}
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
	var out ptl.StoryStrImg
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to build media response", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "media": out})
}

// CreateStory godoc
// @Summary      스토리 생성
// @Description  JWT 인증 사용자의 스토리를 DB에 저장합니다. 파일 업로드는 하지 않으며, upload API의 `data.media`를 body로 받습니다.
// @Description
// @Description  **권장 플로우**
// @Description  1. `POST /story/v01/upload` — Cloudflare 업로드 → `data.media` 수신
// @Description  2. `POST /story/v01/create` — JSON body에 `media` 전달 → `SaveStory` INSERT
// @Description
// @Description  **요청 body** (`CreateStoryReq`)
// @Description  - `stat`: 0=pub, 1=private(팔로우공개), 2=limit(유료공개), 3=resv, 4=del
// @Description  - `sbody`: 본문 (최대 512자)
// @Description  - `media`: `StoryStrImg` 슬롯 맵 (1~5개, upload 응답과 동일)
// @Description    `{"1":{"type":"img","url":"..."},"2":{"type":"vdo","url":"...","thumb":"..."}}`
// @Description
// @Description  **서버에서 JWT로 채우는 필드**: uid, nick, birth, gender, area
// @Description  **검증**: `ValidateStoryMedia` (슬롯·type·url·vdo thumb)
// @Description  **저장**: `BuildStrImgForDB`로 `str_img`/`type` 정규화 후 INSERT
// @Description
// @Description  **응답 data**: `{"msg":"ok","idx":<story.idx>}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        request        body    protocol.CreateStoryReq  true  "stat, sbody, media"
// @Success      200            {object} map[string]interface{} "msg, idx"
// @Failure      400            {object} map[string]interface{} "JSON/media 검증 실패, nick·birth·gender 부족"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "DB 저장 실패"
// @Router       /story/v01/create [post]
// @Example Request: {"stat":1,"sbody":"오늘의 스토리","media":{"1":{"type":"img","url":"https://imagedelivery.net/xxx/public"},"2":{"type":"vdo","url":"https://.../video.m3u8","thumb":"https://imagedelivery.net/xxx/public"}}}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","idx":5}}
func (p *StoryController) CreateStory(c *gin.Context) {
	userInfo, ok := c.Get("user")
	if !ok {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	u := userInfo.(*ptl.UserInfoResp)

	var sinfo ptl.CreateStoryReq
	var err error
	if err = c.ShouldBindJSON(&sinfo); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	var mtype int
	if mtype, err = models.ValidateStoryMedia(sinfo.Media); err != nil {
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

	mainPic, err := p.sdb.GetPFMainPic(u.Uid)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get pf main pic", err)
		return
	}

	simg := &ptl.StoryImage{
		Uid:     u.Uid,
		Nick:    u.Nick,
		Birth:   birth,
		Area:    area,
		Gender:  nGen,
		MainPic: mainPic,
		Body:    sinfo.Sbody,
		MType:   mtype,
		Stat:    sinfo.Stat,
		Price:   sinfo.Price,
	}

	lastID, err := p.sdb.SaveStory(simg, sinfo.Media)
	if err != nil {
		log.Error("Failed to set story: %v", err)
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set story", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "idx": lastID})
}

// UpdateStoryStat godoc
// @Summary      스토리 상태 변경
// @Description  지정한 스토리(`idx`)의 `stat`을 갱신합니다.
// @Description
// @Description  **요청 body** (`UpdateStrStatReq`)
// @Description  - `idx`: 스토리 인덱스 (문자열 숫자)
// @Description  - `stat`: 상태 코드 (문자열 숫자)
// @Description    0=public, 1=private(팔로우공개), 2=limit(유료공개), 3=rsv, 4=del
// @Description
// @Description  **응답 data**: `{"msg":"ok","affected":N}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        request        body    protocol.UpdateStrStatReq  true  "idx, stat"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "JSON/idx/stat 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "상태 갱신 실패"
// @Router       /story/v01/updstat [post]
// @Example Request: POST /story/v01/updstat
// @Example Request Body: {"idx":"15","stat":"2"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// UpdateStrBody godoc
// @Summary      스토리 본문 수정
// @Description  지정한 스토리(`idx`)의 본문(`body`)을 갱신합니다.
// @Description
// @Description  **요청 body** (`UpdateStrBodyReq`)
// @Description  - `idx`: 스토리 인덱스 (문자열 숫자)
// @Description  - `body`: 수정할 본문 (빈 문자열 불가, DB 컬럼 최대 512자)
// @Description
// @Description  **응답 data**: `{"msg":"ok","affected":N}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        request        body    protocol.UpdateStrBodyReq  true  "idx, body"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "JSON/idx/body 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "본문 갱신 실패"
// @Router       /story/v01/updbody [post]
// @Example Request: POST /story/v01/updbody
// @Example Request Body: {"idx":"15","body":"수정된 스토리 본문"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
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

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// DeleteStrPic godoc
// @Summary      스토리 미디어 슬롯 삭제
// @Description  스토리 `str_img`에서 지정 슬롯을 제거하고 `str_imgbak`으로 이동한 뒤, 남은 슬롯을 `1..N`으로 재번호합니다.
// @Description
// @Description  **요청 body** (`DeleteStrPicReq`)
// @Description  - `idx`: 스토리 인덱스 (문자열 숫자)
// @Description  - `pic_idx`: 삭제할 미디어 슬롯 키 (`"1"`~`"5"`)
// @Description
// @Description  **처리**
// @Description  1. `GetStrPicList`로 `str_img` / `str_imgbak` 조회
// @Description  2. `pic_idx` 슬롯을 bak으로 복사
// @Description  3. `ReIdxAfterDel`로 남은 슬롯 재인덱싱
// @Description  4. `DeleteStrPic`으로 DB 갱신
// @Description
// @Description  **응답 data**: `{"msg":"ok","affected":N}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        request        body    protocol.DeleteStrPicReq  true  "idx, pic_idx"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "JSON/idx/pic_idx 오류, 빈 목록, 슬롯 없음, str_img 포맷 오류"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "조회/직렬화/DB 갱신 실패"
// @Router       /story/v01/delpic [post]
// @Example Request: POST /story/v01/delpic
// @Example Request Body: {"idx":"15","pic_idx":"2"}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
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

	var media ptl.StoryStrImg
	if err := json.Unmarshal([]byte(strImg), &media); err != nil {
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
		var parsedBak ptl.StoryStrImg
		if err := json.Unmarshal([]byte(strImgBak), &parsedBak); err == nil {
			bakMedia = parsedBak
		}
	} else {
		bakMedia = make(ptl.StoryStrImg)
	}

	bakMedia[sPic] = item
	media = models.ReIdxAfterDel(media, sPic)
	mediaBytes, err := json.Marshal(media)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to marshal media", err)
		return
	}
	bakMediaBytes, err := json.Marshal(bakMedia)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to marshal bakMedia", err)
		return
	}
	affected, err := p.sdb.DeleteStrPic(strIdxInt, mediaBytes, bakMediaBytes)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to delete str pic", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// FollowUser godoc
// @Summary      팔로우 설정
// @Description  JWT 본인(`follower`)이 대상(`tid`=`followee`)을 팔로우합니다. 신규 생성 또는 기존 행 재활성화(`stat=1`).
// @Description
// @Description  **규칙**
// @Description  - `follower` = JWT uid, `followee` = path `tid`
// @Description  - 자기 자신 팔로우 불가 (`self follow is not allowed`)
// @Description  - 차단(block) 검사는 현재 비활성(주석 처리)
// @Description
// @Description  **응답 data**: `{"msg":"ok","affected":N}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "팔로우 대상 UID (followee)"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid 잘못된 값 또는 자기 자신 팔로우"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "팔로우 저장 실패"
// @Router       /story/v01/follow/set/{tid} [post]
// @Example Request: POST /story/v01/follow/set/4033287471439576593
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
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
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// UnfollowUser godoc
// @Summary      언팔로우 처리
// @Description  JWT 본인(`follower`)이 대상(`tid`=`followee`)에 대한 팔로우를 비활성화합니다 (`stat=0`).
// @Description
// @Description  **규칙**
// @Description  - `follower` = JWT uid, `followee` = path `tid`
// @Description  - 자기 자신 언팔로우 불가 (`self unfollow is not allowed`)
// @Description
// @Description  **응답 data**: `{"msg":"ok","affected":N}`
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "언팔로우 대상 UID (followee)"
// @Success      200            {object} map[string]interface{} "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid 잘못된 값 또는 자기 자신 언팔로우"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "언팔로우 갱신 실패"
// @Router       /story/v01/follow/cancel/{tid} [post]
// @Example Request: POST /story/v01/follow/cancel/4033287471439576593
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
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
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "affected": affected})
}

// GetFollowerList godoc
// @Summary      팔로워 목록 조회
// @Description  JWT 본인을 followee로 하는 활성 팔로워(`user_follow.stat=1`) 목록을 페이징 조회합니다.
// @Description
// @Description  **조회 규칙**
// @Description  - JWT uid = `followee_uid` (나를 팔로우한 사람들)
// @Description  - `at_update DESC` 정렬
// @Description  - `page`/`limit`은 path에서 읽고 빈 값이면 page=1, limit=20
// @Description
// @Description  **응답 data**
// @Description  - `total_count`: 전체 팔로워 수
// @Description  - `list`: `FollowUserItem` 배열 (`uid`, `nick`, `thumb_pic`, `at_update`)
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        page           path    int     true  "페이지 번호(1부터, 기본 1)"
// @Param        limit          path    int     true  "페이지 크기(기본 20)"
// @Success      200            {object} map[string]interface{} "total_count, list(FollowUserItem[])"
// @Failure      400            {object} map[string]interface{} "page/limit 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "팔로워 목록 조회 실패"
// @Router       /story/v01/follower/list/{page}/{limit} [get]
// @Example Request: GET /story/v01/follower/list/1/20
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":2,"list":[{"uid":77645423236541,"nick":"test1","thumb_pic":"https://.../thumb.jpg","at_update":"2026-06-23T14:30:15Z"}]}}
func (p *StoryController) GetFollowerList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	uid64 := user.(*ptc.UserInfoResp).Uid
	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	list, totalCount, err := p.sdb.GetFollowerList(uid64, nPage, nLimit)
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
// @Description  JWT 본인을 follower로 하는 활성 팔로잉(`user_follow.stat=1`) 목록을 페이징 조회합니다.
// @Description
// @Description  **조회 규칙**
// @Description  - JWT uid = `follower_uid` (내가 팔로우한 사람들)
// @Description  - `at_update DESC` 정렬
// @Description
// @Description  **응답 data**
// @Description  - `total_count`: 전체 팔로잉 수
// @Description  - `list`: `FollowUserItem` 배열 (`uid`, `nick`, `thumb_pic`, `at_update`)
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        page           path    int     true  "페이지 번호(1부터, 기본 1)"
// @Param        limit          path    int     true  "페이지 크기(기본 20)"
// @Success      200            {object} map[string]interface{} "total_count, list(FollowUserItem[])"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "팔로잉 목록 조회 실패"
// @Router       /story/v01/following/list/{page}/{limit} [get]
// @Example Request: GET /story/v01/following/list/1/20
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":2,"list":[{"uid":4033287471439576593,"nick":"푸른 아름다운 양","thumb_pic":"https://.../thumb.jpg","at_update":"2026-06-23T14:30:15Z"}]}}
func (p *StoryController) GetFollowingList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	list, totalCount, err := p.sdb.GetFollowingList(uid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get following list", err)
		return
	}
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"total_count": totalCount,
		"list":        *list,
	})
}

// -------------------- user cut out ---------------------------------

// SetCutoutUser godoc
// @Summary      스토리 cutout 설정
// @Description  JWT 본인이 대상(`tid`)을 스토리 피드에서 숨깁니다. 서비스 전체 차단(`user_block`)과 별개입니다.
// @Description
// @Description  **규칙**
// @Description  - JWT uid = cutout 주체, path `tid` = 숨길 대상
// @Description  - 자기 자신 cutout 불가 (`self cutout is not allowed`)
// @Description  - 대상 미존재/성별 파싱 실패 시 400
// @Description
// @Description  **처리**
// @Description  - `GetUserInfoByUID(tid)`로 프로필 스냅샷 조회
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/cutout/set/4033287471439576593
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
// @Tags         Story_Cutout
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "cutout 대상 사용자 UID"
// @Success      200            {object} protocol.CutoutActionResp "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid invalid / self cutout / target user not found / gender invalid"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "str_cutout 저장 실패"
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "affected": affected})
}

// UnsetCutoutUser godoc
// @Summary      스토리 cutout 해제
// @Description  JWT 본인이 대상(`tid`)에 대한 스토리 피드 숨김을 해제합니다 (`str_cutout.stat=0`).
// @Description
// @Description  **규칙**
// @Description  - JWT uid = cutout 주체, path `tid` = 해제 대상
// @Description
// @Description  **처리** (`UnsetCutoutUser`)
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/cutout/cancel/4033287471439576593
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","affected":1}}
// @Tags         Story_Cutout
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        tid            path    string  true  "cutout 해제 대상 사용자 UID"
// @Success      200            {object} protocol.CutoutActionResp "msg, affected"
// @Failure      400            {object} map[string]interface{} "tid invalid"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "str_cutout 갱신 실패"
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "affected": affected})
}

// GetCutoutCount godoc
// @Summary      스토리 cutout 개수 조회
// @Description  JWT 본인의 활성 cutout(`str_cutout.stat=1`) 개수를 반환합니다.
// @Description
// @Description  **Request example**
// @Description  - POST /story/v01/cutout/count
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","count":3}}
// @Tags         Story_Cutout
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Success      200            {object} protocol.CutoutCountResp "msg, count"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "개수 조회 실패"
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "count": count})
}

// GetCutoutList godoc
// @Summary      스토리 cutout 목록 조회
// @Description  JWT 본인의 활성 cutout(`str_cutout.stat=1`) 목록을 페이징 조회합니다.
// @Description
// @Description  **조회 규칙**
// @Description  - JWT uid 기준, `at_upd DESC` 정렬
// @Description  - `page`/`limit`은 path에서 `getPageAndLimit` 파싱 (빈 값이면 page=1, limit=20)
// @Description
// @Description  **응답 data**
// @Description  - `msg`: `"ok"`
// @Description  - `total_count`: 전체 활성 cutout 수
// @Description  - `list`: `CutoutUserItem` 배열
// @Description    (`idx`, `tid`, `stat`, `tnick`, `tgen`, `tbirth`, `age`, `tsp_intro`, `tarea`, `tthmb_pic`, `at_crt`, `at_upd`)
// @Description
// @Description  **Request example**
// @Description  - GET /story/v01/cutout/list/1/10
// @Description  - Header: Authorization: Bearer {access_token}
// @Description
// @Description  **Response example**
// @Description  {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":1,"list":[{"idx":1,"tid":4033287471439576593,"stat":1,"tnick":"푸른 아름다운 양","tgen":1,"tbirth":"1991-12-18","age":34,"tsp_intro":"반가워요 큐피톡에서 만나요!","tarea":1,"tthmb_pic":"https://i.ibb.co/99MhfMXt/icon-male-03.webp","at_crt":"2026-06-23T14:30:15Z","at_upd":"2026-06-23T14:30:15Z"}]}}
// @Tags         Story_Cutout
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization  header  string  true  "Bearer access token"
// @Param        page           path    int     true  "페이지 번호(1부터, 기본 1)"
// @Param        limit          path    int     true  "페이지 크기(기본 20)"
// @Success      200            {object} protocol.CutoutListResp "msg, total_count, list(CutoutUserItem[])"
// @Failure      400            {object} map[string]interface{} "page/limit 잘못된 값"
// @Failure      401            {object} map[string]interface{} "JWT 인증 실패"
// @Failure      500            {object} map[string]interface{} "목록 조회 실패"
// @Router       /story/v01/cutout/list/{page}/{limit} [get]
func (p *StoryController) GetCutoutList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	list, totalCount, err := p.sdb.GetCutoutList(uid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get cutout list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "total_count": totalCount, "list": list})
}

// -------------------- user cut out ---------------------------------
