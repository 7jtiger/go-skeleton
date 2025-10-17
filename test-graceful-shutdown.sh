#!/bin/bash

echo "🧪 TURN 서버 Graceful Shutdown 테스트"
echo "====================================="

# 서버를 백그라운드에서 실행
echo "1️⃣ 서버 시작 중..."
./ms-gateway > server.log 2>&1 &
SERVER_PID=$!

echo "📝 서버 PID: $SERVER_PID"
echo "⏳ 서버 초기화 대기 중... (5초)"
sleep 5

# 서버 상태 확인
echo ""
echo "2️⃣ 서버 상태 확인 중..."
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ HTTP 서버 정상 작동"
else
    echo "❌ HTTP 서버 연결 실패"
fi

# 서버 로그 확인
echo ""
echo "3️⃣ 초기 로그 확인:"
echo "-------------------"
tail -n 10 server.log

# TURN 서버 상태 확인
echo ""
echo "4️⃣ TURN 서버 모니터링 상태 확인 (10초 대기)..."
sleep 10

echo ""
echo "📊 최근 TURN 서버 모니터링 로그:"
echo "------------------------------"
grep "TURN server" server.log | tail -n 5

# Graceful shutdown 테스트
echo ""
echo "5️⃣ Graceful Shutdown 테스트 시작..."
echo "💀 SIGTERM 신호 전송 (PID: $SERVER_PID)"
kill -TERM $SERVER_PID

echo "⏳ 종료 과정 모니터링 (10초)..."
SHUTDOWN_START=$(date +%s)

# 종료 과정 모니터링
while kill -0 $SERVER_PID 2>/dev/null; do
    CURRENT=$(date +%s)
    ELAPSED=$((CURRENT - SHUTDOWN_START))
    
    if [ $ELAPSED -ge 15 ]; then
        echo "⚠️  종료 시간 초과 (15초), 강제 종료 수행"
        kill -KILL $SERVER_PID
        break
    fi
    
    echo "⏳ 종료 진행 중... (${ELAPSED}초 경과)"
    sleep 1
done

echo ""
echo "6️⃣ 종료 과정 로그 분석:"
echo "----------------------"
grep -E "(Shutdown|stopped|shutdown|exiting)" server.log | tail -n 10

echo ""
echo "7️⃣ 최종 결과:"
echo "-------------"
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "✅ 서버가 정상적으로 종료되었습니다!"
    
    # 종료 순서 확인
    echo ""
    echo "📋 종료 순서 검증:"
    echo "- 백그라운드 서비스 중지"
    grep "Stopping background services" server.log && echo "  ✅ 백그라운드 서비스 중지 완료"
    
    echo "- TURN 서버 종료"
    if grep "TURN server is disabled" server.log; then
        echo "  ℹ️  TURN 서버가 비활성화 상태였습니다"
    else
        grep "TURN Server shutdown successfully" server.log && echo "  ✅ TURN 서버 정상 종료"
    fi
    
    echo "- HTTP 서버 종료"
    grep "HTTP Server shutdown successfully" server.log && echo "  ✅ HTTP 서버 정상 종료"
    
    echo "- 모든 서비스 종료"
    grep "All services shutdown successfully" server.log && echo "  ✅ 모든 서비스 정상 종료"
    
else
    echo "❌ 서버가 아직 실행 중입니다. 강제 종료합니다."
    kill -KILL $SERVER_PID
fi

echo ""
echo "📄 전체 로그는 server.log 파일에서 확인할 수 있습니다."
echo "🏁 테스트 완료!"

# 정리
wait $SERVER_PID 2>/dev/null 