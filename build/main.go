package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

const (
	// WebRTC 미디어 타입 상수
	MimeTypeVP8  = "video/VP8"
	MimeTypeVP9  = "video/VP9"
	MimeTypeH264 = "video/H264"
	MimeTypeOpus = "audio/opus"
)

// WebRTCClient WebRTC 및 WebSocket을 활용한 영상/음성 화상 채팅 클라이언트
type WebRTCClient struct {
	// 기본 클라이언트 정보
	UserID    string
	ServerURL string
	Token     string // JWT 토큰

	// WebSocket 관련
	ws               *websocket.Conn
	wsWriteMutex     sync.Mutex // 웹소켓 동시 쓰기 방지용 뮤텍스
	reconnectEnabled bool

	// WebRTC 관련
	peerConnections      map[string]*webrtc.PeerConnection      // 상대방 UserID -> PeerConnection
	peerConnectionsMutex sync.RWMutex                           // PeerConnection 맵 접근 뮤텍스
	localTracks          map[string]*webrtc.TrackLocalStaticRTP // 로컬 미디어 트랙
	remoteTrackChan      chan *webrtc.TrackRemote               // 원격 트랙 수신 채널

	// 콜백
	onConnectedCallback     func()
	onDisconnectedCallback  func()
	onCallStartCallback     func(remoteUserID string)
	onCallEndCallback       func(remoteUserID string)
	onRemoteTrackCallback   func(track *webrtc.TrackRemote)
	onICEConnectionCallback func(peerID string, state webrtc.ICEConnectionState)
	onErrorCallback         func(err error)

	// 상태
	isConnected bool
	activeCalls map[string]bool // 현재 진행 중인 통화 (userID -> true)
	mutex       sync.RWMutex
}

// NewWebRTCClient 새로운 화상 채팅 클라이언트 생성
func NewWebRTCClient(userID, serverURL string) *WebRTCClient {
	return &WebRTCClient{
		UserID:           userID,
		ServerURL:        serverURL,
		reconnectEnabled: true,
		activeCalls:      make(map[string]bool),
		peerConnections:  make(map[string]*webrtc.PeerConnection),
		localTracks:      make(map[string]*webrtc.TrackLocalStaticRTP),
		remoteTrackChan:  make(chan *webrtc.TrackRemote, 16),
	}
}

// Connect 서버에 WebSocket 연결
func (c *WebRTCClient) Connect() error {
	// JWT 토큰 가져오기 (구현에 따라 다름)
	token := c.Token

	// WebSocket URL 구성
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("URL 파싱 실패: %v", err)
	}

	// 웹소켓 프로토콜로 변경
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	// 웹소켓 엔드포인트 지정
	u.Path = "/rtc/v01/signal"

	// 쿼리 파라미터 추가
	q := u.Query()
	q.Set("userId", c.UserID)
	if token != "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()

	// 헤더 설정
	header := http.Header{}
	if token != "" {
		header.Add("Authorization", "Bearer "+token)
	}

	// WebSocket 연결
	log.Printf("WebSocket 연결 시도: %s\n", u.String())
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("WebSocket 연결 실패: %v", err)
	}

	c.ws = ws
	c.isConnected = true

	// 연결 콜백 호출
	if c.onConnectedCallback != nil {
		c.onConnectedCallback()
	}

	// 메시지 수신 고루틴 시작
	go c.receiveMessages()

	// 핑 메시지 전송 고루틴 시작
	go c.pingLoop()

	// 원격 트랙 처리 고루틴 시작
	go c.handleRemoteTracks()

	return nil
}

// Disconnect 서버 연결 종료
func (c *WebRTCClient) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.reconnectEnabled = false

	// 모든 통화 종료
	for userID := range c.activeCalls {
		c.EndCall(userID)
	}

	// 모든 WebRTC 연결 정리
	c.cleanupAllPeerConnections()

	// WebSocket 연결 종료
	if c.ws != nil {
		c.ws.Close()
	}

	c.isConnected = false

	// 연결 종료 콜백 호출
	if c.onDisconnectedCallback != nil {
		c.onDisconnectedCallback()
	}
}

// IsConnected 연결 상태 확인
func (c *WebRTCClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isConnected
}

