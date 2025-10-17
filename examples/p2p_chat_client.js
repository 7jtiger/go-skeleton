/**
 * P2P 채팅 클라이언트 예제
 * WebRTC 데이터 채널을 사용하여 메시지를 서버를 거치지 않고 직접 주고받는 방식
 */

class P2PChatClient {
  constructor(wsUrl) {
    this.wsUrl = wsUrl;
    this.ws = null;
    this.userId = null;
    this.peerConnections = {}; // userId -> RTCPeerConnection
    this.dataChannels = {}; // userId -> RTCDataChannel
    this.messageHandlers = [];
    this.connectionHandlers = [];
    this.lastStatusUpdate = {}; // userId -> timestamp
  }

  /**
   * WebSocket 서버에 연결
   * @param {string} userId 사용자 ID
   */
  connect(userId) {
    this.userId = userId;
    const url = `${this.wsUrl}?userId=${userId}`;
    
    this.ws = new WebSocket(url);
    
    this.ws.onopen = () => {
      console.log('WebSocket 연결됨');
      this._notifyConnectionHandlers('connected', null);
    };
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this._handleServerMessage(message);
    };
    
    this.ws.onclose = () => {
      console.log('WebSocket 연결 종료');
      this._notifyConnectionHandlers('disconnected', null);
      this._cleanupPeerConnections();
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket 오류:', error);
      this._notifyConnectionHandlers('error', error);
    };
    
