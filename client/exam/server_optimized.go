package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// 설정 상수들
const (
	// WebSocket 설정
	writeWait      = 10 * time.Second    // 쓰기 작업 타임아웃
	pongWait       = 60 * time.Second    // Pong 대기 시간
	pingPeriod     = (pongWait * 9) / 10 // Ping 전송 주기
	maxMessageSize = 512 * 1024          // 최대 메시지 크기 (512KB)

	// 연결 제한
	maxConnectionsPerRoom = 3     // 방당 최대 연결 수
	maxRooms              = 10000 // 최대 방 수
	maxTotalConnections   = 50000 // 전체 최대 연결 수

	// 버퍼 크기
	messageBufferSize = 256 // 메시지 버퍼 크기

	// 워커 풀 설정
	numWorkers         = 100 // 메시지 처리 워커 수
	workerQueueSize    = 1000
	broadcastQueueSize = 10000
)

// Room 구조체: 각 방의 사용자 정보를 관리
type Room struct {
	Name         string
	Clients      map[*Client]bool       // 클라이언트 맵
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
	userID   string // 사용자 ID
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

// todo : joindata = userlist
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
}

// CallResponseData 연결 응답 데이터
type CallResponseData struct {
	From   string `json:"from"`    // 응답자 ID
	To     string `json:"to"`      // 요청자 ID
	RoomID string `json:"room_id"` // 방 ID
	Accept bool   `json:"accept"`  // 수락 여부
}

// WaitingRoom 대기방 구조체
type WaitingRoom struct {
	Clients      map[string]*Client // userId -> Client
	mu           sync.RWMutex
	createdAt    time.Time
	lastActivity time.Time
}

// Server 메인 서버 구조체
type Server struct {
	rooms            map[string]*Room
	roomsMu          sync.RWMutex
	waitingRoom      *WaitingRoom // 대기방
	upgrader         websocket.Upgrader
	messagePool      sync.Pool // 메시지 재사용을 위한 풀
	totalConnections int64     // 전체 연결 수 (atomic)
	stats            *Stats
	workerQueue      chan WorkItem
	broadcastQueue   chan *BroadcastJob
	ctx              context.Context
	cancel           context.CancelFunc
}

// Stats 서버 통계
type Stats struct {
	TotalConnections  int64
	TotalMessages     int64
	TotalRooms        int64
	MessagesPerSecond int64
	lastMessageCount  int64
	// 패킷 사이즈 추적 (바이트 단위)
	TotalBytesReceived  int64 // 클라이언트로부터 받은 총 바이트
	TotalBytesSent      int64 // 클라이언트로 보낸 총 바이트
	BytesReceivedPerSec int64 // 초당 수신 바이트
	BytesSentPerSec     int64 // 초당 송신 바이트
	lastBytesReceived   int64
	lastBytesSent       int64
	mu                  sync.RWMutex
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

// NewServer 새로운 서버 인스턴스 생성
func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		rooms: make(map[string]*Room),
		waitingRoom: &WaitingRoom{
			Clients:      make(map[string]*Client),
			createdAt:    time.Now(),
			lastActivity: time.Now(),
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true // CORS 허용 (프로덕션에서는 제한 필요)
			},
		},
		messagePool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, maxMessageSize)
			},
		},
		stats:          &Stats{},
		workerQueue:    make(chan WorkItem, workerQueueSize),
		broadcastQueue: make(chan *BroadcastJob, broadcastQueueSize),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 워커 풀 시작
	for i := 0; i < numWorkers; i++ {
		go s.messageWorker()
	}

	// 브로드캐스트 워커 시작
	for i := 0; i < runtime.NumCPU(); i++ {
		go s.broadcastWorker()
	}

	// 통계 수집 고루틴
	go s.statsCollector()

	// 방 정리 고루틴
	go s.roomCleaner()

	return s
}

