#!/bin/bash

# MS-Gateway P2P 화상채팅 테스트 스크립트
# 이 스크립트는 두 개의 클라이언트를 자동으로 테스트합니다.

echo "========================================="
echo "🎥 MS-Gateway P2P 화상채팅 테스트"
echo "========================================="
echo ""

# 서버 확인
echo "1️⃣  서버 연결 확인 중..."
if ! curl -s http://localhost:8080/test > /dev/null 2>&1; then
    echo "❌ MS-Gateway 서버가 실행되고 있지 않습니다."
    echo "   다음 명령으로 서버를 시작하세요:"
    echo "   cd /home/jino/go/src/ms-gateway && ./ms-gateway"
    exit 1
fi
echo "✅ 서버가 실행 중입니다."
echo ""

# 클라이언트 빌드
echo "2️⃣  클라이언트 빌드 중..."
if [ ! -f "video-chat-client" ]; then
    go build -o video-chat-client video_chat_client.go
    if [ $? -ne 0 ]; then
        echo "❌ 빌드 실패"
        exit 1
    fi
fi
echo "✅ 빌드 완료"
echo ""

# 테스트 안내
echo "3️⃣  테스트 준비 완료"
echo ""
echo "📋 테스트 시나리오:"
echo "   1. 두 개의 터미널을 준비하세요"
echo "   2. 터미널 1: ./video-chat-client user1"
echo "   3. 터미널 2: ./video-chat-client user2"
echo "   4. user1에서 'call user2' 명령 실행"
echo "   5. 연결 후 'send user2 Hello!' 명령으로 메시지 전송"
echo "   6. 'end user2'로 통화 종료"
echo ""
echo "🚀 자동 테스트를 시작하려면 Enter를 누르세요 (Ctrl+C로 취소)"
read

# user1 테스트 (백그라운드)
echo "👤 user1 클라이언트 시작 중..."
./video-chat-client user1 &
USER1_PID=$!
sleep 2

# user2 테스트 (백그라운드)
echo "👤 user2 클라이언트 시작 중..."
./video-chat-client user2 &
USER2_PID=$!
sleep 2

echo ""
echo "✅ 두 클라이언트가 실행 중입니다."
echo "   - user1 PID: $USER1_PID"
echo "   - user2 PID: $USER2_PID"
echo ""
echo "📊 로그를 확인하려면 다음 명령을 사용하세요:"
echo "   tail -f /tmp/video-chat-*.log"
echo ""
echo "🛑 테스트를 종료하려면 Enter를 누르세요"
read

# 클라이언트 종료
echo "🛑 클라이언트 종료 중..."
kill $USER1_PID $USER2_PID 2>/dev/null
echo "✅ 테스트 완료"