// receiveMessages WebSocket 메시지 수신 및 처리
func (c *WebRTCClient) receiveMessages() {
	defer func() {
		c.mutex.Lock()
		c.isConnected = false
		c.mutex.Unlock()

		if c.onDisconnectedCallback != nil {
			c.onDisconnectedCallback()
		}

		// 재연결 시도
		if c.reconnectEnabled {
			go c.reconnect()
		}
	}()

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket 읽기 오류: %v\n", err)
			break
		}

		// 메시지 처리
		c.handleMessage(message)
	}
}

// 메시지 처리
func (c *WebRTCClient) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("메시지 파싱 실패: %v\n", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		log.Println("메시지 타입 없음")
		return
	}

	// 메시지 타입에 따른 처리
	switch msgType {
	case "pong":
		// 서버로부터의 pong 응답

	case "signal":
		// WebRTC 시그널링 메시지 처리
		signalType, _ := msg["signalType"].(string)
		from, _ := msg["from"].(string)
		payload, _ := msg["payload"].(string)

		log.Printf("시그널링 메시지 수신: %s (발신자: %s)\n", signalType, from)

		// 시그널링 메시지 처리
		c.handleSignalingMessage(signalType, from, payload)

	case "user_status":
		// 사용자 상태 변경 메시지 처리
		userID, _ := msg["userId"].(string)
		isConnected, _ := msg["isConnected"].(bool)
		log.Printf("사용자 상태 변경: %s (연결됨: %v)\n", userID, isConnected)

		// 사용자 연결이 끊어진 경우, 해당 사용자와의 통화 종료
		if !isConnected {
			c.mutex.Lock()
			if _, exists := c.activeCalls[userID]; exists {
				c.mutex.Unlock()
				c.EndCall(userID)
			} else {
				c.mutex.Unlock()
			}

			// 연결 정리
			c.cleanupPeerConnection(userID)
		}

	case "call_notification":
		// 통화 관련 알림 처리
		content, _ := msg["content"].(string)
		from, _ := msg["from"].(string)

		if content == "call_start" && c.onCallStartCallback != nil {
			c.onCallStartCallback(from)
		} else if content == "call_end" && c.onCallEndCallback != nil {
			c.onCallEndCallback(from)
		}

	default:
		// 기타 메시지
		log.Printf("알 수 없는 메시지 타입: %s\n", msgType)
	}
}

// WebRTC 시그널링 메시지 처리
func (c *WebRTCClient) handleSignalingMessage(signalType, fromUserID, payload string) {
	if payload == "" {
		log.Printf("빈 페이로드의 시그널링 메시지: %s", signalType)
		return
	}

	switch signalType {
	case "offer":
		// Offer 수신 - 상대방이 연결을 시작
		log.Printf("%s로부터 연결 제안(offer) 수신", fromUserID)

		// SDP Offer 파싱
		var offer webrtc.SessionDescription
		if err := json.Unmarshal([]byte(payload), &offer); err != nil {
			log.Printf("Offer SDP 파싱 실패: %v", err)
			return
		}

		// 새 WebRTC 연결 생성
		pc, err := c.createPeerConnection(fromUserID)
		if err != nil {
			log.Printf("WebRTC 연결 생성 실패: %v", err)
			return
		}

		// 원격 설명(SDP) 설정
		if err := pc.SetRemoteDescription(offer); err != nil {
			log.Printf("원격 설명 설정 실패: %v", err)
			c.cleanupPeerConnection(fromUserID)
			return
		}

		// Answer 생성 및 전송
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			log.Printf("Answer 생성 실패: %v", err)
			c.cleanupPeerConnection(fromUserID)
			return
		}

		if err := pc.SetLocalDescription(answer); err != nil {
			log.Printf("로컬 설명 설정 실패: %v", err)
			c.cleanupPeerConnection(fromUserID)
			return
		}

		// Answer 전송
		answerBytes, _ := json.Marshal(pc.LocalDescription())
		c.sendSignalingMessage(fromUserID, "answer", string(answerBytes))

	case "answer":
		// Answer 수신 - 상대방이 연결을 수락
		log.Printf("%s로부터 연결 응답(answer) 수신", fromUserID)

		// SDP Answer 파싱
		var answer webrtc.SessionDescription
		if err := json.Unmarshal([]byte(payload), &answer); err != nil {
			log.Printf("Answer SDP 파싱 실패: %v", err)
			return
		}

		// PeerConnection 접근
		c.peerConnectionsMutex.RLock()
		pc, exists := c.peerConnections[fromUserID]
		c.peerConnectionsMutex.RUnlock()

		if !exists {
			log.Printf("%s와의 WebRTC 연결이 존재하지 않음", fromUserID)
			return
		}

		// 원격 설명(SDP) 설정
		if err := pc.SetRemoteDescription(answer); err != nil {
			log.Printf("원격 설명 설정 실패: %v", err)
			c.cleanupPeerConnection(fromUserID)
			return
		}

	case "ice-candidate":
		// ICE 후보 수신
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal([]byte(payload), &candidate); err != nil {
			log.Printf("ICE 후보 파싱 실패: %v", err)
			return
		}

		c.peerConnectionsMutex.RLock()
		pc, exists := c.peerConnections[fromUserID]
		c.peerConnectionsMutex.RUnlock()

		if exists {
			// 기존 연결에 ICE 후보 추가
			if err := pc.AddICECandidate(candidate); err != nil {
				log.Printf("ICE 후보 추가 실패: %v", err)
			}
		}
	}
}

