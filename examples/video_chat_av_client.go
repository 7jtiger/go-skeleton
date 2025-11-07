package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

// SignalingMessage 시그널링 메시지 구조
type SignalingMessage struct {
	Type    string      `json:"type"`
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// VideoChatAVClient 오디오/비디오 화상채팅 클라이언트
type VideoChatAVClient struct {
	userID            string
	signalingURL      string
	ws                *websocket.Conn
	peerConnections   map[string]*webrtc.PeerConnection
	peerConnectionsMu sync.RWMutex
	config            webrtc.Configuration

	// 미디어 트랙
	localVideoTrack *webrtc.TrackLocalStaticSample
	localAudioTrack *webrtc.TrackLocalStaticSample

	// 수신된 트랙 저장
	remoteTracks   map[string][]*webrtc.TrackRemote
	remoteTracksMu sync.RWMutex

	// 미디어 소스 파일 (테스트용)
	videoFile string
	audioFile string

	done chan struct{}
}

// NewVideoChatAVClient 클라이언트 생성
func NewVideoChatAVClient(userID, signalingURL string) *VideoChatAVClient {
	return &VideoChatAVClient{
		userID:          userID,
		signalingURL:    signalingURL,
		peerConnections: make(map[string]*webrtc.PeerConnection),
		remoteTracks:    make(map[string][]*webrtc.TrackRemote),
		done:            make(chan struct{}),
		config: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{
						"stun:stun.l.google.com:19302",
						"stun:stun1.l.google.com:19302",
						"stun:stun2.l.google.com:19302",
					},
				},
			},
		},
	}
}

// SetMediaFiles 미디어 파일 설정 (테스트용)
func (c *VideoChatAVClient) SetMediaFiles(videoFile, audioFile string) {
	c.videoFile = videoFile
	c.audioFile = audioFile
}

// Connect 시그널링 서버에 연결
func (c *VideoChatAVClient) Connect() error {
	url := fmt.Sprintf("%s?userId=%s", c.signalingURL, c.userID)
	log.Printf("🔌 시그널링 서버 연결 중: %s", url)

	var err error
	c.ws, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("WebSocket 연결 실패: %v", err)
	}

	log.Printf("✅ 시그널링 서버 연결됨: %s", c.userID)

	// 메시지 수신 고루틴 시작
	go c.readMessages()

	// Ping 고루틴 시작
	go c.pingLoop()

	return nil
}

// Disconnect 연결 종료
func (c *VideoChatAVClient) Disconnect() {
	close(c.done)

	// 모든 PeerConnection 종료
	c.peerConnectionsMu.Lock()
	for userID, pc := range c.peerConnections {
		log.Printf("🔌 PeerConnection 종료: %s", userID)
		pc.Close()
	}
	c.peerConnections = make(map[string]*webrtc.PeerConnection)
	c.peerConnectionsMu.Unlock()

	// WebSocket 종료
	if c.ws != nil {
		c.ws.Close()
	}

	log.Printf("👋 연결 종료됨: %s", c.userID)
}

// InitializeLocalTracks 로컬 미디어 트랙 초기화
func (c *VideoChatAVClient) InitializeLocalTracks() error {
	// 비디오 트랙 생성 (VP8 코덱)
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"video",
		fmt.Sprintf("video-%s", c.userID),
	)
	if err != nil {
		return fmt.Errorf("비디오 트랙 생성 실패: %v", err)
	}
	c.localVideoTrack = videoTrack

	// 오디오 트랙 생성 (Opus 코덱)
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		fmt.Sprintf("audio-%s", c.userID),
	)
	if err != nil {
		return fmt.Errorf("오디오 트랙 생성 실패: %v", err)
	}
	c.localAudioTrack = audioTrack

	log.Printf("📹 로컬 미디어 트랙 초기화 완료")

	// 미디어 파일이 설정되어 있으면 스트리밍 시작
	if c.videoFile != "" {
		go c.streamVideoFromFile()
	} else {
		log.Printf("ℹ️  비디오 파일 미설정 - 더미 프레임 전송")
		go c.streamDummyVideo()
	}

	if c.audioFile != "" {
		go c.streamAudioFromFile()
	} else {
		log.Printf("ℹ️  오디오 파일 미설정 - 더미 오디오 전송")
		go c.streamDummyAudio()
	}

	return nil
}