// getRoom 방을 가져오거나 새로 생성
func (s *Server) getRoom(roomName string) (*Room, error) {
	s.roomsMu.RLock()
	if room, exists := s.rooms[roomName]; exists {
		s.roomsMu.RUnlock()
		return room, nil
	}
	s.roomsMu.RUnlock()

	// 방 수 제한 확인
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	// Double-check
	if room, exists := s.rooms[roomName]; exists {
		return room, nil
	}

	if len(s.rooms) >= maxRooms {
		return nil, &ServerError{Message: "Maximum number of rooms reached"}
	}

	log.Printf("Creating new room: %s", roomName)
	room := &Room{
		Name:         roomName,
		Clients:      make(map[*Client]bool),
		UserNames:    make(map[*Client]string),
		Broadcast:    make(chan *BroadcastMessage, messageBufferSize),
		Unregister:   make(chan *Client),
		createdAt:    time.Now(),
		lastActivity: time.Now(),
	}

	s.rooms[roomName] = room
	atomic.AddInt64(&s.stats.TotalRooms, 1)

	// 방 고루틴 시작
	go room.run(s)

	return room, nil
}

// ServerError 서버 에러
type ServerError struct {
	Message string
}

func (e *ServerError) Error() string {
	return e.Message
}

// run 방 관리 고루틴
func (r *Room) run(s *Server) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case client := <-r.Unregister:
			r.mu.Lock()
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				delete(r.UserNames, client)
				close(client.send)
				r.lastActivity = time.Now()
				log.Printf("Client unregistered from room %s (remaining: %d)", r.Name, len(r.Clients))
			}

			// 방이 비었으면 일정 시간 후 삭제를 위해 마킹
			if len(r.Clients) == 0 {
				r.mu.Unlock()
				return // 방 고루틴 종료
			}
			r.mu.Unlock()

		case msg := <-r.Broadcast:
			r.lastActivity = time.Now()

			// 브로드캐스트 작업을 큐에 추가
			s.broadcastQueue <- &BroadcastJob{
				room:    r,
				message: msg.message,
				sender:  msg.sender,
				all:     msg.all,
			}

		case <-ticker.C:
			// Ping을 모든 클라이언트에게 전송
			pingMsg := []byte(`{"type":"ping"}`)
			pingSize := int64(len(pingMsg))
			sentCount := int64(0)

			r.mu.RLock()
			for client := range r.Clients {
				select {
				case client.send <- pingMsg:
					sentCount++
				default:
					// 버퍼가 가득 찬 클라이언트는 연결 해제
					go func(c *Client) {
						r.Unregister <- c
					}(client)
				}
			}
			r.mu.RUnlock()

			// Ping 메시지 송신 바이트 추적
			if sentCount > 0 {
				atomic.AddInt64(&s.stats.TotalBytesSent, pingSize*sentCount)
			}
		}
	}
}

// messageWorker 메시지 처리 워커
func (s *Server) messageWorker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case item := <-s.workerQueue:
			s.processMessage(item.client, item.message)
			atomic.AddInt64(&s.stats.TotalMessages, 1)
		}
	}
}

// broadcastWorker 브로드캐스트 워커
func (s *Server) broadcastWorker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.broadcastQueue:
			messageSize := int64(len(job.message))
			sentCount := int64(0)

			job.room.mu.RLock()
			for client := range job.room.Clients {
				if !job.all && client == job.sender {
					continue
				}

				select {
				case client.send <- job.message:
					sentCount++
				default:
					// 버퍼가 가득 찬 경우 연결 해제
					log.Printf("Client buffer full, disconnecting")
					go func(c *Client) {
						job.room.Unregister <- c
					}(client)
				}
			}
			job.room.mu.RUnlock()

			// 송신 바이트 수 추적 (전송 성공한 클라이언트 수 * 메시지 크기)
			atomic.AddInt64(&s.stats.TotalBytesSent, messageSize*sentCount)
		}
	}
}

// processMessage 메시지 처리
func (s *Server) processMessage(client *Client, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// 대기방에 있는 클라이언트의 메시지 처리
	if client.room == nil {
		s.handleWaitingRoomMessage(client, &msg)
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
	}
}

