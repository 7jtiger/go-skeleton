package controller

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// STUNTestResult STUN 서버 테스트 결과
type STUNTestResult struct {
	Success    bool   `json:"success"`
	ServerURL  string `json:"serverUrl"`
	PublicIP   string `json:"publicIp,omitempty"`
	PublicPort int    `json:"publicPort,omitempty"`
	// ResponseTime time.Duration `json:"responseTime"`
	ResponseTime int64  `json:"responseTime"`
	Error        string `json:"error,omitempty"`
}

// ICEConnectivityTest ICE 연결성 테스트 결과
type ICEConnectivityTest struct {
	STUNServers []STUNTestResult `json:"stunServers"`
	Summary     struct {
		TotalTested int `json:"totalTested"`
		Successful  int `json:"successful"`
		Failed      int `json:"failed"`
	} `json:"summary"`
}

// TestStunServer STUN 서버 연결 테스트
func TestStunServer(stunURL string, timeout time.Duration) STUNTestResult {
	result := STUNTestResult{
		ServerURL: stunURL,
		Success:   false,
	}

	start := time.Now()
	defer func() {
		result.ResponseTime = time.Since(start).Milliseconds()
	}()

	// STUN URL 파싱 (stun:host:port)
	host, port, err := parseStunURL(stunURL)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid STUN URL: %v", err)
		return result
	}

	// UDP 연결 생성
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer conn.Close()

	// STUN Binding Request 패킷 생성
	packet, transactionID, err := createStunBindingRequest()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create STUN packet: %v", err)
		return result
	}

	// 패킷 전송
	_, err = conn.Write(packet)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to send packet: %v", err)
		return result
	}

	// 응답 대기
	conn.SetReadDeadline(time.Now().Add(timeout))
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	// 응답 파싱
	publicIP, publicPort, err := parseStunResponse(response[:n], transactionID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return result
	}

	result.Success = true
	result.PublicIP = publicIP
	result.PublicPort = publicPort

	return result
}

// TestMultipleStunServers 여러 STUN 서버 동시 테스트
func TestMultipleStunServers(stunURLs []string, timeout time.Duration) ICEConnectivityTest {
	test := ICEConnectivityTest{
		STUNServers: make([]STUNTestResult, len(stunURLs)),
	}

	// 동시 테스트를 위한 채널
	resultChan := make(chan struct {
		index  int
		result STUNTestResult
	}, len(stunURLs))

	// 각 STUN 서버를 고루틴으로 테스트
	for i, stunURL := range stunURLs {
		go func(index int, url string) {
			result := TestStunServer(url, timeout)
			resultChan <- struct {
				index  int
				result STUNTestResult
			}{index, result}
		}(i, stunURL)
	}

	// 결과 수집
	for i := 0; i < len(stunURLs); i++ {
		res := <-resultChan
		test.STUNServers[res.index] = res.result

		if res.result.Success {
			test.Summary.Successful++
		} else {
			test.Summary.Failed++
		}
	}

	test.Summary.TotalTested = len(stunURLs)
	return test
}

// parseStunURL STUN URL 파싱 (stun:host:port)
func parseStunURL(stunURL string) (host string, port int, err error) {
	// stun: prefix 제거
	if len(stunURL) < 5 || stunURL[:5] != "stun:" {
		return "", 0, fmt.Errorf("invalid stun URL format")
	}

	hostPort := stunURL[5:]
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", 0, err
	}

	port = 19302 // 기본 STUN 포트
	if portStr != "" {
		_, err = fmt.Sscanf(portStr, "%d", &port)
		if err != nil {
			return "", 0, err
		}
	}

	return host, port, nil
}

// createStunBindingRequest STUN Binding Request 패킷 생성
func createStunBindingRequest() ([]byte, []byte, error) {
	// Transaction ID 생성 (12바이트)
	transactionID := make([]byte, 12)
	_, err := rand.Read(transactionID)
	if err != nil {
		return nil, nil, err
	}

	// STUN 헤더 구성
	packet := make([]byte, 20)

	// Message Type (Binding Request = 0x0001)
	binary.BigEndian.PutUint16(packet[0:2], 0x0001)

	// Message Length (헤더 제외한 길이, 현재는 0)
	binary.BigEndian.PutUint16(packet[2:4], 0x0000)

	// Magic Cookie (RFC 5389)
	binary.BigEndian.PutUint32(packet[4:8], 0x2112A442)

	// Transaction ID
	copy(packet[8:20], transactionID)

	return packet, transactionID, nil
}