// streamVideoFromFile 비디오 파일에서 스트리밍 (IVF 형식)
func (c *VideoChatAVClient) streamVideoFromFile() {
	file, err := os.Open(c.videoFile)
	if err != nil {
		log.Printf("❌ 비디오 파일 열기 실패: %v", err)
		return
	}
	defer file.Close()

	ivf, header, err := ivfreader.NewWith(file)
	if err != nil {
		log.Printf("❌ IVF 리더 생성 실패: %v", err)
		return
	}

	log.Printf("📹 비디오 스트리밍 시작: %s (fps: %d)", c.videoFile, header.TimebaseNumerator)

	ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseDenominator) / float32(header.TimebaseNumerator) * 1000)))
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			frame, _, err := ivf.ParseNextFrame()
			if err != nil {
				// 파일 끝나면 처음부터 다시
				file.Seek(0, 0)
				ivf, _, _ = ivfreader.NewWith(file)
				continue
			}

			if err := c.localVideoTrack.WriteSample(media.Sample{
				Data:     frame,
				Duration: time.Second / time.Duration(header.TimebaseNumerator),
			}); err != nil {
				log.Printf("⚠️  비디오 샘플 쓰기 오류: %v", err)
			}
		}
	}
}

// streamAudioFromFile 오디오 파일에서 스트리밍 (OGG 형식)
func (c *VideoChatAVClient) streamAudioFromFile() {
	file, err := os.Open(c.audioFile)
	if err != nil {
		log.Printf("❌ 오디오 파일 열기 실패: %v", err)
		return
	}
	defer file.Close()

	ogg, _, err := oggreader.NewWith(file)
	if err != nil {
		log.Printf("❌ OGG 리더 생성 실패: %v", err)
		return
	}

	log.Printf("🎵 오디오 스트리밍 시작: %s", c.audioFile)

	ticker := time.NewTicker(time.Millisecond * 20) // 20ms 간격
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			pageData, _, err := ogg.ParseNextPage()
			if err != nil {
				// 파일 끝나면 처음부터 다시
				file.Seek(0, 0)
				ogg, _, _ = oggreader.NewWith(file)
				continue
			}

			if err := c.localAudioTrack.WriteSample(media.Sample{
				Data:     pageData,
				Duration: time.Millisecond * 20,
			}); err != nil {
				log.Printf("⚠️  오디오 샘플 쓰기 오류: %v", err)
			}
		}
	}
}

// streamDummyVideo 더미 비디오 프레임 전송 (테스트용)
func (c *VideoChatAVClient) streamDummyVideo() {
	ticker := time.NewTicker(time.Millisecond * 33) // ~30fps
	defer ticker.Stop()

	log.Printf("📹 더미 비디오 스트림 시작 (30fps)")

	dummyFrame := make([]byte, 1200) // 더미 프레임
	for i := range dummyFrame {
		dummyFrame[i] = byte(i % 256)
	}

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if err := c.localVideoTrack.WriteSample(media.Sample{
				Data:     dummyFrame,
				Duration: time.Millisecond * 33,
			}); err != nil && err.Error() != "InvalidStateError" {
				// InvalidStateError는 아직 peer가 없을 때 발생하므로 무시
			}
		}
	}
}

