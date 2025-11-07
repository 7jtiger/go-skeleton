# 🚀 화상채팅 빠른 시작 가이드 (오디오 + 비디오)

5분 안에 실제 오디오/비디오 P2P 화상채팅을 테스트해보세요!

## 📋 필요 사항

- ✅ Go 1.21 이상
- ✅ FFmpeg (테스트 미디어 생성용)
- ✅ MS-Gateway 서버 실행 중

## 🎬 1단계: 서버 시작

```bash
cd /home/jino/go/src/ms-gateway
./ms-gateway
```

## 🔨 2단계: 클라이언트 빌드

```bash
cd /home/jino/go/src/ms-gateway/examples
go build -o video-chat-av video_chat_av_client.go
```

## 🎥 3단계: 테스트 미디어 생성

### 방법 A: 자동 생성 스크립트 사용 (권장)

```bash
./generate_test_media.sh
```

이 스크립트는 다음을 자동으로 생성합니다:
- `test_media/test_video.ivf` - 테스트 패턴 비디오
- `test_media/test_audio.ogg` - 440Hz 사인파 오디오
- `test_media/colorbar_video.ivf` - 컬러 바 비디오
- `test_media/melody_audio.ogg` - 멜로디 패턴 오디오

### 방법 B: 수동 생성

```bash
mkdir -p test_media

# 비디오 생성 (VP8 IVF, 640x480, 30fps, 10초)
ffmpeg -f lavfi -i testsrc=duration=10:size=640x480:rate=30 \
    -c:v libvpx -b:v 1M test_media/video.ivf

# 오디오 생성 (Opus OGG, 440Hz, 10초)
ffmpeg -f lavfi -i sine=frequency=440:duration=10 \
    -c:a libopus -b:a 48k test_media/audio.ogg
```

### 방법 C: 더미 스트림 (파일 없이 테스트)

미디어 파일 없이도 실행 가능합니다. 클라이언트가 자동으로 더미 데이터를 생성합니다.

## 👤 4단계: 첫 번째 사용자 실행

### 미디어 파일 사용

```bash
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \
    test_media/test_video.ivf test_media/test_audio.ogg
```

### 더미 스트림 사용

```bash
./video-chat-av alice
```

## 👤 5단계: 두 번째 사용자 실행

새 터미널을 열고:

### 미디어 파일 사용

```bash
cd /home/jino/go/src/ms-gateway/examples
./video-chat-av bob ws://localhost:8080/webrtc/v01/ws \
    test_media/colorbar_video.ivf test_media/melody_audio.ogg
```

### 더미 스트림 사용

```bash
./video-chat-av bob
```

## 📞 6단계: 통화 시작

alice 터미널에서:

```bash
> call bob
```

다음과 같은 로그를 확인하세요:

```
📞 통화 시작: bob
📹 비디오 트랙 추가됨
🎵 오디오 트랙 추가됨
🔗 연결 상태 (bob): connecting
🔗 연결 상태 (bob): connected
✅ P2P 연결 성공: bob
🎬 원격 트랙 수신 (bob): video/VP8 [video-bob]
🎬 원격 트랙 수신 (bob): audio/opus [audio-bob]
📦 수신 중 (bob video/VP8): 100 패킷
📦 수신 중 (bob audio/opus): 100 패킷
```

## 📊 7단계: 통계 확인

```bash
> stats bob
```

출력:
```
📊 연결 통계 (bob):
  - 연결 상태: connected
  - ICE 상태: connected
  - 통계 항목: 12개
```

## 📴 8단계: 통화 종료

```bash
> end bob
```

## 🎉 완료!

실제 오디오와 비디오가 P2P로 전송되는 것을 확인했습니다!

---

## 🆚 비교: DataChannel vs A/V Tracks

### DataChannel 클라이언트 (`video-chat-client`)
- **용도**: 텍스트 메시지 테스트
- **명령어**: `send <userID> <message>`
- **전송**: DataChannel (텍스트)

### A/V 클라이언트 (`video-chat-av`)
- **용도**: 실제 화상채팅 테스트
- **명령어**: `call <userID>` (자동으로 A/V 전송)
- **전송**: RTP Media Tracks (오디오/비디오)

---

## 🔧 문제 해결

### FFmpeg가 없는 경우

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

### 연결 안 됨

1. 서버가 실행 중인지 확인
2. 포트 8080이 열려있는지 확인
3. 방화벽 설정 확인

### 트랙 수신 안 됨

1. 로컬 트랙이 초기화되었는지 확인
   - "로컬 미디어 트랙 초기화 완료" 로그
2. P2P 연결이 성공했는지 확인
   - "P2P 연결 성공" 로그
3. ICE 상태가 `connected`인지 확인

---

## 🎬 고급 테스트

### 자신의 비디오/오디오 파일 사용

```bash
# MP4 비디오를 IVF로 변환
ffmpeg -i your_video.mp4 -c:v libvpx -b:v 1M -r 30 my_video.ivf

# MP3 오디오를 OGG로 변환
ffmpeg -i your_audio.mp3 -c:a libopus -b:a 48k my_audio.ogg

# 실행
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws \
    my_video.ivf my_audio.ogg
```

### 웹캠/마이크 직접 캡처

```bash
# Linux (V4L2)
ffmpeg -f v4l2 -i /dev/video0 -c:v libvpx -b:v 1M -r 30 -t 10 webcam.ivf
ffmpeg -f alsa -i default -c:a libopus -b:a 48k -t 10 mic.ogg

# macOS (AVFoundation)
ffmpeg -f avfoundation -i "0:0" -c:v libvpx -b:v 1M -r 30 -t 10 webcam.ivf
ffmpeg -f avfoundation -i ":0" -c:a libopus -b:a 48k -t 10 mic.ogg
```

### 다중 사용자 테스트

```bash
# 터미널 1
./video-chat-av alice

# 터미널 2
./video-chat-av bob

# 터미널 3
./video-chat-av charlie

# alice에서
> call bob
> call charlie
```

---

## 📚 다음 단계

- **상세 문서**: `README_AV_CLIENT.md` 참조
- **일반 클라이언트**: `README_GO_CLIENT.md` 참조
- **웹 클라이언트**: `README.md` 참조

---

**도움이 필요하신가요?**  
README_AV_CLIENT.md의 "문제 해결" 섹션을 확인하세요.

**최종 업데이트:** 2025-10-20

