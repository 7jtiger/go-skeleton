# MS-Gateway 화상채팅 예제 모음

MS-Gateway WebRTC 시그널링 서버를 활용한 P2P 화상채팅 예제 및 테스트 클라이언트 모음입니다.

## 📋 목차

- [개요](#개요)
- [구성 파일](#구성-파일)
- [클라이언트 종류](#클라이언트-종류)
- [빠른 시작](#빠른-시작)
- [문서](#문서)
- [시스템 요구사항](#시스템-요구사항)

## 개요

이 디렉토리에는 MS-Gateway의 WebRTC 기능을 테스트하고 데모할 수 있는 다양한 클라이언트와 예제가 포함되어 있습니다.

### 주요 특징

- ✅ **Go 기반 테스트 클라이언트**: 터미널에서 실행 가능
- ✅ **실제 A/V 스트리밍**: 오디오 + 비디오 지원
- ✅ **P2P 통신**: WebRTC를 통한 직접 연결
- ✅ **자동화 스크립트**: 테스트 미디어 생성 및 시나리오 실행

## 구성 파일

### Go 클라이언트

| 파일 | 설명 | 크기 |
|------|------|------|
| `video_chat_client.go` | DataChannel 기반 텍스트 채팅 클라이언트 | 14KB |
| `video_chat_av_client.go` | 오디오/비디오 스트리밍 클라이언트 | 26KB |
| `go.mod` | Go 모듈 정의 및 의존성 | 1.1KB |

### 빌드된 바이너리

| 파일 | 설명 | 크기 |
|------|------|------|
| `video-chat-client` | DataChannel 클라이언트 (빌드됨) | 13MB |
| `video-chat-av` | A/V 스트리밍 클라이언트 (빌드됨) | 13MB |

### 유틸리티 스크립트

| 파일 | 설명 | 용도 |
|------|------|------|
| `generate_test_media.sh` | 테스트 미디어 파일 생성 | VP8/Opus 파일 자동 생성 |
| `test_scenario.sh` | 자동화된 테스트 시나리오 | 다중 클라이언트 테스트 |

### 문서

| 파일 | 설명 | 크기 |
|------|------|------|
| `README.md` | 이 파일 - 전체 개요 | - |
| `README_GO_CLIENT.md` | DataChannel 클라이언트 상세 문서 | 9.1KB |
| `README_AV_CLIENT.md` | A/V 클라이언트 상세 문서 | 11KB |
| `QUICKSTART.md` | DataChannel 빠른 시작 가이드 | 1.3KB |
| `QUICKSTART_AV.md` | A/V 빠른 시작 가이드 | 5.1KB |

## 클라이언트 종류

### 1. DataChannel 클라이언트 (`video-chat-client`)

**용도**: 텍스트 메시지 기반 P2P 통신 테스트

**특징**:
- WebRTC DataChannel 사용
- 터미널 기반 채팅
- 간단한 연결 테스트

**실행**:
```bash
./video-chat-client user1
```

**명령어**:
```bash
call <userID>         # 연결 시작
send <userID> <msg>   # 메시지 전송
end <userID>          # 연결 종료
```

**문서**: `README_GO_CLIENT.md`, `QUICKSTART.md`

---

### 2. A/V 스트리밍 클라이언트 (`video-chat-av`)

**용도**: 실제 오디오/비디오 화상채팅 테스트

**특징**:
- 실제 A/V 스트림 송수신
- VP8 비디오 + Opus 오디오
- IVF/OGG 파일 지원
- 더미 스트림 지원

**실행**:
```bash
# 더미 스트림
./video-chat-av user1

# 미디어 파일 사용
./video-chat-av user1 ws://localhost:8080/webrtc/v01/ws video.ivf audio.ogg
```

**명령어**:
```bash
call <userID>     # 화상 통화 시작
end <userID>      # 통화 종료
stats <userID>    # 통계 확인
```

**문서**: `README_AV_CLIENT.md`, `QUICKSTART_AV.md`

## 빠른 시작

### 기본 설정

#### 1. 서버 시작

```bash
cd /home/jino/go/src/ms-gateway
./ms-gateway
```

#### 2. 클라이언트 빌드

```bash
cd examples

# DataChannel 클라이언트
go build -o video-chat-client video_chat_client.go

# A/V 클라이언트
go build -o video-chat-av video_chat_av_client.go
```

### DataChannel 클라이언트 테스트

```bash
# 터미널 1
./video-chat-client alice

# 터미널 2
./video-chat-client bob

# alice에서
> call bob
> send bob Hello!
```

### A/V 클라이언트 테스트

#### 방법 1: 더미 스트림 (파일 없이)

```bash
# 터미널 1
./video-chat-av alice

# 터미널 2
./video-chat-av bob

# alice에서
> call bob
```

#### 방법 2: 실제 미디어 파일

```bash
# 테스트 미디어 생성
./generate_test_media.sh

# 터미널 1
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \
    test_media/test_video.ivf test_media/test_audio.ogg

# 터미널 2
./video-chat-av bob ws://localhost:8080/webrtc/v01/ws \
    test_media/colorbar_video.ivf test_media/melody_audio.ogg

# alice에서
> call bob
```

## 문서

### 상세 문서

각 클라이언트의 상세한 사용법, API, 문제 해결은 다음 문서를 참조하세요:

- **DataChannel 클라이언트**: `README_GO_CLIENT.md`
  - 아키텍처 설명
  - 명령어 레퍼런스
  - 테스트 시나리오
  - 문제 해결

- **A/V 클라이언트**: `README_AV_CLIENT.md`
  - 미디어 코덱 정보
  - 파일 형식 가이드
  - 성능 최적화
  - FFmpeg 사용법

### 빠른 시작 가이드

5분 안에 시작할 수 있는 간단한 가이드:

- **DataChannel**: `QUICKSTART.md`
- **A/V 스트리밍**: `QUICKSTART_AV.md`

## 시스템 요구사항

### 필수

- **Go**: 1.21 이상
- **OS**: Linux, macOS, Windows
- **네트워크**: UDP 포트 열림 (WebRTC용)

### 선택 (A/V 클라이언트)

- **FFmpeg**: 테스트 미디어 생성용
  ```bash
  # Ubuntu/Debian
  sudo apt-get install ffmpeg
  
  # macOS
  brew install ffmpeg
  ```

### 의존성 패키지

```bash
cd examples
go mod download
```

주요 패키지:
- `github.com/gorilla/websocket` - WebSocket 클라이언트
- `github.com/pion/webrtc/v3` - WebRTC 구현

## 📊 비교표

| 특징 | DataChannel | A/V 스트리밍 |
|------|-------------|--------------|
| 전송 방식 | DataChannel | RTP Media Tracks |
| 데이터 타입 | 텍스트 메시지 | 오디오/비디오 스트림 |
| 코덱 | 없음 | VP8 + Opus |
| 파일 지원 | 없음 | IVF, OGG |
| 파일 크기 | 14KB | 26KB |
| 빌드 크기 | 13MB | 13MB |
| 복잡도 | 낮음 | 중간 |
| 테스트 용도 | 연결 테스트 | 실제 화상채팅 테스트 |

## 🔧 개발 팁

### 디버깅

```bash
# 로그 레벨 증가
export PION_LOG_TRACE=all
./video-chat-av alice
```

### 빌드 옵션

```bash
# 최적화 빌드
go build -ldflags="-s -w" -o video-chat-av video_chat_av_client.go

# 크로스 컴파일
GOOS=linux GOARCH=amd64 go build -o video-chat-av-linux video_chat_av_client.go
GOOS=windows GOARCH=amd64 go build -o video-chat-av.exe video_chat_av_client.go
```

### 테스트 미디어 커스터마이징

```bash
# 다른 해상도
ffmpeg -f lavfi -i testsrc=duration=10:size=1280x720:rate=30 \
    -c:v libvpx -b:v 2M hd_video.ivf

# 다른 FPS
ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=60 \
    -c:v libvpx -b:v 1.5M highfps_video.ivf

# 다른 비트레이트
ffmpeg -f lavfi -i sine=frequency=440:duration=10 \
    -c:a libopus -b:a 96k highquality_audio.ogg
```

## 🔍 문제 해결

### 일반적인 문제

#### 1. 빌드 오류

```bash
# 의존성 다시 다운로드
go mod tidy
go mod download
```

#### 2. WebSocket 연결 실패

```bash
# 서버 확인
curl http://localhost:8080/test

# 서버 재시작
cd /home/jino/go/src/ms-gateway
./ms-gateway
```

#### 3. ICE 연결 실패

- 방화벽에서 UDP 포트 열기
- STUN 서버 연결 확인
- NAT 환경이면 TURN 서버 추가

#### 4. 미디어 파일 오류

```bash
# 파일 형식 확인
ffmpeg -i test_media/video.ivf

# 재생성
./generate_test_media.sh
```

### 상세한 문제 해결

각 클라이언트별 상세한 문제 해결은 해당 README를 참조하세요:
- DataChannel: `README_GO_CLIENT.md` → "문제 해결" 섹션
- A/V: `README_AV_CLIENT.md` → "디버깅" 섹션

## 🚀 다음 단계

### 학습 경로

1. **기초**: DataChannel 클라이언트로 P2P 연결 이해
2. **중급**: A/V 클라이언트로 실제 미디어 스트리밍
3. **고급**: 자체 클라이언트 개발

### 확장 아이디어

- [ ] 화면 공유 기능 추가
- [ ] 그룹 통화 지원
- [ ] 파일 전송 기능
- [ ] 웹캠 직접 캡처
- [ ] GUI 클라이언트 개발

### 참고 자료

- [Pion WebRTC](https://github.com/pion/webrtc)
- [WebRTC 표준](https://www.w3.org/TR/webrtc/)
- [FFmpeg 문서](https://ffmpeg.org/documentation.html)

## 📄 라이선스

이 프로젝트는 MS-Gateway의 일부입니다.

## 👥 기여

문제 발견 시 이슈를 등록해주세요.

---

**최종 업데이트:** 2025-10-20  
**작성자:** MS-Gateway Team

**도움이 필요하신가요?**  
- DataChannel 클라이언트: `QUICKSTART.md`부터 시작하세요
- A/V 클라이언트: `QUICKSTART_AV.md`부터 시작하세요
- 상세 정보: 각 `README_*.md` 파일을 참조하세요