// StartCall 사용자에게 화상 통화 시작 요청
func (c *WebRTCClient) StartCall(targetUserID string) error {
	log.Printf("화상 통화 시작 요청: %s\n", targetUserID)

	// 이미 활성화된 통화가 있는지 확인
	c.mutex.RLock()
	_, exists := c.activeCalls[targetUserID]
	c.mutex.RUnlock()

	if exists {
		return fmt.Errorf("이미 해당 사용자와 통화 중입니다")
	}

	// WebRTC 연결 시작
	err := c.initiatePeerConnection(targetUserID)
	if err != nil {
		return fmt.Errorf("WebRTC 연결 시작 실패: %v", err)
	}

	// 통화 시작 신호 전송
	c.sendSignalingMessage(targetUserID, "call-start", nil)

	// 활성 통화 등록
	c.mutex.Lock()
	c.activeCalls[targetUserID] = true
	c.mutex.Unlock()

	return nil
}

// EndCall 통화 종료
func (c *WebRTCClient) EndCall(targetUserID string) error {
	log.Printf("통화 종료: %s\n", targetUserID)

	// 통화 종료 신호 전송
	c.sendSignalingMessage(targetUserID, "call-end", nil)

	// 활성 통화 목록에서 제거
	c.mutex.Lock()
	delete(c.activeCalls, targetUserID)
	c.mutex.Unlock()

	// WebRTC 연결 정리
	c.cleanupPeerConnection(targetUserID)

	return nil
}

// 시그널링 메시지 전송
func (c *WebRTCClient) sendSignalingMessage(targetUserID, signalType string, payload interface{}) error {
	var payloadString string
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("페이로드 직렬화 실패: %v", err)
		}
		payloadString = string(payloadBytes)
	}

	message := map[string]interface{}{
		"type":       "signal",
		"signalType": signalType,
		"to":         targetUserID,
		"payload":    payloadString,
	}

	return c.sendWebSocketMessage(message)
}

// 웹소켓 메시지 전송
func (c *WebRTCClient) sendWebSocketMessage(message interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("WebSocket이 연결되어 있지 않습니다")
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("메시지 직렬화 실패: %v", err)
	}

	c.wsWriteMutex.Lock()
	defer c.wsWriteMutex.Unlock()

	return c.ws.WriteMessage(websocket.TextMessage, messageBytes)
}

// 핑 메시지 전송 루프
func (c *WebRTCClient) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}

			pingMsg := map[string]interface{}{
				"type":      "ping",
				"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
			}

			if err := c.sendWebSocketMessage(pingMsg); err != nil {
				log.Printf("Ping 메시지 전송 실패: %v\n", err)
			}
		}
	}
}

// 서버 재연결
func (c *WebRTCClient) reconnect() {
	if !c.reconnectEnabled {
		return
	}

	delay := 2 * time.Second
	maxDelay := 30 * time.Second

	for {
		log.Printf("%s 후 재연결 시도\n", delay)
		time.Sleep(delay)

		if err := c.Connect(); err != nil {
			log.Printf("재연결 실패: %v\n", err)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		} else {
			log.Println("재연결 성공")
			break
		}
	}
}

