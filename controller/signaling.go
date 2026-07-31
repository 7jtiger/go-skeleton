package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	log "ms-gateway/common/logger"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	READ_BUFFER_SIZE  = 4096
	WRITE_BUFFER_SIZE = 4096
	MAX_VIDEO_ROOMS   = 5000
	MAX_VDWS_CONNECT  = 25000

	MAX_MESSAGE_SIZE = 512 * 1024

	MSG_BUFFER_SIZE = 256

	NUM_BRC_WORKERS = 15

	MAX_CONN_ROOM        = 3
	NUM_WORKERS          = 50
	WORKER_QUEUE_SIZE    = 1000
	BROADCAST_QUEUE_SIZE = 10000

	PONG_WAIT   = 60 * time.Second
	PING_PERIOD = PONG_WAIT * 9 / 10
	WRITE_WAIT  = 10 * time.Second
)

// SignalingMessage 시그널링 메시지 구조
type SignalingMessage struct {
	Type      string      `json:"type"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	TotalConn int64       `json:"totalConn,omitempty"`
}

// Room 구조체: 각 방의 사용자 정보를 관리
type VDRoom struct {
	Name         string
	ID           int64
	Mode         string                 // "video", "text", "audio"
	Clients      map[*WSClient]bool     // 클라이언트 맵
	UserNames    map[*WSClient]string   // 클라이언트 -> 사용자 이름
	Broadcast    chan *BroadcastMessage // 브로드캐스트 채널
	Unregister   chan *WSClient         // 클라이언트 해제 채널
	done         chan struct{}          // 방 고루틴 종료 신호 (roomCleaner가 close)
	mu           sync.RWMutex
	createdAt    time.Time
	lastActivity time.Time
}

// Client 클라이언트 연결 정보
type WSClient struct {
	conn      *websocket.Conn
	send      chan []byte // 송신 버퍼
	room      *VDRoom
	userName  string
	userID    string // 사용자 ID
	mu        sync.Mutex
	closeOnce sync.Once // send 채널 이중 close 방지
	lastSeen  time.Time
}

// closeSend는 send 채널을 단 한 번만 닫는다 (close of closed channel 패닉 방지).
func (w *WSClient) closeSend() {
	w.closeOnce.Do(func() {
		close(w.send)
	})
}

// BroadcastMessage 브로드캐스트할 메시지
type BroadcastMessage struct {
	message []byte
	sender  *WSClient
	all     bool // true면 발신자 포함, false면 제외
}

// Message 클라이언트와 주고받는 메시지 형식
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// JoinData join 이벤트 데이터
type JoinData struct {
	Name   string `json:"name"`
	Room   string `json:"room"`
	UserID string `json:"user_id,omitempty"` // 사용자 ID
}

// CallRequestData 연결 요청 데이터
type CallRequestData struct {
	From   string `json:"from"`    // 요청자 ID
	To     string `json:"to"`      // 대상자 ID
	RoomID string `json:"room_id"` // 생성될 방 ID
	Mode   string `json:"mode"`    // "video", "text", "audio"
}

// CallResponseData 연결 응답 데이터
type CallResponseData struct {
	From   string `json:"from"`    // 응답자 ID
	To     string `json:"to"`      // 요청자 ID
	RoomID string `json:"room_id"` // 방 ID
	Accept bool   `json:"accept"`  // 수락 여부
	Mode   string `json:"mode"`    // "video", "text", "audio"
}

// WorkItem 워커가 처리할 작업
type WorkItem struct {
	client  *WSClient
	message []byte
}

// BroadcastJob 브로드캐스트 작업
type BroadcastJob struct {
	room    *VDRoom
	message []byte
	sender  *WSClient
	all     bool
}

// WaitingRoom 대기방 구조체
type WaitingRoom struct {
	Clients      map[string]*WSClient // userId -> WSClient
	mu           sync.RWMutex
	createdAt    time.Time
	lastActivity time.Time
}

// SignalingController WebRTC 시그널링 컨트롤러
type SignalingController struct {
	ctl       *Controller
	cfg       *conf.Config
	rep       *models.Repositories
	accountDB *models.AccountDB

	rooms       map[string]*VDRoom // roomId -> VDRoom
	roomsMu     sync.RWMutex
	waitingRoom *WaitingRoom // 대기방
	upgrader    websocket.Upgrader

	messagePool sync.Pool // 메시지 재사용을 위한 풀
	totalConn   int64     // 전체 연결 수 (atomic)
	wrkQueue    chan WorkItem
	brcQueue    chan *BroadcastJob
	ctx         context.Context
	cancel      context.CancelFunc
	workerWG    sync.WaitGroup
}

// NewSignalingController 시그널링 컨트롤러 생성
func NewSignalingController(ctl *Controller, rep *models.Repositories) (*SignalingController, error) {
	var accountDB *models.AccountDB
	if err := rep.Get(&accountDB); err != nil {
		return nil, err
	}

	r := &SignalingController{
		ctl:       ctl,
		rep:       rep,
		cfg:       ctl.cfg,
		accountDB: accountDB,
		rooms:     make(map[string]*VDRoom),
		waitingRoom: &WaitingRoom{
			Clients:      make(map[string]*WSClient),
			createdAt:    time.Now(),
			lastActivity: time.Now(),
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  READ_BUFFER_SIZE,
			WriteBufferSize: WRITE_BUFFER_SIZE,
			CheckOrigin: func(r *http.Request) bool {
				return true // 개발 중에는 모든 origin 허용
			},
		},
		messagePool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, MAX_MESSAGE_SIZE)
			},
		},
		wrkQueue: make(chan WorkItem, WORKER_QUEUE_SIZE),
		brcQueue: make(chan *BroadcastJob, BROADCAST_QUEUE_SIZE),
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel

	for i := 0; i < NUM_WORKERS; i++ {
		r.workerWG.Add(1)
		go func() {
			defer r.workerWG.Done()
			r.msgWorker()
		}()
	}

	for i := 0; i < NUM_BRC_WORKERS; i++ {
		r.workerWG.Add(1)
		go func() {
			defer r.workerWG.Done()
			r.brcWorker()
		}()
	}

	r.workerWG.Add(1)
	go func() {
		defer r.workerWG.Done()
		r.roomCleaner()
	}()

	return r, nil
}

// Shutdown은 graceful 종료 시 context·WebSocket·워커를 정리합니다.
func (p *SignalingController) Shutdown() {
	if p.cancel != nil {
		p.cancel()
	}

	p.waitingRoom.mu.Lock()
	for _, client := range p.waitingRoom.Clients {
		client.conn.Close()
	}
	p.waitingRoom.mu.Unlock()

	p.roomsMu.RLock()
	for _, room := range p.rooms {
		room.mu.RLock()
		for client := range room.Clients {
			client.conn.Close()
		}
		room.mu.RUnlock()
	}
	p.roomsMu.RUnlock()

	done := make(chan struct{})
	go func() {
		p.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("SignalingController workers stopped")
	case <-time.After(3 * time.Second):
		log.Warn("SignalingController worker shutdown timeout")
	}
}

// messageWorker 메시지 처리 워커
func (p *SignalingController) msgWorker() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case item := <-p.wrkQueue:
			p.processMessage(item.client, item.message)
		}
	}
}

func (p *SignalingController) processMessage(client *WSClient, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Error("Error unmarshaling message: %v", err)
		return
	}

	// 대기방에 있는 클라이언트의 메시지 처리
	if client.room == nil {
		p.handleWTRoom(client, &msg)
		return
	}

	room := client.room
	switch msg.Type {
	case "offer", "answer", "ice_candidate":
		// WebRTC 시그널링 메시지는 발신자 제외하고 브로드캐스트
		room.Broadcast <- &BroadcastMessage{
			message: data,
			sender:  client,
			all:     false,
		}

	case "chat":
		// 채팅 메시지 처리
		room.mu.RLock()
		senderName := room.UserNames[client]
		room.mu.RUnlock()

		chatMsg := Message{
			Type: "chat",
			Data: map[string]interface{}{
				"sender":  senderName,
				"message": msg.Data["message"],
			},
		}

		chatData, _ := json.Marshal(chatMsg)
		room.Broadcast <- &BroadcastMessage{
			message: chatData,
			sender:  client,
			all:     false,
		}

	case "present":
		// 선물 메시지 처리
		room.mu.RLock()
		senderName := room.UserNames[client]
		room.mu.RUnlock()

		presentMsg := Message{
			Type: "present",
			Data: map[string]interface{}{
				"sender": senderName,
				"item":   msg.Data["item"],
				"amount": msg.Data["amount"],
			},
		}

		presentData, _ := json.Marshal(presentMsg)
		room.Broadcast <- &BroadcastMessage{
			message: presentData,
			sender:  client,
			all:     false,
		}

	case "leave-room":
		// 자발적 퇴장 → 대기실 복귀
		p.returnToWaitingRoom(client)
	}
}

func (p *SignalingController) brcWorker() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case job := <-p.brcQueue:
			job.room.mu.RLock()
			for client := range job.room.Clients {
				if !job.all && client == job.sender {
					continue
				}

				if !p.trySend(client, job.message) {
					// 버퍼가 가득 찼거나 채널이 닫힌 경우 연결 해제
					log.Warn("Client buffer full or closed, disconnecting")
					go func(c *WSClient) {
						job.room.Unregister <- c
					}(client)
				}
			}
			job.room.mu.RUnlock()
		}
	}
}

// handleWTRoom 대기방 메시지 처리
func (p *SignalingController) handleWTRoom(client *WSClient, msg *Message) {
	switch msg.Type {
	case "join-waiting":
		// 대기방 입장 상태 확인용 (목록은 별도 요청 프로토콜로 전송)
	case "get-user-list":
		// 대기방 사용자 목록 요청 시에만 반환
		p.sendUserList(client)

	case "call-request":
		// 연결 요청
		p.handleCallRequest(client, msg)

	case "call-response":
		// 연결 응답 (수락/거절)
		p.handleCallResponse(client, msg)

	case "call-cancel":
		// 대기실 통화 취소
		p.handleCallCancel(client, msg)
	}
}

/*
최초 접속 프로세스
*/
// handleConnection WebSocket 연결 처리
// func (p *SignalingController) HandleConnection(w http.ResponseWriter, r *http.Request) {

// HandleConnection godoc
// @Summary Handles WebSocket connection and upgrades it to WebSocket protocol
// @Description Handles WebSocket connection and upgrades it to WebSocket protocol and puts the client into the waiting room or chat room
// @Tags Signaling
// @Accept json
// @Produce json
// @Success 101 {object} Message "Switching Protocols - WebSocket connection successful"
// @Router /webrtc/v01/ws [get]
// @Param userId query string false "UID"
//
// [WebSocket Protocol] WebRTC 시그널링/통화 WebSocket 엔드포인트
// 상세 프로토콜 설명:
//   ### 단계 1: WebSocket 접속 및 join-waiting 전송
//   **Client → Server**
//   ```json
//   {
//     "type": "join-waiting",
//     "data": {
//       "userID": "user123",
//       "name": "홍길동"
//     }
//   }
//   ```
//
//   ### 단계 2: 서버의 입장 broadcast 및 user-list 요청/응답
//   **Server → 모든 대기방 사용자** (누군가 입장하면 broadcast)
//   ```json
//   {
//     "type": "waiting-user-joined",
//     "data": {
//       "userID": "user123",
//       "name": "홍길동"
//     }
//   }
//   ```
//
//   **Client → Server** (리스트 요청)
//   ```json
//   {
//     "type": "get-user-list"
//   }
//   ```
////   **Server → Client** (user-list 응답)
//   ```json
//   {
//     "type": "waiting-user-list",
//     "data": [
//       { "uid": "user123", "nick": "홍길동" },
//       { "uid": "user456", "nick": "김철수" }
//     ]
//   }
//   ```
//
//   ### 단계 3: 통화 제안 시 call-request
//   **Client → Server**
//   ```json
//   {
//     "type": "call-request",
//     "data": {
//       "target_uid": "user456",
//       "media": { "video": {}, "audio": {} }
//     }
//   }
//   ```
//
//   **Server → 대상자** (call-incoming)
//   ```json
//   {
//     "type": "call-incoming",
//     "data": {
//       "from_uid": "user123",
//       "media": { "video": {}, "audio": {} }
//     }
//   }
//   ```
//
//   ### 단계 4: 통화 수락/거절 응답
//   **상대방 Client → Server**
//   ```json
//   {
//     "type": "call-response",
//     "data": {
//       "from_uid": "user456",
//       "accept": true
//     }
//   }
//   ```
////   **Server → 발신자** (call-result)
//   ```json
//   {
//     "type": "call-result",
//     "data": {
//       "accept": true,
//       "responder_uid": "user456"
//     }
//   }
//   ```
//
//   ### 단계 5: 통화 취소
//   **Client → Server**
//   ```json
//   {
//     "type": "call-cancel",
//     "data": { "target_uid": "user456" }
//   }
//   ```
//
//   **Server → 대상자**
//   ```json
//   {
//     "type": "call-canceled",
//     "data": { "target_uid": "user456" }
//   }
//   ```
//
//   ### 에러 예시
//   **Server → Client**
//   ```json
//   {
//     "type": "error",
//     "data": { "msg": "방이 가득 찼습니다", "code": 4001 }
//   }
//   ```
//
// Messages:
//   - name: join-waiting
//     description: 클라이언트가 최초 입장 시 전송
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: join-waiting }
//         data: { type: object }
//     responses:
//       - name: waiting-user-joined
//         description: 전체 대기방에 새로운 유저 입장 broadcast
//
//   - name: get-user-list
//     description: 대기방 사용자 목록 요청
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: get-user-list }
//     responses:
//       - name: waiting-user-list
//         description: 대기방 사용자 목록 응답
//
//   - name: call-request
//     description: 특정 대상에게 화상/음성 통화 제안시 사용
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: call-request }
//         data:
//           type: object
//           properties:
//             target_uid: { type: string }
//             media: { $ref: '#/components/schemas/MediaConfig' }
//     responses:
//       - name: call-incoming
//         description: 상대방에게 콜 제안 알림 메시지
//
//   - name: call-response
//     description: 통화 요청('call-request')에 대한 응답 (accept/deny)
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: call-response }
//         data:
//           type: object
//           properties:
//             from_uid: { type: string }
//             accept: { type: boolean }
//     responses:
//       - name: call-result
//         description: "콜 요청 발신자에게 전달되는 응답 (수락, 거절, 기타상황)"
//
//   - name: call-cancel
//     description: 통화 요청/진행 중 취소
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: call-cancel }
//         data:
//           type: object
//           properties:
//             target_uid: { type: string }
//     responses:
//       - name: call-canceled
//         description: 상대방에게 통화가 취소되었음을 알림
//
//   - name: error
//     description: 서버에서 발생한 에러 메시지
//     payload:
//       type: object
//       properties:
//         type: { type: string, example: error }
//         data:
//           type: object
//           properties:
//             msg: { type: string }
//             code: { type: integer, nullable: true }
//
// components:
//   schemas:
//     WTRoomUser:
//       type: object
//       properties:
//         uid: { type: string }
//         sid: { type: string }
//         did: { type: string }
//         mainPic: { type: string }
//         thumbPic: { type: string }
//         intro: { type: string }
//         gender: { type: string }
//         nick: { type: string }
//         area: { type: string }
//         age: { type: string }
//         newStat: { type: boolean }
//     MediaConfig:
//       type: object
//       properties:
//         video: { type: object }
//         audio: { type: object }