// handleWaitingRoomMessage 대기방 메시지 처리
func (s *Server) handleWaitingRoomMessage(client *Client, msg *Message) {
	switch msg.Type {
	case "join-waiting":
		// 대기방 입장 (이미 처리됨)
		s.sendUserList(client)

	case "call-request":
		// 연결 요청
		s.handleCallRequest(client, msg)

	case "call-response":
		// 연결 응답 (수락/거절)
		s.handleCallResponse(client, msg)
	}
}

// sendUserList 사용자 목록 전송
func (s *Server) sendUserList(client *Client) {
	s.waitingRoom.mu.RLock()
	userList := make([]string, 0, len(s.waitingRoom.Clients))
	for userID := range s.waitingRoom.Clients {
		if userID != client.userID {
			userList = append(userList, userID)
		}
	}
	s.waitingRoom.mu.RUnlock()

	userListMsg := Message{
		Type: "user-list",
		Data: map[string]interface{}{
			"users": userList,
		},
	}

	data, _ := json.Marshal(userListMsg)
	select {
	case client.send <- data:
	default:
		log.Printf("Failed to send user list: buffer full")
	}
}

// handleCallRequest 연결 요청 처리
func (s *Server) handleCallRequest(client *Client, msg *Message) {
	// 요청 데이터 파싱
	dataBytes, _ := json.Marshal(msg.Data)
	var reqData CallRequestData
	if err := json.Unmarshal(dataBytes, &reqData); err != nil {
		log.Printf("Error parsing call request: %v", err)
		return
	}

	// 대상자 찾기
	s.waitingRoom.mu.RLock()
	targetClient, exists := s.waitingRoom.Clients[reqData.To]
	s.waitingRoom.mu.RUnlock()

	if !exists {
		log.Printf("Target user %s not found in waiting room", reqData.To)
		// 요청자에게 오류 메시지 전송
		errorMsg := Message{
			Type: "call-error",
			Data: map[string]interface{}{
				"message": "대상 사용자를 찾을 수 없습니다.",
			},
		}
		data, _ := json.Marshal(errorMsg)
		select {
		case client.send <- data:
		default:
		}
		return
	}

	// 대상자에게 연결 요청 전송
	requestMsg := Message{
		Type: "call-request",
		Data: map[string]interface{}{
			"from":    reqData.From,
			"to":      reqData.To,
			"room_id": reqData.RoomID,
		},
	}

	data, _ := json.Marshal(requestMsg)
	select {
	case targetClient.send <- data:
		log.Printf("Call request sent from %s to %s", reqData.From, reqData.To)
	default:
		log.Printf("Failed to send call request: buffer full")
	}
}

// handleCallResponse 연결 응답 처리
func (s *Server) handleCallResponse(client *Client, msg *Message) {
	// 응답 데이터 파싱
	dataBytes, _ := json.Marshal(msg.Data)
	var respData CallResponseData
	if err := json.Unmarshal(dataBytes, &respData); err != nil {
		log.Printf("Error parsing call response: %v", err)
		return
	}

	// 요청자 찾기
	s.waitingRoom.mu.RLock()
	requesterClient, exists := s.waitingRoom.Clients[respData.To]
	s.waitingRoom.mu.RUnlock()

	if !exists {
		log.Printf("Requester %s not found in waiting room", respData.To)
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
		},
	}

	data, _ := json.Marshal(responseMsg)
	select {
	case requesterClient.send <- data:
		log.Printf("Call response sent from %s to %s (accept: %v)", respData.From, respData.To, respData.Accept)
	default:
		log.Printf("Failed to send call response: buffer full")
	}

	// 수락한 경우 두 사용자를 방으로 이동
	if respData.Accept {
		go s.moveToRoom(respData.RoomID, client, requesterClient)
	}
}

