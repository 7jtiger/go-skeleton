package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "ms-gateway/common/logger"
	"ms-gateway/common/utils"
	"ms-gateway/conf"
	"ms-gateway/models"
	ptl "ms-gateway/protocol"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	CHAT_READ_BUFFER_SIZE  = 4096
	CHAT_WRITE_BUFFER_SIZE = 4096
	CHAT_MAX_MESSAGE_SIZE  = 64 * 1024 // 64KB
	CHAT_MSG_BUFFER_SIZE   = 256
	CHAT_PING_PERIOD       = 54 * time.Second
	CHAT_PONG_WAIT         = 60 * time.Second
	CHAT_WRITE_WAIT        = 10 * time.Second
)

// ChatClient 텍스트 채팅 WebSocket 클라이언트
type ChatClient struct {
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	mu       sync.Mutex
	lastSeen time.Time
}

// ChatController 텍스트 채팅 컨트롤러
type ChatController struct {
	ctl       *Controller
	cfg       *conf.Config
	rep       *models.Repositories
	rdb       *models.RedisDB
	clients   map[string]*ChatClient // userID -> ChatClient
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
}

// NewChatController ChatController 생성
func NewChatController(ctl *Controller, rep *models.Repositories) (*ChatController, error) {
	rdb := ctl.GetRedis()
	if rdb == nil {
		return nil, fmt.Errorf("redis database not available")
	}

	return &ChatController{
		ctl:     ctl,
		cfg:     ctl.cfg,
		rep:     rep,
		rdb:     rdb,
		clients: make(map[string]*ChatClient),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  CHAT_READ_BUFFER_SIZE,
			WriteBufferSize: CHAT_WRITE_BUFFER_SIZE,
			CheckOrigin: func(r *http.Request) bool {
				return true // 개발 중에는 모든 origin 허용
			},
		},
	}, nil
}

// HandleWebSocket 텍스트 채팅 WebSocket 연결 핸들러
func (cc *ChatController) HandleWebSocket(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		log.Warn("WebSocket connection attempt without userId")
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	// WebSocket 업그레이드
	conn, err := cc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed:", err)
		return
	}

	// 클라이언트 생성 및 등록
	client := &ChatClient{
		conn:     conn,
		send:     make(chan []byte, CHAT_MSG_BUFFER_SIZE),
		userID:   userId,
		lastSeen: time.Now(),
	}

	cc.registerClient(client)

	log.Info(fmt.Sprintf("New chat WebSocket connection: %s", userId))

	// 고루틴 시작
	go cc.writePump(client)
	go cc.readPump(client)
}

// registerClient 클라이언트 등록
func (cc *ChatController) registerClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	// 기존 연결이 있으면 종료
	if oldClient, exists := cc.clients[client.userID]; exists {
		close(oldClient.send)
		oldClient.conn.Close()
	}

	cc.clients[client.userID] = client
	log.Info(fmt.Sprintf("Chat client registered: %s", client.userID))
}

// unregisterClient 클라이언트 등록 해제
func (cc *ChatController) unregisterClient(client *ChatClient) {
	cc.clientsMu.Lock()
	defer cc.clientsMu.Unlock()

	if _, exists := cc.clients[client.userID]; exists {
		delete(cc.clients, client.userID)
		close(client.send)
		log.Info(fmt.Sprintf("Chat client unregistered: %s", client.userID))
	}
}

// readPump 메시지 읽기
func (cc *ChatController) readPump(client *ChatClient) {
	defer func() {
		cc.unregisterClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(CHAT_PONG_WAIT))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(CHAT_PONG_WAIT))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("Chat WebSocket error:", err)
			}
			break
		}

		// 메시지 처리
		cc.handleMessage(client, message)
	}
}

// writePump 메시지 쓰기
func (cc *ChatController) writePump(client *ChatClient) {
	ticker := time.NewTicker(CHAT_PING_PERIOD)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(CHAT_WRITE_WAIT))
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
			client.conn.SetWriteDeadline(time.Now().Add(CHAT_WRITE_WAIT))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 메시지 처리
func (cc *ChatController) handleMessage(client *ChatClient, message []byte) {
	var msg ptl.ChatMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error("Chat message parse error:", err)
		return
	}

	msg.From = client.userID
	msg.Timestamp = time.Now().Unix()

	switch msg.Type {
	case "text-message":
		cc.handleTextMessage(client, &msg)
	case "typing":
		cc.handleTyping(client, &msg)
	case "read-receipt":
		cc.handleReadReceipt(client, &msg)
	case "call-request", "call-accept", "call-reject":
		// 통화 관련 메시지는 수신자에게 전달만 함
		if msg.Type == "call-accept" {
			// 채팅 수락 시 상대방을 채팅방에 추가
			/* if msg.RoomID != "" {
				if err := cc.rdb.AddUserToChatRoom(msg.RoomID, msg.From); err != nil {
					log.Error(fmt.Sprintf("Failed to add user %s to room %s: %v", msg.From, msg.RoomID, err))
				}
			} */
		}
		cc.sendToUser(msg.To, &msg)
		log.Info(fmt.Sprintf("Call message relayed: %s from %s to %s", msg.Type, msg.From, msg.To))
	default:
		log.Warn(fmt.Sprintf("Unknown chat message type: %s from %s", msg.Type, client.userID))
	}
}

