package controller

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	// "gocv.io/x/gocv"
)

// ----  stroy ----------------

func Test_UploadPicStory(t *testing.T) {
	// story.POST("/create", p.ValidateFileUpload(5, 5), p.st.CreateStory)
	// "./bak/tmpimg/qwer.jpg"
	// "./bak/tmpimg/rewq.jpg"
	// "./bak/tmpimg/wet1.jpg"

	// This test demonstrates a multipart/form POST to the /story/v01/create API.
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()

	uploadUrl := fmt.Sprintf("http://%s/story/v01/create", *targetUrl)

	// Prepare test files (ensure these files exist!)
	// img1 := "/home/jino/go/src/ms-gateway/bak/tmpimg/qwer.jpg"
	img1 := "/home/jino/go/src/ms-gateway/bak/tmpimg/wet1.jpg"
	img2 := "/home/jino/go/src/ms-gateway/bak/tmpimg/rewq.jpg"

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add files part (maximum 2 images for this test)
	files := []string{img1, img2}
	for _, fname := range files {
		file, err := os.Open(fname)
		if err != nil {
			t.Fatalf("Failed to open file %s: %v", fname, err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("files", filepath.Base(fname))
		if err != nil {
			t.Fatalf("Failed to create form file for %s: %v", fname, err)
		}
		if _, err := io.Copy(part, file); err != nil {
			t.Fatalf("Failed to copy file data for %s: %v", fname, err)
		}
	}

	// Add sinfo part (story info)
	// Simulate a realistic user
	sinfo := map[string][]string{
		"stat":  {"1"},
		"sbody": {"tasdfsafdest 테스트 스토리 본문 story body from automated test"},
	}
	// Flat form fields (also compatible with sinfo[key] via middleware normalize)
	for k, vs := range sinfo {
		for _, v := range vs {
			if err := writer.WriteField(k, v); err != nil {
				t.Fatalf("Failed to add sinfo field %s: %v", k, err)
			}
		}
	}

	// Set the requested access token in the header for authentication
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Build the HTTP request
	req, err := http.NewRequest("POST", uploadUrl, &b)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("Status: %d, Response: %s\n", resp.StatusCode, string(respBody))

}

// Test_UploadContents POST /story/v01/upload — UploadStoryContent (동영상 + thbnl 1:1)
func Test_UploadContents(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()

	uploadURL := fmt.Sprintf("http://%s/story/v01/upload", *targetUrl)

	vd1 := "/home/jino/tmp/next1.mp4"
	thb1 := "/home/jino/tmp/bc1.jpg"
	videoFiles := []string{vd1}
	thumbFiles := []string{thb1}

	for _, path := range append(videoFiles, thumbFiles...) {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("test media not found %s: %v", path, err)
		}
	}

	var bodyBuf bytes.Buffer
	writer := multipart.NewWriter(&bodyBuf)

	for _, fname := range videoFiles {
		file, err := os.Open(fname)
		if err != nil {
			t.Fatalf("Failed to open video file %s: %v", fname, err)
		}
		func(f *os.File, path string) {
			defer f.Close()
			part, err := writer.CreateFormFile("files", filepath.Base(path))
			if err != nil {
				t.Fatalf("Failed to create form file for %s: %v", path, err)
			}
			if _, err := io.Copy(part, f); err != nil {
				t.Fatalf("Failed to copy file data for %s: %v", path, err)
			}
		}(file, fname)
	}

	for _, tname := range thumbFiles {
		file, err := os.Open(tname)
		if err != nil {
			t.Fatalf("Failed to open thumbnail file %s: %v", tname, err)
		}
		func(f *os.File, path string) {
			defer f.Close()
			part, err := writer.CreateFormFile("thbnl", filepath.Base(path))
			if err != nil {
				t.Fatalf("Failed to create form file for thumbnail %s: %v", path, err)
			}
			if _, err := io.Copy(part, f); err != nil {
				t.Fatalf("Failed to copy thumbnail data for %s: %v", path, err)
			}
		}(file, tname)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

	req, err := http.NewRequest(http.MethodPost, uploadURL, &bodyBuf)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+testToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("Status: %d, Response: %s\n", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Result       int    `json:"result"`
		ResultString string `json:"resultString"`
		Data         struct {
			Msg  string `json:"msg"`
			Vdos string `json:"vdos"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if apiResp.Result != 0 {
		t.Fatalf("expected result=0, got %d (%s)", apiResp.Result, apiResp.ResultString)
	}
	if apiResp.Data.Msg != "ok" {
		t.Fatalf("expected data.msg=ok, got %q", apiResp.Data.Msg)
	}
	if apiResp.Data.Vdos == "" {
		t.Fatalf("expected data.vdos with playback/thumb URLs, got empty")
	}

	fmt.Printf("UploadStoryContent OK — vdos: %s\n", apiResp.Data.Vdos)
}

// Test_CreateStory POST /story/v01/create — CreateStory (upload 응답 vdos 고정값으로 JSON 저장 테스트)

func Test_CreateStory_Test(t *testing.T) {
	targetURL := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()

	createStoryURL := fmt.Sprintf("http://%s/story/v01/create", *targetURL)

	// POST /story/v01/upload 성공 응답의 data.vdos (실제 파일 업로드 생략)
	uploadedVdos := `{"1":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}`

	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

	createPayload, err := json.Marshal(map[string]interface{}{
		"stat":  1,
		"sbody": "테스트 동영상 스토리 — CreateStory from automated video+thumb test",
		"imgs":  "",
		"vdos":  uploadedVdos,
	})
	if err != nil {
		t.Fatalf("Failed to marshal create request: %v", err)
	}

	createReq, err := http.NewRequest(http.MethodPost, createStoryURL, bytes.NewReader(createPayload))
	if err != nil {
		t.Fatalf("Failed to create story request: %v", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+testToken)

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("Failed to perform create request: %v", err)
	}
	defer createResp.Body.Close()

	createRespBody, err := io.ReadAll(createResp.Body)
	if err != nil {
		t.Fatalf("Failed to read create response: %v", err)
	}
	fmt.Printf("Create status: %d, Response: %s\n", createResp.StatusCode, string(createRespBody))

	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create expected status 200, got %d: %s", createResp.StatusCode, string(createRespBody))
	}

	var createAPIResp struct {
		Msg string `json:"msg"`
		Idx int64  `json:"idx"`
	}
	if err := json.Unmarshal(createRespBody, &createAPIResp); err != nil {
		t.Fatalf("failed to parse create response JSON: %v", err)
	}
	if createAPIResp.Msg != "ok" {
		t.Fatalf("expected msg=ok, got %q", createAPIResp.Msg)
	}
	if createAPIResp.Idx <= 0 {
		t.Fatalf("expected idx > 0, got %d", createAPIResp.Idx)
	}

	fmt.Printf("CreateStory OK — idx: %d, vdos: %s\n", createAPIResp.Idx, uploadedVdos)
}

func TestGetStoryDetail(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	idx := "3"
	qurl := fmt.Sprintf("/story/v01/detail/%s", idx)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func TestGetDefStoryList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/list"

	var key = []string{"uid"}
	var value = []string{"7766493213763375817"}

	res, err := Get(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to get story list: %v", err)
	}

	fmt.Println(res)
}

// condition : area / stat / type / latest
// paging
// area :   // all = 0, seoul, gyeonggi, incheon, busan, daejeon/sejong/chungnam
// chungbuk/cheonju/chungju, daegu/gyeongbuk, gyeongnam/ulsan, gwangju/jeonnam
// jeonbuk/jeonju, gangwon/chuncheon, jeju
// stat : 0=PUB(public 전체공개), 1=FLW(follower only), 2=PAY(paid only), 3=PRV(Private 비공개), 4=DEL(Deleted) 5=RSV(reserved)
// type : 0=IMG(only img), 1 = VDO(only video), 2 = ALL(img + video), 3=RSV(reserved)
// order : 0=OLD(oldest), 1=NEW(latest), 2=CMT(comMost), 3=GOD(goodMost), 4=VIEW(viewMost), 5=RSV(reserved)
// gen : target gender 0=WMN(woman), 1=MAN(man), 2=RSV(reserved)
// page : 1, 2, 3, ...
// limit : 10, 20, 30, ...
// response : {"result":0,"resultString":"Success","data":{"story_home_list":{"1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","2":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","3":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5512db27-6340-43cd-b5ab-85d7b9bd4800/public"}}}
func TestGetCondStoryList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	area := "all"
	stat := "pub"
	mtype := "img"
	order := "new" // 최신순
	gen := "1"
	page := "1"
	limit := "5"

	qurl := fmt.Sprintf("/story/v01/condition/list/%s/%s/%s/%s/%s/%s/%s", area, stat, mtype, order, gen, page, limit)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get conditional story list: %v", err)
		return
	}

	fmt.Println(res)
}

func TestGetStoryListByUid(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	uid := "2345091823"
	qurl := fmt.Sprintf("/story/v01/list/%s", uid)

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get story list by uid: %v", err)
	}

	/*
		{"result":0,"resultString":"Success","data":[{"idx":2,"nick":"testnick","str_img":{"bc1.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","bc2.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public","bc3.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"},"at_create":"2026-02-02T07:27:44Z"},{"idx":1,"nick":"testnick","str_img":{"bc1.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public","bc2.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public","bc3.jpg":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"},"at_create":"2026-02-02T07:24:25Z"}]}
	*/
	fmt.Println(res)
}

func TestDeleteStrPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/delpic"

	var key = []string{"idx", "pic_idx"}
	var value = []string{"3", "1"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to delete str pic: %v", err)
	}

	fmt.Println(res)
}

func TestCreateStrComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/comment/create"

	var key = []string{"str_idx", "wuid", "nick", "stat", "body"}
	var value = []string{"3", "77645423234236541", "test", "1", "111111 test comment body"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to create str comment: %v", err)
	}

	fmt.Println(res)
}

func TestGetStrCommentList(t *testing.T) {
	//story.GET("/comment/:idx/:page",
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := fmt.Sprintf("/story/v01/comment/%s/%s", "3", "1")

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get str comment list: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStrBody(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updbody"

	var key = []string{"idx", "body"}
	var value = []string{"3", "updated story body content"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str body: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStoryStat(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updstat"

	var key = []string{"idx", "stat"}
	var value = []string{"3", "2"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update story stat: %v", err)
	}

	fmt.Println(res)
}

func TestUpdateStrBodyComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/updbody"

	var key = []string{"idx", "body"}
	var value = []string{"1", "updated comment body"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str body comment: %v", err)
	}

	fmt.Println(res)
}

func Test_UpdStatComment(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/comment/updstat"

	var key = []string{"idx", "stat"}
	var value = []string{"1", "3"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to update str stat comment: %v", err)
	}

	fmt.Println(res)
}

func TestToggleStoryLike(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/like/toggle"

	var key = []string{"story_idx", "uid"}
	var value = []string{"3", "77645423234236541"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to toggle story like: %v", err)
	}

	fmt.Println(res)
}

func TestFollowUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/create"

	var key = []string{"follower_uid", "followee_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to follow user: %v", err)
	}

	fmt.Println(res)
}

func TestUnfollowUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/cancel"

	var key = []string{"follower_uid", "followee_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to unfollow user: %v", err)
	}

	fmt.Println(res)
}

func TestGetFollowerList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/follower/7766493213763375817/1"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get follower list: %v", err)
	}

	fmt.Println(res)
}

func TestGetFollowingList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/follow/following/77645423234236541/1"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get following list: %v", err)
	}

	fmt.Println(res)
}

// -------------------- user cut out ---------------------------------
// ---- story cutout (JWT: -dm_token, 대상 uid: -dm_peer_uid, 서버: -dm_target) ----
func Test_SetCutoutUser(t *testing.T) {
	flag.Parse()
	qurl := fmt.Sprintf("/story/v01/cutout/set/%s", *dmPeerUID)

	// tk := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI"
	// res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, tk)
	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to set cutout user: %v", err)
		return
	}

	fmt.Println(res)
}

func Test_UnsetCutoutUser(t *testing.T) {
	flag.Parse()
	qurl := fmt.Sprintf("/story/v01/cutout/cancel/%s", *dmPeerUID)

	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to unset cutout user: %v", err)
		return
	}

	fmt.Println(res)
}

func Test_GetCutoutCount(t *testing.T) {
	flag.Parse()
	qurl := "/story/v01/cutout/count"

	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get cutout count: %v", err)
		return
	}

	fmt.Println(res)
}

func Test_GetCutoutList(t *testing.T) {
	flag.Parse()
	qurl := "/story/v01/cutout/list/1/10"

	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get cutout list: %v", err)
		return
	}

	fmt.Println(res)
}

// -------------------- user cut out ---------------------------------
// ---- story end ----

// ---- file upload ----
func TestUploadStoryPic(t *testing.T) {
	accountID := "3a5160a653bf6175ea5c5b24ee01e344"
	apiToken := "lhmeGe8y_2zCunsLcr8g4eUxJdPTkPGNMI_pNEWU"

	// for _, file := range files {
	// Open the file
	src, err := os.Open("/home/jino/tmp/bc1.jpg")
	if err != nil {
		fmt.Println("Failed to open file: ", err.Error())
		return
	}
	defer src.Close()

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file to form
	part, err := writer.CreateFormFile("file", "bc1.jpg")
	if err != nil {
		fmt.Println("Failed to create form file: ", err.Error())
		return
	}

	if _, err := io.Copy(part, src); err != nil {
		fmt.Println("Failed to copy file: ", err.Error())
		return
	}

	/* 	// Add requireSignedURLs field
	   	if err := writer.WriteField("requireSignedURLs", "true"); err != nil {
	   		fmt.Println("Failed to write requireSignedURLs field: ", err.Error())
	   		return
	   	}

	*/ // // Add id field if exists in putInfo
	// if info, ok := putInfo[file.Filename]; ok {
	// 	if id, exists := info["key"]; exists {
	// 		writer.WriteField("id", id)
	// 	}
	// }

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/images/v1", accountID)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		fmt.Println("Failed to create request: ", err.Error())
		return
	}

	// Set headers - Content-Type must be set before Authorization
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiToken))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to upload to Cloudflare: ", err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body: ", err.Error())
		return
	}

	// Check response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Cloudflare upload failed with status %d: %s\n", resp.StatusCode, string(bodyBytes))
		fmt.Printf("Request URL: %s\n", url)
		fmt.Printf("Authorization header: Bearer %s\n", apiToken)
		return
	}

	fmt.Println("Successfully uploaded bc1.jpg to Cloudflare")
	fmt.Println("Response:", string(bodyBytes))
	// }

}

func TestRequestFileUpload(t *testing.T) {
	// Test file upload to story endpoint
	// This test simulates: curl -X POST http://localhost:8080/story/v01/upload -F "files=@/home/jino/tmp/bc1.jpg" -H "X-Totp: test" -v

	targetUrl := "localhost:8080"
	endpoint := "/story/v01/upload"
	filePath := "/home/jino/tmp/bc1.jpg"

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Create form file field
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		t.Errorf("Failed to create form file: %v", err)
		return
	}

	// Copy file content to form
	_, err = io.Copy(part, file)
	if err != nil {
		t.Errorf("Failed to copy file content: %v", err)
		return
	}

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("http://%s%s", targetUrl, endpoint)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Totp", "test")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
		return
	}

	// Check response
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return
	}

	fmt.Println("Successfully uploaded file to story endpoint")

}

func Test_GetFilename(t *testing.T) {
	fullPath := "home/jino/tmp/bc1.jpg"
	parts := strings.Split(fullPath, "/")
	// filename := parts[len(parts)-1]
	var filename string
	if len(parts) > 1 {
		filename = parts[len(parts)-1]
	}
	fmt.Println(filename)
}

func TestMultiFileUpload(t *testing.T) {
	// Test file upload to story endpoint
	// This test simulates: curl -X POST http://localhost:8080/story/v01/upload -F "files=@/home/jino/tmp/bc1.jpg" -F "uid=test_user_123" -F "index_0=0" -H "X-Totp: test" -v

	targetUrl := "localhost:8080"
	endpoint := "/story/v01/upload"
	filePath := "/home/jino/tmp/bc1.jpg"
	filePath2 := "/home/jino/tmp/bc2.jpg"
	filePath3 := "/home/jino/tmp/bc3.jpg"
	testUid := "7766493213763375817" // 테스트용 uid

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// List of files to upload
	filePaths := []string{filePath, filePath2, filePath3}

	// Add uid field first
	err := writer.WriteField("uid", testUid)
	if err != nil {
		t.Errorf("Failed to write uid field: %v", err)
		return
	}

	err = writer.WriteField("stat", "1")
	if err != nil {
		t.Errorf("Failed to write stat field: %v", err)
		return
	}

	err = writer.WriteField("nick", "testnick")
	if err != nil {
		t.Errorf("Failed to write stat field: %v", err)
		return
	}

	err = writer.WriteField("sbody", "teqewrqwreqwreqwerqwreqwreqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerqwerwqerqwerstteststsetsteewststestsetsetestestesetsetestestestsetestseetestestestestsetestestestestsetestestsetestestsetsetestesetestestsetsetestsetsetestestewsterstewrtkldsjgklsdjklsdfajklasfjdlkjaklsjfaljfaskdljfklasjfklasjfdlkasjklfjaslkfjkalsjfklasjfklasjfkljasklfjasklfjlkasfd")
	if err != nil {
		t.Errorf("Failed to write uid field: %v", err)
		return
	}

	// Add each file to the multipart form with index
	for _, fp := range filePaths {
		// Open the file
		file, err := os.Open(fp)
		if err != nil {
			t.Errorf("Failed to open file %s: %v", fp, err)
			return
		}
		defer file.Close()

		// Create form file field
		part, err := writer.CreateFormFile("files", filepath.Base(fp))
		if err != nil {
			t.Errorf("Failed to create form file for %s: %v", fp, err)
			return
		}

		// Copy file content to form
		_, err = io.Copy(part, file)
		if err != nil {
			t.Errorf("Failed to copy file content for %s: %v", fp, err)
			return
		}
		/*
			parts := strings.Split(fp, "/")
			var fname = ""
			if len(parts) > 1 {
				fname = parts[len(parts)-1]
			} else {
				fname = fp
			}

			// Add index field for this file
			indexFieldName := fmt.Sprintf("idx_%d", idx)
			err = writer.WriteField(indexFieldName, fmt.Sprintf("%s", fname))
			if err != nil {
				t.Errorf("Failed to write index field for %s: %v", fp, err)
				return
			} */
	}

	// Get content type before closing writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	url := fmt.Sprintf("http://%s%s", targetUrl, endpoint)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		t.Errorf("Failed to create request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Totp", "test")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed to upload file: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
		return
	}

	// Check response
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return
	}

	fmt.Println("Successfully uploaded multiple files to story endpoint")

}

func Test_StoryProcess(t *testing.T) {
	Test_UploadPicStory(t)
	Test_UploadContents(t)

	TestGetStoryDetail(t)

	TestGetDefStoryList(t)

	TestGetCondStoryList(t)

	TestGetStoryListByUid(t)

	TestDeleteStrPic(t)

	TestCreateStrComment(t)

	TestGetStrCommentList(t)
}