func (p *SignalingController) HandleConnection(c *gin.Context) {
	// 연결 수 제한 확인
	if atomic.LoadInt64(&p.totalConn) >= MAX_VDWS_CONNECT {
		http.Error(c.Writer, "Server at capacity", http.StatusServiceUnavailable)
		log.Error("Connection rejected: server at capacity")
		return
	}

	conn, err := p.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Error upgrading connection: %v", err)
		return
	}

	atomic.AddInt64(&p.totalConn, 1)

	client := &WSClient{
		conn:     conn,
		send:     make(chan []byte, MSG_BUFFER_SIZE),
		lastSeen: time.Now(),
	}

	// join-waiting 메시지를 기다림
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Warn("Error reading join-waiting message: %v", err)
		conn.Close()
		atomic.AddInt64(&p.totalConn, -1)
		return
	}

	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error("Error unmarshaling join-waiting message: %v", err)
		conn.Close()
		atomic.AddInt64(&p.totalConn, -1)
		return
	}

	// join-waiting 또는 join 메시지 처리
	if msg.Type == "join-waiting" {
		// 대기방 입장
		var joinData JoinData
		dataBytes, _ := json.Marshal(msg.Data)
		if err := json.Unmarshal(dataBytes, &joinData); err != nil {
			log.Warn("Error parsing join-waiting data: %v", err)
			conn.Close()
			atomic.AddInt64(&p.totalConn, -1)
			return
		}

		if joinData.UserID == "" {
			log.Warn("UserID is required for waiting room")
			conn.Close()
			atomic.AddInt64(&p.totalConn, -1)
			return
		}

		client.userID = joinData.UserID
		client.userName = joinData.Name

		// 대기방에 추가
		p.waitingRoom.mu.Lock()
		// 이미 존재하는 사용자 ID인 경우 기존 연결 종료
		if existingClient, exists := p.waitingRoom.Clients[joinData.UserID]; exists {
			existingClient.conn.Close()
			delete(p.waitingRoom.Clients, joinData.UserID)
		}
		p.waitingRoom.Clients[joinData.UserID] = client
		p.waitingRoom.lastActivity = time.Now()
		p.waitingRoom.mu.Unlock()

		log.Info("%s (ID: %s) joined waiting room", joinData.Name, joinData.UserID)

		// 다른 사용자들에게 새 사용자 입장 알림
		p.broadcastUserJoined(client)

	} else if msg.Type == "join" {
		// 기존 방식: 직접 방 입장 (하위 호환성)
		var joinData JoinData
		dataBytes, _ := json.Marshal(msg.Data)
		if err := json.Unmarshal(dataBytes, &joinData); err != nil {
			log.Warn("Error parsing join data: %v", err)
			conn.Close()
			atomic.AddInt64(&p.totalConn, -1)
			return
		}

		// 방 가져오기 또는 생성
		room, err := p.getRoom(joinData.Room)
		if err != nil {
			log.Warn("Error getting room: %v", err)
			conn.Close()
			atomic.AddInt64(&p.totalConn, -1)
			return
		}

		// 방 인원 제한 확인
		room.mu.RLock()
		if len(room.Clients) >= MAX_CONN_ROOM {
			room.mu.RUnlock()
			log.Warn("Room %s is full", joinData.Room)
			conn.Close()
			atomic.AddInt64(&p.totalConn, -1)
			return
		}
		room.mu.RUnlock()

		client.room = room
		client.userName = joinData.Name

		// 클라이언트를 먼저 방에 추가 (동기식으로)
		room.mu.Lock()
		room.Clients[client] = true
		room.UserNames[client] = joinData.Name
		userCount := len(room.Clients)
		room.lastActivity = time.Now()
		room.mu.Unlock()

		log.Info("%s joined room %s (total users: %d)", joinData.Name, joinData.Room, userCount)

		// start 시그널 전송 (방에 2명이 된 경우)
		if userCount == 2 {
			room.mu.RLock()
			for otherClient := range room.Clients {
				if otherClient != client {
					startMsg, _ := json.Marshal(Message{Type: "start"})
					if p.trySend(otherClient, startMsg) {
						log.Info("Start signal sent to %s", room.UserNames[otherClient])
					} else {
						log.Warn("Failed to send start signal")
					}
					break
				}
			}
			room.mu.RUnlock()
		}
	} else {
		log.Warn("First message must be 'join-waiting' or 'join'")
		conn.Close()
		atomic.AddInt64(&p.totalConn, -1)
		return
	}

	// Read/Write 펌프 시작
	go client.writePump()
	go client.readPump(p)
}

