package engine

import (
	"strings"
	"testing"
)

func TestBackupConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *BackupConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &BackupConfig{
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
				OutputDir: "/backup",
			},
			wantErr: false,
		},
		{
			name: "missing data dir",
			config: &BackupConfig{
				ZkLogDir:  "/logs",
				OutputDir: "/backup",
			},
			wantErr: true,
			errMsg:  "zk-data-dir is required",
		},
		{
			name: "missing log dir",
			config: &BackupConfig{
				ZkDataDir: "/data",
				OutputDir: "/backup",
			},
			wantErr: true,
			errMsg:  "zk-log-dir is required",
		},
		{
			name: "missing output dir",
			config: &BackupConfig{
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
			},
			wantErr: true,
			errMsg:  "output-dir is required",
		},
		{
			name: "auto-generate backup id",
			config: &BackupConfig{
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
				OutputDir: "/backup",
			},
			wantErr: false,
		},
		{
			name: "default zk host",
			config: &BackupConfig{
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
				OutputDir: "/backup",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BackupConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("BackupConfig.Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
			if !tt.wantErr {
				// Check defaults
				if tt.config.ZkHost == "" {
					t.Error("ZkHost should be set to default")
				}
				if tt.config.BackupID == "" {
					t.Error("BackupID should be auto-generated")
				}
				if !strings.HasPrefix(tt.config.BackupID, "backup-") {
					t.Errorf("BackupID should start with 'backup-', got %v", tt.config.BackupID)
				}
			}
		})
	}
}

func TestRestoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RestoreConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &RestoreConfig{
				BackupDir: "/backup/backup-123",
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
			},
			wantErr: false,
		},
		{
			name: "missing backup dir",
			config: &RestoreConfig{
				ZkDataDir: "/data",
				ZkLogDir:  "/logs",
			},
			wantErr: true,
			errMsg:  "backup-dir is required",
		},
		{
			name: "missing data dir",
			config: &RestoreConfig{
				BackupDir: "/backup/backup-123",
				ZkLogDir:  "/logs",
			},
			wantErr: true,
			errMsg:  "zk-data-dir is required",
		},
		{
			name: "missing log dir",
			config: &RestoreConfig{
				BackupDir: "/backup/backup-123",
				ZkDataDir: "/data",
			},
			wantErr: true,
			errMsg:  "zk-log-dir is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RestoreConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("RestoreConfig.Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestVerifyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *VerifyConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &VerifyConfig{
				BackupDir: "/backup/backup-123",
			},
			wantErr: false,
		},
		{
			name:    "missing backup dir",
			config:  &VerifyConfig{},
			wantErr: true,
			errMsg:  "backup-dir is required",
		},
		{
			name: "default output format",
			config: &VerifyConfig{
				BackupDir: "/backup/backup-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("VerifyConfig.Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
			if !tt.wantErr && tt.config.OutputFormat == "" {
				t.Error("OutputFormat should be set to default")
			}
		})
	}
}
