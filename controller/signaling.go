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

/*
// Client WebSocket 클라이언트
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}
*/

const (
	READ_BUFFER_SIZE            = 4096
	WRITE_BUFFER_SIZE           = 4096
	MAX_VIDEO_ROOMS             = 5000
	MAX_TOTAL_VIDEO_CONNECTIONS = 25000

	MAX_MESSAGE_SIZE = 512 * 1024

	MSG_BUFFER_SIZE = 256

	NUM_BRC_WORKERS = 15

	NUM_WORKERS          = 50
	WORKER_QUEUE_SIZE    = 1000
	BROADCAST_QUEUE_SIZE = 10000
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
type Room struct {
	ID           int64
	UserIDs      map[*Client]bool       // 클라이언트 맵
	UserNames    map[*Client]string     // 클라이언트 -> 사용자 이름
	Broadcast    chan *BroadcastMessage // 브로드캐스트 채널
	Unregister   chan *Client           // 클라이언트 해제 채널
	mu           sync.RWMutex
	createdAt    time.Time
	lastActivity time.Time
}

// Client 클라이언트 연결 정보
type Client struct {
	conn     *websocket.Conn
	send     chan []byte // 송신 버퍼
	room     *Room
	userName string
	mu       sync.Mutex
	lastSeen time.Time
}

// BroadcastMessage 브로드캐스트할 메시지
type BroadcastMessage struct {
	message []byte
	sender  *Client
	all     bool // true면 발신자 포함, false면 제외
}

// Message 클라이언트와 주고받는 메시지 형식
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// JoinData join 이벤트 데이터
type JoinData struct {
	Name string `json:"name"`
	Room string `json:"room"`
}

// WorkItem 워커가 처리할 작업
type WorkItem struct {
	client  *Client
	message []byte
}

// BroadcastJob 브로드캐스트 작업
type BroadcastJob struct {
	room    *Room
	message []byte
	sender  *Client
	all     bool
}

// SignalingController WebRTC 시그널링 컨트롤러
type SignalingController struct {
	ctl *Controller
	cfg *conf.Config
	rep *models.Repositories

	rooms       map[string]*Room // roomId -> Room
	roomsMu     sync.RWMutex
	upgrader    websocket.Upgrader
	messagePool sync.Pool // 메시지 재사용을 위한 풀
	totalConn   int64     // 전체 연결 수 (atomic)
	// stats          *Stats
	wrkQueue chan WorkItem
	brcQueue chan *BroadcastJob
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSignalingController 시그널링 컨트롤러 생성
func NewSignalingController(ctl *Controller, rep *models.Repositories) (*SignalingController, error) {
	r := &SignalingController{
		ctl:   ctl,
		rep:   rep,
		cfg:   ctl.cfg,
		rooms: make(map[string]*Room),
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
		ctx:      context.Background(),
		cancel:   context.CancelFunc(func() {}),
	}

	for i := 0; i < NUM_WORKERS; i++ {
		go r.msgWorker()
	}

	for i := 0; i < NUM_BRC_WORKERS; i++ {
		go r.brcWorker()
	}

	go r.roomCleaner()

	return r, nil
}

func (r *SignalingController) msgWorker() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case item := <-r.wrkQueue:
			r.processMessage(item.client, item.message)
		}
	}
}

func (r *SignalingController) processMessage(client *Client, message []byte) {
	var msg SignalingMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error("Message parse error:", err)
		return
	}

	msg.From = client.userName

	r.handleMessage(client, &msg)
}

func (r *SignalingController) brcWorker() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case item := <-r.brcQueue:
			r.broadcastMessage(item)
		}
	}
}

func (r *SignalingController) broadcastMessage(item *BroadcastJob) {
	item.room.Broadcast <- &BroadcastMessage{
		message: item.message,
		sender:  item.sender,
		all:     item.all,
	}
}

func (r *SignalingController) roomCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.cleanEmptyRooms()
		}
	}
}

func (r *SignalingController) cleanEmptyRooms() {
	r.roomsMu.Lock()
	defer r.roomsMu.Unlock()

	for name, room := range r.rooms {
		room.mu.RLock()
		isEmpty := len(room.UserIDs) == 0
		inactive := time.Since(room.lastActivity) > 5*time.Minute
		room.mu.RUnlock()

		if isEmpty && inactive {
			delete(r.rooms, name)
			atomic.AddInt64(&r.totalConn, -1)
			log.Info("Cleaned up inactive room: %s", name)
		}
	}
}

// HandleWebSocket WebSocket 연결 핸들러
func (p *SignalingController) HandleWebSocket(c *gin.Context) {
	// userId 파라미터 가져오기
	userId := c.Query("userId")
	if userId == "" {
		log.Warn("WebSocket connection attempt without userId")
		c.JSON(400, gin.H{"error": "userId required"})
		return
	}

	// WebSocket 업그레이드
	conn, err := p.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed:", err)
		return
	}

	// 클라이언트 생성
	client := &Client{
		conn: conn,
		send: make(chan []byte, MSG_BUFFER_SIZE),
	}

	// 클라이언트 등록
	p.registerClient(client)

	log.Info(fmt.Sprintf("New WebSocket connection: %s", userId))

	// 고루틴 시작
	go p.writePump(client)
	go p.readPump(client)
}

// registerClient 클라이언트 등록
func (p *SignalingController) registerClient(client *Client) {
	p.roomsMu.Lock()
	defer p.roomsMu.Unlock()

	// 기존 연결이 있으면 종료
	if oldClient, exists := p.rooms[strconv.FormatInt(client.room.ID, 10)]; exists {
		close(oldClient.Broadcast)
		for client := range oldClient.UserIDs {
			client.conn.Close()
		}
	}

	p.rooms[strconv.FormatInt(client.room.ID, 10)] = client.room

	// 사용자 목록 브로드캐스트
	go p.broadcastUserList()
}