// sendUserList 사용자 목록 전송
func (p *SignalingController) sendUserList(client *WSClient) {
	p.waitingRoom.mu.RLock()
	type userInfo struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}
	userList := make([]userInfo, 0, len(p.waitingRoom.Clients))
	for userID, c := range p.waitingRoom.Clients {
		if userID != client.userID {
			userList = append(userList, userInfo{UserID: userID, Name: c.userName})
		}
	}
	p.waitingRoom.mu.RUnlock()

	userListMsg := Message{
		Type: "user-list",
		Data: map[string]interface{}{
			"users": userList,
		},
	}

	data, _ := json.Marshal(userListMsg)
	if !p.trySend(client, data) {
		log.Warn("Failed to send user list")
	}
}

// handleCallRequest 연결 요청 처리
func (p *SignalingController) handleCallRequest(client *WSClient, msg *Message) {
	// 요청 데이터 파싱
	dataBytes, _ := json.Marshal(msg.Data)
	var reqData CallRequestData
	if err := json.Unmarshal(dataBytes, &reqData); err != nil {
		log.Error("Error parsing call request: %v", err)
		return
	}

	// 대상자 찾기
	p.waitingRoom.mu.RLock()
	targetClient, exists := p.waitingRoom.Clients[reqData.To]
	p.waitingRoom.mu.RUnlock()

	if !exists {
		log.Warn("Target user %s not found in waiting room", reqData.To)

		if p.ctl != nil && p.ctl.ChatCtl != nil && p.ctl.ChatCtl.IsUserOnline(reqData.To) {
			p.ctl.ChatCtl.SendCallNotification(&ptl.ChatMessage{
				Type:      "call-incoming",
				From:      reqData.From,
				To:        reqData.To,
				RoomID:    reqData.RoomID,
				CallMode:  reqData.Mode,
				Timestamp: time.Now().Unix(),
			})
			infoMsg := Message{
				Type: "call-info",
				Data: map[string]interface{}{
					"message": "상대방이 대기실에 없어 채팅 채널로 통화 알림을 전달했습니다.",
				},
			}
			infoData, _ := json.Marshal(infoMsg)
			p.trySend(client, infoData)
			return
		}

		toUID, toErr := strconv.ParseUint(reqData.To, 10, 64)
		fromUID, fromErr := strconv.ParseUint(reqData.From, 10, 64)
		if toErr == nil && fromErr == nil && p.ctl != nil && p.ctl.FCMPusher != nil && p.accountDB != nil {
			caller, cErr := p.accountDB.GetUserInfoByUID(fromUID)
			target, tErr := p.accountDB.GetUserInfoByUID(toUID)
			if cErr == nil && tErr == nil {
				p.ctl.FCMPusher.SendCallPush(caller.Nick, caller.MainPic, reqData.Mode, target.Did)
				infoMsg := Message{
					Type: "call-info",
					Data: map[string]interface{}{
						"message": "상대방이 오프라인 상태여서 푸시 알림을 전송했습니다.",
					},
				}
				infoData, _ := json.Marshal(infoMsg)
				p.trySend(client, infoData)
				return
			}
		}

		errorMsg := Message{
			Type: "call-error",
			Data: map[string]interface{}{
				"message": "대상 사용자를 찾을 수 없습니다.",
			},
		}
		data, _ := json.Marshal(errorMsg)
		p.trySend(client, data)
		return
	}

	// 대상자에게 연결 요청 전송
	requestMsg := Message{
		Type: "call-request",
		Data: map[string]interface{}{
			"from":    reqData.From,
			"to":      reqData.To,
			"room_id": reqData.RoomID,
			"mode":    reqData.Mode,
		},
	}

	data, _ := json.Marshal(requestMsg)
	if p.trySend(targetClient, data) {
		log.Info("Call request sent from %s to %s", reqData.From, reqData.To)
	} else {
		log.Warn("Failed to send call request")
	}
}

