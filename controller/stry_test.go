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
	"strconv"
	"strings"
	"testing"
	"time"

	ptl "ms-gateway/protocol"
)

// Every endpoint URL is fixed to localhost:8080 and the literal qurl below.
// Server and URL parameters do not use command-line flags.
// Run selected tests against test accounts only: writes change stories/relations.
//
//	go test ./controller -run '^TestGetStoryViewCount$' -args \
//	  -dm_token <JWT>
//
// Profile endpoints already covered in stry_prf_test.go remain there.
var (
	storyTestIdx        = flag.String("story_idx", "3", "test story ID for view/like/comment/edit tests (original default: 3)")
	storyTestCommentIdx = flag.String("story_comment_idx", "1", "existing test comment ID")
	storyTestPicIdx     = flag.String("story_pic_idx", "1", "media slot to delete from test story")
	storyTestFiles      = flag.String("story_files", "/home/jino/tmp/next1.mp4", "comma-separated upload file paths, maximum 5")
	storyTestThumbs     = flag.String("story_thumbnails", "/home/jino/tmp/bc1.jpg", "comma-separated video thumbnail paths in video order")
	storyTestMedia      = flag.String("story_media_json", "", "JSON file containing upload response data.media only")
)

func storyFixture(t *testing.T, value, name string) string {
	t.Helper()
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || n == 0 {
		t.Skipf("set -%s to a positive test fixture ID", name)
	}
	return value
}

// Multipart/create requests retain the original explicit HTTP request template.
func storyRequest(t *testing.T, target, qurl, contentType string, body io.Reader) string {
	t.Helper()
	return storyHTTPResponse(t, http.MethodPost, target, qurl, contentType, body, *dmToken)
}

