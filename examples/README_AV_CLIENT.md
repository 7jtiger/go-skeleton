# Go P2P 화상채팅 클라이언트 (오디오 + 비디오)

실제 오디오와 비디오 스트림을 송수신하는 P2P 화상채팅 클라이언트입니다.

## 📋 개요

`video_chat_av_client.go`는 WebRTC를 사용하여 실제 오디오/비디오 스트림을 P2P로 전송하고 수신하는 클라이언트입니다.

### 주요 특징

- ✅ **실제 A/V 스트림**: 오디오(Opus) + 비디오(VP8) 전송
- ✅ **파일 기반 스트리밍**: IVF/OGG 파일에서 미디어 읽기
- ✅ **더미 스트림 지원**: 파일 없이도 테스트 가능
- ✅ **RTCP 지원**: 수신 확인 및 품질 모니터링
- ✅ **원격 트랙 수신**: 상대방의 A/V 스트림 수신
- ✅ **통계 정보**: 연결 상태 및 통계 확인

## 🎬 미디어 코덱

### 비디오
- **코덱**: VP8
- **파일 형식**: IVF (IVF container with VP8)
- **FPS**: 파일 헤더에서 자동 감지

### 오디오
- **코덱**: Opus
- **파일 형식**: OGG (OGG container with Opus)
- **샘플링**: 20ms 간격

## 🚀 빠른 시작

### 1. 빌드

```bash
cd /home/jino/go/src/ms-gateway/examples
go build -o video-chat-av video_chat_av_client.go
```

### 2. 더미 스트림으로 테스트

미디어 파일 없이 더미 데이터로 테스트:

```bash
# 터미널 1
./video-chat-av alice

# 터미널 2
./video-chat-av bob
```

### 3. 실제 미디어 파일로 테스트

IVF/OGG 파일이 있는 경우:

```bash
# 터미널 1
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws video1.ivf audio1.ogg

# 터미널 2
./video-chat-av bob ws://localhost:8080/webrtc/v01/ws video2.ivf audio2.ogg
```

## 📝 사용 방법

### 명령어

#### 1. 화상 통화 걸기

```bash
> call bob
```

오디오와 비디오를 포함한 P2P 통화를 시작합니다.

#### 2. 통화 종료

```bash
> end bob
```

#### 3. 연결 통계 확인

```bash
> stats bob
```

연결 상태와 WebRTC 통계를 표시합니다.

#### 4. 도움말

```bash
> help
```

#### 5. 종료

```bash
> quit
```

## 🎥 미디어 파일 생성

### VP8 비디오 파일 (IVF) 생성

FFmpeg를 사용하여 비디오를 VP8 IVF로 변환:

```bash
# 웹캠에서 캡처
ffmpeg -f v4l2 -i /dev/video0 -c:v libvpx -b:v 1M -r 30 -t 10 output.ivf

# 기존 비디오 파일 변환
ffmpeg -i input.mp4 -c:v libvpx -b:v 1M -r 30 -t 10 output.ivf

# 테스트 패턴 생성
ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=30 -c:v libvpx -b:v 1M output.ivf
```

### Opus 오디오 파일 (OGG) 생성

FFmpeg를 사용하여 오디오를 Opus OGG로 변환:

```bash
# 마이크에서 캡처
ffmpeg -f alsa -i default -c:a libopus -b:a 48k -t 10 output.ogg

# 기존 오디오 파일 변환
ffmpeg -i input.mp3 -c:a libopus -b:a 48k output.ogg

# 테스트 사인파 생성
ffmpeg -f lavfi -i sine=frequency=440:duration=10 -c:a libopus -b:a 48k output.ogg
```

## 📊 로그 출력 예시

```
🔌 시그널링 서버 연결 중: ws://localhost:8080/webrtc/v01/ws?userId=alice
✅ 시그널링 서버 연결됨: alice
📹 로컬 미디어 트랙 초기화 완료
📹 비디오 스트리밍 시작: video.ivf (fps: 30)
🎵 오디오 스트리밍 시작: audio.ogg

========================================
🎥 MS-Gateway P2P 화상채팅 클라이언트 (A/V)
========================================
명령어:
  call <userID>    - 사용자에게 화상 통화 걸기
  end <userID>     - 통화 종료
  stats <userID>   - 연결 통계 확인
  help             - 도움말 표시
  quit             - 종료
========================================

👥 접속 중인 사용자 (2명):
  1. bob

> call bob
📞 통화 시작: bob
📹 비디오 트랙 추가됨
🎵 오디오 트랙 추가됨
🔗 연결 상태 (bob): connecting
🧊 ICE 상태 (bob): checking
🔗 연결 상태 (bob): connected
✅ P2P 연결 성공: bob

> 🎬 원격 트랙 수신 (bob): video/VP8 [video-bob]
🎬 원격 트랙 수신 (bob): audio/opus [audio-bob]
📦 수신 중 (bob video/VP8): 100 패킷
📦 수신 중 (bob audio/opus): 100 패킷

> stats bob

📊 연결 통계 (bob):
  - 연결 상태: connected
  - ICE 상태: connected
  - 통계 항목: 12개

> end bob
📴 통화 종료: bob
```

