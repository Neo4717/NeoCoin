package main

import (
	"strings"
	"testing"
)

func TestWSClient_LegacyBroadcastAll(t *testing.T) {
	c := &wsClient{sub: wsSubscriptions{legacyAll: true}}
	if !c.acceptsEvent(WSEvent{Type: "new_block", Data: map[string]any{"height": 1}}) {
		t.Fatal("expected legacy client to accept all events")
	}
}

func TestWSClient_SubscribeAddressFilters(t *testing.T) {
	addrA := strings.Repeat("a", 64)
	addrB := strings.Repeat("b", 64)
	addrC := strings.Repeat("c", 64)

	c := &wsClient{sub: wsSubscriptions{legacyAll: true}}
	c.handleText([]byte(`{"type":"subscribe","topic":"address","address":"` + addrA + `"}`))

	if c.acceptsEvent(WSEvent{
		Type: "mempool_added",
		Data: map[string]any{"fromAddr": addrA, "toAddress": addrB},
	}) != true {
		t.Fatal("expected address subscriber to receive matching event")
	}

	if c.acceptsEvent(WSEvent{
		Type: "mempool_added",
		Data: map[string]any{"fromAddr": addrB, "toAddress": addrC},
	}) != false {
		t.Fatal("expected address subscriber to not receive non-matching event")
	}
}

func TestWSClient_SubscribeTypeFilters(t *testing.T) {
	c := &wsClient{sub: wsSubscriptions{legacyAll: true}}
	c.handleText([]byte(`{"type":"subscribe","topic":"type","event":"new_block"}`))

	if !c.acceptsEvent(WSEvent{Type: "new_block", Data: map[string]any{"height": 1}}) {
		t.Fatal("expected type subscriber to receive matching type")
	}
	if c.acceptsEvent(WSEvent{Type: "mempool_added", Data: map[string]any{"txId": "x"}}) {
		t.Fatal("expected type subscriber to not receive other types")
	}
}

func TestWSClient_SubscribeAllOverridesFilters(t *testing.T) {
	addrA := strings.Repeat("a", 64)
	c := &wsClient{sub: wsSubscriptions{legacyAll: true}}
	c.handleText([]byte(`{"type":"subscribe","topic":"address","address":"` + addrA + `"}`))
	c.handleText([]byte(`{"type":"subscribe","topic":"all"}`))

	if !c.acceptsEvent(WSEvent{Type: "anything", Data: map[string]any{"x": "y"}}) {
		t.Fatal("expected topic=all to accept all events")
	}
}
