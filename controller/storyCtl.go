package controller

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptc "ms-gateway/protocol"
	"net/http"
	"strconv"
	"strings"
	"time"

	haredis "ms-gateway/common/ha-redis"
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
	hch *haredis.HAChecker

	adb *models.AccountDB
	hdb *models.HistoryDB
	sdb *models.StoryDB
	rdb *models.RedisDB
}

func NewStoryController(ctl *Controller, hca *haredis.HAChecker, rep *models.Repositories) (*StoryController, error) {
	r := &StoryController{
		ctl: ctl,
		rep: rep,
		hch: hca,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.hdb, &r.sdb, &r.rdb); err != nil {
		return nil, err
	}

	go r.JobStoryList()
	return r, nil
}

func (p *StoryController) JobStoryList() {
	if err := p.initStory(); err != nil {
		log.Warn("Failed to initialize data: ", err)
	}

	tickStoryList := time.NewTicker(3 * time.Minute)
	defer tickStoryList.Stop()

	for range tickStoryList.C {
		if err := p.initStory(); err != nil {
			log.Warn("Failed to initialize data: ", err)
		}
	}
}

func (p *StoryController) initStory() error {
	if p.hch == nil || !p.hch.IsLeader() {
		return nil
	}

	list, total, err := p.sdb.GetStory1000()
	if err != nil {
		return err
	}
	return p.rdb.SetStory1000(list, total)
}

func (p *StoryController) GetCachedStory(viewerUID, ownerUID uint64, nPage int, nLimit int) (*[]ptc.StoryDetailResp, int, error) {
	if nPage < 1 || nLimit < 1 {
		return nil, 0, fmt.Errorf("page or limit is invalid")
	}

	current, total, err := p.sdb.GetStoryPage(viewerUID, ownerUID, nPage, nLimit)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int, len(current))
	for i, entry := range current {
		ids[i] = entry.Story.Idx
	}
	cached, _ := p.rdb.GetStoryEntries(ids) // Redis failure falls back to a single batch load.
	missing := make([]int, 0)
	for _, ref := range current {
		if entry, ok := cached[ref.Story.Idx]; !ok || !entry.Matches(ref) {
			missing = append(missing, ref.Story.Idx)
			delete(cached, ref.Story.Idx)
		}
	}
	refreshed, err := p.sdb.GetStoryCacheEntries(missing)
	if err != nil {
		return nil, 0, err
	}
	if cached == nil {
		cached = make(map[int]models.StoryCacheEntry)
	}
	// Cache only viewer-independent originals.
	if len(refreshed) > 0 {
		_ = p.rdb.SetStory1000(refreshed, len(refreshed))
	}
	for _, entry := range refreshed {
		cached[entry.Story.Idx] = entry
	}
	list := make([]ptc.StoryDetailResp, 0, len(current))
	for _, ref := range current {
		entry, ok := cached[ref.Story.Idx]
		if !ok {
			continue
		}

		story := entry.Story
		if entry.Matches(ref) {
			story.QtLike, story.QtView, story.Price = ref.Story.QtLike, ref.Story.QtView, ref.Story.Price
		}
		list = append(list, story)
	}
	list, err = p.sdb.RefreshStoryList(list, viewerUID)
	if err != nil {
		return nil, 0, err
	}

	return &list, total, nil
}

// storyActorUID reads only the identity established by JWT authentication.
func (p *StoryController) storyActorUID(c *gin.Context) (uint64, bool) {
	value, exists := c.Get("user")
	user, valid := value.(*ptc.UserInfoResp)
	if !exists || !valid || user == nil || user.Uid == 0 {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return 0, false
	}
	return user.Uid, true
}

func (p *StoryController) requireProfileAdmin(c *gin.Context) bool {
	uid, ok := p.storyActorUID(c)
	if !ok {
		return false
	}
	if p.cfg != nil {
		for _, adminUID := range p.cfg.Server.ProfileAdminUIDs {
			if uid == adminUID {
				return true
			}
		}
	}
	p.ctl.SimpleError(c, http.StatusForbidden, "Profile administrator required")
	return false
}

