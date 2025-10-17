# ICE 서버와 STUN 서버 구성 가이드

## 개요

WebRTC P2P 연결에서 ICE(Interactive Connectivity Establishment) 서버와 STUN(Session Traversal Utilities for NAT) 서버는 NAT/방화벽 환경에서 클라이언트 간 직접 연결을 가능하게 하는 핵심 구성요소입니다.

## 1. ICE 서버란?

### ICE(Interactive Connectivity Establishment)의 역할
- **NAT 트래버설**: NAT/방화벽 뒤의 클라이언트들이 서로 연결할 수 있도록 지원
- **연결 경로 탐색**: 가능한 모든 연결 경로를 수집하고 최적의 경로 선택
- **Candidate 수집**: 로컬, 서버 반사형, 릴레이형 주소 후보들 수집

### ICE Candidate 유형
1. **Host Candidate**: 로컬 네트워크 인터페이스 주소
2. **Server Reflexive Candidate**: STUN 서버를 통해 얻은 공인 IP 주소
3. **Relay Candidate**: TURN 서버를 통한 릴레이 주소
4. **Peer Reflexive Candidate**: 연결 중 발견된 피어 주소

## 2. STUN 서버 구성

### STUN(Session Traversal Utilities for NAT) 서버 역할
- **공인 IP 발견**: NAT 뒤의 클라이언트가 자신의 공인 IP 주소 확인
- **NAT 타입 감지**: NAT의 종류와 특성 파악
- **포트 매핑 정보**: NAT에서 사용하는 포트 매핑 정보 제공

### Google STUN 서버 장점
- **무료 사용**: Google에서 무료로 제공
- **높은 가용성**: 전 세계 분산된 서버로 안정적 서비스
- **빠른 응답**: 최적화된 인프라로 빠른 STUN 응답

## 3. Google STUN 서버 설정

### 사용 가능한 Google STUN 서버들
```javascript
const iceServers = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
  { urls: 'stun:stun2.l.google.com:19302' },
  { urls: 'stun:stun3.l.google.com:19302' },
  { urls: 'stun:stun4.l.google.com:19302' }
];
```

### WebRTC 설정 예시
```javascript
// RTCPeerConnection 설정
const configuration = {
  iceServers: [
    {
      urls: [
        'stun:stun.l.google.com:19302',
        'stun:stun1.l.google.com:19302'
      ]
    }
  ],
  iceCandidatePoolSize: 10,
  bundlePolicy: 'max-bundle',
  rtcpMuxPolicy: 'require'
};

const peerConnection = new RTCPeerConnection(configuration);
```

## 4. 현재 프로젝트의 ICE 서버 구성

### Go 코드에서의 ICE 서버 설정
```go
// WebRTC 설정 구조체
type WebRTCConfig struct {
    ICEServers      []ICEServer `json:"iceServers"`
    SignalingServer string      `json:"signalingServer"`
    StunServers     []string    `json:"stunServers"`
    MediaSettings   MediaConfig `json:"mediaSettings"`
}

// ICE 서버 정보
type ICEServer struct {
    URLs       []string `json:"urls"`
    Username   string   `json:"username,omitempty"`
    Credential string   `json:"credential,omitempty"`
    Type       string   `json:"type"` // stun, turn
}

// 기본 ICE 서버 구성
func (r *RedisDB) GetWebRTCConfig(userID string) (*WebRTCConfig, error) {
    config := &WebRTCConfig{
        ICEServers: []ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
                Type: "stun",
            },
            {
                URLs: []string{"stun:stun1.l.google.com:19302"},
                Type: "stun",
            },
            {
                URLs: []string{"stun:stun2.l.google.com:19302"},
                Type: "stun",
            },
        },
        SignalingServer: "ws://localhost:8080/ws",
        StunServers: []string{
            "stun:stun.l.google.com:19302",
            "stun:stun1.l.google.com:19302",
        },
    }
    
    return config, nil
}
```

