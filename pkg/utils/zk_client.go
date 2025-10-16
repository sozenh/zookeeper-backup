package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"

	"github.com/zookeeper-backup/pkg/zkfile"
)

// ZKClient wraps zookeeper client
type ZKClient struct {
	host string
	conn *zk.Conn
}

// NewZKClient creates a new ZooKeeper client
func NewZKClient(host string, timeout time.Duration) (*ZKClient, error) {
	conn, _, err := zk.Connect([]string{host}, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	return &ZKClient{conn: conn, host: host}, nil
}

// Close closes the client connection
func (c *ZKClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsAlive checks if ZooKeeper is alive
func (c *ZKClient) IsAlive() bool {
	_, _, err := c.conn.Get("/")
	return err == nil
}

// GetVersion retrieves ZooKeeper version
func (c *ZKClient) GetVersion() (string, error) {
	stats, err := c.getStats("mntr")
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(stats, "\n") {
		if strings.HasPrefix(line, "zk_version") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "unknown", nil
}

// GetCurrentZXID retrieves the current ZXID from ZooKeeper
func (c *ZKClient) GetCurrentZXID() (zkfile.ZXID, error) {
	// Use the 'mntr' four-letter word command
	stats, err := c.getStats("mntr")
	if err != nil {
		return 0, err
	}

	// Parse zk_zxid from stats
	for _, line := range strings.Split(stats, "\n") {
		if strings.HasPrefix(line, "zk_zxid") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// ZXID is in hex format 0x...
				zxidStr := strings.TrimPrefix(parts[1], "0x")
				zxid, err := strconv.ParseUint(zxidStr, 16, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse zxid: %w", err)
				}
				return zkfile.ZXID(zxid), nil
			}
		}
	}

	return 0, fmt.Errorf("zk_zxid not found in mntr output")
}

// GetStats retrieves ZooKeeper stats using four-letter word command
func (c *ZKClient) GetStats(command string) (string, error) {
	return c.getStats(command)
}

// getStats internal method to get stats
func (c *ZKClient) getStats(command string) (string, error) {
	// Note: go-zookeeper library doesn't directly support four-letter words
	// We need to use raw TCP connection for this
	// For now, return a placeholder
	// In production, you'd implement raw TCP connection to send four-letter words

	return "", fmt.Errorf("four-letter words not implemented in this client")
}