// moveToRoom 사용자들을 방으로 이동
func (s *Server) moveToRoom(roomID string, client1, client2 *Client) {
	// 방 가져오기 또는 생성
	room, err := s.getRoom(roomID)
	if err != nil {
		log.Printf("Error getting room: %v", err)
		return
	}

	// 대기방에서 제거
	s.waitingRoom.mu.Lock()
	delete(s.waitingRoom.Clients, client1.userID)
	delete(s.waitingRoom.Clients, client2.userID)
	s.waitingRoom.mu.Unlock()

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

	// 두 클라이언트에게 방 입장 완료 메시지 전송
	joinMsg := Message{
		Type: "room-joined",
		Data: map[string]interface{}{
			"room_id": roomID,
		},
	}

	data, _ := json.Marshal(joinMsg)

	// client1에게 room-joined 전송
	select {
	case client1.send <- data:
		log.Printf("room-joined sent to %s", client1.userID)
	default:
		log.Printf("Failed to send room-joined to %s: buffer full", client1.userID)
		// 버퍼가 가득 찬 경우 재시도
		go func() {
			time.Sleep(100 * time.Millisecond)
			select {
			case client1.send <- data:
				log.Printf("room-joined sent to %s (retry)", client1.userID)
			default:
				log.Printf("Failed to send room-joined to %s after retry", client1.userID)
			}
		}()
	}

	// client2에게 room-joined 전송
	select {
	case client2.send <- data:
		log.Printf("room-joined sent to %s", client2.userID)
	default:
		log.Printf("Failed to send room-joined to %s: buffer full", client2.userID)
		// 버퍼가 가득 찬 경우 재시도
		go func() {
			time.Sleep(100 * time.Millisecond)
			select {
			case client2.send <- data:
				log.Printf("room-joined sent to %s (retry)", client2.userID)
			default:
				log.Printf("Failed to send room-joined to %s after retry", client2.userID)
			}
		}()
	}

	// 약간의 지연 후 start 시그널 전송 (화상채팅 시작)
	// room-joined가 먼저 처리되도록 약간의 지연 추가
	time.Sleep(50 * time.Millisecond)

	startMsg, _ := json.Marshal(Message{Type: "start"})
	select {
	case client1.send <- startMsg:
		atomic.AddInt64(&s.stats.TotalBytesSent, int64(len(startMsg)))
		log.Printf("start signal sent to %s", client1.userID)
	default:
		log.Printf("Failed to send start signal to %s: buffer full", client1.userID)
		// 버퍼가 가득 찬 경우 재시도
		go func() {
			time.Sleep(100 * time.Millisecond)
			select {
			case client1.send <- startMsg:
				atomic.AddInt64(&s.stats.TotalBytesSent, int64(len(startMsg)))
				log.Printf("start signal sent to %s (retry)", client1.userID)
			default:
				log.Printf("Failed to send start signal to %s after retry", client1.userID)
			}
		}()
	}

	select {
	case client2.send <- startMsg:
		atomic.AddInt64(&s.stats.TotalBytesSent, int64(len(startMsg)))
		log.Printf("start signal sent to %s", client2.userID)
	default:
		log.Printf("Failed to send start signal to %s: buffer full", client2.userID)
		// 버퍼가 가득 찬 경우 재시도
		go func() {
			time.Sleep(100 * time.Millisecond)
			select {
			case client2.send <- startMsg:
				atomic.AddInt64(&s.stats.TotalBytesSent, int64(len(startMsg)))
				log.Printf("start signal sent to %s (retry)", client2.userID)
			default:
				log.Printf("Failed to send start signal to %s after retry", client2.userID)
			}
		}()
	}

	log.Printf("Users %s and %s moved to room %s", client1.userID, client2.userID, roomID)
}

// broadcastUserJoined 새 사용자 입장 알림
func (s *Server) broadcastUserJoined(newClient *Client) {
	s.waitingRoom.mu.RLock()
	userList := make([]string, 0, len(s.waitingRoom.Clients))
	for userID := range s.waitingRoom.Clients {
		if userID != newClient.userID {
			userList = append(userList, userID)
		}
	}
	s.waitingRoom.mu.RUnlock()

	// 다른 사용자들에게 새 사용자 알림
	joinMsg := Message{
		Type: "user-joined",
		Data: map[string]interface{}{
			"user_id": newClient.userID,
			"name":    newClient.userName,
		},
	}

	data, _ := json.Marshal(joinMsg)
	s.waitingRoom.mu.RLock()
	for userID, client := range s.waitingRoom.Clients {
		if userID != newClient.userID {
			select {
			case client.send <- data:
			default:
			}
		}
	}
	s.waitingRoom.mu.RUnlock()
}

