package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// 벤치마크 설정
const (
	serverURL      = "ws://localhost:3000/ws"
	numClients     = 100
	messagesPerClient = 100
)

// TestConnection 기본 연결 테스트
func TestConnection(t *testing.T) {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// join 메시지 전송
	joinMsg := Message{
		Type: "join",
		Data: map[string]interface{}{
			"name": "TestUser",
			"room": "test-room",
		},
	}

	if err := conn.WriteJSON(joinMsg); err != nil {
		t.Fatalf("Failed to send join message: %v", err)
	}

	t.Log("Successfully connected and joined room")
}

// BenchmarkConcurrentConnections 동시 연결 벤치마크
func BenchmarkConcurrentConnections(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var wg sync.WaitGroup
		errors := make(chan error, numClients)

		for i := 0; i < numClients; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
				if err != nil {
					errors <- fmt.Errorf("Client %d: connection failed: %v", id, err)
					return
				}
				defer conn.Close()

				// join 메시지 전송
				joinMsg := Message{
					Type: "join",
					Data: map[string]interface{}{
						"name": fmt.Sprintf("User%d", id),
						"room": "bench-room",
					},
				}

				if err := conn.WriteJSON(joinMsg); err != nil {
					errors <- fmt.Errorf("Client %d: join failed: %v", id, err)
					return
				}

				// 메시지 수신 대기
				var response Message
				if err := conn.ReadJSON(&response); err != nil {
					// 타임아웃은 정상
					return
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// 에러 수집
		errorCount := 0
		for err := range errors {
			b.Logf("Error: %v", err)
			errorCount++
		}

		if errorCount > numClients/10 { // 10% 이상 실패 시
			b.Fatalf("Too many errors: %d/%d", errorCount, numClients)
		}
	}
}

// BenchmarkMessageThroughput 메시지 처리량 벤치마크
func BenchmarkMessageThroughput(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var wg sync.WaitGroup
		totalMessages := int64(0)
		startTime := time.Now()

		for i := 0; i < numClients; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				// join
				joinMsg := Message{
					Type: "join",
					Data: map[string]interface{}{
						"name": fmt.Sprintf("User%d", id),
						"room": "throughput-room",
					},
				}
				conn.WriteJSON(joinMsg)

				// 채팅 메시지 전송
				for j := 0; j < messagesPerClient; j++ {
					chatMsg := Message{
						Type: "chat",
						Data: map[string]interface{}{
							"message": fmt.Sprintf("Message %d from User %d", j, id),
						},
					}
					
					if err := conn.WriteJSON(chatMsg); err != nil {
						return
					}
					totalMessages++
				}

				time.Sleep(100 * time.Millisecond) // 정리 대기
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		messagesPerSecond := float64(totalMessages) / duration.Seconds()
		b.ReportMetric(messagesPerSecond, "msgs/sec")
	}
}

// BenchmarkRoomBroadcast 방 브로드캐스트 성능 벤치마크
func BenchmarkRoomBroadcast(b *testing.B) {
	roomSize := 50 // 방당 사용자 수

	for n := 0; n < b.N; n++ {
		var wg sync.WaitGroup
		conns := make([]*websocket.Conn, roomSize)

		// 모든 클라이언트 연결
		for i := 0; i < roomSize; i++ {
			conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
			if err != nil {
				b.Fatalf("Failed to connect: %v", err)
			}
			conns[i] = conn

			joinMsg := Message{
				Type: "join",
				Data: map[string]interface{}{
					"name": fmt.Sprintf("User%d", i),
					"room": "broadcast-room",
				},
			}
			conn.WriteJSON(joinMsg)
		}

		// 첫 번째 클라이언트가 메시지 전송
		time.Sleep(500 * time.Millisecond) // 연결 안정화 대기

		startTime := time.Now()
		
		chatMsg := Message{
			Type: "chat",
			Data: map[string]interface{}{
				"message": "Broadcast test message",
			},
		}
		conns[0].WriteJSON(chatMsg)

		// 모든 클라이언트가 메시지 수신했는지 확인
		received := 0
		for i := 1; i < roomSize; i++ { // 발신자 제외
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				conns[idx].SetReadDeadline(time.Now().Add(2 * time.Second))
				var response Message
				if err := conns[idx].ReadJSON(&response); err == nil {
					if response.Type == "chat" {
						received++
					}
				}
			}(i)
		}

		wg.Wait()
		latency := time.Since(startTime)

		// 연결 정리
		for _, conn := range conns {
			conn.Close()
		}

		b.ReportMetric(float64(latency.Milliseconds()), "latency_ms")
		b.Logf("Broadcast latency: %v, Received: %d/%d", latency, received, roomSize-1)
	}
}