// handleTextMessage 텍스트 메시지 처리
func (cc *ChatController) handleTextMessage(client *ChatClient, msg *ptl.ChatMessage) {
	// Redis에 메시지 저장
	/* if err := cc.rdb.SaveChatMessage(msg.RoomID, msg.From, msg.Content); err != nil {
		log.Error("Failed to save chat message:", err)
		return
	} */

	// 수신자에게 메시지 전송
	cc.sendToUser(msg.To, msg)
	log.Info(fmt.Sprintf("Text message sent: %s -> %s in room %s", msg.From, msg.To, msg.RoomID))
}

// handleTyping 타이핑 상태 처리
func (cc *ChatController) handleTyping(client *ChatClient, msg *ptl.ChatMessage) {
	// 수신자에게 타이핑 상태 전송
	cc.sendToUser(msg.To, msg)
}

// handleReadReceipt 읽음 확인 처리
func (cc *ChatController) handleReadReceipt(client *ChatClient, msg *ptl.ChatMessage) {
	// 발신자에게 읽음 확인 전송
	cc.sendToUser(msg.From, msg)
}

// sendToUser 특정 사용자에게 메시지 전송
func (cc *ChatController) sendToUser(userID string, msg *ptl.ChatMessage) {
	cc.clientsMu.RLock()
	client, exists := cc.clients[userID]
	cc.clientsMu.RUnlock()

	if !exists {
		log.Warn(fmt.Sprintf("User %s not connected to chat WebSocket", userID))
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Error("Message marshal error:", err)
		return
	}

	select {
	case client.send <- data:
	default:
		log.Warn(fmt.Sprintf("Client %s send buffer full", userID))
	}
}

// CreateChatRoom 채팅방 생성
func (cc *ChatController) CreateChatRoom(c *gin.Context) {
	var req struct {
		RoomName  string `json:"roomName"`
		UserID    string `json:"userId"`
		IsPrivate bool   `json:"isPrivate"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// 채팅방 ID 생성
	roomID := utils.GenUuid()

	// Redis에 채팅방 저장
	/* if err := cc.rdb.SetChatRoom(roomID, req.RoomName, req.UserID, req.IsPrivate); err != nil {
		cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to create chat room", err)
		return
	} */

	cc.ctl.SendDataResponse(c, http.StatusOK, gin.H{
		"roomId":   roomID,
		"roomName": req.RoomName,
	})
}

// GetChatRooms 사용자의 채팅방 목록 조회
func (cc *ChatController) GetChatRooms(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "userId required")
		return
	}

	/*
		 	// Redis에서 사용자의 채팅방 목록 조회
			rooms, err := cc.rdb.GetChatRooms()
			if err != nil {
				cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get chat rooms", err)
				return
			}

			// 사용자가 참여한 채팅방만 필터링
			userRooms := make([]models.ChatRoomData, 0)
			for _, room := range rooms {
				for _, participant := range room.Participants {
					if participant == userID {
						userRooms = append(userRooms, room)
						break
					}
				}
			}

			cc.ctl.SendDataResponse(c, http.StatusOK, userRooms)
	*/
}

// GetChatHistory 채팅 기록 조회
func (cc *ChatController) GetChatHistory(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "roomId required")
		return
	}
	/*
		// 페이지네이션 파라미터
		page := c.DefaultQuery("page", "1")
		limit := c.DefaultQuery("limit", "50")

		pageNum, err := strconv.Atoi(page)
		if err != nil {
			pageNum = 1
		}

		limitNum, err := strconv.Atoi(limit)
		if err != nil {
			limitNum = 50
		}

		offset := (pageNum - 1) * limitNum

		// Redis에서 메시지 조회
		messages, err := cc.rdb.GetChatMessages(roomID, offset, limitNum)
		if err != nil {
			cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to get chat history", err)
			return
		}

		cc.ctl.SendDataResponse(c, http.StatusOK, messages)
	*/
}

// SendMessage REST API를 통한 메시지 전송
func (cc *ChatController) SendMessage(c *gin.Context) {
	var req struct {
		RoomID  string `json:"roomId"`
		UserID  string `json:"userId"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		cc.ctl.SimpleError(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	/*
		 	// Redis에 메시지 저장
			if err := cc.rdb.SaveChatMessage(req.RoomID, req.UserID, req.Content); err != nil {
				cc.ctl.SimpleError(c, http.StatusInternalServerError, "Failed to save message", err)
				return
			}

			cc.ctl.SendResponse(c, http.StatusOK, "Message sent")
	*/
}

// SendCallNotification 통화 알림 전송 (SignalingController에서 호출)
func (cc *ChatController) SendCallNotification(msg *ptl.ChatMessage) {
	cc.sendToUser(msg.To, msg)
	log.Info(fmt.Sprintf("Call notification sent: %s -> %s (type: %s)", msg.From, msg.To, msg.Type))
}
