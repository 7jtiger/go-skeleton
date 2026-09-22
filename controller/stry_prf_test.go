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
	"testing"
)

const prfTestToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

// JWT userId — GetPrfInfo / GetPrfStoryLists 본인·타인 조회에 사용
const prfTestUID = "8697414060736839837"

// Test_GetPrfInfo GET /story/v01/prf/info/:tid
func Test_GetPrfInfo(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/info/" + prfTestUID

	res, err := GetWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get prf info: %v", err)
	}

	fmt.Println(res)
	//{"result":0,"resultString":"Success","data":{"flw_cnt":0,"prf_Info":{"uid":8697414060736839837,"nick":"건강한 꼬마 오렌지","gender":"1","birth":"1990-01-01T00:00:00Z","age":"-2","area":"1","intro":"반가워요 큐피톡에서 만나요!","main_pic":""}}}

}

// Test_GetPrfAssets GET /story/v01/prf/assets
func Test_GetPrfAssets(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/assets"

	res, err := GetWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get prf assets: %v", err)
	}

	fmt.Println(res)
	//{"result":0,"resultString":"Success","data":{"hold_cash":0,"hold_point":0,"msg_count":0}}
}

// Test_GetPrfPicList GET /story/v01/prf/pic/list
// 핸들러가 아직 스텁이면 빈 본문일 수 있음 — 라우트·JWT 통과만 확인
func Test_GetPrfPicList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/pic/list"

	res, err := GetWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get prf pic list: %v", err)
	}

	fmt.Println(res)
	// {"result":0,"resultString":"Success","data":{"1":{"url":"https://imagedelivery.net/.../public","stat":3},"2":{"url":"https://imagedelivery.net/.../public","stat":3}}}
}

// Test_GetPrfStoryLists GET /story/v01/prf/story/lists/:tid
func Test_GetPrfStoryLists(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/story/lists/" + prfTestUID

	res, err := GetWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get prf story lists: %v", err)
	}

	fmt.Println(res)
	//{"result":0,"resultString":"Success","data":{"str_imgs":{"4":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public","5":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/e7b2b58b-a7ec-49a1-5e6d-175e42609b00/public"},"str_thmbls":{"11":"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"}}}
}

// Test_UpdatePrfInfo POST /story/v01/prf/upd — JSON partial update
func Test_UpdatePrfInfo(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/upd"

	var key = []string{"nick", "intro", "area"}
	var value = []string{"prf_test_nick", "프로필 소개 — automated prf test", "서울"}

	res, err := PostWithToken(*targetUrl, qurl, key, value, *dmToken)
	if err != nil {
		t.Errorf("Failed to update prf info: %v", err)
	}

	fmt.Println(res)
	//{"result":0,"resultString":"Success","data":{"msg":"success"}}
}

// Test_DeletePrfPic POST /story/v01/prf/pic/delete/:idx
func Test_DeletePrfPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/prf/pic/delete/2"

	res, err := PostWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to delete prf pic: %v", err)
	}
	fmt.Println(res)
}

// Test_SetMainPic POST /story/v01/prf/set/mainpic/:url
func Test_SetMainPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	mainURL := "https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public"
	qurl := "/story/v01/prf/set/mainpic/" + mainURL

	res, err := PostWithToken(*targetUrl, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to set main pic: %v", err)
	}

	fmt.Println(res)
}

// Test_UploadPrfPic POST /story/v01/prf/pic/upload — multipart files + urls=[]
func Test_UploadPrfPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()
	qurl := "/story/v01/prf/pic/upload"

	// img1 := "/home/jino/go/src/ms-gateway/bak/tmpimg/wet1.jpg"
	// img2 := "/home/jino/go/src/ms-gateway/bak/tmpimg/rewq.jpg"
	img1 := ""
	img2 := ""
	for _, path := range []string{img1, img2} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("test image not found %s: %v", path, err)
		}
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, fname := range []string{img1, img2} {
		file, err := os.Open(fname)
		if err != nil {
			t.Fatalf("open %s: %v", fname, err)
		}
		func(f *os.File, path string) {
			defer f.Close()
			part, err := writer.CreateFormFile("files", filepath.Base(path))
			if err != nil {
				t.Fatalf("form file %s: %v", path, err)
			}
			if _, err := io.Copy(part, f); err != nil {
				t.Fatalf("copy %s: %v", path, err)
			}
		}(file, fname)
	}

	// 신규 업로드만: 기존 URL 없음 → urls=[]
	if err := writer.WriteField("urls", "[\"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/5bca857f-75b1-4548-af93-ffa9fdd05600/public\", \"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/510f55ea-3262-46ba-4fa3-b92c7c738000/public\"]"); err != nil {
		t.Fatalf("urls field: %v", err)
	}
	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	url := fmt.Sprintf("http://%s%s", *targetUrl, qurl)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// story/v01 그룹은 JwtAuth — Authorization Bearer 필수 (X-Totp만으로는 401)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+*dmToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(bodyBytes))
	//Response: {"result":0,"resultString":"Success","data":{"msg":"success"}}

	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("JWT auth failed: %s", string(bodyBytes))
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

// Test_PrfProcess 프로필 API 시나리오 순차 실행
func Test_PrfProcess(t *testing.T) {
	Test_GetPrfInfo(t)
	Test_GetPrfAssets(t)
	Test_GetPrfPicList(t)
	Test_GetPrfStoryLists(t)
	Test_UpdatePrfInfo(t)
	Test_SetMainPic(t)
	Test_UploadPrfPic(t)
}

func dataKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