// streamDummyAudio 더미 오디오 전송 (테스트용)
func (c *VideoChatAVClient) streamDummyAudio() {
	ticker := time.NewTicker(time.Millisecond * 20) // 20ms 간격
	defer ticker.Stop()

	log.Printf("🎵 더미 오디오 스트림 시작")

	dummyAudio := make([]byte, 960) // 더미 오디오
	for i := range dummyAudio {
		dummyAudio[i] = byte(i % 256)
	}

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if err := c.localAudioTrack.WriteSample(media.Sample{
				Data:     dummyAudio,
				Duration: time.Millisecond * 20,
			}); err != nil && err.Error() != "InvalidStateError" {
				// InvalidStateError는 무시
			}
		}
	}
}

// readMessages 메시지 수신
func (c *VideoChatAVClient) readMessages() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Panic in readMessages: %v", r)
		}
	}()

	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("❌ WebSocket 읽기 오류: %v", err)
				}
				return
			}

			var msg SignalingMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("❌ 메시지 파싱 오류: %v", err)
				continue
			}

			c.handleMessage(&msg)
		}
	}
}

// pingLoop Ping 메시지 전송
func (c *VideoChatAVClient) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Ping 전송 실패: %v", err)
				return
			}
		}
	}
}

// handleMessage 메시지 처리
func (c *VideoChatAVClient) handleMessage(msg *SignalingMessage) {
	switch msg.Type {
	case "user-list":
		c.handleUserList(msg)
	case "offer":
		c.handleOffer(msg)
	case "answer":
		c.handleAnswer(msg)
	case "ice-candidate":
		c.handleIceCandidate(msg)
	case "hangup":
		c.handleHangup(msg)
	case "error":
		log.Printf("❌ 서버 오류: %v", msg.Payload)
	default:
		log.Printf("⚠️  알 수 없는 메시지 타입: %s", msg.Type)
	}
}

// handleUserList 사용자 목록 처리
func (c *VideoChatAVClient) handleUserList(msg *SignalingMessage) {
	users, ok := msg.Payload.([]interface{})
	if !ok {
		log.Printf("❌ 잘못된 사용자 목록 형식")
		return
	}

	log.Printf("👥 접속 중인 사용자 (%d명):", len(users))
	for i, u := range users {
		if userID, ok := u.(string); ok {
			if userID != c.userID {
				log.Printf("  %d. %s", i+1, userID)
			}
		}
	}
}

// StartCall 통화 시작 (발신)
func (c *VideoChatAVClient) StartCall(targetUserID string) error {
	log.Printf("📞 통화 시작: %s", targetUserID)

	// PeerConnection 생성
	pc, err := c.createPeerConnection(targetUserID)
	if err != nil {
		return fmt.Errorf("PeerConnection 생성 실패: %v", err)
	}

	// 로컬 트랙 추가
	if c.localVideoTrack != nil {
		if _, err := pc.AddTrack(c.localVideoTrack); err != nil {
			return fmt.Errorf("비디오 트랙 추가 실패: %v", err)
		}
		log.Printf("📹 비디오 트랙 추가됨")
	}

	if c.localAudioTrack != nil {
		if _, err := pc.AddTrack(c.localAudioTrack); err != nil {
			return fmt.Errorf("오디오 트랙 추가 실패: %v", err)
		}
		log.Printf("🎵 오디오 트랙 추가됨")
	}

	// Offer 생성
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("Offer 생성 실패: %v", err)
	}

	// Local Description 설정
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("Local Description 설정 실패: %v", err)
	}

	// Offer 전송
	return c.sendSignal(targetUserID, "offer", offer)
}