// broadcastUserLeft 사용자 나감 알림
func (s *Server) broadcastUserLeft(leftClient *Client) {
	leftMsg := Message{
		Type: "user-left",
		Data: map[string]interface{}{
			"user_id": leftClient.userID,
		},
	}

	data, _ := json.Marshal(leftMsg)
	s.waitingRoom.mu.RLock()
	for _, client := range s.waitingRoom.Clients {
		select {
		case client.send <- data:
		default:
		}
	}
	s.waitingRoom.mu.RUnlock()
}

// readPump 클라이언트로부터 메시지 읽기
func (c *Client) readPump(s *Server) {
	defer func() {
		if c.room != nil {
			c.room.Unregister <- c
		} else if c.userID != "" {
			// 대기방에서 제거
			s.waitingRoom.mu.Lock()
			delete(s.waitingRoom.Clients, c.userID)
			s.waitingRoom.mu.Unlock()
			// 다른 사용자들에게 사용자 나감 알림
			s.broadcastUserLeft(c)
		}
		c.conn.Close()
		atomic.AddInt64(&s.totalConnections, -1)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastSeen = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.lastSeen = time.Now()

		// 수신 바이트 수 추적
		messageSize := int64(len(message))
		atomic.AddInt64(&s.stats.TotalBytesReceived, messageSize)

		// 메시지를 워커 큐에 추가
		select {
		case s.workerQueue <- WorkItem{client: c, message: message}:
		default:
			log.Printf("Worker queue full, dropping message")
		}
	}
}

