package controller

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	// "gocv.io/x/gocv"
)

// ----------------- dm test
var (
	dmTargetHost = flag.String("dm_target", "localhost:8080", "dm test target server url")
	// dmToken      = flag.String("dm_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjoxODEyNzc1ODQ5fQ.ZKkpklhevJVbMXKqPYHPu1sbp_sPtAaSuhTRpygLbOc", "dm test bearer token")
	dmToken     = flag.String("dm_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE", "dm test bearer token")
	dmUID       = flag.String("dm_uid", "8697414060736839837", "dm test uid (required for dm room/list/unread/ws query)")
	dmPeerUID   = flag.String("dm_peer_uid", "4033287471439576593", "dm test peer uid (required for dm room create)")
	dmPeerToken = flag.String("dm_peer_token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjo0OTMxMTQ3Mjk2fQ.cphoVA_rhSlXTEgbnYsHIGV4PxKvePGh1e-Y7mJmPkI", "dm test peer bearer token (required for peer ws auth)")
)

//"acTok":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjoxNzc3NDYxMjI2fQ.em-dbS_WqiW3BWYj33cz0RjZfUEO2bRQF_GpVnCyVZ0",
// "refTok":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI0MDMzMjg3NDcxNDM5NTc2NTkzIiwiZXhwIjoxNzc4NTg0NDI2fQ.eLgd6RLtddrNVwM_YrYhRahNKNL7qGQjP5-lr28cJ1I"
//

func Test_ConnectWWS(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run websocket dm test")
	}

	u := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	q := u.Query()
	q.Set("userId", *dmUID)
	u.RawQuery = q.Encode()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+*dmToken)

	conn, res, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		if res != nil {
			t.Fatalf("Failed to connect dm websocket: %v (status=%d)", err, res.StatusCode)
		}
		t.Fatalf("Failed to connect dm websocket: %v", err)
		return
	}
	defer conn.Close()

	// t.Logf("dm websocket connected: %s", u.String())
}

func Test_CreateDMRoom(t *testing.T) {
	if *dmToken == "" || *dmUID == "" || *dmPeerUID == "" {
		t.Skip("set -dm_token, -dm_uid, -dm_peer_uid to run dm room create test")
	}

	qurl := "/dm/v01/mkroom/" + *dmPeerUID
	res, err := PostWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to create dm room: %v", err)
		return
	}
	fmt.Println("create dm room:", res)
}

func Test_GetDMRooms(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm room list test")
	}

	qurl := "/dm/v01/list/1"
	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get dm rooms: %v", err)
		return
	}
	fmt.Println("dm rooms:", res)
}

func Test_GetTotalUnread(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm unread test")
	}

	qurl := "/dm/v01/total/unread"
	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get total unread: %v", err)
		return
	}
	fmt.Println("dm unread:", res)
}

func Test_GetChatList(t *testing.T) {
	if *dmToken == "" || *dmUID == "" {
		t.Skip("set -dm_token and -dm_uid to run dm chat list test")
	}

	qurl := "/dm/v01/history/10?limit=20"
	res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get chat list: %v", err)
		return
	}
	fmt.Println("dm chat list:", res)
}