// WebRTC PeerConnection 생성 및 설정
func (c *WebRTCClient) createPeerConnection(remoteUserID string) (*webrtc.PeerConnection, error) {
	// 기존 연결이 있으면 제거
	c.cleanupPeerConnection(remoteUserID)

	// WebRTC 설정
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun1.l.google.com:19302"},
			},
			{
				URLs:       []string{"turn:turn.example.com:3478"},
				Username:   "username",
				Credential: "password",
			},
		},
		ICETransportPolicy:   webrtc.ICETransportPolicyAll,
		ICECandidatePoolSize: 10,
	}

	// PeerConnection 생성
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("PeerConnection 생성 실패: %v", err)
	}

	// ICE 후보 생성 이벤트 핸들러
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		// ICE 후보 직렬화 및 전송
		candidateBytes, _ := json.Marshal(candidate.ToJSON())
		c.sendSignalingMessage(remoteUserID, "ice-candidate", string(candidateBytes))
	})

	// ICE 연결 상태 변경 이벤트 핸들러
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("%s와의 ICE 연결 상태 변경: %s", remoteUserID, state.String())

		if state == webrtc.ICEConnectionStateFailed {
			// ICE 실패 시 재시작 요청
			log.Printf("%s와의 ICE 연결 실패, 재협상 요청", remoteUserID)
			c.sendSignalingMessage(remoteUserID, "ice-restart", "")
		}
	})

	// 원격 트랙 수신 이벤트 핸들러
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("원격 미디어 트랙 수신: %s (%s)\n", track.ID(), track.Kind().String())

		// 원격 트랙 채널로 전송
		c.remoteTrackChan <- track
	})

	// ICE 연결 상태 변경 이벤트 핸들러
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("%s와의 ICE 연결 상태 변경: %s", remoteUserID, state.String())

		// 상태 변경 콜백 호출
		if c.onICEConnectionCallback != nil {
			c.onICEConnectionCallback(remoteUserID, state)
		}

		if state == webrtc.ICEConnectionStateDisconnected ||
			state == webrtc.ICEConnectionStateFailed ||
			state == webrtc.ICEConnectionStateClosed {
			// 연결이 종료된 경우 정리
			c.cleanupPeerConnection(remoteUserID)
		}
	})

	// 로컬 트랙 추가
	for _, track := range c.localTracks {
		if _, err := pc.AddTrack(track); err != nil {
			log.Printf("로컬 트랙 추가 실패: %v", err)
		}
	}

	// P2P 연결에 저장
	c.peerConnectionsMutex.Lock()
	c.peerConnections[remoteUserID] = pc
	c.peerConnectionsMutex.Unlock()

	return pc, nil
}

// WebRTC 연결 시작 및 Offer 생성
func (c *WebRTCClient) initiatePeerConnection(remoteUserID string) error {
	// PeerConnection 생성
	pc, err := c.createPeerConnection(remoteUserID)
	if err != nil {
		return err
	}

	// Offer 생성 및 전송
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		c.cleanupPeerConnection(remoteUserID)
		return fmt.Errorf("Offer 생성 실패: %v", err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		c.cleanupPeerConnection(remoteUserID)
		return fmt.Errorf("로컬 설명 설정 실패: %v", err)
	}

	// Offer 전송
	offerBytes, _ := json.Marshal(pc.LocalDescription())
	c.sendSignalingMessage(remoteUserID, "offer", string(offerBytes))

	return nil
}

// AddTrack 로컬 미디어 트랙 추가
func (c *WebRTCClient) AddTrack(trackID string, kind webrtc.RTPCodecType, codec string) (*webrtc.TrackLocalStaticRTP, error) {
	// 트랙 생성
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: mimeTypeForCodec(kind, codec),
	}, trackID, c.UserID)

	if err != nil {
		return nil, fmt.Errorf("트랙 생성 실패: %v", err)
	}

	// 로컬 트랙 저장
	c.localTracks[trackID] = track

	// 기존 모든 연결에 트랙 추가
	c.peerConnectionsMutex.RLock()
	defer c.peerConnectionsMutex.RUnlock()

	for peerID, pc := range c.peerConnections {
		if _, err := pc.AddTrack(track); err != nil {
			log.Printf("%s와의 연결에 트랙 추가 실패: %v", peerID, err)
		}
	}

	return track, nil
}