// unregisterClient 클라이언트 등록 해제
func (p *SignalingController) unregisterClient(client *Client) {
	p.roomsMu.Lock()
	defer p.roomsMu.Unlock()

	if _, exists := p.rooms[strconv.FormatInt(client.room.ID, 10)]; exists {
		delete(p.rooms, strconv.FormatInt(client.room.ID, 10))
		close(client.send)
		log.Info(fmt.Sprintf("Client disconnected: %d", client.room.ID))
	}

	// 사용자 목록 브로드캐스트
	go p.broadcastUserList()
}

// readPump 메시지 읽기
func (p *SignalingController) readPump(client *Client) {
	defer func() {
		p.unregisterClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket error:", err)
			}
			break
		}

		// 메시지 파싱
		var msg SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Error("Message parse error:", err)
			continue
		}

		// 발신자 설정
		msg.From = client.userName

		// 메시지 처리
		p.handleMessage(client, &msg)
	}
}

// writePump 메시지 쓰기
func (p *SignalingController) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 대기 중인 메시지 일괄 전송
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 메시지 처리
func (p *SignalingController) handleMessage(client *Client, msg *SignalingMessage) {
	switch msg.Type {
	case "offer", "answer", "ice-candidate", "hangup":
		// 대상 사용자에게 메시지 전달
		p.relayMessage(client, msg)

	default:
		log.Warn(fmt.Sprintf("Unknown message type: %s from %s", msg.Type, client.userName))
	}
}

// relayMessage 메시지 중계
func (p *SignalingController) relayMessage(from *Client, msg *SignalingMessage) {
	if msg.To == "" {
		log.Warn("Message without target user")
		return
	}

	p.roomsMu.RLock()
	targetClient, exists := p.rooms[msg.To]
	p.roomsMu.RUnlock()

	if !exists {
		// 대상 사용자를 찾을 수 없음
		errorMsg := SignalingMessage{
			Type:    "error",
			Payload: fmt.Sprintf("User %s is not available", msg.To),
		}
		p.sendToClient(from, &errorMsg)
		return
	}

	// 메시지 전달
	// p.sendToRoom(targetClient, msg)
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Message marshal error:", err)
		return
	}

	targetClient.Broadcast <- &BroadcastMessage{
		message: data,
		sender:  nil,
		all:     false,
	}

	log.Info(fmt.Sprintf("Message relayed: %s from %s to %s", msg.Type, from.userName, msg.To))
}

// sendToClient 클라이언트에게 메시지 전송
func (p *SignalingController) sendToClient(client *Client, msg *SignalingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Message marshal error:", err)
		return
	}

	select {
	case client.send <- data:
	default:
		// 버퍼가 가득 찬 경우 클라이언트 제거
		p.unregisterClient(client)
	}
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

// GetConnectedUsers 연결된 사용자 목록 조회 (HTTP API)
func (p *SignalingController) GetConnectedUsers(c *gin.Context) {
	p.roomsMu.RLock()
	userList := make([]string, 0, len(p.rooms))
	for roomId := range p.rooms {
		userList = append(userList, roomId)
	}
	p.roomsMu.RUnlock()

	p.ctl.SimpleRespOK(c, gin.H{
		"users": userList,
		"count": len(userList),
	})
}

// CallRequest 통화 요청 처리
func (p *SignalingController) CallRequest(c *gin.Context) {
	var req ptl.CallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// 채팅방 ID가 없으면 생성
	if req.RoomID == "" {
		req.RoomID = fmt.Sprintf("call_%s_%s_%d", req.From, req.To, time.Now().Unix())
	}

	// ChatController를 통해 통화 요청 알림 전송
	if p.ctl.ChatCtl != nil {
		chatMsg := ptl.ChatMessage{
			Type:      "call-request",
			From:      req.From,
			To:        req.To,
			RoomID:    req.RoomID,
			Content:   fmt.Sprintf(`{"callType":"%s"}`, req.CallType),
			Timestamp: time.Now().Unix(),
		}
		p.ctl.ChatCtl.SendCallNotification(&chatMsg)
	}

	// Redis에 통화 요청 상태 저장 (선택적)
	// p.rdb.SetCallRequest(req.RoomID, req.From, req.To, req.CallType)

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"roomId":   req.RoomID,
		"callType": req.CallType,
		"status":   "requested",
	})
}

// CallAccept 통화 수락 처리
func (p *SignalingController) CallAccept(c *gin.Context) {
	var req ptl.CallResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	req.Accepted = true

	// ChatController를 통해 통화 수락 알림 전송
	if p.ctl.ChatCtl != nil {
		chatMsg := ptl.ChatMessage{
			Type:      "call-accept",
			From:      req.From,
			To:        req.To,
			RoomID:    req.RoomID,
			Content:   "accepted",
			Timestamp: time.Now().Unix(),
		}
		p.ctl.ChatCtl.SendCallNotification(&chatMsg)
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"roomId": req.RoomID,
		"status": "accepted",
	})
}

// CallReject 통화 거절 처리
func (p *SignalingController) CallReject(c *gin.Context) {
	var req ptl.CallResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		p.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	req.Accepted = false

	// ChatController를 통해 통화 거절 알림 전송
	if p.ctl.ChatCtl != nil {
		chatMsg := ptl.ChatMessage{
			Type:      "call-reject",
			From:      req.From,
			To:        req.To,
			RoomID:    req.RoomID,
			Content:   "rejected",
			Timestamp: time.Now().Unix(),
		}
		p.ctl.ChatCtl.SendCallNotification(&chatMsg)
	}

	p.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"roomId": req.RoomID,
		"status": "rejected",
	})
}