// writePump 클라이언트로 메시지 쓰기
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 버퍼에 대기 중인 메시지들을 배치로 전송
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleConnection WebSocket 연결 처리
func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	// 연결 수 제한 확인
	if atomic.LoadInt64(&s.totalConnections) >= maxTotalConnections {
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		log.Printf("Connection rejected: server at capacity")
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	atomic.AddInt64(&s.totalConnections, 1)
	atomic.AddInt64(&s.stats.TotalConnections, 1)

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, messageBufferSize),
		lastSeen: time.Now(),
	}

	// join-waiting 메시지를 기다림
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Error reading join-waiting message: %v", err)
		conn.Close()
		atomic.AddInt64(&s.totalConnections, -1)
		return
	}

	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling join-waiting message: %v", err)
		conn.Close()
		atomic.AddInt64(&s.totalConnections, -1)
		return
	}

	// join-waiting 또는 join 메시지 처리
	if msg.Type == "join-waiting" {
		// 대기방 입장
		var joinData JoinData
		dataBytes, _ := json.Marshal(msg.Data)
		if err := json.Unmarshal(dataBytes, &joinData); err != nil {
			log.Printf("Error parsing join-waiting data: %v", err)
			conn.Close()
			atomic.AddInt64(&s.totalConnections, -1)
			return
		}

		if joinData.UserID == "" {
			log.Printf("UserID is required for waiting room")
			conn.Close()
			atomic.AddInt64(&s.totalConnections, -1)
			return
		}

		client.userID = joinData.UserID
		client.userName = joinData.Name

		// 대기방에 추가
		s.waitingRoom.mu.Lock()
		// 이미 존재하는 사용자 ID인 경우 기존 연결 종료
		if existingClient, exists := s.waitingRoom.Clients[joinData.UserID]; exists {
			existingClient.conn.Close()
			delete(s.waitingRoom.Clients, joinData.UserID)
		}
		s.waitingRoom.Clients[joinData.UserID] = client
		s.waitingRoom.lastActivity = time.Now()
		s.waitingRoom.mu.Unlock()

		log.Printf("%s (ID: %s) joined waiting room", joinData.Name, joinData.UserID)

		// 사용자 목록 전송
		s.sendUserList(client)

		// 다른 사용자들에게 새 사용자 입장 알림
		s.broadcastUserJoined(client)

	} else if msg.Type == "join" {
		// 기존 방식: 직접 방 입장 (하위 호환성)
		var joinData JoinData
		dataBytes, _ := json.Marshal(msg.Data)
		if err := json.Unmarshal(dataBytes, &joinData); err != nil {
			log.Printf("Error parsing join data: %v", err)
			conn.Close()
			atomic.AddInt64(&s.totalConnections, -1)
			return
		}

		// 방 가져오기 또는 생성
		room, err := s.getRoom(joinData.Room)
		if err != nil {
			log.Printf("Error getting room: %v", err)
			conn.Close()
			atomic.AddInt64(&s.totalConnections, -1)
			return
		}

		// 방 인원 제한 확인
		room.mu.RLock()
		if len(room.Clients) >= maxConnectionsPerRoom {
			room.mu.RUnlock()
			log.Printf("Room %s is full", joinData.Room)
			conn.Close()
			atomic.AddInt64(&s.totalConnections, -1)
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

		log.Printf("%s joined room %s (total users: %d)", joinData.Name, joinData.Room, userCount)

		// start 시그널 전송 (방에 2명이 된 경우)
		if userCount == 2 {
			room.mu.RLock()
			for otherClient := range room.Clients {
				if otherClient != client {
					startMsg, _ := json.Marshal(Message{Type: "start"})
					select {
					case otherClient.send <- startMsg:
						log.Printf("Start signal sent to %s", room.UserNames[otherClient])
						// start 시그널 송신 바이트 추적
						atomic.AddInt64(&s.stats.TotalBytesSent, int64(len(startMsg)))
					default:
						log.Printf("Failed to send start signal: buffer full")
					}
					break
				}
			}
			room.mu.RUnlock()
		}
	} else {
		log.Printf("First message must be 'join-waiting' or 'join'")
		conn.Close()
		atomic.AddInt64(&s.totalConnections, -1)
		return
	}

	// Read/Write 펌프 시작
	go client.writePump()
	go client.readPump(s)
}

// statsCollector 통계 수집 고루틴
func (s *Server) statsCollector() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.stats.mu.Lock()
			// 메시지 초당 처리량 계산
			currentMessages := atomic.LoadInt64(&s.stats.TotalMessages)
			s.stats.MessagesPerSecond = currentMessages - s.stats.lastMessageCount
			s.stats.lastMessageCount = currentMessages

			// 바이트 초당 처리량 계산
			currentBytesReceived := atomic.LoadInt64(&s.stats.TotalBytesReceived)
			currentBytesSent := atomic.LoadInt64(&s.stats.TotalBytesSent)
			s.stats.BytesReceivedPerSec = currentBytesReceived - s.stats.lastBytesReceived
			s.stats.BytesSentPerSec = currentBytesSent - s.stats.lastBytesSent
			s.stats.lastBytesReceived = currentBytesReceived
			s.stats.lastBytesSent = currentBytesSent
			s.stats.mu.Unlock()

			// 5초마다 상세 통계 출력
			if time.Now().Unix()%5 == 0 {
				s.printStats()
			}
		}
	}
}

// roomCleaner 빈 방 정리 고루틴
func (s *Server) roomCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanEmptyRooms()
		}
	}
}

// cleanEmptyRooms 비어있는 방 정리
func (s *Server) cleanEmptyRooms() {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()

	for name, room := range s.rooms {
		room.mu.RLock()
		isEmpty := len(room.Clients) == 0
		inactive := time.Since(room.lastActivity) > 5*time.Minute
		room.mu.RUnlock()

		if isEmpty && inactive {
			delete(s.rooms, name)
			atomic.AddInt64(&s.stats.TotalRooms, -1)
			log.Printf("Cleaned up inactive room: %s", name)
		}
	}
}

