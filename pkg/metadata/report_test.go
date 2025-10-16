package metadata

import (
	"strings"
	"testing"
	"time"

	"github.com/zookeeper-backup/pkg/utils"
	"github.com/zookeeper-backup/pkg/zkfile"
)

func TestBackupInfo_GenerateTextReport(t *testing.T) {
	info := NewBackupInfo("test-backup-123", zkfile.ZXID(0x100000000))
	info.ZooKeeper.Version = "3.8.0"
	info.ZooKeeper.Host = "localhost:2181"
	info.ZooKeeper.DataDir = "/data"
	info.ZooKeeper.LogDir = "/logs"

	info.AddSnapshot(&zkfile.SnapshotInfo{
		Name: "snapshot.100000000",
		Zxid: zkfile.ZXID(0x100000000),
		Size: 1024,
	})

	info.AddTxnLog(&zkfile.TxnLogInfo{
		Name:             "log.100000000",
		StartZxid:        zkfile.ZXID(0x100000000),
		EndZxid:          zkfile.ZXID(0x100000010),
		Size:             2048,
		TransactionCount: 10,
	})

	info.UpdateValidation(2, 0, 0)
	info.UpdateStatistics(3072, 1536, 10*time.Second)

	report := info.GenerateTextReport()

	// Check report contains key information
	if !strings.Contains(report, "test-backup-123") {
		t.Error("Report should contain backup ID")
	}
	if !strings.Contains(report, "3.8.0") {
		t.Error("Report should contain ZooKeeper version")
	}
	if !strings.Contains(report, "localhost:2181") {
		t.Error("Report should contain ZK host")
	}
	if !strings.Contains(report, "Snapshots: 1") {
		t.Error("Report should contain snapshot count")
	}
	if !strings.Contains(report, "TxnLogs: 1") {
		t.Error("Report should contain txnlog count")
	}
	if !strings.Contains(report, "Total Files: 2") {
		t.Error("Report should contain total file count")
	}
	if !strings.Contains(report, "10.00 seconds") {
		t.Error("Report should contain duration")
	}
}

func TestBackupInfo_GenerateManifest(t *testing.T) {
	info := NewBackupInfo("test-backup-456", zkfile.ZXID(0x200000000))

	info.AddSnapshot(&zkfile.SnapshotInfo{
		Name: "snapshot.200000000",
		Zxid: zkfile.ZXID(0x200000000),
		Size: 5120,
	})

	info.AddTxnLog(&zkfile.TxnLogInfo{
		Name:             "log.200000000",
		StartZxid:        zkfile.ZXID(0x200000000),
		EndZxid:          zkfile.ZXID(0x200000020),
		Size:             10240,
		TransactionCount: 20,
		Status:           "valid",
	})

	manifest := info.GenerateManifest()

	// Check manifest contains key information
	if !strings.Contains(manifest, "test-backup-456") {
		t.Error("Manifest should contain backup ID")
	}
	if !strings.Contains(manifest, "ZooKeeper Backup Manifest") {
		t.Error("Manifest should contain title")
	}
	if !strings.Contains(manifest, "snapshot.200000000") {
		t.Error("Manifest should contain snapshot file name")
	}
	if !strings.Contains(manifest, "log.200000000") {
		t.Error("Manifest should contain txnlog file name")
	}
	if !strings.Contains(manifest, "## Snapshot Files") {
		t.Error("Manifest should have snapshot section")
	}
	if !strings.Contains(manifest, "## TxnLog Files") {
		t.Error("Manifest should have txnlog section")
	}
	if !strings.Contains(manifest, "Status: valid") {
		t.Error("Manifest should contain status")
	}
	if !strings.Contains(manifest, "Txns: 20") {
		t.Error("Manifest should contain transaction count")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 100,
			want:  "100 B",
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
		{
			name:  "terabytes",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.0 TB",
		},
		{
			name:  "1.5 MB",
			bytes: 1536 * 1024,
			want:  "1.5 MB",
		},
		{
			name:  "500 KB",
			bytes: 512 * 1024,
			want:  "512.0 KB",
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