// GetStoryList godoc
// @Summary      작성자별 스토리 목록 조회
// @Description  작성자별 스토리 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100. tid가 있는 경로는 해당 작성자, 없는 경로는 JWT 본인을 조회합니다.
// @Description
// @Description  **처리**: viewer는 항상 JWT UID입니다. 목록과 COUNT에 owner UID를 적용합니다. 삭제 글과 타인의 비공개·예약 글은 제외합니다. FLW 글은 팔로우 권한, PAY 글은 구매 권한이 없으면 본문·댓글을 숨기며 미디어 URL·썸네일은 유지하고 is_locked로 클라이언트에서 표시를 제어합니다. GET은 조회수를 변경하지 않습니다.
// @Description
// @Description  목록의 댓글·댓글 수·팔로워 수는 캐시 적중 시에도 DB에서 최신 값으로 갱신합니다.
// @Description  **응답 data**: page, limit, total_count, story(StoryDetailResp 배열). 관계 필드 is_locked는 최종 잠금 여부(1=잠김, 0=열림), good_byme는 JWT 사용자의 좋아요 여부(1=좋아요, 0=아님)입니다. is_flwd/is_paid는 글 공개 상태와 별도로 계산합니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=object{page=int,limit=int,total_count=int,story=[]protocol.StoryDetailResp}} "data: total_count, story(StoryDetailResp 배열). 관계 필드 is_locked는 최종 잠금 여부(1=잠김, 0=열림), good_byme는 JWT 사용자의 좋아요 여부(1=좋아요, 0=아님)입니다. is_flwd/is_paid는 글 공개 상태와 별도로 계산합니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/list/{page}/{limit} [get]
// @Router       /story/v01/list/{page}/{limit}/{tid} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @Example Request: GET /story/v01/list/1/10
func (p *StoryController) GetStoryList(c *gin.Context) {
	// 스토리 리스트 클릭후 -> 해당 사용자의 스토리 모음으로 이동 -> 해당 스토리 인덱스가 최상위, 그 밑으로 최신순으로 스토리 리스트 출력
	// 출력 내역 : 해당대상자 프로필아이콘, 닉네임, 나이, 지역, 팔로잉 여부, 스토리컨텐츠 리스트, 전체 컨텐츠수, 본문, 좋아요, 댓글, 조회수, 생성일자, 댓글 최신 2개,
	// 댓글 내역 : 아이콘프로필, 성별, 닉네임, 지역, 나이, 댓글내역 - ptc.StrComment
	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	// A path tid selects the owner, never the authenticated viewer.
	viewerUID := user.(*ptc.UserInfoResp).Uid
	ownerUID := viewerUID
	if tid := c.Param("tid"); tid != "" {
		ownerUID, err = strconv.ParseUint(tid, 10, 64)
		if err != nil || ownerUID == 0 {
			p.ctl.SimpleError(c, http.StatusBadRequest, "tid is invalid")
			return
		}
	}

	story, cnt, err := p.GetCachedStory(viewerUID, ownerUID, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story detail", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"page": nPage, "limit": nLimit, "total_count": cnt, "story": story})
}

// GetStoryConditionList godoc
// @Summary      조건별 스토리 목록 조회
// @Description  조건별 스토리 목록 조회 API입니다.
// @Description
// @Description  **요청**: path area/stat/type/order/gen/page/limit. stat: all 또는 pub=0/flw=1/pay=2/prv=3/del=4/rsv=5. type: all 또는 img=0/vdo=1/txt=2/mix=3/rsv=4.
// @Description
// @Description  **처리**: all은 필터 생략이며 저장 enum이 아닙니다. 잘못된 stat/type은 400입니다. txt=2는 미디어 없는 텍스트 글, mix=3은 이미지·동영상 혼합입니다. JWT viewer의 가시성·본문 권한과 활성 cutout 제외를 적용합니다. order: old=오래된순, god=qt_like, cmt=공개 댓글 수, view=qt_view, 나머지=최신순이며 모든 정렬에 idx 보조키를 적용합니다. 조건 피드는 PUB=0, FLW=1, PAY=2와 본인 PRV=3을 조회합니다. 6=all은 상태 필터를 생략하며 삭제·예약 및 타인의 비공개 글은 제외합니다. FLW/PAY는 권한이 없으면 원문을 숨깁니다. gen 파싱 실패 시 1입니다.
// @Description
// @Description  **응답 data**: page, limit, total_count, story(StoryDetailResp 배열).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        area path string true "지역 코드 또는 all"
// @Param        stat path string true "상태: 0=pub, 1=flw, 2=pay, 3=prv, 6=all (문자열 이름도 허용)" Enums(0,1,2,3,6,pub,flw,pay,prv,all)
// @Param        type path string true "미디어: all/img/vdo/txt/mix/rsv"
// @Param        order path string true "정렬: new/old/cmt/god/view/rsv"
// @Param        gen path string true "성별: 0=여성, 1=남성"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{page=int,limit=int,total_count=int,story=[]protocol.StoryDetailResp}} "data: total_count, story(StoryDetailResp 배열)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/cndt/list/{area}/{stat}/{type}/{order}/{gen}/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @Example Request: GET /story/v01/cndt/list/all/6/all/new/1/1/10
func (p *StoryController) GetStoryConditionList(c *gin.Context) {
	// 홈 디폴트 리스트, 최신 등록순으로 전체 고객 스토리 리스트 출력, 해당 리스트 출력시 대상 사용자의 스토리로 이동해야함
	area := switchDefParam(c.Param("area"), "all")
	stat := c.Param("stat")
	mtype := c.Param("type")
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

	nMType, err := ptl.GetTypeCode(mtype)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "type is invalid", err)
		return
	}
	if nMType != ptl.StoryFilterAll {
		conds = append(conds, "type = ?")
		args = append(args, nMType)
	}

	nStat, err := ptl.GetStatCode(stat)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "stat is invalid", err)
		return
	}
	if nStat != ptl.StoryStatAll {
		conds = append(conds, "stat = ?")
		args = append(args, nStat)
	}

	// 요청 사용자 cutout 목록은 story feed에서 제외한다.
	var uid64 uint64
	user, exists := c.Get("user")
	if exists {
		uid64 = user.(*ptc.UserInfoResp).Uid
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
	storyList, cnt, err := p.sdb.GetCondStoryList(uid64, conds, orderQuery, args)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	// p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"story_home_list": storyList})
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"page": nPage, "limit": nLimit, "total_count": cnt, "story": storyList})
}