// parseStunResponse STUN 응답 파싱
func parseStunResponse(response []byte, expectedTransactionID []byte) (publicIP string, publicPort int, err error) {
	if len(response) < 20 {
		return "", 0, fmt.Errorf("response too short")
	}

	// Transaction ID 검증
	responseTransactionID := response[8:20]
	for i := 0; i < 12; i++ {
		if responseTransactionID[i] != expectedTransactionID[i] {
			return "", 0, fmt.Errorf("transaction ID mismatch")
		}
	}

	// Message Type 확인 (Binding Success Response = 0x0101)
	messageType := binary.BigEndian.Uint16(response[0:2])
	if messageType != 0x0101 {
		return "", 0, fmt.Errorf("not a binding success response")
	}

	// Message Length
	messageLength := binary.BigEndian.Uint16(response[2:4])

	// Attributes 파싱
	offset := 20
	for offset < int(messageLength)+20 {
		if offset+4 > len(response) {
			break
		}

		attrType := binary.BigEndian.Uint16(response[offset : offset+2])
		attrLength := binary.BigEndian.Uint16(response[offset+2 : offset+4])

		if offset+4+int(attrLength) > len(response) {
			break
		}

		// MAPPED-ADDRESS (0x0001) 또는 XOR-MAPPED-ADDRESS (0x0020)
		if attrType == 0x0001 || attrType == 0x0020 {
			attrData := response[offset+4 : offset+4+int(attrLength)]

			if len(attrData) >= 8 {
				family := binary.BigEndian.Uint16(attrData[0:2])
				port := binary.BigEndian.Uint16(attrData[2:4])

				if family == 0x01 { // IPv4
					ip := net.IPv4(attrData[4], attrData[5], attrData[6], attrData[7])

					// XOR-MAPPED-ADDRESS인 경우 XOR 디코딩
					if attrType == 0x0020 {
						port ^= 0x2112 // Magic Cookie 상위 16비트와 XOR
						ip[0] ^= 0x21
						ip[1] ^= 0x12
						ip[2] ^= 0xA4
						ip[3] ^= 0x42
					}

					return ip.String(), int(port), nil
				}
			}
		}

		// 다음 attribute로 이동 (4바이트 정렬)
		paddedLength := (int(attrLength) + 3) & ^3
		offset += 4 + paddedLength
	}

	return "", 0, fmt.Errorf("no mapped address found in response")
}

// GetRecommendedStunServers 권장 STUN 서버 목록 반환
func GetRecommendedStunServers() []string {
	return []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
		"stun:stun3.l.google.com:19302",
		"stun:stun4.l.google.com:19302",
	}
}

// ValidateWebRTCConfig WebRTC 설정 유효성 검증
func ValidateWebRTCConfig(config map[string]interface{}) error {
	// ICE 서버 설정 검증
	if iceServers, ok := config["iceServers"].([]interface{}); ok {
		if len(iceServers) == 0 {
			return fmt.Errorf("at least one ICE server must be configured")
		}

		for i, server := range iceServers {
			serverMap, ok := server.(map[string]interface{})
			if !ok {
				return fmt.Errorf("ice server %d: invalid format", i)
			}

			urls, ok := serverMap["urls"].([]interface{})
			if !ok || len(urls) == 0 {
				return fmt.Errorf("ice server %d: urls field is required", i)
			}

			// URL 형식 검증
			for j, url := range urls {
				urlStr, ok := url.(string)
				if !ok {
					return fmt.Errorf("ice server %d, url %d: must be string", i, j)
				}

				if len(urlStr) < 5 {
					return fmt.Errorf("ice server %d, url %d: invalid format", i, j)
				}

				prefix := urlStr[:4]
				if prefix != "stun" && prefix != "turn" {
					return fmt.Errorf("ice server %d, url %d: must start with stun: or turn:", i, j)
				}
			}
		}
	} else {
		return fmt.Errorf("iceServers field is required")
	}

	return nil
}

// CreateStunTestReport STUN 테스트 보고서 생성
func CreateStunTestReport(ctx context.Context, stunServers []string) (map[string]interface{}, error) {
	timeout := 5 * time.Second

	// 테스트 실행
	testResult := TestMultipleStunServers(stunServers, timeout)

	// 보고서 생성
	report := map[string]interface{}{
		"timestamp": time.Now(),
		"summary": map[string]interface{}{
			"totalTested": testResult.Summary.TotalTested,
			"successful":  testResult.Summary.Successful,
			"failed":      testResult.Summary.Failed,
			"successRate": float64(testResult.Summary.Successful) / float64(testResult.Summary.TotalTested) * 100,
		},
		"details": make([]map[string]interface{}, len(testResult.STUNServers)),
	}

	// 상세 결과
	for i, result := range testResult.STUNServers {
		detail := map[string]interface{}{
			"serverUrl":    result.ServerURL,
			"success":      result.Success,
			"responseTime": result.ResponseTime,
		}

		if result.Success {
			detail["publicIp"] = result.PublicIP
			detail["publicPort"] = result.PublicPort
		} else {
			detail["error"] = result.Error
		}

		report["details"].([]map[string]interface{})[i] = detail
	}

	return report, nil
}
