package engine

import (
	"fmt"
	"time"
)

// BackupConfig backup configuration
type BackupConfig struct {
	ZkDataDir   string
	ZkLogDir    string
	OutputDir   string
	ZkHost      string
	BackupID    string
	Verify      bool
	Compression string
	Verbose     bool
}

// Validate validates the backup configuration
func (c *BackupConfig) Validate() error {
	if c.ZkHost == "" {
		c.ZkHost = "localhost:2181"
	}
	if c.BackupID == "" {
		c.BackupID = generateBackupID()
	}
	if c.ZkLogDir == "" {
		return fmt.Errorf("zk-log-dir is required")
	}
	if c.ZkDataDir == "" {
		return fmt.Errorf("zk-data-dir is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output-dir is required")
	}
	return nil
}

// RestoreConfig restore configuration
type RestoreConfig struct {
	BackupDir      string
	ZkDataDir      string
	ZkLogDir       string
	Force          bool
	DryRun         bool
	SkipVerify     bool
	TruncateToZxid string
	Verbose        bool
}

// Validate validates the restore configuration
func (c *RestoreConfig) Validate() error {
	if c.BackupDir == "" {
		return fmt.Errorf("backup-dir is required")
	}
	if c.ZkLogDir == "" {
		return fmt.Errorf("zk-log-dir is required")
	}
	if c.ZkDataDir == "" {
		return fmt.Errorf("zk-data-dir is required")
	}
	return nil
}

// VerifyConfig verify configuration
type VerifyConfig struct {
	BackupDir    string
	Fix          bool
	OutputFormat string
	Verbose      bool
}

// Validate validates the verify configuration
func (c *VerifyConfig) Validate() error {
	if c.OutputFormat == "" {
		c.OutputFormat = "text"
	}
	if c.BackupDir == "" {
		return fmt.Errorf("backup-dir is required")
	}
	return nil
}

// generateBackupID generates a backup ID with timestamp
func generateBackupID() string {
	return fmt.Sprintf("backup-%s", time.Now().Format("20060102-150405"))
}
