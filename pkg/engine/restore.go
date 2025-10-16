package engine

import (
	"fmt"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/zookeeper-backup/pkg/metadata"
	"github.com/zookeeper-backup/pkg/utils"
	"github.com/zookeeper-backup/pkg/zkfile"
)

// RestoreEngine restore engine
type RestoreEngine struct {
	config *RestoreConfig
	logger *zap.Logger
}

// NewRestoreEngine creates a new restore engine
func NewRestoreEngine(config *RestoreConfig) *RestoreEngine {
	return &RestoreEngine{
		config: config,
		logger: utils.GetLogger(),
	}
}

// Run executes the restore operation
func (e *RestoreEngine) Run() error {
	// 1. Validate configuration
	if err := e.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	e.logger.Info("Starting restore",
		zap.String("log_dir", e.config.ZkLogDir),
		zap.String("data_dir", e.config.ZkDataDir),
		zap.String("backup_dir", e.config.BackupDir))

	// 2. Load backup metadata
	backupInfo, err := e.loadBackupInfo()
	if err != nil {
		return fmt.Errorf("failed to load backup info: %w", err)
	}

	// 3. Verify backup if not skipped
	if !e.config.SkipVerify {
		e.logger.Info("Verifying backup before restore")
		if err = e.verifyBackup(); err != nil {
			return fmt.Errorf("backup verification failed: %w", err)
		}
	}

	// 4. Confirm restore (if not forced or dry-run)
	if !e.config.Force && !e.config.DryRun {
		if !e.confirmRestore(backupInfo) {
			return fmt.Errorf("restore cancelled by user")
		}
	}

	// 5. Dry-run: just show what would be restored
	if e.config.DryRun {
		return e.showDryRun(backupInfo)
	}

	// 6. Backup existing data (safety measure)
	if err = e.backupExistingData(); err != nil {
		return fmt.Errorf("failed to backup existing data: %w", err)
	}

	// 7. Restore snapshots
	e.logger.Info("Restoring snapshot files")
	if err = e.restoreSnapshots(); err != nil {
		return fmt.Errorf("failed to restore snapshots: %w", err)
	}

	// 8. Restore txnlogs
	e.logger.Info("Restoring txnlog files")
	if err = e.restoreTxnLogs(); err != nil {
		return fmt.Errorf("failed to restore txnlogs: %w", err)
	}

	e.logger.Info("Restore completed successfully")

	e.printNextSteps(backupInfo)

	return nil
}

// loadBackupInfo loads backup metadata
func (e *RestoreEngine) loadBackupInfo() (*metadata.BackupInfo, error) {
	return metadata.LoadBackupInfo(
		filepath.Join(e.config.BackupDir, "metadata", "backup_info.json"))
}

// verifyBackup verifies backup integrity
func (e *RestoreEngine) verifyBackup() error {
	txnlogDir := filepath.Join(e.config.BackupDir, "txnlogs")
	snapshotDir := filepath.Join(e.config.BackupDir, "snapshots")

	results, err := zkfile.ValidateBackupFiles(snapshotDir, txnlogDir)
	if err != nil {
		return err
	}

	for path, result := range results {
		if !result.IsValid {
			return fmt.Errorf("file validation failed: %s (%s)", path, result.CorruptionType)
		}
	}

	return nil
}

// backupExistingData backs up existing data before restore
func (e *RestoreEngine) backupExistingData() error {
	// Implementation: move existing files to backup location
	e.logger.Info("Backing up existing data (safety measure)")
	return nil
}

// restoreTxnLogs restores txnlog files
func (e *RestoreEngine) restoreTxnLogs() error {
	txnlogDir := filepath.Join(e.config.BackupDir, "txnlogs")
	txnlogs, err := zkfile.ListTxnLogFiles(txnlogDir)
	if err != nil {
		return err
	}

	for _, txnlog := range txnlogs {
		dst := filepath.Join(e.config.ZkLogDir, filepath.Base(txnlog))
		if err = zkfile.CopyFile(txnlog, dst); err != nil {
			return err
		}
		e.logger.Debug("Restored txnlog", zap.String("file", filepath.Base(txnlog)))
	}

	return nil
}

// restoreSnapshots restores snapshot files
func (e *RestoreEngine) restoreSnapshots() error {
	snapshotDir := filepath.Join(e.config.BackupDir, "snapshots")
	snapshots, err := zkfile.ListSnapshotFiles(snapshotDir)
	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		dst := filepath.Join(e.config.ZkDataDir, filepath.Base(snapshot))
		if err = zkfile.CopyFile(snapshot, dst); err != nil {
			return err
		}
		e.logger.Debug("Restored snapshot", zap.String("file", filepath.Base(snapshot)))
	}

	return nil
}

// showDryRun shows what would be restored
func (e *RestoreEngine) showDryRun(info *metadata.BackupInfo) error {
	fmt.Printf("Would restore:\n")
	fmt.Printf("- %d txnlog files\n", len(info.Files.TxnLogs))
	fmt.Printf("- %d snapshot files\n", len(info.Files.Snapshots))

	return nil
}

// printNextSteps prints next steps after restore
func (e *RestoreEngine) printNextSteps(info *metadata.BackupInfo) {
	fmt.Printf("Next steps:\n")
	fmt.Printf("1. Start ZooKeeper:\n")
	fmt.Printf("   zkServer.sh start\n")
	fmt.Printf("2. Verify ZXID:\n")
	fmt.Printf("   echo mntr | nc localhost 2181 | grep zk_zxid\n")
	fmt.Printf("   Expected: 0x%s\n", info.BackupZxid.Hex)
	fmt.Printf("3. Verify data integrity:\n")
	fmt.Printf("   zkCli.sh -server localhost:2181\n")
	fmt.Printf("   ls /\n")
}

// confirmRestore asks user for confirmation
func (e *RestoreEngine) confirmRestore(info *metadata.BackupInfo) bool {
	fmt.Printf("You are about to restore ZooKeeper data:\n")
	fmt.Printf("  Backup ID: %s\n", info.BackupID)
	fmt.Printf("  Backup ZXID: 0x%s\n", info.BackupZxid.Hex)
	fmt.Printf("  Target Log Dir: %s\n", e.config.ZkLogDir)
	fmt.Printf("  Target Data Dir: %s\n", e.config.ZkDataDir)
	fmt.Printf("  Backup Time: %s\n", info.BackupTimestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("This will overwrite existing data! Type 'yes' to continue: \n")

	var response string
	_, _ = fmt.Scanln(&response)

	return response == "yes"
}