## 🏗️ 아키텍처

### 컴포넌트

```
VideoChatAVClient
├── WebSocket (시그널링)
├── PeerConnection (WebRTC)
│   ├── 로컬 비디오 트랙 (VP8)
│   ├── 로컬 오디오 트랙 (Opus)
│   ├── 원격 비디오 트랙 수신
│   └── 원격 오디오 트랙 수신
├── 미디어 스트리밍
│   ├── IVF 리더 (비디오)
│   ├── OGG 리더 (오디오)
│   ├── 더미 비디오 생성기
│   └── 더미 오디오 생성기
└── RTCP (품질 모니터링)
```

### 데이터 흐름

```
[IVF 파일] ──┐
             ├──> [VP8 인코더] ──> [RTP] ──┐
[더미 비디오]─┘                            │
                                           ├──> PeerConnection ──> 원격 Peer
[OGG 파일] ──┐                             │
             ├──> [Opus 인코더] ──> [RTP] ─┘
[더미 오디오]─┘

원격 Peer ──> PeerConnection ──> [RTP 수신] ──> [디코딩] ──> [로깅/저장]
```

## 🔧 주요 기능 상세

### 1. 로컬 트랙 초기화

```go
client.InitializeLocalTracks()
```

- 비디오 트랙 생성 (VP8 MimeType)
- 오디오 트랙 생성 (Opus MimeType)
- 스트리밍 고루틴 시작

### 2. 비디오 스트리밍

```go
// IVF 파일에서
streamVideoFromFile()

// 더미 프레임
streamDummyVideo()  // 30fps
```

### 3. 오디오 스트리밍

```go
// OGG 파일에서
streamAudioFromFile()

// 더미 오디오
streamDummyAudio()  // 20ms 간격
```

### 4. 원격 트랙 수신

```go
pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
    // 트랙 정보 로깅
    // RTP 패킷 수신 및 처리
    // RTCP 피드백 전송
})
```

### 5. RTCP 피드백

```go
// 3초마다 Picture Loss Indication 전송
pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{
    MediaSSRC: uint32(track.SSRC()),
}})
```

## 🧪 테스트 시나리오

### 시나리오 1: 더미 스트림 테스트

```bash
# 준비: 미디어 파일 없이 테스트

# 터미널 1
./video-chat-av alice

# 터미널 2
./video-chat-av bob

# alice에서
> call bob

# 확인: 더미 스트림 송수신
# 로그에서 "더미 비디오 스트림 시작" 확인
# 로그에서 "원격 트랙 수신" 확인
```

### 시나리오 2: 실제 미디어 파일 테스트

```bash
# 준비: IVF/OGG 파일 생성
ffmpeg -f lavfi -i testsrc=duration=30:size=640x480:rate=30 -c:v libvpx video.ivf
ffmpeg -f lavfi -i sine=frequency=440:duration=30 -c:a libopus audio.ogg

# 터미널 1
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws video.ivf audio.ogg

# 터미널 2
./video-chat-av bob ws://localhost:8080/webrtc/v01/ws video.ivf audio.ogg

# alice에서
> call bob

# 확인: 실제 파일에서 스트리밍
# 로그에서 "비디오 스트리밍 시작: video.ivf" 확인
```

### 시나리오 3: 연결 통계 확인

```bash
# 통화 중에
> stats bob

# 출력 예시:
📊 연결 통계 (bob):
  - 연결 상태: connected
  - ICE 상태: connected
  - 통계 항목: 12개
```

### 시나리오 4: 다중 사용자

```bash
# 3개 터미널에서
./video-chat-av alice
./video-chat-av bob
./video-chat-av charlie

# alice에서
> call bob
> call charlie

# 두 사용자와 동시에 A/V 스트리밍
```

## 📈 성능 및 리소스

### 리소스 사용량

- **메모리**: ~30-50MB (스트리밍 활성 시)
- **CPU**: 5-10% (VP8 인코딩/디코딩)
- **네트워크**: 
  - 비디오: ~1Mbps (640x480@30fps)
  - 오디오: ~48Kbps