// handleOffer Offer 처리 (수신)
func (c *VideoChatAVClient) handleOffer(msg *SignalingMessage) {
	log.Printf("📨 Offer 수신: %s", msg.From)

	// Payload를 SDP로 변환
	offerMap, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Printf("❌ 잘못된 Offer 형식")
		return
	}

	sdpStr, _ := offerMap["sdp"].(string)
	typeStr, _ := offerMap["type"].(string)

	offer := webrtc.SessionDescription{
		Type: webrtc.NewSDPType(typeStr),
		SDP:  sdpStr,
	}

	// PeerConnection 생성
	pc, err := c.createPeerConnection(msg.From)
	if err != nil {
		log.Printf("❌ PeerConnection 생성 실패: %v", err)
		return
	}

	// 로컬 트랙 추가
	if c.localVideoTrack != nil {
		if _, err := pc.AddTrack(c.localVideoTrack); err != nil {
			log.Printf("❌ 비디오 트랙 추가 실패: %v", err)
			return
		}
		log.Printf("📹 비디오 트랙 추가됨")
	}

	if c.localAudioTrack != nil {
		if _, err := pc.AddTrack(c.localAudioTrack); err != nil {
			log.Printf("❌ 오디오 트랙 추가 실패: %v", err)
			return
		}
		log.Printf("🎵 오디오 트랙 추가됨")
	}

	// Remote Description 설정
	if err := pc.SetRemoteDescription(offer); err != nil {
		log.Printf("❌ Remote Description 설정 실패: %v", err)
		return
	}

	// Answer 생성
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("❌ Answer 생성 실패: %v", err)
		return
	}

	// Local Description 설정
	if err := pc.SetLocalDescription(answer); err != nil {
		log.Printf("❌ Local Description 설정 실패: %v", err)
		return
	}

	// Answer 전송
	if err := c.sendSignal(msg.From, "answer", answer); err != nil {
		log.Printf("❌ Answer 전송 실패: %v", err)
	}
}

// handleAnswer Answer 처리
func (c *VideoChatAVClient) handleAnswer(msg *SignalingMessage) {
	log.Printf("📨 Answer 수신: %s", msg.From)

	c.peerConnectionsMu.RLock()
	pc, exists := c.peerConnections[msg.From]
	c.peerConnectionsMu.RUnlock()

	if !exists {
		log.Printf("❌ PeerConnection을 찾을 수 없음: %s", msg.From)
		return
	}

	// Payload를 SDP로 변환
	answerMap, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Printf("❌ 잘못된 Answer 형식")
		return
	}

	sdpStr, _ := answerMap["sdp"].(string)
	typeStr, _ := answerMap["type"].(string)

	answer := webrtc.SessionDescription{
		Type: webrtc.NewSDPType(typeStr),
		SDP:  sdpStr,
	}

	// Remote Description 설정
	if err := pc.SetRemoteDescription(answer); err != nil {
		log.Printf("❌ Remote Description 설정 실패: %v", err)
	}
}

// handleIceCandidate ICE Candidate 처리
func (c *VideoChatAVClient) handleIceCandidate(msg *SignalingMessage) {
	c.peerConnectionsMu.RLock()
	pc, exists := c.peerConnections[msg.From]
	c.peerConnectionsMu.RUnlock()

	if !exists {
		log.Printf("❌ PeerConnection을 찾을 수 없음: %s", msg.From)
		return
	}

	// Payload를 ICE Candidate로 변환
	candidateMap, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Printf("❌ 잘못된 ICE Candidate 형식")
		return
	}

	candidateStr, _ := candidateMap["candidate"].(string)
	sdpMid, _ := candidateMap["sdpMid"].(string)
	sdpMLineIndex, _ := candidateMap["sdpMLineIndex"].(float64)

	candidate := webrtc.ICECandidateInit{
		Candidate:     candidateStr,
		SDPMid:        &sdpMid,
		SDPMLineIndex: func() *uint16 { v := uint16(sdpMLineIndex); return &v }(),
	}

	if err := pc.AddICECandidate(candidate); err != nil {
		log.Printf("❌ ICE Candidate 추가 실패: %v", err)
	}
}

// handleHangup 통화 종료 처리
func (c *VideoChatAVClient) handleHangup(msg *SignalingMessage) {
	log.Printf("📴 통화 종료됨: %s", msg.From)
	c.EndCall(msg.From)
}