// printStats 통계 출력
func (s *Server) printStats() {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	s.roomsMu.RLock()
	roomCount := len(s.rooms)
	s.roomsMu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 총 바이트를 KB로 변환
	totalReceivedKB := float64(atomic.LoadInt64(&s.stats.TotalBytesReceived)) / 1024
	totalSentKB := float64(atomic.LoadInt64(&s.stats.TotalBytesSent)) / 1024
	receivedPerSecKB := float64(s.stats.BytesReceivedPerSec) / 1024
	sentPerSecKB := float64(s.stats.BytesSentPerSec) / 1024

	log.Printf("=== Server Stats ===")
	log.Printf("Active Connections: %d", atomic.LoadInt64(&s.totalConnections))
	log.Printf("Total Connections: %d", s.stats.TotalConnections)
	log.Printf("Active Rooms: %d", roomCount)
	log.Printf("Total Messages: %d", s.stats.TotalMessages)
	log.Printf("Messages/sec: %d", s.stats.MessagesPerSecond)
	log.Printf("--- Packet Size ---")
	log.Printf("Total Received: %.2f KB (%.2f MB)", totalReceivedKB, totalReceivedKB/1024)
	log.Printf("Total Sent: %.2f KB (%.2f MB)", totalSentKB, totalSentKB/1024)
	log.Printf("Received/sec: %.2f KB/s", receivedPerSecKB)
	log.Printf("Sent/sec: %.2f KB/s", sentPerSecKB)
	log.Printf("--- Memory ---")
	log.Printf("Memory (Alloc): %.2f MB", float64(m.Alloc)/1024/1024)
	log.Printf("Memory (Sys): %.2f MB", float64(m.Sys)/1024/1024)
	log.Printf("Goroutines: %d", runtime.NumGoroutine())
	log.Printf("==================")
}

// Shutdown 서버 종료
func (s *Server) Shutdown() {
	log.Println("Shutting down server...")
	s.cancel()

	// 모든 방 닫기
	s.roomsMu.Lock()
	for _, room := range s.rooms {
		room.mu.Lock()
		for client := range room.Clients {
			close(client.send)
			client.conn.Close()
		}
		room.mu.Unlock()
	}
	s.roomsMu.Unlock()

	log.Println("Server shutdown complete")
}

func main_test() {
	// CPU 코어 수에 맞게 GOMAXPROCS 설정
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("Using %d CPU cores", runtime.NumCPU())

	// 서버 생성
	server := NewServer()

	// HTTP 핸들러 설정
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte("WebRTC Signaling Server (Optimized)"))
	})

	http.HandleFunc("/ws", server.handleConnection)

	// 헬스 체크 엔드포인트
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		server.stats.mu.RLock()
		receivedKB := float64(atomic.LoadInt64(&server.stats.TotalBytesReceived)) / 1024
		sentKB := float64(atomic.LoadInt64(&server.stats.TotalBytesSent)) / 1024
		receivedPerSecKB := float64(server.stats.BytesReceivedPerSec) / 1024
		sentPerSecKB := float64(server.stats.BytesSentPerSec) / 1024
		server.stats.mu.RUnlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "ok",
			"active_connections": atomic.LoadInt64(&server.totalConnections),
			"total_rooms":        atomic.LoadInt64(&server.stats.TotalRooms),
			"total_messages":     atomic.LoadInt64(&server.stats.TotalMessages),
			"bandwidth": map[string]interface{}{
				"total_received_kb":   receivedKB,
				"total_sent_kb":       sentKB,
				"received_kb_per_sec": receivedPerSecKB,
				"sent_kb_per_sec":     sentPerSecKB,
			},
		})
	})

	// Graceful shutdown 설정
	srv := &http.Server{
		Addr:         ":3000",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 시그널 핸들링
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		server.Shutdown()
	}()

	log.Printf("Optimized server is running on port 3000")
	log.Printf("Max connections: %d", maxTotalConnections)
	log.Printf("Max rooms: %d", maxRooms)
	log.Printf("Workers: %d", numWorkers)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe error: ", err)
	}
}