// GetFollowingStory godoc
// @Summary      팔로우 작성자의 스토리 목록 조회
// @Description  팔로우 작성자의 스토리 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: JWT 사용자의 활성 팔로우 관계로 조회하며 스토리 가시성과 구매 권한을 적용합니다.
// @Description
// @Description  **응답 data**: page, limit, total_count, story(StoryDetailResp 배열).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{page=int,limit=int,total_count=int,story=[]protocol.StoryDetailResp}} "data: total_count, story(StoryDetailResp 배열)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/flwlist/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @Example Request: GET /story/v01/flwlist/1/10
func (p *StoryController) GetFollowingStory(c *gin.Context) {
	// 내가 팔로우 -> 팔로윙중인 컨텐츠
	nPage, nLimit, err := getPageAndLimit(c.Param("page"), c.Param("limit"))
	if err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "page or limit is invalid", err)
		return
	}

	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	uid64 := user.(*ptc.UserInfoResp).Uid
	story, cnt, err := p.sdb.GetFlwStoryList(uid64, nPage, nLimit)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story list", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"page": nPage, "limit": nLimit, "total_count": cnt, "story": story})
}

// GetStoryDetail godoc
// @Summary      스토리 단건 상세 조회
// @Description  스토리 단건 상세 조회 API입니다.
// @Description
// @Description  **요청**: path idx: 스토리 인덱스.
// @Description
// @Description  **처리**: JWT viewer의 가시성·활성 팔로우·완료 구매 권한을 적용합니다. 접근 가능한 행이 없으면 404입니다. GET은 조회수를 변경하지 않습니다.
// @Description
// @Description  **응답 data**: StoryDetailResp. 실제 JSON 필드는 ctnt_total, qt_like, is_locked(0/1), good_byme(0/1), cmt_list, is_flwd, is_paid입니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Success      200 {object} protocol.RespDataHeader{data=protocol.StoryDetailResp} "data: StoryDetailResp. 실제 JSON 필드는 ctnt_total, qt_like, is_locked(0/1), good_byme(0/1), cmt_list, is_flwd, is_paid입니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "대상 없음 또는 조회 가능한 대상 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/detail/{idx} [get]
// @Example Request: GET /story/v01/detail/15
func (p *StoryController) GetStoryDetail(c *gin.Context) {
	// 단건 디테일 조회
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
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	uid64 := user.(*ptc.UserInfoResp).Uid
	story, err := p.sdb.GetStoryDetail(uid64, idxInt)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
		return
	}
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story detail", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, story)
}

// GetPurchaseList godoc
// @Summary      완료 구매 스토리 목록 조회
// @Description  완료 구매 스토리 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: 구매자는 JWT UID입니다. psc_stat=1인 유효한 완료 구매만 조회하며 스토리 가시성을 적용합니다. 금액이나 구매 행 존재만으로 권한을 부여하지 않습니다.
// @Description
// @Description  **응답 data**: page, limit, total_count, story(StoryDetailResp 배열).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{page=int,limit=int,total_count=int,story=[]protocol.StoryDetailResp}} "data: total_count, story(StoryDetailResp 배열)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/pcs/list/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"page":1,"limit":10,"total_count":0,"story":[]}}
// @Example Request: GET /story/v01/pcs/list/1/10
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"page": nPage, "limit": nLimit, "total_count": totalCount, "story": list})
}

