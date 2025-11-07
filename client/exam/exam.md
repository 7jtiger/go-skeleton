# WebRTC 시그널링 서버 및 클라이언트

## 프로젝트 개요

Node.js로 작성된 WebRTC 시그널링 서버를 Golang으로 변환한 프로젝트입니다.

## 파일 목록

### 서버
- **server.go**: Golang으로 작성된 WebRTC 시그널링 서버
  - WebSocket 기반 실시간 통신
  - 방(Room) 기반 사용자 관리
  - Goroutine-safe 구현 (sync.RWMutex 사용)
  - Offer/Answer/ICE Candidate 시그널링 처리
  - **텍스트 채팅 메시지 브로드캐스트 기능**

### 클라이언트
- **client.html**: HTML/JavaScript 웹 클라이언트
  - WebRTC API를 사용한 비디오/오디오 채팅
  - **실시간 텍스트 채팅 기능** (💬)
  - 현대적인 UI/UX 디자인
  - 음소거, 비디오 토글 기능
  - 연결 상태 실시간 표시
  - 채팅 메시지 송수신 및 표시
  - Enter 키로 빠른 전송

### 문서
- **README.md**: 상세한 설치 및 사용 가이드
  - 설치 방법
  - 사용 방법
  - 코드 설명 (초보자용)
  - 문제 해결 가이드

- **go.mod**: Go 모듈 의존성 파일
  - gorilla/websocket v1.5.1

## 주요 변경 사항 (Node.js → Golang)

### 1. Socket.IO → WebSocket
- Node.js의 Socket.IO를 Golang의 gorilla/websocket으로 대체
- 메시지 형식을 JSON으로 통일

### 2. 이벤트 처리
- Socket.IO의 이벤트 기반 → WebSocket의 메시지 타입 기반
- `socket.on('event')` → `switch msg.Type { case "event": ... }`

### 3. 동시성 처리
- JavaScript의 싱글 스레드 → Go의 goroutine
- 데이터 경합 방지를 위한 sync.RWMutex 사용

### 4. 방 관리
- Map 구조는 유사하지만, Go에서는 명시적 동기화 필요
- `rooms = new Map()` → `rooms = make(map[string]*Room)`

## 기술적 특징

### 서버 (server.go)
1. **동시성 안전성**
   - 전역 roomMu로 rooms 맵 보호
   - 각 Room의 mu로 Users 맵 보호
   - RWMutex로 읽기/쓰기 분리

2. **WebSocket 통신**
   - HTTP를 WebSocket으로 업그레이드
   - JSON 메시지 기반 통신
   - CORS 허용

3. **시그널링 로직**
   - join: 방 입장 및 사용자 등록
   - offer: WebRTC offer 중계
   - answer: WebRTC answer 중계
   - ice_candidate: ICE candidate 중계
   - **chat: 텍스트 채팅 메시지 브로드캐스트**

### 클라이언트 (client.html)
1. **WebRTC 구현**
   - RTCPeerConnection API 사용
   - STUN 서버 설정 (구글 공개 서버)
   - Offer/Answer 패턴

2. **미디어 스트림**
   - getUserMedia로 카메라/마이크 접근
   - Track 추가 및 수신
   - 동적 음소거/비디오 토글

3. **UI/UX**
   - 반응형 디자인
   - 상태 표시 (연결됨, 대기 중, 연결 끊김)
   - 직관적인 컨트롤

4. **채팅 기능**
   - 실시간 텍스트 채팅
   - 메시지 구분 (보낸 메시지 / 받은 메시지)
   - 자동 스크롤
   - Enter 키 전송 지원
   - 메시지 애니메이션 효과

## 실행 방법 요약

```bash
# 1. 서버 실행
cd /home/jino/go/src/ms-gateway/client/exam
go run server.go

# 2. 클라이언트 접속 (브라우저 2개)
# - 브라우저에서 client.html 열기
# - 같은 방 이름으로 입장
```

## 테스트 체크리스트

### 기본 기능
- [ ] 서버 실행 (포트 3000)
- [ ] 클라이언트 2개 접속
- [ ] 같은 방 입장 시 자동 연결
- [ ] 비디오/오디오 스트리밍
- [ ] 음소거 기능
- [ ] 비디오 토글 기능
- [ ] 방 나가기 기능
- [ ] 연결 끊김 처리
- [ ] 빈 방 자동 삭제

### 채팅 기능 ✨ (신규)
- [ ] 채팅 메시지 전송
- [ ] 채팅 메시지 수신
- [ ] 보낸 메시지 / 받은 메시지 구분 표시
- [ ] Enter 키로 메시지 전송
- [ ] 자동 스크롤 (최신 메시지로)
- [ ] 메시지 애니메이션 효과
- [ ] 방 나가기 시 채팅 내역 초기화

## 다음 단계 개선 사항

1. **보안**
   - HTTPS/WSS 지원
   - 인증 및 권한 관리

2. **확장성**
   - 3명 이상 그룹 채팅
   - ~~채팅 메시지 기능~~ ✅ **완료**
   - 화면 공유 기능
   - 파일 전송 기능
   - 이모지 지원

3. **프로덕션**
   - TURN 서버 추가
   - 에러 처리 강화
   - 로깅 개선
   - 모니터링 추가

4. **UI/UX**
   - 사용자 목록 표시
   - 연결 품질 표시
   - 녹화 기능
   - 채팅 알림 기능
   - 읽지 않은 메시지 표시

## 참고사항

- 이 프로젝트는 교육 및 학습 목적으로 제작되었습니다.
- 프로덕션 환경에서는 추가적인 보안 및 최적화가 필요합니다.
- 자세한 내용은 README.md를 참조하세요.