### 최적화 팁

1. **비디오 비트레이트 조정**
   ```bash
   ffmpeg -i input.mp4 -c:v libvpx -b:v 500k output.ivf  # 낮은 품질
   ffmpeg -i input.mp4 -c:v libvpx -b:v 2M output.ivf    # 높은 품질
   ```

2. **해상도 조정**
   ```bash
   ffmpeg -i input.mp4 -c:v libvpx -s 320x240 output.ivf  # 저해상도
   ffmpeg -i input.mp4 -c:v libvpx -s 1280x720 output.ivf # 고해상도
   ```

3. **FPS 조정**
   ```bash
   ffmpeg -i input.mp4 -c:v libvpx -r 15 output.ivf  # 낮은 FPS
   ffmpeg -i input.mp4 -c:v libvpx -r 60 output.ivf  # 높은 FPS
   ```

## 🔍 디버깅

### 연결 문제

**증상**: `ICE 상태: failed`

**해결**:
1. STUN 서버 연결 확인
2. 방화벽 UDP 포트 확인
3. NAT 환경이면 TURN 서버 추가

### 트랙 수신 안 됨

**증상**: "원격 트랙 수신" 로그 없음

**해결**:
1. 로컬 트랙이 추가되었는지 확인
2. SDP 교환이 완료되었는지 확인
3. 연결 상태가 `connected`인지 확인

### 비디오/오디오 파일 오류

**증상**: "IVF 리더 생성 실패"

**해결**:
1. 파일 형식 확인 (IVF for video, OGG for audio)
2. 코덱 확인 (VP8 for video, Opus for audio)
3. FFmpeg로 다시 인코딩

```bash
# 파일 정보 확인
ffmpeg -i video.ivf

# 재인코딩
ffmpeg -i video.ivf -c:v libvpx -b:v 1M output.ivf
```

## ⚙️ 고급 설정

### TURN 서버 추가

```go
config := webrtc.Configuration{
    ICEServers: []webrtc.ICEServer{
        {
            URLs: []string{"stun:stun.l.google.com:19302"},
        },
        {
            URLs:       []string{"turn:turn.example.com:3478"},
            Username:   "username",
            Credential: "password",
        },
    },
}
```

### 비디오 품질 설정

```go
// 비트레이트 제한
pc.SetBandwidth(1_000_000) // 1Mbps
```

### 통계 수집

```go
stats := pc.GetStats()
for _, stat := range stats {
    fmt.Printf("%s: %v\n", stat.Type(), stat)
}
```

## 🆚 비교: DataChannel vs Media Tracks

| 기능 | video_chat_client.go | video_chat_av_client.go |
|------|----------------------|-------------------------|
| 전송 방식 | DataChannel | Media Tracks (RTP) |
| 데이터 타입 | 텍스트 메시지 | 오디오/비디오 스트림 |
| 코덱 | 없음 | VP8 (비디오), Opus (오디오) |
| 파일 지원 | 없음 | IVF, OGG |
| 더미 스트림 | 없음 | 지원 |
| 용도 | 텍스트 채팅 테스트 | 실제 화상채팅 테스트 |

## 📚 참고 자료

### Pion WebRTC 예제
- [Pion Media](https://github.com/pion/webrtc/tree/master/examples)
- [IVF Reader](https://github.com/pion/webrtc/tree/master/pkg/media/ivfreader)
- [OGG Reader](https://github.com/pion/webrtc/tree/master/pkg/media/oggreader)

### FFmpeg 문서
- [VP8 인코딩](https://trac.ffmpeg.org/wiki/Encode/VP8)
- [Opus 인코딩](https://trac.ffmpeg.org/wiki/Encode/HighQualityAudio)

### WebRTC 표준
- [WebRTC 1.0](https://www.w3.org/TR/webrtc/)
- [RTCP 피드백](https://datatracker.ietf.org/doc/html/rfc4585)

## 🔮 향후 개선 사항

- [ ] 실시간 화면 캡처 (스크린 공유)
- [ ] 웹캠/마이크 직접 캡처
- [ ] 수신 스트림 파일로 저장
- [ ] 비디오 인코더 설정 조정
- [ ] Simulcast 지원
- [ ] 품질 통계 UI
- [ ] 네트워크 적응형 비트레이트

## 📄 라이선스

이 프로젝트는 MS-Gateway의 일부입니다.

---

**최종 업데이트:** 2025-10-20  
**작성자:** MS-Gateway Team