// BuyStory godoc
// @Summary      스토리 구매 완료
// @Description  **요청**: path idx는 양의 스토리 인덱스입니다. body는 사용하지 않습니다.
// @Description
// @Description  **처리**: 구매자는 JWT UID입니다. 타인의 유료 공개(stat=2) 글만 요청할 수 있으며 양방향 차단을 검사합니다.
// @Description  현재 결제 없는 완료 정책으로 psc_stat=1, qt_payed=0을 저장하여 열람 권한을 부여합니다. 결제 및 포인트 차감은 수행하지 않습니다.
// @Description  기존 대기(0) 요청은 재요청 시 완료(1)로 변경합니다. 완료 요청은 중복 저장하지 않으며 취소·회수 상태는 409로 거절합니다.
// @Description
// @Description  **응답 data**: msg=ok, psc_stat=1. HTTP 200으로 반환합니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "구매 요청할 스토리 인덱스"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,psc_stat=int}} "구매 완료"
// @Failure      400 {object} protocol.RespHeader "잘못된 idx 또는 구매 요청 불가 대상"
// @Failure      401 {object} protocol.RespHeader "JWT 인증 실패"
// @Failure      403 {object} protocol.RespHeader "차단 관계"
// @Failure      404 {object} protocol.RespHeader "스토리 없음"
// @Failure      409 {object} protocol.RespHeader "취소·회수된 구매"
// @Failure      500 {object} protocol.RespHeader "조회 또는 저장 실패"
// @Router       /story/v01/buy/{idx} [post]
// @Example Request: POST /story/v01/buy/15
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","psc_stat":1}}
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","psc_stat":1}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","psc_stat":1}}
func (p *StoryController) BuyStory(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}
	idx, err := strconv.Atoi(c.Param("idx"))
	if err != nil || idx <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is invalid")
		return
	}
	ownerUID, err := p.sdb.GetStoryOwnerUID(idx)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
		return
	}
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story owner", err)
		return
	}
	if ownerUID == actorUID {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Cannot purchase own story")
		return
	}
	blocked, err := p.adb.IsBlockedPair(actorUID, ownerUID)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check block relation", err)
		return
	}
	if blocked {
		p.ctl.SimpleError(c, http.StatusForbidden, "Blocked relation")
		return
	}
	status, err := p.sdb.BuyStory(actorUID, idx)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
	case errors.Is(err, models.ErrStoryPurchaseInvalid), errors.Is(err, models.ErrStoryPurchaseNotPaid):
		p.ctl.SimpleError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrStoryPurchaseRevoked):
		p.ctl.SimpleError(c, http.StatusConflict, err.Error())
	case err != nil:
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to save purchase request", err)
	default:
		p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "psc_stat": status})
	}
}

// UpdateStoryViewCnt godoc
// @Summary      스토리 조회 이력 갱신
// @Description  스토리 조회 이력 갱신 API입니다.
// @Description
// @Description  **요청**: path idx: 양의 스토리 인덱스.
// @Description
// @Description  **처리**: JWT 사용자별 24시간 중복 조회를 억제하여 조회 이력과 qt_view를 갱신합니다. 없는 글은 404입니다.
// @Description
// @Description  **응답 data**: msg만 반환합니다. 조회수는 /view/count/{idx}에서 조회합니다.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string}} "data.msg만 반환합니다. 조회수는 /view/count/{idx}에서 조회합니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "대상 없음 또는 조회 가능한 대상 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/view/update/{idx} [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok"}}
// @Example Request: POST /story/v01/view/update/15
func (p *StoryController) UpdateStoryViewCnt(c *gin.Context) {
	idx := c.Param("idx")
	idxInt, err := strconv.Atoi(idx)
	if err != nil || idxInt <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx is required")
		return
	}

	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	uid64 := user.(*ptc.UserInfoResp).Uid

	_, err = p.UpdateViewHistory(uid64, idxInt)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
		return
	}
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story view", err)
		return
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok"})
}

func (p *StoryController) UpdateViewHistory(uid uint64, idx int) (bool, error) {
	_, err := p.sdb.UpdateStoryView(uid, idx)
	// Both a new count and an already-counted view are successful requests.
	return err == nil, err
}

