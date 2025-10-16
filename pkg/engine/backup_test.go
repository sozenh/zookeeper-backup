package engine

import (
	"testing"

	"github.com/zookeeper-backup/pkg/utils"
)

func TestNewBackupEngine(t *testing.T) {
	config := &BackupConfig{
		ZkDataDir: "/data",
		ZkLogDir:  "/logs",
		OutputDir: "/output",
	}

	engine := NewBackupEngine(config)
	if engine == nil {
		t.Error("NewBackupEngine() should return non-nil engine")
	}
	if engine.config != config {
		t.Error("Engine should store config reference")
	}
	if engine.logger == nil {
		t.Error("Engine should have logger initialized")
	}
}

func TestBackupEngine_preCheck(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		config  *BackupConfig
		wantErr bool
		setup   func()
	}{
		{
			name: "nonexistent data dir",
			config: &BackupConfig{
				ZkDataDir: "/nonexistent/data",
				ZkLogDir:  tmpDir,
				OutputDir: tmpDir,
			},
			wantErr: true,
		},
		{
			name: "nonexistent log dir",
			config: &BackupConfig{
				ZkDataDir: tmpDir,
				ZkLogDir:  "/nonexistent/logs",
				OutputDir: tmpDir,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			engine := NewBackupEngine(tt.config)
			err := engine.preCheck()

			if (err != nil) != tt.wantErr {
				t.Errorf("preCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "bytes",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.0 MB",
		},
		{
			name:  "gigabytes",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}