// 원격 트랙 처리 고루틴
func (c *WebRTCClient) handleRemoteTracks() {
	for {
		select {
		case track := <-c.remoteTrackChan:
			log.Printf("새 원격 트랙 처리: %s (%s)", track.ID(), track.Kind().String())

			// 원격 트랙 콜백 호출
			if c.onRemoteTrackCallback != nil {
				c.onRemoteTrackCallback(track)
			}
		}
	}
}

// WebRTC 연결 정리
func (c *WebRTCClient) cleanupPeerConnection(remoteUserID string) {
	c.peerConnectionsMutex.Lock()
	defer c.peerConnectionsMutex.Unlock()

	// PeerConnection 정리
	if pc, exists := c.peerConnections[remoteUserID]; exists {
		pc.Close()
		delete(c.peerConnections, remoteUserID)
	}
}

// 모든 WebRTC 연결 정리
func (c *WebRTCClient) cleanupAllPeerConnections() {
	c.peerConnectionsMutex.Lock()
	defer c.peerConnectionsMutex.Unlock()

	// 모든 PeerConnection 정리
	for remoteUserID, pc := range c.peerConnections {
		pc.Close()
		delete(c.peerConnections, remoteUserID)
	}
}

// 콜백 설정 메서드들
func (c *WebRTCClient) OnConnected(callback func()) {
	c.onConnectedCallback = callback
}

func (c *WebRTCClient) OnDisconnected(callback func()) {
	c.onDisconnectedCallback = callback
}

func (c *WebRTCClient) OnCallStart(callback func(remoteUserID string)) {
	c.onCallStartCallback = callback
}

func (c *WebRTCClient) OnCallEnd(callback func(remoteUserID string)) {
	c.onCallEndCallback = callback
}

func (c *WebRTCClient) OnRemoteTrack(callback func(track *webrtc.TrackRemote)) {
	c.onRemoteTrackCallback = callback
}

func (c *WebRTCClient) OnICEConnectionStateChange(callback func(peerID string, state webrtc.ICEConnectionState)) {
	c.onICEConnectionCallback = callback
}

func (c *WebRTCClient) OnError(callback func(err error)) {
	c.onErrorCallback = callback
}

// 유틸리티 함수
func mimeTypeForCodec(kind webrtc.RTPCodecType, codec string) string {
	if kind == webrtc.RTPCodecTypeVideo {
		switch strings.ToLower(codec) {
		case "vp8":
			return MimeTypeVP8
		case "vp9":
			return MimeTypeVP9
		case "h264":
			return MimeTypeH264
		default:
			return MimeTypeVP8
		}
	} else if kind == webrtc.RTPCodecTypeAudio {
		return MimeTypeOpus
	}
	return ""
}

// ===== 메인 애플리케이션 =====

var (
	userID     string
	serverURL  string
	targetUser string
)

func init() {
	// 명령행 인자 파싱
	flag.StringVar(&userID, "id", "", "사용자 ID (필수)")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "서버 URL")
	flag.StringVar(&targetUser, "target", "", "통화 대상 사용자 ID (선택사항)")
}

func main() {
	flag.Parse()

	// 필수 인자 확인
	if userID == "" {
		fmt.Println("사용자 ID는 필수입니다. -id 플래그를 사용하세요.")
		flag.PrintDefaults()
		return
	}

	log.Printf("화상 채팅 클라이언트 시작 (사용자 ID: %s)\n", userID)
	log.Printf("서버: %s\n", serverURL)

	// 클라이언트 생성
	client := NewWebRTCClient(userID, serverURL)

	// 콜백 설정
	setupCallbacks(client)

	// 시그널 핸들링 (Ctrl+C 등)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 클라이언트 연결
	log.Println("서버에 연결 중...")
	if err := client.Connect(); err != nil {
		log.Fatalf("서버 연결 실패: %v", err)
	}

	// 로컬 미디어 트랙 추가 (비디오)
	_, err := client.AddTrack("video", webrtc.RTPCodecTypeVideo, "VP8")
	if err != nil {
		log.Printf("비디오 트랙 추가 실패: %v", err)
	} else {
		log.Println("비디오 트랙 추가 성공")
	}

	// 로컬 미디어 트랙 추가 (오디오)
	// _, err := client.AddTrack("audio", webrtc.RTPCodecTypeAudio, "opus")
	if err != nil {
		log.Printf("오디오 트랙 추가 실패: %v", err)
	} else {
		log.Println("오디오 트랙 추가 성공")
	}

	// 대상 사용자가 지정되었으면 자동으로 통화 시작
	if targetUser != "" {
		log.Printf("%s에게 통화 요청 중...\n", targetUser)
		if err := client.StartCall(targetUser); err != nil {
			log.Printf("통화 시작 실패: %v", err)
		}
	}

	// 대화형 프롬프트 고루틴 시작
	stopChan := make(chan struct{})
	go handleUserInput(client, stopChan)

	// 종료 대기
	select {
	case <-sigChan:
		log.Println("종료 신호 수신...")
	case <-stopChan:
		log.Println("사용자 종료 요청...")
	}

	// 연결 종료
	client.Disconnect()
	log.Println("클라이언트 종료됨")
}

