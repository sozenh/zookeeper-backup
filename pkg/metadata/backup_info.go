package metadata

import (
	"encoding/json"
	"os"
	"time"

	"github.com/zookeeper-backup/pkg/zkfile"
)

// BackupInfo backup metadata structure
type BackupInfo struct {
	Version         string         `json:"version"`
	BackupID        string         `json:"backup_id"`
	BackupTimestamp time.Time      `json:"backup_timestamp"`
	BackupZxid      ZxidInfo       `json:"backup_zxid"`
	ZooKeeper       ZooKeeperInfo  `json:"zookeeper"`
	Files           FilesInfo      `json:"files"`
	Validation      ValidationInfo `json:"validation"`
	Statistics      StatisticsInfo `json:"statistics"`
}

// ZxidInfo ZXID information
type ZxidInfo struct {
	Hex     string `json:"hex"`
	Decimal uint64 `json:"decimal"`
}

// ZooKeeperInfo ZooKeeper server information
type ZooKeeperInfo struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	DataDir string `json:"data_dir"`
	LogDir  string `json:"log_dir"`
}

// FilesInfo backup files information
type FilesInfo struct {
	TxnLogs   []*zkfile.TxnLogInfo   `json:"txnlogs"`
	Snapshots []*zkfile.SnapshotInfo `json:"snapshots"`
}

// ValidationInfo validation results
type ValidationInfo struct {
	Enabled            bool `json:"enabled"`
	TotalFiles         int  `json:"total_files"`
	ValidFiles         int  `json:"valid_files"`
	CorruptedFiles     int  `json:"corrupted_files"`
	RepairedFiles      int  `json:"repaired_files"`
	UnrecoverableFiles int  `json:"unrecoverable_files"`
}

// StatisticsInfo backup statistics
type StatisticsInfo struct {
	TotalSize       int64   `json:"total_size"`
	CompressedSize  int64   `json:"compressed_size,omitempty"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// LoadBackupInfo loads BackupInfo from a JSON file
func LoadBackupInfo(path string) (*BackupInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info BackupInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// NewBackupInfo creates a new BackupInfo instance
func NewBackupInfo(backupID string, zxid zkfile.ZXID) *BackupInfo {
	return &BackupInfo{
		Version:         "1.0",
		BackupID:        backupID,
		BackupTimestamp: time.Now(),
		BackupZxid: ZxidInfo{
			Hex:     zxid.Hex(),
			Decimal: uint64(zxid),
		},
		Files: FilesInfo{
			TxnLogs:   make([]*zkfile.TxnLogInfo, 0),
			Snapshots: make([]*zkfile.SnapshotInfo, 0),
		},
		Validation: ValidationInfo{
			Enabled: true,
		},
	}
}

// SaveToFile saves BackupInfo to a JSON file
func (bi *BackupInfo) SaveToFile(path string) error {
	data, err := json.MarshalIndent(bi, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddSnapshot adds a snapshot file info
func (bi *BackupInfo) AddSnapshot(info *zkfile.SnapshotInfo) {
	bi.Files.Snapshots = append(bi.Files.Snapshots, info)
}

// AddTxnLog adds a txnlog file info
func (bi *BackupInfo) AddTxnLog(info *zkfile.TxnLogInfo) {
	bi.Files.TxnLogs = append(bi.Files.TxnLogs, info)
}

// UpdateValidation updates validation info
func (bi *BackupInfo) UpdateValidation(validFiles, corruptedFiles, repairedFiles int) {
	bi.Validation.TotalFiles = validFiles + corruptedFiles
	bi.Validation.ValidFiles = validFiles
	bi.Validation.CorruptedFiles = corruptedFiles
	bi.Validation.RepairedFiles = repairedFiles
}

// UpdateStatistics updates statistics info
func (bi *BackupInfo) UpdateStatistics(totalSize, compressedSize int64, duration time.Duration) {
	bi.Statistics.TotalSize = totalSize
	bi.Statistics.CompressedSize = compressedSize
	bi.Statistics.DurationSeconds = duration.Seconds()
}