// handleCallResponse 연결 응답 처리
func (p *SignalingController) handleCallResponse(client *WSClient, msg *Message) {
	// 응답 데이터 파싱
	dataBytes, _ := json.Marshal(msg.Data)
	var respData CallResponseData
	if err := json.Unmarshal(dataBytes, &respData); err != nil {
		log.Error("Error parsing call response: %v", err)
		return
	}

	// 요청자 찾기
	p.waitingRoom.mu.RLock()
	requesterClient, exists := p.waitingRoom.Clients[respData.To]
	p.waitingRoom.mu.RUnlock()

	if !exists {
		log.Warn("Requester %s not found in waiting room", respData.To)
		return
	}

	// 요청자에게 응답 전송
	responseType := "call-reject"
	if respData.Accept {
		responseType = "call-accept"
	}

	responseMsg := Message{
		Type: responseType,
		Data: map[string]interface{}{
			"from":    respData.From,
			"to":      respData.To,
			"room_id": respData.RoomID,
			"accept":  respData.Accept,
			"mode":    respData.Mode,
		},
	}
	if respData.Accept {
		partner, err := p.buildPartnerInfoByUserID(respData.From)
		if err == nil {
			responseMsg.Data["partner"] = partner
		}
	}

	data, _ := json.Marshal(responseMsg)
	if p.trySend(requesterClient, data) {
		log.Info("Call response sent from %s to %s (accept: %v)", respData.From, respData.To, respData.Accept)
	} else {
		log.Warn("Failed to send call response")
	}

	// 수락한 경우 두 사용자를 방으로 이동
	if respData.Accept {
		go p.moveToRoom(respData.RoomID, client, requesterClient, respData.Mode)
	}
}

