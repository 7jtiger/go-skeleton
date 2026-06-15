package controller

import (
	"bytes"
	"flag"
	"fmt"
	"mime/multipart"
	"ms-gateway/common/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	// "gocv.io/x/gocv"
)

func TestAccAddPartner(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// targetUrl := flag.String("target", "192.168.48.182:8080", "target server url")
	qurl := "/acc/v01/regist"

	var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var value = []string{"test01", "usdx", "1111111123456", "test01@test.com", "0x0AFfB0a96FBefAa97dCe488DfD97512346cf3Ab8"}
	// var value = []string{"test23", "123456789", "홍길동", "1", "20", "1990-01-01", "서울"}
	var value = []string{"test13", "123456789", "홍길", "1", "22", "1990-01-01", "서울"}

	res, err := PostJson(*targetUrl, qurl, key, value)

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type RegistReq struct {
	Id     string `json:"id"`
	Pw     string `json:"pw"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    string `json:"age"`
	Birth  string `json:"birth"`
	Area   string `json:"area"`
	Email  string `json:"email"`
}

func Test_CalcBirth2Age(t *testing.T) {
	birth := "1990-01-01"
	// Parse birth date
	birthDate, err := time.Parse("2006-01-02", birth)
	if err != nil {
		t.Errorf("Failed to parse birth date: %v", err)
		return
	}

	// Get current date
	now := time.Now()

	// Calculate age
	age := now.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred this year yet
	if now.Month() < birthDate.Month() || (now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}

	fmt.Printf("Birth: %s, Current Age: %d\n", birth, age)
}

func TestAccRegist(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/regist"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var key = []string{"info"}
	var key = []string{"data"}
	// var value = []string{"test243", "123456789", "홍두께", "1", "24", "1990-01-01", "서울", "test243@test.com"}
	var value = []string{"qqqq1111", "qqqq1111!", "동욱", "0", "24", "1990-01-01", "서울", "test243@test.com"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := RegistReq{
		Id:     value[0],
		Pw:     value[1],
		Name:   value[2],
		Gender: value[3],
		Age:    value[4],
		Birth:  value[5],
		Area:   value[6],
		Email:  value[7],
	}

	// Convert the map to JSON
	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type LoginReq struct {
	Id string `json:"id"`
	Pw string `json:"pw"`
}

func TestAccLogin(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/login"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test243", "123456789"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := LoginReq{
		Id: value[0],
		Pw: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}
	fmt.Println(encryptedData)

	decryptedData, err := DecryptData(encryptedData)
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
		return
	}
	fmt.Println(decryptedData)

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	/*
		"{\"result\":0,\"resultString\":\"Success\",
		\"data\":{\"msg\":\"success\",\"acTok\":\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNjEyNiwibmJmIjoxNzcwMTI5NzI2LCJpYXQiOjE3NzAxMjk3MjYsImp0aSI6IjM5OTIwNTE2LWQ0OTgtNDhiNi1iNTdhLTliOGMwOWI5NmZlMiJ9.6qz-WDaNQVeR7X1YyYLr4_n6QFtH3rLl-DSkhOIC2dI\",
		\"refTok\":\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNjEyNiwibmJmIjoxNzcwMTI5NzI2LCJpYXQiOjE3NzAxMjk3MjYsImp0aSI6IjM5OTIwNTE2LWQ0OTgtNDhiNi1iNTdhLTliOGMwOWI5NmZlMiJ9.6qz-WDaNQVeR7X1YyYLr4_n6QFtH3rLl-DSkhOIC2dI\",
		\"uid\":\"8697414060736839837\",
		\"meta\":\"8697414060736839837/test243//건강한 꼬마 오렌지/1/24/서울/test243@test.com/https://i.ibb.co/QF37KRST/male-ai-02.webp/https://i.ibb.co/C5c51dYg/icon-male-04.webp/반가워요 큐피톡에서 만나요!\",
		\"wrtc\":{\"iceServers\":[{\"urls\":[\"stun:stun.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun1.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun2.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun3.l.google.com:19302\"],\"type\":\"stun\"},{\"urls\":[\"stun:stun4.l.google.com:19302\"],\"type\":\"stun\"}],\"signalingServer\":\"ws://localhost:8080/ws\",
		\"stunServers\":[\"stun:stun.l.google.com:19302\",\"stun:stun1.l.google.com:19302\",\"stun:stun2.l.google.com:19302\",\"stun:stun3.l.google.com:19302\",\"stun:stun4.l.google.com:19302\"],
		\"mediaSettings\":{\"video\":{\"enabled\":true,\"width\":1280,\"height\":720,\"frameRate\":30,\"maxBitrate\":2000000},
		\"audio\":{\"enabled\":true,\"echoCancellation\":true,\"noiseSuppression\":true,\"autoGainControl\":true,\"maxBitrate\":96000}}}}}"
	*/
	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func Test_FBlogin(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/login"

	var key = []string{"data"}
	var value = []string{"test243", "123456789"}
	// var value = []string{"qqqq1111", "qqqq1111!"}

	dataMap := LoginReq{
		Id: value[0],
		Pw: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}
	fmt.Println(encryptedData)

	decryptedData, err := DecryptData(encryptedData)
	if err != nil {
		t.Errorf("Failed to decrypt data: %v", err)
		return
	}
	fmt.Println(decryptedData)

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	if err != nil {
		t.Errorf("Failed to login: %v", err)
		return
	}

	fmt.Println(res)
}

func Test_Logout(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/inserv/v01/logout"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDczMDQ4NCwibmJmIjoxNzcwNjQ0MDg0LCJpYXQiOjE3NzA2NDQwODQsImp0aSI6Ijg2OTc0MTQwNjA3MzY4Mzk4MzcifQ.6nVD1doG4AkViunYrQ9CST6_fRda_xS9A38wqA-ROg0"

	// var key = []string{"Authorization"}
	// var value = []string{"Bearer " + token}

	res, err := PostWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to logout: %v", err)
	}
	fmt.Println(res)
}

///// test token
// test243 uid: 8697414060736839837
//act := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"
//rft := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

// qqqq11111 uid: 4033287471439576593
//act := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI"
//ref := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI"

func Test_GetSetting(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := fmt.Sprintf("/user/v01/getset/%s", "8697414060736839837")
	// qurl := fmt.Sprintf("/user/v01/mdata", "8697414060736839837")
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxODEyNzc1ODQ5fQ.ZKkpklhevJVbMXKqPYHPu1sbp_sPtAaSuhTRpygLbOc"

	//"77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"

	// meta := "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
	res, err := GetWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func Test_GetRefreshToken(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/refresh"
	// qurl := fmt.Sprintf("/user/v01/mdata", "8697414060736839837")
	// qurl := "/user/v01/mdata"
	// token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiaXNzIjoiY3VwaXRvay5jb20iLCJzdWIiOiJBdXRoZW50aWNhdGlvbiIsImF1ZCI6WyI4Njk3NDE0MDYwNzM2ODM5ODM3Il0sImV4cCI6MTc3MDIxNTc1NywibmJmIjoxNzcwMTI5MzU3LCJpYXQiOjE3NzAxMjkzNTcsImp0aSI6ImE0Zjc1OGVjLWQyZWEtNGQzOS05ZDVmLTNjMzQyMTc4M2ZhMCJ9.EMC2dOpcuIr33UBbjhfe8ahbqvXSpgr2-cfBkGbCLns"
	// token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxNzc2NjczNzA0LCJuYmYiOjE3NzY2NzM0MDQsImlhdCI6MTc3NjY3MzQwNH0.wnemA-Uz5TJtJqhZ7Nt8isEvPbAAxQUrzs4a-cUaxQo"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxNzc2NjczOTQyfQ.ibww4Q7zsXlo_548JsAjcmIx1AEAfqDUazoK8v5WhJM"

	//"77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"

	// meta := "77665817/test123/device123/nickname/1/25/Seoul/test@email.com/pic.jpg/thumb.jpg/Hello!"
	// res, err := GetWithToken(*targetUrl, qurl, nil, nil, token)
	res, err := PostWithToken(*targetUrl, qurl, nil, nil, token)
	if err != nil {
		t.Errorf("Failed to get story detail: %v", err)
	}

	fmt.Println(res)
}

func Test_SetSetting(t *testing.T) {
	// 0
	base := 0

	// noti=1, call=2, msg=4, rerv=8\n0 = all alert\n1111 = no alert\n0001 = no noti\n0011 = no noti call\n0111 = no noti, call, msg\n
	noti := 1
	call := 2
	msg := 4
	rvr := 8

	noNoti := noti | base
	noNotiCall := noti | call | base
	noNotiMsg := noti | msg | base
	noNotiRvr := noti | rvr | base

	fmt.Println(noNoti)
	fmt.Println(noNotiCall)
	fmt.Println(noNotiMsg)
	fmt.Println(noNotiRvr)

	noNotiRvrValue := 15 & noti //1
	fmt.Println("noNotiRvr value:", noNotiRvrValue)

	noNotiRvre := noti & 15 //
	fmt.Println("noNotiRvre value:", noNotiRvre)

	noNotiValue := 12 & noti //0
	fmt.Println("noNoti value:", noNotiValue)

	noValue := 15 &^ noti //14
	fmt.Println("noValue value:", noValue)

	res1 := 15 ^ 7
	fmt.Println("res1 value:", res1)

	res3 := res1 ^ 7
	fmt.Println("res3 value:", res3)

	res2 := 7 ^ 15
	fmt.Println("res2 value:", res2)

}

/*
{
    "sid": "test243",
    "uid": 8697414060736839837,
    "did": "",
    "email": "test243@test.com",
    "name": "홍두께",
    "nick": "건강한 꼬마 오렌지",
    "gender": "1",
    "age": "24",
    "birthday": "1990-01-01T00:00:00Z",
    "area": "서울",
    "stat": "0",
    "main_pic": "https://i.ibb.co/QF37KRST/male-ai-02.webp",
    "thmb_pic": "https://i.ibb.co/C5c51dYg/icon-male-04.webp",
    "sp_intro": "반가워요 큐피톡에서 만나요!"
}
*/

func Test_ToMetaHeader(t *testing.T) {

	meta := "test243 8697414060736839837  test243@test.com 홍두께 건강한 꼬마 오렌지 1 24 1990-01-01T00:00:00Z 서울 0 https://i.ibb.co/QF37KRST/male-ai-02.webp https://i.ibb.co/C5c51dYg/icon-male-04.webp 반가워요 큐피톡에서 만나요!"
	parts := strings.Split(meta, " ")
	if len(parts) != 13 {
		return
	}

	fmt.Println(meta)

}

// ---- story end ----

func TestBlockUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/create"

	var key = []string{"blocker_uid", "blocked_uid", "reason"}
	var value = []string{"77645423234236541", "7766493213763375817", "spam"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to block user: %v", err)
	}

	fmt.Println(res)
}

func TestUnblockUser(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/cancel"

	var key = []string{"blocker_uid", "blocked_uid"}
	var value = []string{"77645423234236541", "7766493213763375817"}

	res, err := PostJson(*targetUrl, qurl, key, value)
	if err != nil {
		t.Errorf("Failed to unblock user: %v", err)
	}

	fmt.Println(res)
}

func TestGetBlockList(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/story/v01/block/list/77645423234236541/1/20"

	res, err := Get(*targetUrl, qurl, nil, nil)
	if err != nil {
		t.Errorf("Failed to get block list: %v", err)
	}

	fmt.Println(res)
}

type ModifyMainPicReq struct {
	UID string `json:"uid"`
}

func TestSimpleMainPic(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/upd/mpic"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	// var key = []string{"data"}
	value := "test23"

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := ModifyMainPicReq{
		UID: value,
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	// Create a new file upload request
	filePath := "/home/jino/tmp/bc1.jpg" // 이미지 파일 경로 설정
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("파일 열기 실패: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("파일 정보 가져오기 실패: %v", err)
	}

	// 파일을 읽어들여 바이트 배열로 변환
	fileBytes := make([]byte, fileInfo.Size())
	_, err = file.Read(fileBytes)
	if err != nil {
		t.Fatalf("파일 읽기 실패: %v", err)
	}

	// 파일 업로드 요청 생성
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		t.Fatalf("폼 파일 생성 실패: %v", err)
	}
	part.Write(fileBytes)

	// 인코딩 된 파라메터 추가
	err = writer.WriteField("data", encryptedData)
	if err != nil {
		t.Fatalf("폼 필드 추가 실패: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", *targetUrl+qurl, body)
	if err != nil {
		t.Fatalf("HTTP 요청 생성 실패: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("이미지 업로드 실패: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("이미지 업로드 실패, 상태 코드: %d", res.StatusCode)
	} else {
		fmt.Println("이미지 업로드 성공")
	}

}

// ---- account ----
func TestAccLogin2(t *testing.T) {
	hash := "$2a$11$tO.ziYPkqMgVY28iiTn54ufX3TmIJlm13UiRIchaYQ80EUlu.HhZC"
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("123456789"))
	if err != nil {
		t.Errorf("Failed to compare hash and password: %v", err)
		return
	}
	fmt.Println("Password matches")
}

func TestAccLogout(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	// targetUrl := flag.String("target", "192.168.48.182:8080", "target server url")
	qurl := "/acc/v01/logout"

	var key = []string{"id"}
	// var value = []string{"test01", "usdx", "1111111123456", "test01@test.com", "0x0AFfB0a96FBefAa97dCe488DfD97512346cf3Ab8"}
	// var value = []string{"test23", "123456789", "홍길동", "1", "20", "1990-01-01", "서울"}
	var value = []string{"test13"}

	res, err := PostJson(*targetUrl, qurl, key, value)

	if err != nil {
		t.Errorf("Failed to logout: %v", err)
	}

	fmt.Println(res)
}

type LeaveReq struct {
	Id string `json:"id"`
}

func TestAccLeave(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/leave"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test43"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := LeaveReq{
		Id: value[0],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type FindIDReq struct {
	Name  string `json:"name"`
	Birth string `json:"birth"`
}

func TestAccFindID(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/fnid"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"홍길동", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindIDReq{
		Name:  value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func TestGetEmail(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/ckemail"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"홍길동", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindIDReq{
		Name:  value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})

	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

type FindPWReq struct {
	ID    string `json:"id"`
	Birth string `json:"birth"`
}

func TestAccFindPW(t *testing.T) {
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	qurl := "/acc/v01/fnpw"

	// var key = []string{"id", "pw", "name", "gender", "age", "birth", "area"}
	var key = []string{"data"}
	var value = []string{"test23", "1990-01-01"}

	// Convert the key-value pairs to a map for easier JSON handling
	dataMap := FindPWReq{
		ID:    value[0],
		Birth: value[1],
	}

	encryptedData, err := EncryptData(dataMap)
	if err != nil {
		t.Errorf("Failed to encrypt data: %v", err)
		return
	}

	res, err := PostEncJson(*targetUrl, qurl, key, []string{encryptedData})
	if err != nil {
		t.Errorf("Failed to add partner: %v", err)
	}

	fmt.Println(res)
}

func TestGenUuid(t *testing.T) {
	uuid := utils.GenUuid()
	fmt.Println(uuid)
}