// GetStoryViewCount godoc
// @Summary      스토리 조회수 조회
// @Description  스토리 조회수 조회 API입니다.
// @Description
// @Description  **요청**: path idx: 스토리 인덱스.
// @Description
// @Description  **처리**: 해당 스토리의 카운트를 조회합니다.
// @Description
// @Description  **응답 data**: msg, count.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,count=int}} "data: msg, count."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/view/count/{idx} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","count":17}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","count":17}}
// @Example Request: GET /story/v01/view/count/15
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
// @Description  스토리 좋아요 수 조회 API입니다.
// @Description
// @Description  **요청**: path idx: 스토리 인덱스.
// @Description
// @Description  **처리**: 해당 스토리의 카운트를 조회합니다.
// @Description
// @Description  **응답 data**: msg, count.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        idx path int true "대상 인덱스"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,count=int}} "data: msg, count."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/like/count/{idx} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","count":17}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","count":17}}
// @Example Request: GET /story/v01/like/count/15
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
// @Description  스토리 좋아요 토글 API입니다.
// @Description
// @Description  **요청**: body StoryLikeToggleReq: story_idx는 양의 정수 문자열. uid는 호환용으로 허용하지만 무시합니다.
// @Description
// @Description  **처리**: JWT UID를 행위자로 사용합니다. 작성자 조회 후 양방향 차단을 검사하고 ToggleStoryLikeTx로 좋아요와 카운트를 갱신합니다. 없는 글은 404, 차단 관계는 403입니다.
// @Description
// @Description  **응답**: 최상위 msg, like_cnt. data 래퍼가 없습니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.StoryLikeToggleReq true "body StoryLikeToggleReq: story_idx는 양의 정수 문자열. uid는 호환용으로 허용하지만 무시합니다."
// @Success      200 {object} object{msg=string,like_cnt=int} "최상위 msg, like_cnt. data 래퍼가 없습니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      403 {object} protocol.RespHeader "권한 없음 또는 차단 관계"
// @Failure      404 {object} protocol.RespHeader "대상 없음 또는 조회 가능한 대상 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/like/toggle [post]
// @Example Response: {"msg":"ok","like_cnt":17}
// @x-response-example {"msg":"ok","like_cnt":17}
// @Example Request: POST /story/v01/like/toggle
// @Example Request Body: {"story_idx":"15","uid":"4033287471439576593"}
// @x-request-example {"story_idx":"15","uid":"4033287471439576593"}
func (p *StoryController) UpdateStoryLike(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}

	var req ptl.StoryLikeToggleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}
	if req.StoryIdx == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StoryIdx is required")
		return
	}

	storyIdx, err := strconv.Atoi(req.StoryIdx)
	if err != nil || storyIdx <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "StoryIdx is invalid")
		return
	}
	ownerUID, err := p.sdb.GetStoryOwnerUID(storyIdx)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
		return
	}
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get story owner", err)
		return
	}
	blocked, err := p.adb.IsBlockedPair(actorUID, ownerUID)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to check block relation", err)
		return
	}
	if blocked {
		p.ctl.SimpleError(c, http.StatusForbidden, "Blocked relation")
		return
	}

	_, likeCount, err := p.sdb.ToggleStoryLikeTx(storyIdx, actorUID)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to toggle story like", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "like_cnt": likeCount})
}

// UploadStoryContent godoc
// @Summary      스토리 미디어 업로드 (Cloudflare)
// @Description  스토리 미디어 업로드 (Cloudflare) API입니다.
// @Description
// @Description  **요청**: multipart files: 최대 5개, 이미지·thbnl 각 5MiB 이하, MP4 각 100MiB 이하. 동영상마다 thbnl 이미지가 필요합니다.
// @Description
// @Description  **처리**: ValidateFileUpload(5,5) 검증 후 업로드하고 DB 저장 없이 슬롯 맵을 반환합니다. 현재 UpTotalCldFlr의 동영상 분기는 실제 Stream 업로드 대신 예시 URL을 반환합니다. **현재 제한**: 동영상 업로드 완료를 보장하지 않습니다. 반환 media를 create body로 전달합니다.
// @Description
// @Description  **응답 data**: msg, media. 슬롯 키는 1..5, type은 img/vdo, url은 본편, 동영상 썸네일 JSON 키는 **thumb1**입니다(Go 필드 Thumb).
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Description  업로드 미들웨어 오류는 JSON 문자열 또는 error 객체일 수 있으며 핸들러의 RespHeader 오류와 형식이 다릅니다.
// @Description  **업로드 공통 제한**: 요청 전체 크기·동시 업로드 제한을 초과하면 413/503입니다. 기본값은 530MiB/8건이며 서버 설정에 따라 달라집니다.
// @Tags         Story
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        files formData file true "업로드 파일 (동일 필드명으로 여러 파일 전송 가능)"
// @Param        thbnl formData file false "동영상마다 필수인 썸네일 이미지, 순서 1:1"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,media=protocol.StoryStrImg}} "data: msg, media. 슬롯 키는 1..5, type은 img/vdo, url은 본편, 동영상 썸네일 JSON 키는 **thumb1**입니다(Go 필드 Thumb)."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      413 {object} map[string]interface{} "업로드 요청 크기 제한 초과"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Failure      503 {object} map[string]interface{} "서버 동시 처리·연결 한도 초과"
// @Router       /story/v01/upload [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","media":{"1":{"type":"img","url":"https://imagedelivery.net/example/public"}}}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","media":{"1":{"type":"img","url":"https://imagedelivery.net/example/public"}}}}
// @Example Request: POST /story/v01/upload
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
// @Description  스토리 생성 API입니다.
// @Description
// @Description  **요청**: body CreateStoryReq: stat(int, 0=pub/1=flw/2=pay/3=prv/4=del/5=rsv), sbody, price(int), media(0~5 슬롯, 없으면 sbody 필수). 동영상 항목은 thumb1이 필요합니다. stat 생략 시 0(PUB)입니다. 6(ALL)은 조회 전용입니다.
// @Description
// @Description  **처리**: JWT에서 uid/nick/birth/gender/area를 채웁니다. 상태 범위, 프로필 필수 값, 미디어 슬롯·type·url·thumb1을 검증하고 str_img/type을 정규화하여 INSERT합니다. 파일 업로드는 별도 /story/v01/upload입니다. sbody 512자 제한은 이 핸들러에서 직접 검증하지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg, idx. data 래퍼가 없습니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.CreateStoryReq true "body CreateStoryReq: stat(int, 0=pub/1=flw/2=pay/3=prv/4=del/5=rsv), sbody, price(int), media(0~5 슬롯, 없으면 sbody 필수). 동영상 항목은 thumb1이 필요합니다. stat 생략 시 0(PUB)입니다. 6(ALL)은 조회 전용입니다."
// @Success      200 {object} object{msg=string,idx=int} "최상위 msg, idx. data 래퍼가 없습니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/create [post]
// @Example Response: {"msg":"ok","idx":5}
// @x-response-example {"msg":"ok","idx":5}
// @Example Request: POST /story/v01/create
// @Example Request Body: {"stat":1,"sbody":"오늘의 스토리","price":0,"media":{"1":{"type":"img","url":"https://imagedelivery.net/example/public"},"2":{"type":"vdo","url":"https://example.com/video.m3u8","thumb1":"https://imagedelivery.net/example-thumb/public"}}}
// @x-request-example {"stat":1,"sbody":"오늘의 스토리","price":0,"media":{"1":{"type":"img","url":"https://imagedelivery.net/example/public"},"2":{"type":"vdo","url":"https://example.com/video.m3u8","thumb1":"https://imagedelivery.net/example-thumb/public"}}}
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

	if !ptl.IsValidStoryStat(sinfo.Stat) {
		p.ctl.SimpleError(c, http.StatusBadRequest, "stat must be 0=PUB, 1=FLW, 2=PAY, 3=PRV, 4=DEL or 5=RSV; 6=ALL is query-only")
		return
	}

	if len(sinfo.Media) == 0 && strings.TrimSpace(sinfo.Sbody) == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "sbody is required for text stories")
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

	// p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "idx": lastID})
	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "idx": lastID, "data": simg})
}