func (p *SignalingController) handleCallCancel(client *WSClient, msg *Message) {
	dataBytes, _ := json.Marshal(msg.Data)
	var reqData CallRequestData
	if err := json.Unmarshal(dataBytes, &reqData); err != nil {
		log.Error("Error parsing call cancel: %v", err)
		return
	}

	cancelMsg := Message{
		Type: "call-cancel",
		Data: map[string]interface{}{
			"from":    reqData.From,
			"to":      reqData.To,
			"room_id": reqData.RoomID,
			"mode":    reqData.Mode,
		},
	}
	cancelData, _ := json.Marshal(cancelMsg)

	targetClient, exists := p.findClientByUserID(reqData.To)
	if exists {
		p.trySend(targetClient, cancelData)

		// 수신자가 이미 통화방에 있는 상태에서 취소가 들어오면
		// 종료 이벤트를 전달하고 대기실로 복귀시켜 통화를 정리한다.
		if targetClient.room != nil {
			endMsg := Message{
				Type: "partner-left",
				Data: map[string]interface{}{
					"message": "발신자가 통화를 취소했습니다. 통화를 종료합니다.",
				},
			}
			endData, _ := json.Marshal(endMsg)
			p.trySend(targetClient, endData)
			go p.returnToWaitingRoom(targetClient)
		}
	}

	if p.ctl != nil && p.ctl.ChatCtl != nil {
		p.ctl.ChatCtl.SendCallNotification(&ptl.ChatMessage{
			Type:      "call-cancel",
			From:      reqData.From,
			To:        reqData.To,
			RoomID:    reqData.RoomID,
			CallMode:  reqData.Mode,
			Timestamp: time.Now().Unix(),
		})
	}

	ack := Message{
		Type: "call-info",
		Data: map[string]interface{}{
			"message": "통화 요청을 취소했습니다.",
		},
	}
	ackData, _ := json.Marshal(ack)
	p.trySend(client, ackData)
}