// EndCall 통화 종료
func (c *VideoChatAVClient) EndCall(targetUserID string) {
	c.peerConnectionsMu.Lock()
	defer c.peerConnectionsMu.Unlock()

	if pc, exists := c.peerConnections[targetUserID]; exists {
		pc.Close()
		delete(c.peerConnections, targetUserID)
		log.Printf("🔌 통화 종료: %s", targetUserID)
	}

	// 종료 신호 전송
	c.sendSignal(targetUserID, "hangup", nil)
}

// createPeerConnection PeerConnection 생성
func (c *VideoChatAVClient) createPeerConnection(targetUserID string) (*webrtc.PeerConnection, error) {
	pc, err := webrtc.NewPeerConnection(c.config)
	if err != nil {
		return nil, err
	}

	// ICE Candidate 이벤트
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			c.sendSignal(targetUserID, "ice-candidate", candidate.ToJSON())
		}
	})

	// 연결 상태 변경 이벤트
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("🔗 연결 상태 (%s): %s", targetUserID, state.String())

		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Printf("✅ P2P 연결 성공: %s", targetUserID)
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateClosed:
			c.EndCall(targetUserID)
		}
	})

	// ICE 연결 상태 변경 이벤트
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("🧊 ICE 상태 (%s): %s", targetUserID, state.String())
	})

	// 원격 트랙 수신 이벤트
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		codec := track.Codec()
		log.Printf("🎬 원격 트랙 수신 (%s): %s [%s]", targetUserID, codec.MimeType, track.ID())

		c.remoteTracksMu.Lock()
		c.remoteTracks[targetUserID] = append(c.remoteTracks[targetUserID], track)
		c.remoteTracksMu.Unlock()

		// RTCP 패킷 전송 (수신 확인)
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			defer ticker.Stop()

			for range ticker.C {
				if pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
					return
				}

				rtcpSendErr := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{
					MediaSSRC: uint32(track.SSRC()),
				}})
				if rtcpSendErr != nil {
					return
				}
			}
		}()

		// 트랙 데이터 읽기 및 저장 (선택적)
		go c.handleRemoteTrack(targetUserID, track)
	})

	c.peerConnectionsMu.Lock()
	c.peerConnections[targetUserID] = pc
	c.peerConnectionsMu.Unlock()

	return pc, nil
}

// handleRemoteTrack 원격 트랙 처리
func (c *VideoChatAVClient) handleRemoteTrack(userID string, track *webrtc.TrackRemote) {
	codec := track.Codec()

	// RTP 패킷 읽기
	buf := make([]byte, 1500)
	packetCount := 0

	for {
		select {
		case <-c.done:
			return
		default:
			_, _, err := track.Read(buf)
			if err != nil {
				return
			}

			packetCount++
			if packetCount%100 == 0 {
				log.Printf("📦 수신 중 (%s %s): %d 패킷", userID, codec.MimeType, packetCount)
			}
		}
	}
}

