package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zookeeper-backup/pkg/zkfile"
)

func TestNewBackupInfo(t *testing.T) {
	backupID := "test-backup-123"
	zxid := zkfile.ZXID(0x100000000)

	info := NewBackupInfo(backupID, zxid)

	if info.Version != "1.0" {
		t.Errorf("Version = %v, want %v", info.Version, "1.0")
	}
	if info.BackupID != backupID {
		t.Errorf("BackupID = %v, want %v", info.BackupID, backupID)
	}
	if info.BackupZxid.Decimal != uint64(zxid) {
		t.Errorf("BackupZxid.Decimal = %v, want %v", info.BackupZxid.Decimal, uint64(zxid))
	}
	if info.BackupZxid.Hex != zxid.Hex() {
		t.Errorf("BackupZxid.Hex = %v, want %v", info.BackupZxid.Hex, zxid.Hex())
	}
	if info.Files.Snapshots == nil {
		t.Error("Files.Snapshots should be initialized")
	}
	if info.Files.TxnLogs == nil {
		t.Error("Files.TxnLogs should be initialized")
	}
	if !info.Validation.Enabled {
		t.Error("Validation.Enabled should be true")
	}
}

func TestBackupInfo_AddSnapshot(t *testing.T) {
	info := NewBackupInfo("test", zkfile.ZXID(100))

	snapshot := &zkfile.SnapshotInfo{
		Name: "snapshot.100",
		Zxid: zkfile.ZXID(100),
		Size: 1024,
	}

	info.AddSnapshot(snapshot)

	if len(info.Files.Snapshots) != 1 {
		t.Errorf("Snapshots count = %v, want 1", len(info.Files.Snapshots))
	}
	if info.Files.Snapshots[0] != snapshot {
		t.Error("Snapshot not added correctly")
	}
}

func TestBackupInfo_AddTxnLog(t *testing.T) {
	info := NewBackupInfo("test", zkfile.ZXID(100))

	txnlog := &zkfile.TxnLogInfo{
		Name:      "log.100",
		StartZxid: zkfile.ZXID(100),
		EndZxid:   zkfile.ZXID(200),
		Size:      2048,
	}

	info.AddTxnLog(txnlog)

	if len(info.Files.TxnLogs) != 1 {
		t.Errorf("TxnLogs count = %v, want 1", len(info.Files.TxnLogs))
	}
	if info.Files.TxnLogs[0] != txnlog {
		t.Error("TxnLog not added correctly")
	}
}

func TestBackupInfo_UpdateValidation(t *testing.T) {
	info := NewBackupInfo("test", zkfile.ZXID(100))

	info.UpdateValidation(10, 2, 1)

	if info.Validation.TotalFiles != 12 {
		t.Errorf("TotalFiles = %v, want 12", info.Validation.TotalFiles)
	}
	if info.Validation.ValidFiles != 10 {
		t.Errorf("ValidFiles = %v, want 10", info.Validation.ValidFiles)
	}
	if info.Validation.CorruptedFiles != 2 {
		t.Errorf("CorruptedFiles = %v, want 2", info.Validation.CorruptedFiles)
	}
	if info.Validation.RepairedFiles != 1 {
		t.Errorf("RepairedFiles = %v, want 1", info.Validation.RepairedFiles)
	}
}

func TestBackupInfo_UpdateStatistics(t *testing.T) {
	info := NewBackupInfo("test", zkfile.ZXID(100))

	totalSize := int64(1024 * 1024)
	compressedSize := int64(512 * 1024)
	duration := 10 * time.Second

	info.UpdateStatistics(totalSize, compressedSize, duration)

	if info.Statistics.TotalSize != totalSize {
		t.Errorf("TotalSize = %v, want %v", info.Statistics.TotalSize, totalSize)
	}
	if info.Statistics.CompressedSize != compressedSize {
		t.Errorf("CompressedSize = %v, want %v", info.Statistics.CompressedSize, compressedSize)
	}
	if info.Statistics.DurationSeconds != 10.0 {
		t.Errorf("DurationSeconds = %v, want 10.0", info.Statistics.DurationSeconds)
	}
}

func TestBackupInfo_SaveToFile_LoadBackupInfo(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "backup_info.json")

	// Create backup info
	info := NewBackupInfo("test-backup", zkfile.ZXID(0x100000000))
	info.ZooKeeper.Version = "3.8.0"
	info.ZooKeeper.Host = "localhost:2181"
	info.AddSnapshot(&zkfile.SnapshotInfo{
		Name: "snapshot.100000000",
		Zxid: zkfile.ZXID(0x100000000),
		Size: 1024,
	})

	// Save
	err := info.SaveToFile(filePath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("File should exist after SaveToFile")
	}

	// Load
	loaded, err := LoadBackupInfo(filePath)
	if err != nil {
		t.Fatalf("LoadBackupInfo() error = %v", err)
	}

	// Verify
	if loaded.BackupID != info.BackupID {
		t.Errorf("BackupID = %v, want %v", loaded.BackupID, info.BackupID)
	}
	if loaded.BackupZxid.Decimal != info.BackupZxid.Decimal {
		t.Errorf("BackupZxid.Decimal = %v, want %v", loaded.BackupZxid.Decimal, info.BackupZxid.Decimal)
	}
	if loaded.ZooKeeper.Version != info.ZooKeeper.Version {
		t.Errorf("ZooKeeper.Version = %v, want %v", loaded.ZooKeeper.Version, info.ZooKeeper.Version)
	}
	if len(loaded.Files.Snapshots) != 1 {
		t.Errorf("Snapshots count = %v, want 1", len(loaded.Files.Snapshots))
	}
}

func TestLoadBackupInfo_FileNotExists(t *testing.T) {
	_, err := LoadBackupInfo("/nonexistent/path/backup_info.json")
	if err == nil {
		t.Error("LoadBackupInfo() should return error for nonexistent file")
	}
}