// broadcastUserList 사용자 목록 브로드캐스트
func (p *SignalingController) broadcastUserList() {
	p.roomsMu.RLock()
	userList := make([]string, 0, len(p.rooms))
	for roomId := range p.rooms {
		userList = append(userList, roomId)
	}
	p.roomsMu.RUnlock()

	msg := SignalingMessage{
		Type:    "user-list",
		Payload: userList,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("User list marshal error:", err)
		return
	}

	p.roomsMu.RLock()
	defer p.roomsMu.RUnlock()

	for _, room := range p.rooms {
		select {
		case room.Broadcast <- &BroadcastMessage{
			message: data,
			sender:  nil,
			all:     true,
		}:
		default:
			// 전송 실패 시 무시
		}
	}

	log.Info(fmt.Sprintf("User list broadcasted: %v", userList))
}

// broadcastUserJoined 새 사용자 입장 알림
func (p *SignalingController) broadcastUserJoined(newClient *WSClient) {
	p.waitingRoom.mu.RLock()
	userList := make([]string, 0, len(p.waitingRoom.Clients))
	for userID := range p.waitingRoom.Clients {
		if userID != newClient.userID {
			userList = append(userList, userID)
		}
	}
	p.waitingRoom.mu.RUnlock()

	// 다른 사용자들에게 새 사용자 알림
	joinMsg := Message{
		Type: "user-joined",
		Data: map[string]interface{}{
			"user_id": newClient.userID,
			"name":    newClient.userName,
		},
	}

	data, _ := json.Marshal(joinMsg)
	p.waitingRoom.mu.RLock()
	for userID, client := range p.waitingRoom.Clients {
		if userID != newClient.userID {
			p.trySend(client, data)
		}
	}
	p.waitingRoom.mu.RUnlock()
}

// broadcastUserLeft 사용자 나감 알림
func (p *SignalingController) broadcastUserLeft(leftClient *WSClient) {
	leftMsg := Message{
		Type: "user-left",
		Data: map[string]interface{}{
			"user_id": leftClient.userID,
		},
	}

	data, _ := json.Marshal(leftMsg)
	p.waitingRoom.mu.RLock()
	for _, client := range p.waitingRoom.Clients {
		p.trySend(client, data)
	}
	p.waitingRoom.mu.RUnlock()
}

// IsUserInWaitingRoom 특정 사용자가 대기실에 있는지 확인
func (p *SignalingController) IsUserInWaitingRoom(userID string) bool {
	p.waitingRoom.mu.RLock()
	_, exists := p.waitingRoom.Clients[userID]
	p.waitingRoom.mu.RUnlock()
	return exists
}

// findClientByUserID는 대기실/통화방 전체에서 사용자 클라이언트를 찾는다.
func (p *SignalingController) findClientByUserID(userID string) (*WSClient, bool) {
	p.waitingRoom.mu.RLock()
	if client, exists := p.waitingRoom.Clients[userID]; exists {
		p.waitingRoom.mu.RUnlock()
		return client, true
	}
	p.waitingRoom.mu.RUnlock()

	p.roomsMu.RLock()
	defer p.roomsMu.RUnlock()

	for _, room := range p.rooms {
		room.mu.RLock()
		for c := range room.Clients {
			if c.userID == userID {
				room.mu.RUnlock()
				return c, true
			}
		}
		room.mu.RUnlock()
	}

	return nil, false
}

// ForwardCallRequest ChatController -> SignalingController 브릿지
func (p *SignalingController) ForwardCallRequest(msg *ptl.ChatMessage) {
	if msg == nil {
		return
	}

	p.waitingRoom.mu.RLock()
	targetClient, exists := p.waitingRoom.Clients[msg.To]
	p.waitingRoom.mu.RUnlock()
	if !exists {
		return
	}

	mode := msg.CallMode
	if mode == "" {
		mode = "video"
	}
	requestMsg := Message{
		Type: "call-request",
		Data: map[string]interface{}{
			"from":    msg.From,
			"to":      msg.To,
			"room_id": msg.RoomID,
			"mode":    mode,
		},
	}
	data, _ := json.Marshal(requestMsg)
	p.trySend(targetClient, data)
}

func (p *SignalingController) buildPartnerInfoByUserID(userID string) (map[string]interface{}, error) {
	if p.accountDB == nil {
		return nil, fmt.Errorf("account db unavailable")
	}
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, err
	}

	user, err := p.accountDB.GetUserInfoByUID(uid)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"uid":      user.Uid,
		"nick":     user.Nick,
		"mainPic":  user.MainPic,
		"thumbPic": user.ThumbPic,
		"spIntro":  user.SPIntro,
		"gender":   user.Gender,
		"age":      user.Age,
		"area":     user.Area,
	}, nil
}

