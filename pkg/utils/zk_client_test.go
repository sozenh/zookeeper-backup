package utils

import (
	"testing"

	"github.com/zookeeper-backup/pkg/zkfile"
)

func TestZKClient_IsAlive(t *testing.T) {
	// Note: These tests require a running ZooKeeper instance
	// In a real environment, you would use mocks or test containers

	t.Run("client creation", func(t *testing.T) {
		// This will fail to connect, but tests the client creation
		client, err := NewZKClient("nonexistent:2181", 1)
		if err == nil && client != nil {
			defer client.Close()
			// If somehow connected, test IsAlive
			_ = client.IsAlive()
		}
		// Test passes if no panic occurs
	})
}

func TestZKClient_GetCurrentZXID(t *testing.T) {
	t.Run("placeholder implementation", func(t *testing.T) {
		// Current implementation returns error
		// This test documents the current behavior
		client := &ZKClient{
			conn: nil,
			host: "test:2181",
		}

		_, err := client.GetCurrentZXID()
		// Current implementation returns error for four-letter words
		if err == nil {
			t.Error("GetCurrentZXID() should return error for unimplemented four-letter words")
		}
	})
}

func TestZKClient_GetVersion(t *testing.T) {
	t.Run("placeholder implementation", func(t *testing.T) {
		client := &ZKClient{
			conn: nil,
			host: "test:2181",
		}

		_, err := client.GetVersion()
		if err == nil {
			t.Error("GetVersion() should return error for unimplemented four-letter words")
		}
	})
}

func TestZKClient_GetStats(t *testing.T) {
	t.Run("placeholder implementation", func(t *testing.T) {
		client := &ZKClient{
			conn: nil,
			host: "test:2181",
		}

		_, err := client.GetStats("mntr")
		if err == nil {
			t.Error("GetStats() should return error for unimplemented four-letter words")
		}
	})
}

func TestZKClient_Close(t *testing.T) {
	t.Run("close nil connection", func(t *testing.T) {
		client := &ZKClient{
			conn: nil,
			host: "test:2181",
		}

		// Should not panic
		client.Close()
	})
}

// Test that ZXID type from zkfile package works with client
func TestZKClientWithZXID(t *testing.T) {
	t.Run("zxid type compatibility", func(t *testing.T) {
		var zxid zkfile.ZXID = 0x100000000

		if zxid == 0 {
			t.Error("ZXID should not be zero")
		}

		// Test ZXID can be used with client types
		_ = zxid.String()
		_ = zxid.Hex()
	})
}
