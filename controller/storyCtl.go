package controller

import (
	"encoding/json"
	"mime/multipart"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptc "ms-gateway/protocol"
	"net/http"
	"strconv"

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
}

func NewStoryController(ctl *Controller, rep *models.Repositories) (*StoryController, error) {
	r := &StoryController{
		ctl: ctl,
		rep: rep,
		cfg: ctl.cfg,
	}

	if err := rep.Get(&r.adb, &r.hdb, &r.sdb); err != nil {
		return nil, err
	}

	return r, nil
}

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
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved story list - returns array of story objects with idx, nick, str_img (JSON array), at_create fields"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid or missing uid"
// @Failure 500 {object} protocol.RespHeader "Internal server error - failed to get story list"
// @Router /story/v01/list/{uid} [get]
// @Example Request: GET /story/v01/list/5817
// @Example Response: {"result":0,"resultString":"Success","data":[{"idx":2,"nick":"testnick","str_img":{"1":"https://iy.net/bLg/26c0/lic","2":"https://iy.net/g/94d61/public","3":"https://iy.net/bg/e80/public"},"at_create":"2026-02-02T07:27:44Z"},{"idx":1,"nick":"testnick","str_img":{"1":"https://i.net/bg/ec3d00/public","2":"https://imaet/bh7hqLg/9200/public","3":"https://inet/bhyxVLg/e811fef00/public"},"at_create":"2026-02-02T07:24:25Z"}]}
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

	// TODO: 블록 유저 조회

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

// GetStoryDetail godoc
// @Summary Get story detail by story index
// @Description Retrieve detailed information of a specific story including comments
// @Tags Story
// @Accept json
// @Produce json
// @Param idx path string true "Story Index"
// @Success 200 {object} protocol.RespDataHeader "Successfully retrieved story detail - returns story object with nick, body, str_img (JSON array), at_create fields and commentList array with idx, wuid, nick, body, at_create fields"
// @Failure 400 {object} protocol.RespHeader "Bad request - invalid or missing idx"
// @Failure 500 {object} protocol.RespHeader "Internal server error - failed to get story detail or comment list"
// @Router /story/v01/detail/{idx} [get]
// @Example Request: GET /story/v01/detail/3
// @Example Response: {"code":200,"message":"success","data":{"story":{"nick":"testnick","body":"test story body","str_img":["https://example.com/img1.jpg","https://example.com/img2.jpg"],"at_create":"2024-01-01T12:00:00Z"},"commentList":[{"idx":1,"wuid":"77645423236541","nick":"test","body":"111111 test comment body","at_create":"2024-01-01T12:30:00Z"}]}}
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