// getRoom 방을 가져오거나 새로 생성
func (p *SignalingController) getRoom(roomName string) (*VDRoom, error) {
	p.roomsMu.RLock()
	if room, exists := p.rooms[roomName]; exists {
		p.roomsMu.RUnlock()
		return room, nil
	}
	p.roomsMu.RUnlock()

	// 방 수 제한 확인
	p.roomsMu.Lock()
	defer p.roomsMu.Unlock()

	// Double-check
	if room, exists := p.rooms[roomName]; exists {
		return room, nil
	}

	if len(p.rooms) >= MAX_VIDEO_ROOMS {
		return nil, fmt.Errorf("maximum number of rooms reached")
	}

	log.Info("Creating new room: %s", roomName)
	room := &VDRoom{
		Name:         roomName,
		Clients:      make(map[*WSClient]bool),
		UserNames:    make(map[*WSClient]string),
		Broadcast:    make(chan *BroadcastMessage, MSG_BUFFER_SIZE),
		Unregister:   make(chan *WSClient, MAX_CONN_ROOM*2), // run() 종료 후에도 전송자가 블록되지 않도록 버퍼링
		done:         make(chan struct{}),
		createdAt:    time.Now(),
		lastActivity: time.Now(),
	}

	p.rooms[roomName] = room
	atomic.AddInt64(&p.totalConn, 1)

	// 방 고루틴 시작
	go room.run(p)

	return room, nil
}

// moveToRoom 사용자들을 방으로 이동
func (p *SignalingController) moveToRoom(roomID string, client1, client2 *WSClient, mode string) {
	// 방 가져오기 또는 생성
	room, err := p.getRoom(roomID)
	if err != nil {
		log.Error("Error getting room: %v", err)
		return
	}

	room.Mode = mode

	// 대기방에서 제거
	p.waitingRoom.mu.Lock()
	delete(p.waitingRoom.Clients, client1.userID)
	delete(p.waitingRoom.Clients, client2.userID)
	p.waitingRoom.mu.Unlock()

	// 대기방 사용자들에게 user-left 알림
	p.broadcastUserLeft(client1)
	p.broadcastUserLeft(client2)

	// 방에 추가
	room.mu.Lock()
	room.Clients[client1] = true
	room.Clients[client2] = true
	room.UserNames[client1] = client1.userName
	room.UserNames[client2] = client2.userName
	room.lastActivity = time.Now()
	room.mu.Unlock()

	// 클라이언트의 room 설정
	client1.room = room
	client2.room = room

	// 두 클라이언트에게 방 입장 완료 메시지 전송 (mode 포함)
	joinMsg := Message{
		Type: "room-joined",
		Data: map[string]interface{}{
			"room_id": roomID,
			"mode":    mode,
		},
	}

	data, _ := json.Marshal(joinMsg)

	if p.trySend(client1, data) {
		log.Info("room-joined sent to %s", client1.userID)
	} else {
		log.Warn("Failed to send room-joined to %s", client1.userID)
	}

	if p.trySend(client2, data) {
		log.Info("room-joined sent to %s", client2.userID)
	} else {
		log.Warn("Failed to send room-joined to %s", client2.userID)
	}

	// text 모드가 아닌 경우에만 start 시그널 전송 (WebRTC 필요)
	if mode != "text" {
		time.Sleep(50 * time.Millisecond)

		startMsg, _ := json.Marshal(Message{Type: "start"})
		if p.trySend(client1, startMsg) {
			log.Info("start signal sent to %s", client1.userID)
		} else {
			log.Warn("Failed to send start signal to %s", client1.userID)
		}

		if p.trySend(client2, startMsg) {
			log.Info("start signal sent to %s", client2.userID)
		} else {
			log.Warn("Failed to send start signal to %s", client2.userID)
		}
	}

	log.Info("Users %s and %s moved to room %s (mode: %s)", client1.userID, client2.userID, roomID, mode)
}

func (p *SignalingController) roomCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanEmptyRooms()
		}
	}
}

func (p *SignalingController) cleanEmptyRooms() {
	p.roomsMu.Lock()
	defer p.roomsMu.Unlock()

	for name, room := range p.rooms {
		room.mu.RLock()
		isEmpty := len(room.Clients) == 0
		inactive := time.Since(room.lastActivity) > 5*time.Minute
		room.mu.RUnlock()

		if isEmpty && inactive {
			delete(p.rooms, name)
			close(room.done) // run() 고루틴 종료
			atomic.AddInt64(&p.totalConn, -1)
			log.Info("Cleaned up inactive room: %s", name)
		}
	}
}

// returnToWaitingRoom 클라이언트를 방에서 제거하고 대기실로 복귀
func (p *SignalingController) returnToWaitingRoom(client *WSClient) {
	room := client.room
	if room == nil {
		return
	}

	// 1. 방에서 제거 (send 채널 닫지 않음)
	room.mu.Lock()
	delete(room.Clients, client)
	delete(room.UserNames, client)
	remaining := len(room.Clients)
	room.mu.Unlock()

	// 2. client.room = nil
	client.room = nil

	// 3. 대기방에 재추가
	p.waitingRoom.mu.Lock()
	p.waitingRoom.Clients[client.userID] = client
	p.waitingRoom.lastActivity = time.Now()
	p.waitingRoom.mu.Unlock()

	// 4. 본인에게 "returned-to-waiting" + user-list 전송
	returnMsg := Message{
		Type: "returned-to-waiting",
		Data: map[string]interface{}{
			"message": "대기실로 복귀했습니다.",
		},
	}
	returnData, _ := json.Marshal(returnMsg)
	p.trySend(client, returnData)

	p.sendUserList(client)

	// 5. 대기방 사용자들에게 user-joined 브로드캐스트
	p.broadcastUserJoined(client)

	log.Info("User %s returned to waiting room", client.userID)

	// 6. 남은 상대방도 대기실로 복귀
	if remaining > 0 {
		go p.forceReturnToWaitingRoom(room, client)
	}
}