// sendSignal 시그널링 메시지 전송
func (c *VideoChatAVClient) sendSignal(to, msgType string, payload interface{}) error {
	msg := SignalingMessage{
		Type:    msgType,
		To:      to,
		From:    c.userID,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("메시지 직렬화 실패: %v", err)
	}

	if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("메시지 전송 실패: %v", err)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("사용법: go run video_chat_av_client.go <사용자ID> [WebSocket URL] [비디오파일] [오디오파일]")
		fmt.Println("예제: go run video_chat_av_client.go user1")
		fmt.Println("      go run video_chat_av_client.go user1 ws://localhost:8080/webrtc/v01/ws")
		fmt.Println("      go run video_chat_av_client.go user1 ws://localhost:8080/webrtc/v01/ws video.ivf audio.ogg")
		fmt.Println("")
		fmt.Println("미디어 파일 형식:")
		fmt.Println("  - 비디오: IVF (VP8 코덱)")
		fmt.Println("  - 오디오: OGG (Opus 코덱)")
		fmt.Println("")
		fmt.Println("미디어 파일이 없으면 더미 스트림을 전송합니다.")
		os.Exit(1)
	}

	userID := os.Args[1]
	signalingURL := "ws://localhost:8080/webrtc/v01/ws"
	videoFile := ""
	audioFile := ""

	if len(os.Args) > 2 {
		signalingURL = os.Args[2]
	}
	if len(os.Args) > 3 {
		videoFile = os.Args[3]
	}
	if len(os.Args) > 4 {
		audioFile = os.Args[4]
	}

	// 클라이언트 생성
	client := NewVideoChatAVClient(userID, signalingURL)

	// 미디어 파일 설정
	if videoFile != "" || audioFile != "" {
		client.SetMediaFiles(videoFile, audioFile)
	}

	// 시그널링 서버 연결
	if err := client.Connect(); err != nil {
		log.Fatalf("❌ 연결 실패: %v", err)
	}
	defer client.Disconnect()

	// 로컬 미디어 트랙 초기화
	if err := client.InitializeLocalTracks(); err != nil {
		log.Fatalf("❌ 미디어 트랙 초기화 실패: %v", err)
	}

	// 인터럽트 시그널 처리
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 사용법 출력
	fmt.Println("\n========================================")
	fmt.Println("🎥 MS-Gateway P2P 화상채팅 클라이언트 (A/V)")
	fmt.Println("========================================")
	fmt.Println("명령어:")
	fmt.Println("  call <userID>    - 사용자에게 화상 통화 걸기")
	fmt.Println("  end <userID>     - 통화 종료")
	fmt.Println("  stats <userID>   - 연결 통계 확인")
	fmt.Println("  help             - 도움말 표시")
	fmt.Println("  quit             - 종료")
	fmt.Println("========================================\n")

	// 명령어 입력 처리
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)

			if len(parts) == 0 {
				continue
			}

			cmd := parts[0]
			var arg1 string
			if len(parts) > 1 {
				arg1 = parts[1]
			}

			switch cmd {
			case "call":
				if arg1 == "" {
					fmt.Println("❌ 사용법: call <userID>")
					continue
				}
				if err := client.StartCall(arg1); err != nil {
					fmt.Printf("❌ 통화 시작 실패: %v\n", err)
				} else {
					fmt.Printf("📞 화상 통화 중... %s\n", arg1)
				}

			case "end":
				if arg1 == "" {
					fmt.Println("❌ 사용법: end <userID>")
					continue
				}
				client.EndCall(arg1)
				fmt.Printf("📴 통화 종료: %s\n", arg1)

			case "stats":
				if arg1 == "" {
					fmt.Println("❌ 사용법: stats <userID>")
					continue
				}
				client.peerConnectionsMu.RLock()
				pc, exists := client.peerConnections[arg1]
				client.peerConnectionsMu.RUnlock()

				if !exists {
					fmt.Printf("❌ %s와 연결되어 있지 않습니다\n", arg1)
					continue
				}

				stats := pc.GetStats()
				fmt.Printf("\n📊 연결 통계 (%s):\n", arg1)
				fmt.Printf("  - 연결 상태: %s\n", pc.ConnectionState())
				fmt.Printf("  - ICE 상태: %s\n", pc.ICEConnectionState())
				fmt.Printf("  - 통계 항목: %d개\n\n", len(stats))

			case "help":
				fmt.Println("\n명령어 목록:")
				fmt.Println("  call <userID>    - 사용자에게 화상 통화 걸기")
				fmt.Println("  end <userID>     - 통화 종료")
				fmt.Println("  stats <userID>   - 연결 통계 확인")
				fmt.Println("  help             - 도움말 표시")
				fmt.Println("  quit             - 종료")
				fmt.Println()

			case "quit", "exit":
				interrupt <- os.Interrupt
				return

			default:
				fmt.Printf("❌ 알 수 없는 명령어: %s (help 입력으로 도움말 확인)\n", cmd)
			}

			fmt.Print("> ")
		}
	}()

	fmt.Print("> ")

	// 종료 대기
	<-interrupt
	fmt.Println("\n\n👋 종료 중...")
}