func Test_SendDMMessage(t *testing.T) {
	// if *dmToken == "" || *dmUID == "" || *dmPeerUID == "" || *dmPeerToken == "" {
	// 	t.Skip("set -dm_token, -dm_uid, -dm_peer_uid, -dm_peer_token to run dm message send test")
	// }

	senderURL := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	senderQ := senderURL.Query()
	senderQ.Set("userId", *dmUID)
	senderURL.RawQuery = senderQ.Encode()

	senderHeader := http.Header{}
	senderHeader.Set("Authorization", "Bearer "+*dmToken)
	senderConn, senderRes, err := websocket.DefaultDialer.Dial(senderURL.String(), senderHeader)
	if err != nil {
		if senderRes != nil {
			t.Fatalf("Failed to connect sender websocket: %v (status=%d)", err, senderRes.StatusCode)
		}
		t.Fatalf("Failed to connect sender websocket: %v", err)
	}
	defer senderConn.Close()

	receiverURL := url.URL{
		Scheme: "ws",
		Host:   *dmTargetHost,
		Path:   "/dm/v01/ws",
	}
	receiverQ := receiverURL.Query()
	receiverQ.Set("userId", *dmPeerUID)
	receiverURL.RawQuery = receiverQ.Encode()

	receiverHeader := http.Header{}
	receiverHeader.Set("Authorization", "Bearer "+*dmPeerToken)
	receiverConn, receiverRes, err := websocket.DefaultDialer.Dial(receiverURL.String(), receiverHeader)
	if err != nil {
		if receiverRes != nil {
			t.Fatalf("Failed to connect receiver websocket: %v (status=%d)", err, receiverRes.StatusCode)
		}
		t.Fatalf("Failed to connect receiver websocket: %v", err)
	}
	defer receiverConn.Close()

	roomRes, err := PostWithToken(*dmTargetHost, "/dm/v01/mkroom/"+*dmPeerUID, nil, nil, *dmToken)
	if err != nil {
		t.Fatalf("Failed to create dm room: %v", err)
	}
	t.Logf("create dm room response: %s", roomRes)

	msgID := fmt.Sprintf("dm-it-msg-%d", time.Now().UnixNano())
	wantContent := "hello from dm integration test"
	sendPayload := map[string]interface{}{
		"type":    "text-message",
		"to":      *dmPeerUID,
		"content": wantContent,
		"msgId":   msgID,
	}

	if err := senderConn.WriteJSON(sendPayload); err != nil {
		t.Fatalf("Failed to send websocket dm message: %v", err)
	}

	receiverConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, recvRaw, err := receiverConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read dm message on receiver: %v", err)
	}

	var recvMsg map[string]interface{}
	if err := json.Unmarshal(recvRaw, &recvMsg); err != nil {
		t.Fatalf("Failed to parse receiver message: %v raw=%s", err, string(recvRaw))
	}

	if recvMsg["type"] != "text-message" {
		t.Fatalf("Unexpected receiver message type: %v raw=%s", recvMsg["type"], string(recvRaw))
	}
	if recvMsg["from"] != *dmUID {
		t.Fatalf("Unexpected receiver from uid: got=%v want=%s", recvMsg["from"], *dmUID)
	}
	if recvMsg["to"] != *dmPeerUID {
		t.Fatalf("Unexpected receiver to uid: got=%v want=%s", recvMsg["to"], *dmPeerUID)
	}
	if recvMsg["content"] != wantContent {
		t.Fatalf("Unexpected receiver content: got=%v want=%s", recvMsg["content"], wantContent)
	}

	senderConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, ackRaw, err := senderConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read msg-ack on sender: %v", err)
	}

	var ackMsg map[string]interface{}
	if err := json.Unmarshal(ackRaw, &ackMsg); err != nil {
		t.Fatalf("Failed to parse sender ack message: %v raw=%s", err, string(ackRaw))
	}

	if ackMsg["type"] != "msg-ack" {
		t.Fatalf("Unexpected ack message type: %v raw=%s", ackMsg["type"], string(ackRaw))
	}
	if ackMsg["msgId"] != msgID {
		t.Fatalf("Unexpected ack msgId: got=%v want=%s", ackMsg["msgId"], msgID)
	}
	if ackMsg["to"] != *dmUID {
		t.Fatalf("Unexpected ack to uid: got=%v want=%s", ackMsg["to"], *dmUID)
	}
}

func Test_UploadChatImage(t *testing.T) {
	// story.POST("/create", p.ValidateFileUpload(5, 5), p.st.CreateStory)
	// "./bak/tmpimg/qwer.jpg"
	// "./bak/tmpimg/rewq.jpg"
	// "./bak/tmpimg/wet1.jpg"

	// This test demonstrates a multipart/form POST to the /story/v01/create API.
	targetUrl := flag.String("target", "localhost:8080", "target server url")
	flag.Parse()

	uploadUrl := fmt.Sprintf("http://%s/dm/v01/upload/img", *targetUrl)

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

	// Set the requested access token in the header for authentication
	// testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI4Njk3NDE0MDYwNzM2ODM5ODM3IiwiZXhwIjo0OTMxMTQ3NDQ4fQ.3_r7sDc90IoeLEXO78d5MIp4Ejn3RHpYWixjdrHFrfE"

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	// Build the HTTP request
	req, err := http.NewRequest("POST", uploadUrl, &b)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+*dmToken)

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

	//"{\"result\":0,\"resultString\":\"Success\",\"data\":{\"data\":{\"rewq.jpg\":\"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/1356a6c4-6a5a-4627-a316-268a452d1700/public\",\"wet1.jpg\":\"https://imagedelivery.net/bhnuJ7hC7hq1zO__1yxVLg/4ec7eb26-a181-43fd-efdd-caf8f8669b00/public\"},\"msg\":\"success\"}}"
	fmt.Printf("Status: %d, Response: %s\n", resp.StatusCode, string(respBody))

}

func Test_chatProcess(t *testing.T) {
	if *dmToken == "" || *dmUID == "" || *dmPeerUID == "" || *dmPeerToken == "" {
		t.Skip("set -dm_token, -dm_uid, -dm_peer_uid, -dm_peer_token to run chat process test")
	}

	qurl := "/dm/v01/list/1"
	// res, err := GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	res, err := GetWithToken("5e3983f3a310.ngrok-free.app", qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get dm rooms: %v", err)
		return
	}
	fmt.Println("dm rooms:", res)

	qurl = "/dm/v01/total/unread"
	res, err = GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get total unread: %v", err)
		return
	}
	fmt.Println("dm unread:", res)

	qurl = "/dm/v01/history/10?limit=20"
	res, err = GetWithToken(*dmTargetHost, qurl, nil, nil, *dmToken)
	if err != nil {
		t.Errorf("Failed to get chat list: %v", err)
		return
	}
	fmt.Println("dm chat list:", res)
}
