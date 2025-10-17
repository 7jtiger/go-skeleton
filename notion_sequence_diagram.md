# 📞 WebRTC P2P 화상채팅 연결 절차

## 🎭 참여자
- 👤 **사용자 A**: 호출을 시작하는 사용자
- 🖥️ **WebSocket 서버**: 시그널링 중계 서버
- 👤 **사용자 B**: 호출을 받는 사용자

---

## 🔄 연결 절차

### 1️⃣ 사전 연결 설정
> **목적**: 양쪽 사용자가 서버에 연결하고 통화 가능한 상태가 됨

- **사용자 A → 서버**
  - 로그인 요청 (`POST /acc/v01/login`)
  - JWT 토큰 수신
  - WebSocket 연결 (`ws://server/ws?token=JWT`)
  
- **사용자 B → 서버**
  - 로그인 요청 (`POST /acc/v01/login`) 
  - JWT 토큰 수신
  - WebSocket 연결 (`ws://server/ws?token=JWT`)

- **서버 → 사용자 A & B**
  - 연결된 사용자 목록 전송
  - WebRTC 설정 정보 전송 (STUN/TURN 서버)
  - ICE 서버 정보 전송

### 2️⃣ WebRTC 호출 시작
> **목적**: A가 B에게 화상통화를 요청하고 B가 응답함

- **사용자 A → 서버**
  ```json
  {
    "type": "signal",
    "signalType": "offer",
    "from": "userA_id",
    "to": "userB_id", 
    "payload": "SDP_OFFER_STRING"
  }
  ```

- **서버 → 사용자 B**
  - A의 offer 시그널 전달
  - 호출 알림 표시

- **사용자 B → 서버**
  ```json
  {
    "type": "signal",
    "signalType": "answer",
    "from": "userB_id",
    "to": "userA_id",
    "payload": "SDP_ANSWER_STRING"
  }
  ```

- **서버 → 사용자 A**
  - B의 answer 시그널 전달
  - 통화 수락 확인

### 3️⃣ ICE 후보 교환
> **목적**: 최적의 네트워크 연결 경로를 찾기 위한 정보 교환

- **양방향 ICE Candidate 교환**
  ```json
  {
    "type": "signal",
    "signalType": "ice-candidate",
    "from": "sender_id",
    "to": "receiver_id",
    "payload": "ICE_CANDIDATE_STRING"
  }
  ```

- **서버의 역할**
  - ICE candidate 메시지 중계
  - 연결 상태 모니터링
  - TURN 서버 대역폭 사용량 기록

### 4️⃣ P2P 연결 완료
> **목적**: 서버를 거치지 않는 직접 연결 확립

- **데이터 채널 설정 완료**
  ```json
  {
    "type": "signal", 
    "signalType": "datachannel-ready",
    "from": "user_id",
    "to": "peer_id"
  }
  ```

- **직접 P2P 통신 시작**
  - 🎥 영상 스트림: 사용자 A ←→ 사용자 B
  - 🎤 음성 스트림: 사용자 A ←→ 사용자 B  
  - 💬 채팅 메시지: 사용자 A ←→ 사용자 B
  - ⚡ **서버 우회**: 이후 모든 미디어 데이터는 서버를 거치지 않음

---

## 🚨 예외 상황 처리

### 📴 사용자가 오프라인인 경우
- 서버가 시그널링 메시지를 임시 저장
- 발신자에게 "대상 사용자가 오프라인" 알림
- 수신자가 온라인 상태가 되면 저장된 메시지 전달

### ❌ ICE 연결 실패
- TURN 서버를 통한 릴레이 연결 시도
- 연결 실패 로그 기록
- 사용자에게 네트워크 상태 확인 안내

### 🔄 재연결 처리
- WebSocket 연결 끊김 시 자동 재연결
- P2P 연결 상태 모니터링
- 하트비트를 통한 연결 상태 확인 