// Keep HTTP and application errors from passing as successful integration tests.
func storyHTTPResponse(t *testing.T, method, target, qurl, contentType string, body io.Reader, token string) string {
	t.Helper()
	req, err := http.NewRequest(method, "http://"+target+qurl, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: HTTP %d: %s", method, qurl, resp.StatusCode, raw)
	}
	var result struct {
		Result       *int            `json:"result"`
		ResultString string          `json:"resultString"`
		Data         json.RawMessage `json:"data"`
		Msg          string          `json:"msg"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("%s %s: invalid JSON: %s", method, qurl, raw)
	}
	if result.Result != nil {
		if *result.Result != 0 || result.ResultString != "Success" || len(result.Data) == 0 || string(result.Data) == "null" {
			t.Fatalf("%s %s: unsuccessful response: %s", method, qurl, raw)
		}
	} else if result.Msg != "ok" {
		t.Fatalf("%s %s: unexpected response: %s", method, qurl, raw)
	}
	return string(raw)
}

func storyJSONRequest(t *testing.T, method, target, qurl string, keys, values []string, token string) string {
	t.Helper()
	if len(keys) != len(values) {
		t.Fatal("request keys and values must have equal lengths")
	}
	payload := make(map[string]string, len(keys))
	for i, key := range keys {
		payload[key] = values[i]
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var body io.Reader
	if method == http.MethodPost {
		body = bytes.NewReader(raw)
	}
	return storyHTTPResponse(t, method, target, qurl, "application/json", body, token)
}

// POST /story/v01/upload
func Test_UploadContents(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/upload"
	if *storyTestFiles == "" {
		t.Skip("set -story_files and, for videos, -story_thumbnails")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for field, paths := range map[string]string{"files": *storyTestFiles, "thbnl": *storyTestThumbs} {
		if paths == "" {
			continue
		}
		for _, path := range strings.Split(paths, ",") {
			path = strings.TrimSpace(path)
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			part, err := writer.CreateFormFile(field, filepath.Base(path))
			if err == nil {
				_, err = io.Copy(part, file)
			}
			file.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	res := storyRequest(t, targetUrl, qurl, writer.FormDataContentType(), &body)
	fmt.Println(res)
}

// POST /story/v01/create
func Test_CreateStory_Test(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/create"
	// Original upload-response fixture; an optional file can override it.
	media := ptl.StoryStrImg{"1": {Type: ptl.StoryMediaTypeVdo, URL: "https://storage.googleapis.com/stream-example-bucket/video.mp4", Thumb: "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}}
	if *storyTestMedia != "" {
		raw, err := os.ReadFile(*storyTestMedia)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &media); err != nil {
			t.Fatal(err)
		}
	}
	payload, err := json.Marshal(ptl.CreateStoryReq{Stat: 1, Sbody: "테스트 동영상 스토리 — CreateStory from automated video+thumb test", Price: 100, Media: media})
	if err != nil {
		t.Fatal(err)
	}
	res := storyRequest(t, targetUrl, qurl, "application/json", bytes.NewReader(payload))
	fmt.Println(res)
}

// GET /story/v01/detail/15
func TestGetStoryDetail(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/detail/20"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"is_locked":0,"good_byme":0,"idx":20,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"1123","stat":0,"qt_like":21,"qt_view":2,"ctnt_total":5,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/99bb2943-201b-4c20-9851-8ba2146cdc00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1f5d729f-2a65-4352-9dc3-825550ee4300/public"},"3":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/c05cad48-7448-493e-6653-55980b54f600/public"},"4":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0ca94b70-b7f6-4282-1d00-910a1c0caf00/public"},"5":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4a8b1692-78fa-4216-889d-aef67d024400/public"}},"at_create":"2026-09-09T12:56:23Z","cmt_count":0,"cmt_list":null}}
	fmt.Println(res)
}

// GET /story/v01/list/1/10/7766493213763375817
func TestGetDefStoryList(t *testing.T) {
	targetUrl := "localhost:8080"
	// qurl := "/story/v01/list/1/10/7766493213763375817"
	qurl := "/story/v01/list/1/10/"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmPeerToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":10,"page":1,"story":[{"is_locked":0,"good_byme":0,"idx":26,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 2번 재배선 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/68598dcf-315f-483e-403a-87b3af033a00/public"},"2":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/004c2b76-559e-46ba-0ffe-0a304604d900/public"}},"at_create":"2026-10-20T14:28:35Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":25,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 슬롯2-thumb1","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"},"2":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5863ede1-a716-4f60-b58f-83050f59e000/public"}},"at_create":"2026-10-20T13:34:33Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":24,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] stat9 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"}},"at_create":"2026-10-20T13:32:43Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":23,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 슬롯 2,3 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"}},"at_create":"2026-10-20T13:32:42Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":22,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] thumb1 검증","stat":3,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5863ede1-a716-4f60-b58f-83050f59e000/public"}},"at_create":"2026-10-20T13:32:42Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":21,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude 1020] create 응답 형태 확인용","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b9322450-c140-4e1b-b174-985b7c597e00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/a31bc924-d655-4274-43f8-622e2b776b00/public"}},"at_create":"2026-10-20T13:23:26Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":20,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"1123","stat":0,"qt_like":21,"qt_view":2,"ctnt_total":5,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/99bb2943-201b-4c20-9851-8ba2146cdc00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1f5d729f-2a65-4352-9dc3-825550ee4300/public"},"3":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/c05cad48-7448-493e-6653-55980b54f600/public"},"4":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0ca94b70-b7f6-4282-1d00-910a1c0caf00/public"},"5":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4a8b1692-78fa-4216-889d-aef67d024400/public"}},"at_create":"2026-09-09T12:56:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":19,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"P5-0 B: imgs 손조립","stat":0,"qt_like":20,"qt_view":2,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/269023c3-7e1b-4bac-c89e-65e0c12a6000/public"}},"at_create":"2026-09-09T12:08:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":18,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"P5-0 A: vdos 손조립(순서반전)","stat":0,"qt_like":19,"qt_view":3,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ff91b39d-bb4f-47a7-f2a2-cf1ff98a5300/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public"}},"at_create":"2026-09-09T12:08:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":17,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"12","stat":2,"qt_like":18,"qt_view":4,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5339e07e-cd0b-4a8d-d419-02c64cc91e00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/abd900bf-4d25-4ae5-547c-494144963000/public"}},"at_create":"2026-09-08T13:16:21Z","cmt_count":0,"cmt_list":null}],"total_count":20}}
	fmt.Println(res)
}

// GET /story/v01/list/1/10/2345091823
func TestGetStoryListByUid(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/list/1/10/2345091823"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":10,"page":1,"story":[{"is_locked":1,"good_byme":0,"idx":2,"uid":2345091823,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"","stat":1,"qt_like":2,"qt_view":20,"ctnt_total":3,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public"},"3":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"}},"at_create":"2026-02-02T07:27:44Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":1,"uid":2345091823,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"테스트 스토리 내용입니다","stat":0,"qt_like":1,"qt_view":20,"ctnt_total":3,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ec0a726c-132f-4f0e-c603-35cfefd13d00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/919d8a0d-37a2-4d61-099b-a879322fc200/public"},"3":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e811ff8b-6646-4fee-b5f6-9aba8e393f00/public"}},"at_create":"2026-02-02T07:24:25Z","cmt_count":0,"cmt_list":null}],"total_count":2}}
	fmt.Println(res)
}

// GET /story/v01/cndt/list/all/pub/img/new/1/1/20
func TestGetCondStoryList(t *testing.T) {
	targetUrl := "localhost:8080"
	// qurl := "/story/v01/cndt/list/all/pub/img/new/1/1/20"
	qurl := "/story/v01/cndt/list/all/all/all/new/1/1/20"

	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmPeerToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":20,"page":1,"story":[{"is_locked":0,"good_byme":0,"idx":28,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"테스트 동영상 스토리 — CreateStory from automated video+thumb test","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":100,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}},"at_create":"2026-10-28T08:48:47Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":26,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 2번 재배선 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/68598dcf-315f-483e-403a-87b3af033a00/public"},"2":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/004c2b76-559e-46ba-0ffe-0a304604d900/public"}},"at_create":"2026-10-20T14:28:35Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":25,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 슬롯2-thumb1","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"},"2":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5863ede1-a716-4f60-b58f-83050f59e000/public"}},"at_create":"2026-10-20T13:34:33Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":24,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] stat9 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"}},"at_create":"2026-10-20T13:32:43Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":23,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] 슬롯 2,3 검증","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public"}},"at_create":"2026-10-20T13:32:42Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":22,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] thumb1 검증","stat":3,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5863ede1-a716-4f60-b58f-83050f59e000/public"}},"at_create":"2026-10-20T13:32:42Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":21,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude 1020] create 응답 형태 확인용","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b9322450-c140-4e1b-b174-985b7c597e00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/a31bc924-d655-4274-43f8-622e2b776b00/public"}},"at_create":"2026-10-20T13:23:26Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":20,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"1123","stat":0,"qt_like":21,"qt_view":2,"ctnt_total":5,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/99bb2943-201b-4c20-9851-8ba2146cdc00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1f5d729f-2a65-4352-9dc3-825550ee4300/public"},"3":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/c05cad48-7448-493e-6653-55980b54f600/public"},"4":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0ca94b70-b7f6-4282-1d00-910a1c0caf00/public"},"5":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4a8b1692-78fa-4216-889d-aef67d024400/public"}},"at_create":"2026-09-09T12:56:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":19,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"P5-0 B: imgs 손조립","stat":0,"qt_like":20,"qt_view":2,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/269023c3-7e1b-4bac-c89e-65e0c12a6000/public"}},"at_create":"2026-09-09T12:08:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":18,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"P5-0 A: vdos 손조립(순서반전)","stat":0,"qt_like":19,"qt_view":3,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ff91b39d-bb4f-47a7-f2a2-cf1ff98a5300/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public"}},"at_create":"2026-09-09T12:08:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":17,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"12","stat":2,"qt_like":18,"qt_view":4,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5339e07e-cd0b-4a8d-d419-02c64cc91e00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/abd900bf-4d25-4ae5-547c-494144963000/public"}},"at_create":"2026-09-08T13:16:21Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":16,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"P2 배선 검증","stat":1,"qt_like":17,"qt_view":8,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/269023c3-7e1b-4bac-c89e-65e0c12a6000/public"}},"at_create":"2026-09-04T10:57:04Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":15,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"E 둘다","stat":2,"qt_like":16,"qt_view":7,"ctnt_total":3,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4cf842e8-0818-415b-39e3-afe1a23ede00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b31063b0-5475-4df7-3a52-959acb050f00/public"},"3":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/d1028e90-de2e-4662-1acf-2226999a9c00/public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"cmt_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","wgender":"0","wage":"2019-01-02","warea":"10","body":"dsafasdfqwerwqe","stat":1,"at_create":"2026-02-02T13:50:06Z","at_update":"2026-02-02T13:50:06Z"}]},{"is_locked":0,"good_byme":0,"idx":14,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"D 혼합","stat":2,"qt_like":15,"qt_view":7,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public"},"2":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":13,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"C 동영상만","stat":1,"qt_like":14,"qt_view":8,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/d1028e90-de2e-4662-1acf-2226999a9c00/public"}},"at_create":"2026-09-02T13:35:23Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":12,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"테스트 B 디코딩JSON","stat":1,"qt_like":13,"qt_view":9,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4cf842e8-0818-415b-39e3-afe1a23ede00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b31063b0-5475-4df7-3a52-959acb050f00/public"}},"at_create":"2026-09-02T13:34:58Z","cmt_count":0,"cmt_list":null},{"is_locked":1,"good_byme":0,"idx":11,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"","stat":1,"qt_like":12,"qt_view":11,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}},"at_create":"2026-08-28T15:01:54Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":10,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"[claude] img+thbnl","stat":1,"qt_like":11,"qt_view":11,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/669744da-bd95-4226-b618-291f36529c00/public"}},"at_create":"2026-08-26T15:15:18Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":9,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나나","stat":1,"qt_like":9,"qt_view":12,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/38fff373-1d2e-4f15-d7bc-bbf6516ebd00/public"}},"at_create":"2026-08-26T14:15:37Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":8,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"t","stat":1,"qt_like":8,"qt_view":13,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/014e40aa-9eb6-4fce-fdbc-95db19fa4600/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/014e40aa-9eb6-4fce-fdbc-95db19fa4600/public"}},"at_create":"2026-08-26T14:15:18Z","cmt_count":0,"cmt_list":null}],"total_count":27}}
	fmt.Println(res)
}

// GET /story/v01/flwlist/1/10
func TestGetFollowingStory(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/flwlist/1/10"

	// res, err := GetWithToken(targetUrl, qurl, nil, nil, *dmToken)
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmPeerToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":10,"page":1,"story":[],"total_count":0}}
	fmt.Println(res)
}

// POST /story/v01/buy/15
func TestBuyStory(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/buy/15"

	// res, err := PostWithToken(targetUrl, qurl, nil, nil, *dmPeerToken)
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"msg":"ok","psc_stat":1}}
	fmt.Println(res)
}

// GET /story/v01/pcs/list/1/10
func TestGetPurchaseList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/pcs/list/1/10"

	// res, err := GetWithToken(targetUrl, qurl, nil, nil, *dmPeerToken)
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":10,"page":1,"story":[{"is_locked":0,"good_byme":0,"idx":15,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","body":"E 둘다","stat":2,"qt_like":16,"qt_view":7,"ctnt_total":3,"price":0,"is_paid":1,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4cf842e8-0818-415b-39e3-afe1a23ede00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b31063b0-5475-4df7-3a52-959acb050f00/public"},"3":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/d1028e90-de2e-4662-1acf-2226999a9c00/public"}},"at_create":"2026-09-02T13:35:24Z","cmt_count":1,"cmt_list":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","wgender":"0","wage":"2019-01-02","warea":"10","body":"dsafasdfqwerwqe","stat":1,"at_create":"2026-02-02T13:50:06Z","at_update":"2026-02-02T13:50:06Z"}]}],"total_count":1}}
	fmt.Println(res)
}

// GET /story/v01/view/count/3
func TestGetStoryViewCount(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/view/count/3"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"count":21,"msg":"ok"}}
	fmt.Println(res)
}

// POST /story/v01/view/update/3
func TestUpdateStoryView(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/view/update/3"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"msg":"ok"}}
	fmt.Println(res)
}

// GET /story/v01/like/count/3
func TestGetStoryLikeCount(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/like/count/3"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"count":4,"msg":"ok"}}
	fmt.Println(res)
}

// POST /story/v01/like/toggle
func TestUpdateStoryLike(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/like/toggle"
	key := []string{"story_idx"}
	value := []string{"3"}

	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"like_cnt":3,"msg":"ok"}
	fmt.Println(res)
}

// POST /story/v01/delpic
func TestDeleteStrPic(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := storyFixture(t, *storyTestIdx, "story_idx")
	qurl := "/story/v01/delpic"
	key := []string{"idx", "pic_idx"}
	value := []string{idx, *storyTestPicIdx}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 404; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Story not found]"}
	fmt.Println(res)
}

// POST /story/v01/comment/create
func TestCreateStrComment(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := storyFixture(t, *storyTestIdx, "story_idx")
	qurl := "/story/v01/comment/create"
	key := []string{"str_idx", "wuid", "nick", "stat", "body"}
	value := []string{idx, "77645423234236541", "test", "1", "111111 test comment body"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 403; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Story content is locked]"}
	fmt.Println(res)
}

// GET /story/v01/comment/3/1/10
func TestGetStrCommentList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/comment/15/1/10"

	// res, err := GetWithToken(targetUrl, qurl, nil, nil, *dmToken)
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmPeerToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"comment":[{"idx":3,"str_idx":15,"wuid":77645423236541,"nick":"test1","thumb_url":"https://icon.prof.com","wgender":"0","wage":"2019-01-02T00:00:00Z","warea":"10","body":"dsafasdfqwerwqe","stat":1,"at_create":"2026-02-02T13:50:06Z","at_update":"2026-02-02T13:50:06Z"}],"limit":10,"page":1,"total_cnt":1}}
	fmt.Println(res)
}

// POST /story/v01/updbody
func TestUpdateStrBody(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := "4" // Existing story owned by the default dmToken user.
	qurl := "/story/v01/updbody"
	key := []string{"idx", "body"}
	value := []string{idx, "updated story body content"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"msg":"ok","res":1}
	fmt.Println(res)
}

// POST /story/v01/updstat
func TestUpdateStoryStat(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := "4" // Existing story owned by the default dmToken user.
	qurl := "/story/v01/updstat"
	key := []string{"idx", "stat"}
	value := []string{idx, "2"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"msg":"ok","res":1}
	fmt.Println(res)
}

// POST /story/v01/comment/updbody
func TestUpdateStrBodyComment(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := storyFixture(t, *storyTestCommentIdx, "story_comment_idx")
	qurl := "/story/v01/comment/updbody"
	key := []string{"idx", "body"}
	value := []string{idx, "updated comment body"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 404; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Story or comment not found]"}
	fmt.Println(res)
}

// POST /story/v01/comment/updstat
func Test_UpdStatComment(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := storyFixture(t, *storyTestCommentIdx, "story_comment_idx")
	qurl := "/story/v01/comment/updstat"
	key := []string{"idx", "stat"}
	value := []string{idx, "3"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 404; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Story or comment not found]"}
	fmt.Println(res)
}

// POST /story/v01/like/toggle
func TestToggleStoryLike(t *testing.T) {
	targetUrl := "localhost:8080"
	idx := storyFixture(t, *storyTestIdx, "story_idx")
	qurl := "/story/v01/like/toggle"
	key := []string{"story_idx", "uid"}
	value := []string{idx, "77645423234236541"}
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, key, value, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"like_cnt":4,"msg":"ok"}
	fmt.Println(res)
}

// POST /story/v01/follow/set/7766493213763375817
func TestFollowUser(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/follow/set/7766493213763375817"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"msg":"ok","res":2}
	fmt.Println(res)
}

// POST /story/v01/follow/cancel/7766493213763375817
func TestUnfollowUser(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/follow/cancel/7766493213763375817"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"msg":"ok","res":1}
	fmt.Println(res)
}

// GET /story/v01/follower/list/1/10
func TestGetFollowerList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/follower/list/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"list":[],"total_count":0}}
	fmt.Println(res)
}

// GET /story/v01/following/list/1/10
func TestGetFollowingList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/following/list/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"list":[],"total_count":0}}
	fmt.Println(res)
}

// POST /story/v01/cutout/set/4033287471439576593
func Test_SetCutoutUser(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/cutout/set/4033287471439576593"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"msg":"ok","res":2}}
	fmt.Println(res)
}

// POST /story/v01/cutout/cancel/4033287471439576593
func Test_UnsetCutoutUser(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/cutout/cancel/4033287471439576593"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"msg":"ok","res":1}}
	fmt.Println(res)
}

// POST /story/v01/cutout/count
func Test_GetCutoutCount(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/cutout/count"
	res := storyJSONRequest(t, http.MethodPost, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"count":0,"msg":"ok"}}
	fmt.Println(res)
}

// GET /story/v01/cutout/list/1/10
func Test_GetCutoutList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/cutout/list/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"list":[],"msg":"ok","total_count":0}}
	fmt.Println(res)
}

// GET /story/v01/prf/story/list/1/10
func TestGetPrfStoryList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/prf/story/list/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":[{"is_locked":0,"good_byme":0,"idx":28,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","stat":0,"cnt_count":1,"cnt_url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","cnt_type":"vdo","cnt_thumb":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","at_create":"2026-10-28T08:48:47Z"},{"is_locked":0,"good_byme":0,"idx":27,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","stat":3,"cnt_count":1,"cnt_url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","cnt_type":"vdo","cnt_thumb":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","at_create":"2026-10-23T07:26:47Z"},{"is_locked":0,"good_byme":0,"idx":11,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","stat":1,"cnt_count":1,"cnt_url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","cnt_type":"vdo","cnt_thumb":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public","at_create":"2026-08-28T15:01:54Z"},{"is_locked":0,"good_byme":0,"idx":5,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","stat":1,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e7b2b58b-a7ec-49a1-5e6d-175e42609b00/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-05-29T09:09:24Z"},{"is_locked":0,"good_byme":0,"idx":4,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","stat":2,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-05-29T09:08:15Z"}]}
	fmt.Println(res)
}

// GET /story/v01/prf/story/list/1/10/4033287471439576593
func TestGetPrfStoryListByUID(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/prf/story/list/1/10/4033287471439576593"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":[{"is_locked":0,"good_byme":0,"idx":26,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/68598dcf-315f-483e-403a-87b3af033a00/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-10-20T14:28:35Z"},{"is_locked":0,"good_byme":0,"idx":25,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-10-20T13:34:33Z"},{"is_locked":0,"good_byme":0,"idx":24,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":1,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-10-20T13:32:43Z"},{"is_locked":0,"good_byme":0,"idx":23,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/998098f0-dd4f-40e1-bcea-052b742dc500/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-10-20T13:32:42Z"},{"is_locked":0,"good_byme":0,"idx":21,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/b9322450-c140-4e1b-b174-985b7c597e00/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-10-20T13:23:26Z"},{"is_locked":0,"good_byme":0,"idx":20,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":5,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/99bb2943-201b-4c20-9851-8ba2146cdc00/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-09T12:56:23Z"},{"is_locked":0,"good_byme":0,"idx":19,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1fd64301-d801-4dcf-4081-e7b0fb2da500/public","cnt_type":"img","cnt_thumb":"","at_create":"2026-09-09T12:08:23Z"},{"is_locked":0,"good_byme":0,"idx":18,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":0,"cnt_count":2,"cnt_url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","cnt_type":"vdo","cnt_thumb":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/ff91b39d-bb4f-47a7-f2a2-cf1ff98a5300/public","at_create":"2026-09-09T12:08:23Z"},{"is_locked":1,"good_byme":0,"idx":17,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":2,"cnt_count":2,"cnt_url":"","cnt_type":"","cnt_thumb":"","at_create":"2026-09-08T13:16:21Z"},{"is_locked":1,"good_byme":0,"idx":16,"uid":4033287471439576593,"nick":"","main_pic":"","gender":0,"area":"all","age":"","stat":1,"cnt_count":1,"cnt_url":"","cnt_type":"","cnt_thumb":"","at_create":"2026-09-04T10:57:04Z"}]}
	fmt.Println(res)
}

// GET /story/v01/prf/pic/waiting/1/10
func TestGetPrfPicWaitingList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/prf/pic/waiting/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 403; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Profile administrator required]"}
	fmt.Println(res)
}

// POST /story/v01/prf/pic/stat
func TestSetPrfPicStat(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/prf/pic/stat"
	uid, err := strconv.ParseUint(storyFixture(t, *dmPeerUID, "dm_peer_uid"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	slot, err := strconv.Atoi(*storyTestPicIdx)
	if err != nil || slot < 1 || slot > 5 {
		t.Fatal("story_pic_idx must be 1..5")
	}
	// This request requires numeric uid/slot, unlike PostWithToken's string map.
	payload, err := json.Marshal(ptl.SetPrfPicStatReq{Uid: uid, Slot: slot})
	if err != nil {
		t.Fatal(err)
	}
	res := storyRequest(t, targetUrl, qurl, "application/json", bytes.NewReader(payload))
	// 실제 응답 (2026-11-10, HTTP 403; 검증 실패로 res 대입 전 종료):
	// {"result":1,"resultString":"Failed","msg":"[Profile administrator required]"}
	fmt.Println(res)
}

// GET /story/v01/list/1/10
func TestGetMyStoryList(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/list/1/10"
	res := storyJSONRequest(t, http.MethodGet, targetUrl, qurl, nil, nil, *dmToken)
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"result":0,"resultString":"Success","data":{"limit":10,"page":1,"story":[{"is_locked":0,"good_byme":0,"idx":28,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"테스트 동영상 스토리 — CreateStory from automated video+thumb test","stat":0,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":100,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}},"at_create":"2026-10-28T08:48:47Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":27,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"테스트 동영상 스토리 — CreateStory from automated video+thumb test","stat":3,"qt_like":0,"qt_view":0,"ctnt_total":1,"price":100,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}},"at_create":"2026-10-23T07:26:47Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":11,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"테스트 동영상 스토리 — CreateStory from automated video+thumb test","stat":1,"qt_like":12,"qt_view":11,"ctnt_total":1,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"vdo","url":"https://storage.googleapis.com/stream-example-bucket/video.mp4","thumb1":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}},"at_create":"2026-08-28T15:01:54Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":5,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"tasdfsafdest 테스트 스토리 본문 story body from automated test","stat":1,"qt_like":5,"qt_view":16,"ctnt_total":2,"price":0,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e7b2b58b-a7ec-49a1-5e6d-175e42609b00/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0ccf1266-7407-4612-8c5a-0d8af75dd500/public"}},"at_create":"2026-05-29T09:09:24Z","cmt_count":0,"cmt_list":null},{"is_locked":0,"good_byme":0,"idx":4,"uid":8697414060736839837,"nick":"prf_test_nick","main_pic":"https://cdn.artkoreatv.com/news/photo/202604/101974_322779_5337.jpg","gender":1,"area":"seoul","age":"34","body":"updated story body content","stat":2,"qt_like":4,"qt_view":17,"ctnt_total":2,"price":100,"is_paid":0,"flw_cnt":0,"is_flwd":0,"str_img":{"1":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public"},"2":{"type":"img","url":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/0125d18e-aa56-45a0-98ac-853ecd3c1600/public"}},"at_create":"2026-05-29T09:08:15Z","cmt_count":0,"cmt_list":null}],"total_count":5}}
	fmt.Println(res)
}

// POST /story/v01/updprice
func TestUpdStoryPriceInfo(t *testing.T) {
	targetUrl := "localhost:8080"
	qurl := "/story/v01/updprice"
	// Same owned fixture as TestUpdateStrBody and TestUpdateStoryStat.
	price := 100
	payload, err := json.Marshal(ptl.UpdateStoryPriceReq{Idx: "4", Stat: "2", Price: &price})
	if err != nil {
		t.Fatal(err)
	}
	res := storyRequest(t, targetUrl, qurl, "application/json", bytes.NewReader(payload))
	var result struct {
		Res *int64 `json:"res"`
	}
	if err := json.Unmarshal([]byte(res), &result); err != nil || result.Res == nil || *result.Res < 0 || *result.Res > 1 {
		t.Fatalf("expected res=0 or res=1: %s", res)
	}
	// 실제 응답 (2026-11-10, HTTP 200):
	// {"msg":"ok","res":1}
	fmt.Println(res)
}