// UpdateStoryStat godoc
// @Summary      본인 스토리 상태 변경
// @Description  본인 스토리 상태 변경 API입니다.
// @Description
// @Description  **요청**: body UpdateStrStatReq: idx와 stat은 숫자 문자열. stat은 0=pub/1=flw/2=pay/3=prv/4=del/5=rsv입니다.
// @Description
// @Description  **처리**: JWT UID를 행위자로 사용하고 idx AND uid 조건으로 갱신합니다. 콘텐츠 revision도 갱신합니다. 다른 작성자 또는 없는 글은 404입니다.
// @Description res = 1 : 정상 업데이트, res = 0 : 정상이나, 변경된 내용은 없음,  본인 글의 동일 값 재요청은 res=0일 수 있습니다.
// @Description
// @Description  **응답**: 최상위 msg, res. data 래퍼가 없습니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.UpdateStrStatReq true "body UpdateStrStatReq: idx와 stat은 숫자 문자열. stat은 0=pub/1=flw/2=pay/3=prv/4=del/5=rsv입니다."
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res. data 래퍼가 없습니다."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "스토리가 없거나 본인 소유가 아님"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/updstat [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/updstat
// @Example Request Body: {"idx":"15","stat":"2"}
// @x-request-example {"idx":"15","stat":"2"}
func (p *StoryController) UpdateStoryStat(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}

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
	if err != nil || strIdx <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx must be a positive integer")
		return
	}

	nstat, err := strconv.Atoi(req.Stat)
	if err != nil || !ptl.IsValidStoryStat(nstat) {
		p.ctl.SimpleError(c, http.StatusBadRequest, "stat must be 0=PUB, 1=FLW, 2=PAY, 3=PRV, 4=DEL or 5=RSV; 6=ALL is query-only")
		return
	}

	affected, err := p.sdb.UpdateStoryStat(strIdx, actorUID, nstat)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found or not owned by current user")
		return
	}

	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story stat", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// UpdStoryPriceInfo godoc
