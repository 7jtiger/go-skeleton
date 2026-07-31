package controller

import (
	"testing"
)

// trySend가 닫힌 채널에 대해 panic 없이 false를 반환하는지 확인
func Test_trySend_closedChannel(t *testing.T) {
	p := &SignalingController{}
	client := &WSClient{
		send:   make(chan []byte, 1),
		userID: "test-user",
	}
	client.closeSend()

	if sent := p.trySend(client, []byte("hello")); sent {
		t.Fatal("expected trySend to return false on closed channel")
	}
}

// trySend 정상 전송 및 버퍼 가득 참 처리 확인
func Test_trySend_bufferFull(t *testing.T) {
	p := &SignalingController{}
	client := &WSClient{
		send:   make(chan []byte, 1),
		userID: "test-user",
	}

	if sent := p.trySend(client, []byte("first")); !sent {
		t.Fatal("expected first send to succeed")
	}
	if sent := p.trySend(client, []byte("second")); sent {
		t.Fatal("expected second send to fail: buffer full")
	}
}

// closeSend가 여러 번 호출돼도 panic이 없어야 함
func Test_closeSend_idempotent(t *testing.T) {
	client := &WSClient{send: make(chan []byte, 1)}

	client.closeSend()
	client.closeSend()

	if _, ok := <-client.send; ok {
		t.Fatal("send channel should be closed")
	}
}
