package engine

import (
	"testing"
)

func TestNewRestoreEngine(t *testing.T) {
	config := &RestoreConfig{
		BackupDir: "/backup/backup-123",
		ZkDataDir: "/data",
		ZkLogDir:  "/logs",
	}

	engine := NewRestoreEngine(config)
	if engine == nil {
		t.Error("NewRestoreEngine() should return non-nil engine")
	}
	if engine.config != config {
		t.Error("Engine should store config reference")
	}
	if engine.logger == nil {
		t.Error("Engine should have logger initialized")
	}
}

func TestRestoreEngine_Structure(t *testing.T) {
	tmpDir := t.TempDir()

	config := &RestoreConfig{
		BackupDir: tmpDir,
		ZkDataDir: "/data",
		ZkLogDir:  "/logs",
	}

	engine := NewRestoreEngine(config)

	// Test that engine has necessary components
	t.Run("engine structure", func(t *testing.T) {
		if engine.config == nil {
			t.Error("Engine config should not be nil")
		}
		if engine.logger == nil {
			t.Error("Engine logger should not be nil")
		}
	})
}