    // 10초마다 P2P 상태 업데이트
    this.statusUpdateInterval = setInterval(() => {
      this._sendP2PStatusUpdates();
    }, 10000);
  }

  /**
   * WebSocket 연결 종료
   */
  disconnect() {
    if (this.statusUpdateInterval) {
      clearInterval(this.statusUpdateInterval);
    }
    
    this._cleanupPeerConnections();
    
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * 다른 사용자에게 메시지 전송 (P2P)
   * @param {string} toUserId 수신자 ID
   * @param {string} content 메시지 내용
   */
  sendMessage(toUserId, content) {
    // 데이터 채널이 있으면 P2P로 전송
    if (this.dataChannels[toUserId] && this.dataChannels[toUserId].readyState === 'open') {
      const message = {
        type: 'chat',
        from: this.userId,
        to: toUserId,
        content: content,
        timestamp: new Date()
      };
      
      this.dataChannels[toUserId].send(JSON.stringify(message));
      
      // 자신의 메시지 핸들러에도 알림
      this._notifyMessageHandlers(message);
      
      // 마지막 상태 업데이트 시간 기록
      this.lastStatusUpdate[toUserId] = Date.now();
      return true;
    } else {
      // 데이터 채널이 없으면 연결 시도
      console.log(`${toUserId}에게 P2P 연결 시도 중...`);
      this._initiatePeerConnection(toUserId);
      return false;
    }
  }

  /**
   * 메시지 수신 핸들러 등록
   * @param {function} handler (message) => void
   */
  onMessage(handler) {
    this.messageHandlers.push(handler);
  }

  /**
   * 연결 상태 변경 핸들러 등록
   * @param {function} handler (status, data) => void
   */
  onConnectionStatus(handler) {
    this.connectionHandlers.push(handler);
  }

  /**
   * 사용자 연결 초기화 (호출자)
   * @param {string} targetUserId 대상 사용자 ID
   */
  _initiatePeerConnection(targetUserId) {
    if (this.peerConnections[targetUserId]) {
      // 이미 연결 시도 중이면 무시
      return;
    }

    // RTCPeerConnection 객체 생성
    const pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    });
    
    this.peerConnections[targetUserId] = pc;
    
    // 데이터 채널 생성 (호출자)
    const dataChannel = pc.createDataChannel('chat');
    this._setupDataChannel(dataChannel, targetUserId);
    
    // ICE 후보 이벤트 처리
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        this._sendSignal(targetUserId, 'ice-candidate', JSON.stringify(event.candidate));
      }
    };
    
    // 연결 상태 변경 처리
    pc.onconnectionstatechange = () => {
      console.log(`P2P 연결 상태 (${targetUserId}):`, pc.connectionState);
      this._notifyConnectionHandlers('peerConnectionState', { peer: targetUserId, state: pc.connectionState });
      
      if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed' || pc.connectionState === 'closed') {
        this._cleanupPeerConnection(targetUserId);
      }
    };
    
    // Offer 생성 및 전송
    pc.createOffer()
      .then(offer => pc.setLocalDescription(offer))
      .then(() => {
        this._sendSignal(targetUserId, 'offer', JSON.stringify(pc.localDescription));
      })
      .catch(error => {
        console.error('Offer 생성 오류:', error);
        this._cleanupPeerConnection(targetUserId);
      });
  }

  /**
   * 데이터 채널 설정
   * @param {RTCDataChannel} dataChannel 데이터 채널
   * @param {string} peerId 상대방 ID
   */
  _setupDataChannel(dataChannel, peerId) {
    dataChannel.onopen = () => {
      console.log(`${peerId}와의 데이터 채널 열림`);
      this.dataChannels[peerId] = dataChannel;
      
      // 데이터 채널이 준비되었음을 서버에 알림
      this._sendSignal(peerId, 'datachannel-ready', '');
      
      // 연결 상태 핸들러에 알림
      this._notifyConnectionHandlers('dataChannelOpen', { peer: peerId });
    };
    
    dataChannel.onclose = () => {
      console.log(`${peerId}와의 데이터 채널 닫힘`);
      delete this.dataChannels[peerId];
      
      // P2P 연결 해제 상태 서버에 알림
      this._sendP2PStatus(peerId, 'disconnected');
      
      // 연결 상태 핸들러에 알림
      this._notifyConnectionHandlers('dataChannelClose', { peer: peerId });
    };
    
    dataChannel.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        // 마지막 상태 업데이트 시간 기록
        this.lastStatusUpdate[peerId] = Date.now();
        
        // 메시지 핸들러에 알림
        this._notifyMessageHandlers(message);
      } catch (error) {
        console.error('메시지 처리 오류:', error);
      }
    };
  }

  /**
   * 서버에서 수신한 메시지 처리
   * @param {Object} message 서버 메시지
   */
  _handleServerMessage(message) {
    switch (message.type) {
      case 'signal':
        this._handleSignalingMessage(message);
        break;
        
      case 'p2p_active_connections':
        // 서버에서 보낸 P2P 활성 연결 정보 처리
        this._notifyConnectionHandlers('activeConnections', message);
        break;
        
      default:
        // 다른 메시지는 등록된 핸들러에게 전달
        this._notifyMessageHandlers(message);
        break;
    }
  }

  /**
   * WebRTC 시그널링 메시지 처리
   * @param {Object} message 시그널링 메시지
   */
  _handleSignalingMessage(message) {
    const { from, signalType, payload } = message;
    
    if (!from || !signalType) {
      return;
    }
    
    let pc = this.peerConnections[from];
    
    if (signalType === 'offer') {
      // Offer 수신 (피어 연결 생성 - 수신자)
      if (!pc) {
        pc = new RTCPeerConnection({
          iceServers: [
            { urls: 'stun:stun.l.google.com:19302' }
          ]
        });
        
        this.peerConnections[from] = pc;
        
        // ICE 후보 이벤트 처리
        pc.onicecandidate = (event) => {
          if (event.candidate) {
            this._sendSignal(from, 'ice-candidate', JSON.stringify(event.candidate));
          }
        };
        
        // 데이터 채널 수신 처리 (수신자)
        pc.ondatachannel = (event) => {
          this._setupDataChannel(event.channel, from);
        };
        
        // 연결 상태 변경 처리
        pc.onconnectionstatechange = () => {
          console.log(`P2P 연결 상태 (${from}):`, pc.connectionState);
          this._notifyConnectionHandlers('peerConnectionState', { peer: from, state: pc.connectionState });
          
          if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed' || pc.connectionState === 'closed') {
            this._cleanupPeerConnection(from);
          }
        };
      }
      
      const offerDesc = JSON.parse(payload);
      pc.setRemoteDescription(new RTCSessionDescription(offerDesc))
        .then(() => pc.createAnswer())
        .then(answer => pc.setLocalDescription(answer))
        .then(() => {
          this._sendSignal(from, 'answer', JSON.stringify(pc.localDescription));
        })
        .catch(error => {
          console.error('Answer 생성 오류:', error);
          this._cleanupPeerConnection(from);
        });
        
    } else if (signalType === 'answer' && pc) {
      // Answer 수신
      const answerDesc = JSON.parse(payload);
      pc.setRemoteDescription(new RTCSessionDescription(answerDesc))
        .catch(error => {
          console.error('Answer 처리 오류:', error);
          this._cleanupPeerConnection(from);
        });
        
    } else if (signalType === 'ice-candidate' && pc) {
      // ICE 후보 수신
      try {
        const candidate = JSON.parse(payload);
        pc.addIceCandidate(new RTCIceCandidate(candidate))
          .catch(error => {
            console.error('ICE 후보 추가 오류:', error);
          });
      } catch (error) {
        console.error('ICE 후보 처리 오류:', error);
      }
    }
  }

  /**
   * 시그널링 메시지 서버로 전송
   * @param {string} to 수신자 ID
   * @param {string} signalType 시그널 유형
   * @param {string} payload 시그널 데이터
   */
  _sendSignal(to, signalType, payload) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    
    const signal = {
      type: 'signal',
      to: to,
      signalType: signalType,
      payload: payload
    };
    
    this.ws.send(JSON.stringify(signal));
  }

  /**
   * P2P 연결 상태 서버로 전송
   * @param {string} toUserId 연결 대상 ID
   * @param {string} status 상태 ('connected' 또는 'disconnected')
   */
  _sendP2PStatus(toUserId, status) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    
    const statusMsg = {
      type: 'p2p_status',
      toUserId: toUserId,
      status: status
    };
    
    this.ws.send(JSON.stringify(statusMsg));
  }

  /**
   * 모든 활성 P2P 연결 상태를 서버로 전송
   */
  _sendP2PStatusUpdates() {
    const now = Date.now();
    
    // 최근 30초 이내에 메시지를 주고받은 연결만 활성으로 간주
    for (const [peerId, lastTime] of Object.entries(this.lastStatusUpdate)) {
      if (now - lastTime < 30000) {
        this._sendP2PStatus(peerId, 'connected');
      }
    }
  }

  /**
   * P2P 연결 정리
   * @param {string} peerId 상대방 ID
   */
  _cleanupPeerConnection(peerId) {
    const pc = this.peerConnections[peerId];
    if (pc) {
      pc.close();
      delete this.peerConnections[peerId];
    }
    
    if (this.dataChannels[peerId]) {
      this.dataChannels[peerId].close();
      delete this.dataChannels[peerId];
    }
    
    // P2P 연결 해제 상태 서버에 알림
    this._sendP2PStatus(peerId, 'disconnected');
  }

  /**
   * 모든 P2P 연결 정리
   */
  _cleanupPeerConnections() {
    for (const peerId in this.peerConnections) {
      this._cleanupPeerConnection(peerId);
    }
  }

  /**
   * 메시지 핸들러에 알림
   * @param {Object} message 메시지
   */
  _notifyMessageHandlers(message) {
    for (const handler of this.messageHandlers) {
      try {
        handler(message);
      } catch (error) {
        console.error('메시지 핸들러 오류:', error);
      }
    }
  }

  /**
   * 연결 상태 핸들러에 알림
   * @param {string} status 상태
   * @param {Object} data 추가 데이터
   */
  _notifyConnectionHandlers(status, data) {
    for (const handler of this.connectionHandlers) {
      try {
        handler(status, data);
      } catch (error) {
        console.error('연결 상태 핸들러 오류:', error);
      }
    }
  }
}

// 사용 예시
/*
const chatClient = new P2PChatClient('ws://localhost:8080/chat/v01/ws');

// 연결 상태 변경 핸들러
chatClient.onConnectionStatus((status, data) => {
  console.log('연결 상태 변경:', status, data);
});

// 메시지 수신 핸들러
chatClient.onMessage((message) => {
  console.log('메시지 수신:', message);
});

// 연결
chatClient.connect('user123');

// 메시지 전송
chatClient.sendMessage('user456', '안녕하세요!');

// 연결 종료
// chatClient.disconnect();
*/ 