// TestMessageOrdering 메시지 순서 보장 테스트
func TestMessageOrdering(t *testing.T) {
	numMessages := 100

	// 두 개의 클라이언트 연결
	conn1, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("Client 1 connection failed: %v", err)
	}
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("Client 2 connection failed: %v", err)
	}
	defer conn2.Close()

	// join
	joinMsg1 := Message{
		Type: "join",
		Data: map[string]interface{}{
			"name": "User1",
			"room": "order-test-room",
		},
	}
	conn1.WriteJSON(joinMsg1)

	time.Sleep(100 * time.Millisecond)

	joinMsg2 := Message{
		Type: "join",
		Data: map[string]interface{}{
			"name": "User2",
			"room": "order-test-room",
		},
	}
	conn2.WriteJSON(joinMsg2)

	time.Sleep(100 * time.Millisecond)

	// User1이 순차적으로 메시지 전송
	go func() {
		for i := 0; i < numMessages; i++ {
			chatMsg := Message{
				Type: "chat",
				Data: map[string]interface{}{
					"message": fmt.Sprintf("Message %d", i),
				},
			}
			conn1.WriteJSON(chatMsg)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// User2가 메시지 수신 및 순서 확인
	receivedMessages := make([]string, 0, numMessages)
	for i := 0; i < numMessages; i++ {
		conn2.SetReadDeadline(time.Now().Add(5 * time.Second))
		var response Message
		if err := conn2.ReadJSON(&response); err != nil {
			t.Logf("Failed to read message %d: %v", i, err)
			continue
		}

		if response.Type == "chat" {
			if msg, ok := response.Data["message"].(string); ok {
				receivedMessages = append(receivedMessages, msg)
			}
		}
	}

	// 순서 확인
	for i, msg := range receivedMessages {
		expected := fmt.Sprintf("Message %d", i)
		if msg != expected {
			t.Errorf("Message order mismatch at index %d: got %s, want %s", i, msg, expected)
		}
	}

	t.Logf("Received %d/%d messages in order", len(receivedMessages), numMessages)
}

// TestMemoryLeak 메모리 누수 테스트
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	iterations := 10
	connectionsPerIteration := 100

	var initialStats, finalStats map[string]interface{}

	// 초기 통계
	initialStats = getServerStats(t)

	for iter := 0; iter < iterations; iter++ {
		var wg sync.WaitGroup

		// 연결 생성 및 해제
		for i := 0; i < connectionsPerIteration; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
				if err != nil {
					return
				}

				joinMsg := Message{
					Type: "join",
					Data: map[string]interface{}{
						"name": fmt.Sprintf("User%d", id),
						"room": fmt.Sprintf("leak-test-%d", id%10),
					},
				}
				conn.WriteJSON(joinMsg)

				// 잠시 대기 후 종료
				time.Sleep(100 * time.Millisecond)
				conn.Close()
			}(i)
		}

		wg.Wait()
		time.Sleep(1 * time.Second) // 정리 시간
		t.Logf("Iteration %d/%d completed", iter+1, iterations)
	}

	// 최종 통계
	time.Sleep(5 * time.Second) // GC 시간
	finalStats = getServerStats(t)

	// 활성 연결 수 확인
	initialConns := int(initialStats["active_connections"].(float64))
	finalConns := int(finalStats["active_connections"].(float64))

	if finalConns > initialConns+10 { // 약간의 여유
		t.Errorf("Possible memory leak: initial=%d, final=%d", initialConns, finalConns)
	} else {
		t.Logf("Memory leak test passed: initial=%d, final=%d", initialConns, finalConns)
	}
}

// getServerStats 서버 통계 조회
func getServerStats(t *testing.T) map[string]interface{} {
	resp, err := http.Get("http://localhost:3000/health")
	if err != nil {
		t.Logf("Failed to get stats: %v", err)
		return make(map[string]interface{})
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)
	return stats
}