### 설정 파일에서의 ICE 서버 관리
```toml
# config.toml
[webrtc]
signalingServerUrl = "ws://localhost:8080/ws"
stunServers = [
    "stun:stun.l.google.com:19302",
    "stun:stun1.l.google.com:19302",
    "stun:stun2.l.google.com:19302"
]
turnServer = ""
turnUsername = ""
turnPassword = ""
```

## 5. 환경별 ICE 서버 구성 전략

### 개발 환경
```javascript
// 개발용 - 간단한 STUN만 사용
const devConfig = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' }
  ]
};
```

### 프로덕션 환경
```javascript
// 프로덕션 - STUN + TURN 서버 조합
const prodConfig = {
  iceServers: [
    // STUN 서버 (무료)
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
    
    // TURN 서버 (유료 - 대칭 NAT 환경용)
    {
      urls: ['turn:your-turn-server.com:3478'],
      username: 'username',
      credential: 'password'
    }
  ]
};
```

## 6. NAT 환경별 대응 전략

### NAT 타입별 특성
1. **Full Cone NAT**: STUN으로 충분
2. **Restricted Cone NAT**: STUN으로 대부분 해결
3. **Port Restricted Cone NAT**: STUN + 추가 기법 필요
4. **Symmetric NAT**: TURN 서버 필수

### 대응 코드 예시
```javascript
// ICE 연결 상태 모니터링
peerConnection.oniceconnectionstatechange = () => {
  console.log('ICE connection state:', peerConnection.iceConnectionState);
  
  switch(peerConnection.iceConnectionState) {
    case 'checking':
      console.log('ICE candidates를 확인 중...');
      break;
    case 'connected':
      console.log('P2P 연결 성공!');
      break;
    case 'failed':
      console.log('ICE 연결 실패 - TURN 서버 필요할 수 있음');
      handleConnectionFailure();
      break;
  }
};

// ICE candidate 수집 모니터링
peerConnection.onicecandidate = (event) => {
  if (event.candidate) {
    console.log('ICE Candidate:', event.candidate);
    // 시그널링 서버를 통해 상대방에게 전송
    sendCandidateToRemotePeer(event.candidate);
  } else {
    console.log('ICE candidate 수집 완료');
  }
};
```

## 7. STUN 서버 연결 테스트

### 브라우저에서 STUN 테스트
```javascript
async function testStunServer(stunUrl) {
  const configuration = {
    iceServers: [{ urls: stunUrl }]
  };
  
  const pc = new RTCPeerConnection(configuration);
  
  return new Promise((resolve, reject) => {
    let candidateFound = false;
    
    pc.onicecandidate = (event) => {
      if (event.candidate && event.candidate.type === 'srflx') {
        candidateFound = true;
        console.log('STUN 서버 응답:', event.candidate);
        resolve({
          success: true,
          candidate: event.candidate,
          publicIP: event.candidate.address
        });
      }
    };
    
    pc.onicegatheringstatechange = () => {
      if (pc.iceGatheringState === 'complete' && !candidateFound) {
        reject(new Error('STUN 서버에서 응답을 받지 못했습니다'));
      }
    };
    
    // 더미 데이터 채널 생성하여 ICE 수집 시작
    pc.createDataChannel('test');
    pc.createOffer().then(offer => pc.setLocalDescription(offer));
  });
}

// 사용 예시
testStunServer('stun:stun.l.google.com:19302')
  .then(result => console.log('STUN 테스트 성공:', result))
  .catch(error => console.error('STUN 테스트 실패:', error));
```

