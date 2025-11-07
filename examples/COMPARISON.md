# 클라이언트 비교표

MS-Gateway 화상채팅 클라이언트 선택 가이드

## 📊 기능 비교

| 특징 | DataChannel 클라이언트 | A/V 스트리밍 클라이언트 |
|------|----------------------|----------------------|
| **파일명** | `video_chat_client.go` | `video_chat_av_client.go` |
| **바이너리** | `video-chat-client` | `video-chat-av` |
| **전송 방식** | DataChannel | RTP Media Tracks |
| **데이터 타입** | 텍스트 메시지 | 오디오/비디오 스트림 |
| **코덱** | 없음 | VP8 (비디오) + Opus (오디오) |
| **파일 지원** | 없음 | IVF (비디오), OGG (오디오) |
| **더미 스트림** | 없음 | 지원 |
| **파일 크기** | 14KB (소스) | 22KB (소스) |
| **빌드 크기** | 13MB | 13MB |
| **복잡도** | 낮음 ⭐ | 중간 ⭐⭐ |
| **학습 곡선** | 쉬움 | 보통 |

## 🎯 사용 사례

### DataChannel 클라이언트

**언제 사용?**
- ✅ WebRTC 연결 테스트
- ✅ 시그널링 서버 검증
- ✅ P2P 연결 확인
- ✅ 간단한 메시지 교환
- ✅ 개발 초기 단계 테스트

**장점**:
- 간단한 구조
- 빠른 실행
- 적은 리소스 사용
- 쉬운 디버깅

**단점**:
- 실제 미디어 스트림 없음
- 제한적인 기능

---

### A/V 스트리밍 클라이언트

**언제 사용?**
- ✅ 실제 화상채팅 테스트
- ✅ 미디어 코덱 검증
- ✅ 성능 테스트
- ✅ 품질 측정
- ✅ 프로덕션 준비 테스트

**장점**:
- 실제 A/V 스트림
- 완전한 WebRTC 기능
- 통계 및 모니터링
- 파일 기반 테스트

**단점**:
- 복잡한 설정
- 더 많은 리소스 필요
- 미디어 파일 필요 (또는 더미)

## 📋 명령어 비교

### DataChannel 클라이언트

```bash
# 실행
./video-chat-client alice

# 명령어
> call bob           # 연결 시작
> send bob Hello!    # 메시지 전송
> end bob            # 연결 종료
> help               # 도움말
> quit               # 종료
```

### A/V 스트리밍 클라이언트

```bash
# 실행 (더미 스트림)
./video-chat-av alice

# 실행 (미디어 파일)
./video-chat-av alice ws://localhost:8080/webrtc/v01/ws video.ivf audio.ogg

# 명령어
> call bob        # 화상 통화 시작
> stats bob       # 통계 확인
> end bob         # 통화 종료
> help            # 도움말
> quit            # 종료
```

## 🔧 준비 사항

### DataChannel 클라이언트

**필수**:
- Go 1.21+
- MS-Gateway 서버

**선택**:
- 없음

**빌드**:
```bash
go build -o video-chat-client video_chat_client.go
```

---

### A/V 스트리밍 클라이언트

**필수**:
- Go 1.21+
- MS-Gateway 서버

**선택** (미디어 파일 사용 시):
- FFmpeg (테스트 미디어 생성)
- IVF 비디오 파일
- OGG 오디오 파일

**빌드**:
```bash
go build -o video-chat-av video_chat_av_client.go
```

**미디어 생성**:
```bash
./generate_test_media.sh
```

## 📈 성능 비교

### DataChannel 클라이언트

| 항목 | 수치 |
|------|------|
| 메모리 사용 | ~20MB |
| CPU 사용 | ~1-2% |
| 네트워크 | ~1KB/s (메시지) |
| 시작 시간 | < 1초 |

### A/V 스트리밍 클라이언트

| 항목 | 수치 |
|------|------|
| 메모리 사용 | ~30-50MB |
| CPU 사용 | ~5-10% |
| 네트워크 | ~1Mbps (A/V) |
| 시작 시간 | 2-3초 |

## 🧪 테스트 시나리오 추천

### 개발 초기 단계
**추천**: DataChannel 클라이언트
```bash
# 빠른 연결 테스트
./video-chat-client user1
./video-chat-client user2
```

### 기능 개발 단계
**추천**: 둘 다
```bash
# 연결 테스트
./video-chat-client test1

# 미디어 스트리밍 테스트
./video-chat-av test2
```

### 프로덕션 준비
**추천**: A/V 스트리밍 클라이언트
```bash
# 실제 미디어로 종합 테스트
./video-chat-av alice ws://server/webrtc/v01/ws video.ivf audio.ogg
```

## 🎓 학습 경로

### 초보자
1. **시작**: DataChannel 클라이언트
   - P2P 연결 이해
   - 시그널링 흐름 파악
   - 기본 WebRTC 개념

2. **다음**: A/V 클라이언트 (더미 스트림)
   - 미디어 트랙 개념
   - 코덱 이해
   - RTCP 피드백

3. **고급**: A/V 클라이언트 (실제 파일)
   - 미디어 인코딩
   - 품질 최적화
   - 성능 튜닝

### 숙련자
- 두 클라이언트를 동시에 사용
- 특정 기능별 선택적 사용
- 커스터마이징 및 확장

## 🔍 선택 가이드

### DataChannel을 선택하세요:
- [ ] WebRTC 처음 시작
- [ ] 빠른 연결 테스트 필요
- [ ] 텍스트 메시지만 충분
- [ ] 간단한 구조 선호
- [ ] 리소스 제약 환경

### A/V를 선택하세요:
- [ ] 실제 화상채팅 구현 예정
- [ ] 미디어 품질 테스트 필요
- [ ] 코덱 검증 필요
- [ ] 성능 측정 필요
- [ ] 프로덕션 환경 준비

### 둘 다 사용하세요:
- [ ] 전체 시스템 테스트
- [ ] 기능별 검증
- [ ] 학습 및 연습

## 📚 관련 문서

### DataChannel
- `README_GO_CLIENT.md` - 상세 문서
- `QUICKSTART.md` - 빠른 시작

### A/V 스트리밍
- `README_AV_CLIENT.md` - 상세 문서
- `QUICKSTART_AV.md` - 빠른 시작

### 전체 개요
- `README.md` - 예제 모음 소개

---

**최종 업데이트:** 2025-10-20