// @Summary 본인 스토리 상태 및 가격 변경
// @Description JWT UID와 idx로 소유권을 확인해 stat과 price를 함께 변경합니다. idx/stat은 숫자 문자열, price는 0 이상의 정수입니다.
// @Tags Story
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body protocol.UpdateStoryPriceReq true "스토리 ID, 상태, 가격"
// @Success 200 {object} object{msg=string,res=int}
// @Failure 400 {object} protocol.RespHeader
// @Failure 401 {object} protocol.RespHeader
// @Failure 404 {object} protocol.RespHeader "스토리가 없거나 본인 소유가 아님"
// @Failure 500 {object} protocol.RespHeader
// @Router /story/v01/updprice [post]
func (p *StoryController) UpdStoryPriceInfo(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}

	var req ptl.UpdateStoryPriceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Failed to bind JSON", err)
		return
	}

	if req.Idx == "" || req.Stat == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx and Stat are required")
		return
	}

	strIdx, err := strconv.Atoi(req.Idx)
	if err != nil || strIdx <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx must be a positive integer")
		return
	}

	nstat, err := strconv.Atoi(req.Stat)
	if err != nil || !ptl.IsValidStoryStat(nstat) {
		p.ctl.SimpleError(c, http.StatusBadRequest, "stat must be 0=PUB, 1=FLW, 2=PAY, 3=PRV, 4=DEL or 5=RSV; 6=ALL is query-only")
		return
	}

	if req.Price == nil || *req.Price < 0 || *req.Price > 2147483647 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "price must be an integer between 0 and 2147483647")
		return
	}

	affected, err := p.sdb.UpdatePriceStat(strIdx, actorUID, nstat, *req.Price)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found or not owned by current user")
		return
	}

	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story price and stat", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// UpdateStrBody godoc
// @Summary      본인 스토리 본문 변경
// @Description  본인 스토리 본문 변경 API입니다.
// @Description
// @Description  **요청**: body UpdateStrBodyReq: idx는 숫자 문자열, body는 비어 있지 않은 문자열.
// @Description
// @Description  **처리**: JWT UID와 idx를 함께 조건으로 갱신하고 콘텐츠 revision을 갱신합니다. 다른 작성자 또는 없는 글은 404입니다. 본인 글의 동일 값 재요청은 res=0일 수 있습니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.UpdateStrBodyReq true "body UpdateStrBodyReq: idx는 숫자 문자열, body는 비어 있지 않은 문자열."
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "스토리가 없거나 본인 소유가 아님"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/updbody [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/updbody
// @Example Request Body: {"idx":"15","body":"수정한 본문"}
// @x-request-example {"idx":"15","body":"수정한 본문"}
func (p *StoryController) UpdateStrBody(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}

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
	if err != nil || strIdx <= 0 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Idx must be a positive integer")
		return
	}

	affected, err := p.sdb.UpdateStrBody(strIdx, actorUID, req.Body)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found or not owned by current user")
		return
	}

	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update story body", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// DeleteStrPic godoc
// @Summary      본인 스토리 미디어 삭제
// @Description  본인 스토리 미디어 삭제 API입니다.
// @Description
// @Description  **요청**: body DeleteStrPicReq: idx와 pic_idx는 문자열. pic_idx는 삭제할 슬롯입니다.
// @Description
// @Description  **처리**: JWT 소유자 조건으로 스토리를 조회합니다. 없는 글·타인 글은 404, 유효하지 않은 슬롯은 400입니다. 삭제 후 슬롯을 재정렬하고 콘텐츠 revision을 갱신합니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        request body protocol.DeleteStrPicReq true "body DeleteStrPicReq: idx와 pic_idx는 문자열. pic_idx는 삭제할 슬롯입니다."
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      404 {object} protocol.RespHeader "대상 없음 또는 조회 가능한 대상 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/delpic [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/delpic
// @Example Request Body: {"idx":"15","pic_idx":"1"}
// @x-request-example {"idx":"15","pic_idx":"1"}
func (p *StoryController) DeleteStrPic(c *gin.Context) {
	actorUID, ok := p.storyActorUID(c)
	if !ok {
		return
	}

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
	strImg, strImgBak, err := p.sdb.GetStrPicList(strIdxInt, actorUID)
	if errors.Is(err, sql.ErrNoRows) {
		p.ctl.SimpleError(c, http.StatusNotFound, "Story not found")
		return
	}
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
	affected, err := p.sdb.DeleteStrPic(strIdxInt, actorUID, mediaBytes, bakMediaBytes)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to delete str pic", err)
		return
	}
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// FollowUser godoc
// @Summary      팔로우 등록
// @Description  팔로우 등록 API입니다.
// @Description
// @Description  **요청**: path tid: 대상 UID. 본인 UID는 허용하지 않습니다. 요청 body는 없습니다.
// @Description
// @Description  **처리**: 행위자는 JWT UID입니다. 팔로우 관계를 활성 상태(stat=1)로 등록합니다. 현재 이 핸들러에서는 양방향 차단을 검사하지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/follow/set/{tid} [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/follow/set/4033287471439576593
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
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// UnfollowUser godoc
// @Summary      팔로우 해제
// @Description  팔로우 해제 API입니다.
// @Description
// @Description  **요청**: path tid: 대상 UID. 본인 UID는 허용하지 않습니다. 요청 body는 없습니다.
// @Description
// @Description  **처리**: 행위자는 JWT UID입니다. 팔로우 관계를 비활성 상태(stat=0)로 변경합니다. 현재 이 핸들러에서는 양방향 차단을 검사하지 않습니다.
// @Description
// @Description  **응답**: 최상위 msg, res.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} object{msg=string,res=int} "최상위 msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/follow/cancel/{tid} [post]
// @Example Response: {"msg":"ok","res":1}
// @x-response-example {"msg":"ok","res":1}
// @Example Request: POST /story/v01/follow/cancel/4033287471439576593
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
	p.ctl.SimpleRespOK(c, gin.H{"msg": "ok", "res": affected})
}