// 콜백 설정
func setupCallbacks(client *WebRTCClient) {
	client.OnConnected(func() {
		log.Println("서버에 연결되었습니다")
	})

	client.OnDisconnected(func() {
		log.Println("서버와의 연결이 끊어졌습니다")
	})

	client.OnCallStart(func(remoteUserID string) {
		log.Printf("%s님과 통화가 시작되었습니다", remoteUserID)
	})

	client.OnCallEnd(func(remoteUserID string) {
		log.Printf("%s님과의 통화가 종료되었습니다", remoteUserID)
	})

	client.OnRemoteTrack(func(track *webrtc.TrackRemote) {
		log.Printf("원격 미디어 트랙 수신: %s (%s)", track.ID(), track.Kind().String())
		// 여기서 트랙 처리 (표시, 저장 등)
	})

	client.OnICEConnectionStateChange(func(peerID string, state webrtc.ICEConnectionState) {
		log.Printf("%s와의 연결 상태 변경: %s", peerID, state.String())
	})

	client.OnError(func(err error) {
		log.Printf("오류 발생: %v", err)
	})
}

// 사용자 입력 처리
func handleUserInput(client *WebRTCClient, stop chan<- struct{}) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n----------------------------------------------------------")
	fmt.Println("명령어:")
	fmt.Println("  call <사용자ID> - 사용자에게 화상 통화 요청")
	fmt.Println("  end <사용자ID>  - 현재 통화 종료")
	fmt.Println("  list            - 연결된 피어 목록 표시")
	fmt.Println("  quit            - 프로그램 종료")
	fmt.Println("----------------------------------------------------------\n")

	for scanner.Scan() {
		input := scanner.Text()
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]

		switch cmd {
		case "call":
			if len(parts) < 2 {
				fmt.Println("사용자 ID를 입력하세요: call <사용자ID>")
				continue
			}
			targetID := parts[1]
			fmt.Printf("%s에게 통화 요청 중...\n", targetID)
			if err := client.StartCall(targetID); err != nil {
				fmt.Printf("통화 요청 실패: %v\n", err)
			}

		case "end":
			if len(parts) < 2 {
				fmt.Println("사용자 ID를 입력하세요: end <사용자ID>")
				continue
			}
			targetID := parts[1]
			fmt.Printf("%s와의 통화 종료 중...\n", targetID)
			if err := client.EndCall(targetID); err != nil {
				fmt.Printf("통화 종료 실패: %v\n", err)
			}

		case "list":
			// 연결된 피어 목록 표시
			client.peerConnectionsMutex.RLock()
			if len(client.peerConnections) == 0 {
				fmt.Println("현재 연결된 피어가 없습니다.")
			} else {
				fmt.Println("연결된 피어 목록:")
				for peerID := range client.peerConnections {
					client.mutex.RLock()
					active := client.activeCalls[peerID]
					client.mutex.RUnlock()
					status := "연결됨"
					if active {
						status = "통화 중"
					}
					fmt.Printf("  - %s (%s)\n", peerID, status)
				}
			}
			client.peerConnectionsMutex.RUnlock()

		case "quit":
			fmt.Println("프로그램 종료 중...")
			close(stop)
			return

		default:
			fmt.Println("알 수 없는 명령어입니다. 도움말을 보려면 'help'를 입력하세요.")
		}
	}
}