// forceReturnToWaitingRoom 방에 남은 사용자들을 모두 대기실로 복귀
func (p *SignalingController) forceReturnToWaitingRoom(room *VDRoom, skipClient *WSClient) {
	room.mu.Lock()
	remainingClients := make([]*WSClient, 0)
	for c := range room.Clients {
		if c == skipClient {
			continue
		}
		remainingClients = append(remainingClients, c)
	}
	room.mu.Unlock()

	for _, c := range remainingClients {
		// 상대방에게 partner-left 알림
		partnerLeftMsg := Message{
			Type: "partner-left",
			Data: map[string]interface{}{
				"message": "상대방이 나갔습니다.",
			},
		}
		partnerData, _ := json.Marshal(partnerLeftMsg)
		p.trySend(c, partnerData)

		p.returnToWaitingRoom(c)
	}
}

// trySend는 닫힌 채널 전송 panic을 방지하는 안전 전송 래퍼다.
// - hotfix: send on closed channel 크래시를 막는다.
// - refactor 보강: 레이스 상황에서도 서버 프로세스가 종료되지 않도록 한다.
func (p *SignalingController) trySend(client *WSClient, data []byte) (sent bool) {
	if client == nil {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			// 채널이 이미 닫힌 경우(send on closed channel) 서버 패닉 대신 경고 로그만 남긴다.
			log.Warn("safe send recovered (user: %s): %v", client.userID, r)
			sent = false
		}
	}()

	select {
	case client.send <- data:
		return true
	default:
		return false
	}
}

// ===== VDRoom =============================================================

// run 방 관리 고루틴
func (v *VDRoom) run(p *SignalingController) {
	ticker := time.NewTicker(PING_PERIOD)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return

		case <-v.done:
			return

		case client := <-v.Unregister:
			v.mu.Lock()
			if _, ok := v.Clients[client]; ok {
				delete(v.Clients, client)
				delete(v.UserNames, client)
				client.closeSend()
				v.lastActivity = time.Now()
				log.Info(fmt.Sprintf("Client unregistered from room %s (remaining: %d)", v.Name, len(v.Clients)))
			}
			v.mu.Unlock()
			// 방이 비어도 고루틴은 유지한다. roomCleaner가 done을 닫을 때 종료
			// (즉시 종료하면 이후 Unregister 전송자가 영원히 블록되는 누수가 발생)

		case msg := <-v.Broadcast:
			v.lastActivity = time.Now()

			// 브로드캐스트 작업을 큐에 추가
			p.brcQueue <- &BroadcastJob{
				room:    v,
				message: msg.message,
				sender:  msg.sender,
				all:     msg.all,
			}

		case <-ticker.C:
			// Ping을 모든 클라이언트에게 전송
			pingMsg := []byte(`{"type":"ping"}`)
			v.mu.RLock()
			for client := range v.Clients {
				if !p.trySend(client, pingMsg) {
					// 버퍼가 가득 찼거나 채널이 닫힌 클라이언트는 연결 해제
					go func(c *WSClient) {
						v.Unregister <- c
					}(client)
				}
			}
			v.mu.RUnlock()
		}
	}
}

// ===== WSClient =============================================================
func (w *WSClient) readPump(p *SignalingController) {
	defer func() {
		if w.room != nil {
			room := w.room
			w.room.Unregister <- w
			// 남은 사용자들을 대기실로 복귀
			go p.forceReturnToWaitingRoom(room, w)
		} else if w.userID != "" {
			// 대기방에서 제거
			p.waitingRoom.mu.Lock()
			delete(p.waitingRoom.Clients, w.userID)
			p.waitingRoom.mu.Unlock()
			// 다른 사용자들에게 사용자 나감 알림
			p.broadcastUserLeft(w)
		}
		w.conn.Close()
		atomic.AddInt64(&p.totalConn, -1)
	}()

	w.conn.SetReadLimit(MAX_MESSAGE_SIZE)
	w.conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
	w.conn.SetPongHandler(func(string) error {
		w.conn.SetReadDeadline(time.Now().Add(PONG_WAIT))
		w.lastSeen = time.Now()
		return nil
	})

	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket error:", err)
			}
			break
		}

		w.lastSeen = time.Now()

		// 메시지를 워커 큐에 추가
		select {
		case p.wrkQueue <- WorkItem{client: w, message: message}:
		default:
			log.Error("Worker queue full, dropping message")
		}
	}
}

func (w *WSClient) writePump() {
	ticker := time.NewTicker(PING_PERIOD)
	defer func() {
		ticker.Stop()
		w.conn.Close()
	}()

	for {
		select {
		case message, ok := <-w.send:
			w.conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if !ok {
				w.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			wt, err := w.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			wt.Write(message)

			// 버퍼에 대기 중인 메시지들을 배치로 전송
			n := len(w.send)
			for i := 0; i < n; i++ {
				wt.Write([]byte{'\n'})
				wt.Write(<-w.send)
			}

			if err := wt.Close(); err != nil {
				return
			}

		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