### Node.js에서 STUN 테스트
```javascript
const dgram = require('dgram');
const crypto = require('crypto');

function testStunServer(host, port = 19302) {
  return new Promise((resolve, reject) => {
    const socket = dgram.createSocket('udp4');
    
    // STUN Binding Request 패킷 생성
    const transactionId = crypto.randomBytes(12);
    const packet = Buffer.concat([
      Buffer.from([0x00, 0x01]), // Message Type: Binding Request
      Buffer.from([0x00, 0x00]), // Message Length
      Buffer.from([0x21, 0x12, 0xA4, 0x42]), // Magic Cookie
      transactionId
    ]);
    
    socket.on('message', (msg, rinfo) => {
      console.log(`STUN 응답 받음: ${rinfo.address}:${rinfo.port}`);
      socket.close();
      resolve({ success: true, response: rinfo });
    });
    
    socket.on('error', (err) => {
      socket.close();
      reject(err);
    });
    
    socket.send(packet, port, host, (err) => {
      if (err) {
        socket.close();
        reject(err);
      }
    });
    
    // 5초 타임아웃
    setTimeout(() => {
      socket.close();
      reject(new Error('STUN 서버 응답 타임아웃'));
    }, 5000);
  });
}
```

## 8. 성능 최적화 및 모니터링

### ICE 수집 최적화
```javascript
const optimizedConfig = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' }
  ],
  iceCandidatePoolSize: 10,        // 미리 수집할 candidate 수
  bundlePolicy: 'max-bundle',      // 미디어 번들링 최적화
  rtcpMuxPolicy: 'require',        // RTCP mux 강제
  iceTransportPolicy: 'all'        // 모든 ICE candidate 사용
};
```

### 연결 통계 모니터링
```javascript
async function getConnectionStats(peerConnection) {
  const stats = await peerConnection.getStats();
  
  stats.forEach(report => {
    if (report.type === 'candidate-pair' && report.state === 'succeeded') {
      console.log('활성 연결 경로:', {
        localCandidate: report.localCandidateId,
        remoteCandidate: report.remoteCandidateId,
        bytesReceived: report.bytesReceived,
        bytesSent: report.bytesSent,
        roundTripTime: report.currentRoundTripTime
      });
    }
  });
}
```

## 9. 트러블슈팅 가이드

### 일반적인 문제들
1. **STUN 서버 연결 실패**
   - 방화벽 설정 확인 (UDP 19302 포트)
   - DNS 해석 문제 확인
   - 대체 STUN 서버 사용

2. **대칭 NAT 환경**
   - TURN 서버 추가 필요
   - ICE-TCP 활용 고려

3. **기업 방화벽 환경**
   - TURN over TLS 사용
   - 화이트리스트 IP 등록 요청

### 디버깅 코드
```javascript
// ICE 연결 진단
function diagnoseIceConnection(peerConnection) {
  const diagnosis = {
    iceConnectionState: peerConnection.iceConnectionState,
    iceGatheringState: peerConnection.iceGatheringState,
    connectionState: peerConnection.connectionState,
    candidates: []
  };
  
  peerConnection.onicecandidate = (event) => {
    if (event.candidate) {
      diagnosis.candidates.push({
        type: event.candidate.type,
        protocol: event.candidate.protocol,
        address: event.candidate.address,
        port: event.candidate.port
      });
    }
  };
  
  return diagnosis;
}
```

## 10. 보안 고려사항

### STUN 서버 보안
- **IP 주소 노출**: STUN을 통해 실제 IP가 노출됨
- **DoS 공격 가능성**: STUN 서버 과부하 주의
- **프라이버시**: 사용자 위치 정보 유추 가능

### 보안 강화 방법
```javascript
// 프라이버시 모드 설정
const privacyConfig = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  iceTransportPolicy: 'relay' // TURN 서버만 사용 (IP 숨김)
};
```

이제 ICE 서버와 STUN 서버를 효과적으로 구성하여 안정적인 WebRTC P2P 연결을 구현할 수 있습니다! 특히 Google STUN 서버는 무료이면서도 안정적이므로 개발 및 테스트 환경에서 매우 유용합니다. 🚀 