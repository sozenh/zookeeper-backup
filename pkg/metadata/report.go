package metadata

import (
	"fmt"
	"strings"
	"time"

	"github.com/zookeeper-backup/pkg/utils"
)

// GenerateTextReport generates a human-readable text report
func (bi *BackupInfo) GenerateTextReport() string {
	var sb strings.Builder

	sb.WriteString("╔════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           Backup Report                                    ║\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════╝\n\n")

	sb.WriteString(fmt.Sprintf("Backup ID: %s\n", bi.BackupID))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", bi.BackupTimestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Backup ZXID: 0x%s (%d)\n\n", bi.BackupZxid.Hex, bi.BackupZxid.Decimal))

	sb.WriteString("ZooKeeper Information:\n")
	sb.WriteString(fmt.Sprintf("  Version: %s\n", bi.ZooKeeper.Version))
	sb.WriteString(fmt.Sprintf("  Host: %s\n", bi.ZooKeeper.Host))
	sb.WriteString(fmt.Sprintf("  Data Dir: %s\n", bi.ZooKeeper.DataDir))
	sb.WriteString(fmt.Sprintf("  Log Dir: %s\n\n", bi.ZooKeeper.LogDir))

	sb.WriteString("Files:\n")
	sb.WriteString(fmt.Sprintf("  Snapshots: %d\n", len(bi.Files.Snapshots)))
	sb.WriteString(fmt.Sprintf("  TxnLogs: %d\n\n", len(bi.Files.TxnLogs)))

	if bi.Validation.Enabled {
		sb.WriteString("Validation:\n")
		sb.WriteString(fmt.Sprintf("  Total Files: %d\n", bi.Validation.TotalFiles))
		sb.WriteString(fmt.Sprintf("  Valid Files: %d\n", bi.Validation.ValidFiles))
		sb.WriteString(fmt.Sprintf("  Corrupted Files: %d\n", bi.Validation.CorruptedFiles))
		sb.WriteString(fmt.Sprintf("  Repaired Files: %d\n\n", bi.Validation.RepairedFiles))
	}

	sb.WriteString("Statistics:\n")
	sb.WriteString(fmt.Sprintf("  Total Size: %s\n", utils.FormatBytes(bi.Statistics.TotalSize)))
	if bi.Statistics.CompressedSize > 0 {
		sb.WriteString(fmt.Sprintf("  Compressed Size: %s\n", utils.FormatBytes(bi.Statistics.CompressedSize)))
	}
	sb.WriteString(fmt.Sprintf("  Duration: %.2f seconds\n", bi.Statistics.DurationSeconds))

	return sb.String()
}

// GenerateManifest generates a simple manifest file content
func (bi *BackupInfo) GenerateManifest() string {
	var sb strings.Builder

	sb.WriteString("# ZooKeeper Backup Manifest\n\n")
	sb.WriteString(fmt.Sprintf("Backup ID: %s\n", bi.BackupID))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", bi.BackupTimestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("ZXID: 0x%s\n\n", bi.BackupZxid.Hex))

	sb.WriteString("## Snapshot Files\n\n")
	for _, s := range bi.Files.Snapshots {
		sb.WriteString(fmt.Sprintf("- %s (ZXID: 0x%s, Size: %s)\n", s.Name, s.Zxid.Hex(), utils.FormatBytes(s.Size)))
	}

	sb.WriteString("\n## TxnLog Files\n\n")
	for _, t := range bi.Files.TxnLogs {
		sb.WriteString(fmt.Sprintf("- %s (ZXID: 0x%s - 0x%s, Txns: %d, Size: %s, Status: %s)\n", t.Name, t.StartZxid.Hex(), t.EndZxid.Hex(), t.TransactionCount, utils.FormatBytes(t.Size), t.Status))
	}

	return sb.String()
}