// GetFollowerList godoc
// @Summary      내 팔로워 목록
// @Description  내 팔로워 목록 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: JWT 사용자의 활성 관계만 조회합니다.
// @Description
// @Description  **응답 data**: total_count, list.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{total_count=int,list=[]protocol.FollowUserItem}} "data: total_count, list."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/follower/list/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":0,"list":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"total_count":0,"list":[]}}
// @Example Request: GET /story/v01/follower/list/1/10
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
// @Summary      내 팔로잉 목록
// @Description  내 팔로잉 목록 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: JWT 사용자의 활성 관계만 조회합니다.
// @Description
// @Description  **응답 data**: total_count, list.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{total_count=int,list=[]protocol.FollowUserItem}} "data: total_count, list."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/following/list/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"total_count":0,"list":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"total_count":0,"list":[]}}
// @Example Request: GET /story/v01/following/list/1/10
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
// @Summary      피드 제외 사용자 등록
// @Description  피드 제외 사용자 등록 API입니다.
// @Description
// @Description  **요청**: path tid: 양의 대상 UID.
// @Description
// @Description  **처리**: JWT UID 기준으로 cutout 관계를 등록합니다. 본인은 제외할 수 없으며 대상 계정과 성별을 검증합니다.
// @Description
// @Description  **응답 data**: msg, res.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,res=int}} "data: msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/cutout/set/{tid} [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @Example Request: POST /story/v01/cutout/set/4033287471439576593
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "res": affected})
}

// UnsetCutoutUser godoc
// @Summary      피드 제외 사용자 해제
// @Description  피드 제외 사용자 해제 API입니다.
// @Description
// @Description  **요청**: path tid: 양의 대상 UID.
// @Description
// @Description  **처리**: JWT UID 기준으로 cutout 관계를 해제합니다.
// @Description
// @Description  **응답 data**: msg, res.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        tid path string true "대상 UID. 이 파라미터가 없는 목록 경로는 JWT 본인"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,res=int}} "data: msg, res."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/cutout/cancel/{tid} [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
// @Example Request: POST /story/v01/cutout/cancel/4033287471439576593
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

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{"msg": "ok", "res": affected})
}

// GetCutoutCount godoc
// @Summary      피드 제외 사용자 수 조회
// @Description  피드 제외 사용자 수 조회 API입니다.
// @Description
// @Description  **요청**: POST 요청이며 body는 없습니다.
// @Description
// @Description  **처리**: JWT 사용자의 활성 cutout 수를 조회합니다.
// @Description
// @Description  **응답 data**: msg, count.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,count=int}} "data: msg, count."
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/cutout/count [post]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","count":0}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","count":0}}
// @Example Request: POST /story/v01/cutout/count
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
// @Summary      피드 제외 사용자 목록 조회
// @Description  피드 제외 사용자 목록 조회 API입니다.
// @Description
// @Description  **요청**: path page ≥ 1, limit 1~100.
// @Description
// @Description  **처리**: JWT 사용자의 활성 cutout 목록을 조회합니다.
// @Description
// @Description  **응답 data**: msg, total_count, list.
// @Description  응답은 `{"result":0,"resultString":"Success","data":...}` 형식입니다.
// @Tags         Story
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer access token"
// @Param        page path int true "페이지 번호(1부터)"
// @Param        limit path int true "페이지 크기(1~100)"
// @Success      200 {object} protocol.RespDataHeader{data=object{msg=string,total_count=int,list=[]protocol.CutoutUserItem}} "data: msg, total_count, list."
// @Failure      400 {object} protocol.RespHeader "요청 형식·값 검증 실패"
// @Failure      401 {object} protocol.RespHeader "인증 실패 또는 사용자 컨텍스트 없음"
// @Failure      500 {object} protocol.RespHeader "내부 처리 실패"
// @Router       /story/v01/cutout/list/{page}/{limit} [get]
// @Example Response: {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":0,"list":[]}}
// @x-response-example {"result":0,"resultString":"Success","data":{"msg":"ok","total_count":0,"list":[]}}
// @Example Request: GET /story/v01/cutout/list/1/10
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