// UploadStoryPic godoc
// @Summary 업로드 스토리 이미지 (Cloudflare Images)
// @Description 최대 5개의 이미지(각 5MB 이하, 총 본문 길이 512자 제한)를 업로드합니다. 인증된 사용자가 Cloudflare Images에 스토리 이미지를 업로드하며, 업로드 성공 시 db에 저장합니다.
// @Tags Story
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param files formData file true "스토리 미디어 파일 (최대 5MB * 5개) — 지원 파일: 이미지/동영상, 최대 5개까지 첨부, 최소 1개 이상 필요, 파일명 중복 불가"
// @Param thumbnails formData file false "동영상 썸네일 이미지 (동영상 개수만큼 필요, 이미지 타입만 허용, files의 동영상 순서와 1:1 매칭)"
// @Param sbody formData string true "스토리 본문 (최대 512자, 필수)"
// @Param stat formData int true "스토리 공개 상태 (0=del, 1=pub, 2=private, 3=limit, 4=resv)" Enums(0,1,2,3,4)
//
//	@Example curl -X POST http://localhost:8080/story/v01/upload \
//	  -H "Authorization: Bearer {jwt-access-token}" \
//	  -F "files=@/home/tmp/bc1.jpg" \
//	  -F "files=@/home/tmp/bc2.jpg" \
//	  -F "files=@/home/tmp/bc3.jpg" \
//	  -F "sbody=이것은 스토리 본문입니다." \
//	  -F "stat=1" \
//	  -v
//
// @Success 200 {object} map[string]interface{} "업로드 성공 시: {\"msg\": \"Successfully uploaded story picture\", \"lastID\":5}"
// @Failure 400 {object} map[string]interface{} "요청 파라미터 누락 또는 파일 미첨부 시 에러"
// @Failure 401 {object} map[string]interface{} "인증 실패 시"
// @Failure 500 {object} map[string]interface{} "Cloudflare 업로드 오류 또는 서버 내부 오류"
// @Router /story/v01/upload [post]
// @Example Response : Status: 200, Response: {"msg":"Successfully uploaded story picture","lastID":5}
func (p *StoryController) CreateStory(c *gin.Context) {
	fileInfo, ok := c.Get("upFiles")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No files uploaded")
		return
	}
	sinfo, ok := c.Get("sinfo")
	if !ok {
		p.ctl.SimpleError(c, http.StatusBadRequest, "No sinfo uploaded")
		return
	}
	// TODO : birth 타입 - 문자열 일자까지만
	// TODO : area 타입 변경
	user, exists := c.Get("user")
	if !exists {
		p.ctl.SimpleError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	sinfoMap := sinfo.(map[string][]string)
	sstat, okStat := firstFormValue(sinfoMap, "stat")
	ssbody, okBody := firstFormValue(sinfoMap, "sbody")
	if !okStat || !okBody || sstat == "" || ssbody == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Story body, Status are required")
		return
	}

	files := fileInfo.([]*multipart.FileHeader)

	// 동영상용 썸네일(선택) — 미들웨어에서 이미지 타입만 통과시킨다.
	var thumbnails []*multipart.FileHeader
	if thumbInfo, exists := c.Get("upThumb"); exists {
		thumbnails, _ = thumbInfo.([]*multipart.FileHeader)
	}

	// 동영상 개수만큼 썸네일이 필요하다. (순서대로 1:1 매칭)
	videoCount := 0
	for _, f := range files {
		if utils.GetFileType(f.Filename) == 1 {
			videoCount++
		}
	}
	if videoCount > len(thumbnails) {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Thumbnail is required for each video")
		return
	}

	// 본편 업로드 (이미지/동영상)
	cldFlrInfos, err := utils.UploadCldFlr(files, p.cfg.Server.CfId, p.cfg.Server.CfToken)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload story picture", err)
		return
	}

	// 썸네일 업로드 (Cloudflare Images)
	var thumbInfos *map[string]string
	if len(thumbnails) > 0 {
		thumbInfos, err = utils.UploadCldFlr(thumbnails, p.cfg.Server.CfId, p.cfg.Server.CfToken)
		if err != nil {
			p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to upload thumbnail", err)
			return
		}
	}

	// files 순서를 유지하면서 str_img 구성.
	// - "N"      : N번째 미디어 URL
	// - "thumbN" : N번째 미디어(동영상)의 썸네일 URL
	dix := map[string]string{}
	var hasImg, hasVid bool
	thumbIdx := 0
	for i, f := range files {
		idx := strconv.Itoa(i + 1)
		dix[idx] = (*cldFlrInfos)[f.Filename]

		switch utils.GetFileType(f.Filename) {
		case 0:
			hasImg = true
		case 1:
			hasVid = true
			// 동영상은 순서대로 썸네일 1장을 매칭한다.
			if thumbInfos != nil && thumbIdx < len(thumbnails) {
				thumbFile := thumbnails[thumbIdx]
				dix["thumb"+idx] = (*thumbInfos)[thumbFile.Filename]
				thumbIdx++
			}
		}
	}

	simg := &ptl.StoryImage{}
	simg.StrImg, err = json.Marshal(dix)
	if err != nil {
		log.Error("Failed to marshal story image: %v", err)
		return
	}

	if hasImg && hasVid {
		simg.MType = 2
	} else if hasVid {
		simg.MType = 1
	} else {
		simg.MType = 0
	}

	simg.Uid = user.(*ptl.UserInfoResp).Uid

	//"1990-01-01T00:00:00Z"
	simg.Birth = utils.Time2StrDay(user.(*ptl.UserInfoResp).Birth)
	if simg.Birth == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, birth is required")
		return
	}
	simg.Nick = user.(*ptl.UserInfoResp).Nick
	if simg.Nick == "" {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, nick is required")
		return
	}
	simg.Area = ptl.GetAreaCode(user.(*ptl.UserInfoResp).Area)
	if simg.Area <= 0 {
		simg.Area = 0
	}

	simg.Gender, err = strconv.Atoi(user.(*ptl.UserInfoResp).Gender)
	if err != nil || simg.Gender < 0 || simg.Gender > 2 {
		p.ctl.SimpleError(c, http.StatusBadRequest, "User not authenticated, gender is required")
		return
	}

	simg.Body = ssbody
	simg.Stat, err = strconv.Atoi(sstat)
	if err != nil {
		log.Error("Failed to convert stat to int: %v", err)
		return
	}

	lastID, err := p.sdb.SaveStory(simg)
	if err != nil {
		log.Error("Failed to set story: %v", err)
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to set story", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully uploaded story picture", "lastID": lastID})
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
// @Description Delete a picture from a story and move it to backup
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

	//todo 분해
	// strImg
	mImg := make(map[string]string)
	mImgBak := make(map[string]string)

	json.Unmarshal(strImg, &mImg)
	json.Unmarshal(strImgBak, &mImgBak)

	durl := mImg[sPic]
	mImgBak[sPic] = durl
	delete(mImg, sPic)

	// 동영상 썸네일(thumb<N>)이 있으면 함께 백업/삭제한다.
	thumbKey := "thumb" + sPic
	if turl, ok := mImg[thumbKey]; ok {
		mImgBak[thumbKey] = turl
		delete(mImg, thumbKey)
	}

	// mImgBytes, err := json.Marshal(mImg)
	mImgBytes, err := json.Marshal(mImg)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to marshal mImg", err)
		return
	}
	mImgBakBytes, err := json.Marshal(mImgBak)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to marshal mImgBak", err)
		return
	}
	affected, err := p.sdb.DeleteStrPic(strIdxInt, mImgBytes, mImgBakBytes)
	if err != nil {
		p.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to update str pic comment", err)
		return
	}

	p.ctl.SimpleRespOK(c, gin.H{"msg": "Successfully updated str pic comment", "affected": affected})
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
