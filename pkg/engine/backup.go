package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/zookeeper-backup/pkg/metadata"
	"github.com/zookeeper-backup/pkg/utils"
	"github.com/zookeeper-backup/pkg/zkfile"
)

// BackupEngine backup engine
type BackupEngine struct {
	config *BackupConfig
	logger *zap.Logger
}

// NewBackupEngine creates a new backup engine
func NewBackupEngine(config *BackupConfig) *BackupEngine {
	return &BackupEngine{
		config: config,
		logger: utils.GetLogger(),
	}
}

// Run executes the backup operation
func (e *BackupEngine) Run() error {
	startTime := time.Now()

	// 1. Validate configuration
	if err := e.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	e.logger.Info("Starting backup",
		zap.String("backup_id", e.config.BackupID),
		zap.String("log_dir", e.config.ZkLogDir),
		zap.String("data_dir", e.config.ZkDataDir),
		zap.String("output_dir", e.config.OutputDir))

	// 2. Pre-check
	if err := e.preCheck(); err != nil {
		return fmt.Errorf("pre-check failed: %w", err)
	}

	// 3. Get current ZXID from ZooKeeper
	currentZxid, zkVersion, err := e.getCurrentZxid()
	if err != nil {
		currentZxid = 0
		e.logger.Warn("Failed to get current ZXID from ZooKeeper, will use local files", zap.Error(err))
	}

	// 4. Create backup directory structure
	backupDir := filepath.Join(e.config.OutputDir, e.config.BackupID)
	if err = e.createBackupDirs(backupDir); err != nil {
		return fmt.Errorf("failed to create backup directories: %w", err)
	}

	// 5. Initialize backup info
	backupInfo := metadata.NewBackupInfo(e.config.BackupID, currentZxid)
	backupInfo.ZooKeeper.Version = zkVersion
	backupInfo.ZooKeeper.Host = e.config.ZkHost
	backupInfo.ZooKeeper.LogDir = e.config.ZkLogDir
	backupInfo.ZooKeeper.DataDir = e.config.ZkDataDir

	// 6. Backup snapshot files
	e.logger.Info("Backing up snapshot files")
	if err = e.backupSnapshots(backupDir, backupInfo); err != nil {
		return fmt.Errorf("failed to backup snapshots: %w", err)
	}

	// 7. Backup txnlog files
	e.logger.Info("Backing up txnlog files")
	if err = e.backupTxnLogs(backupDir, backupInfo); err != nil {
		return fmt.Errorf("failed to backup txnlogs: %w", err)
	}

	// 8. Verify backup if enabled
	if e.config.Verify {
		e.logger.Info("Verifying backup")
		if err = e.verifyBackup(backupDir, backupInfo); err != nil {
			return fmt.Errorf("backup verification failed: %w", err)
		}
	}

	// 9. Calculate statistics
	totalSize, err := zkfile.GetDirSize(backupDir)
	if err != nil {
		e.logger.Warn("Failed to calculate backup size", zap.Error(err))
	}
	backupInfo.UpdateStatistics(totalSize, 0, time.Since(startTime))

	// 10. Save metadata
	if err := e.saveMetadata(backupDir, backupInfo); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	e.logger.Info("Backup completed", zap.Int64("size", totalSize),
		zap.String("backup_id", e.config.BackupID), zap.Duration("duration", time.Since(startTime)))

	return nil
}

// preCheck performs pre-backup checks
func (e *BackupEngine) preCheck() error {
	// Check if output directory is writable
	if err := zkfile.EnsureDir(e.config.OutputDir); err != nil {
		return fmt.Errorf("output directory is not writable: %w", err)
	}

	// Check if log directory exists
	if !zkfile.DirExists(e.config.ZkLogDir) {
		return fmt.Errorf("log directory does not exist: %s", e.config.ZkLogDir)
	}

	// Check if data directory exists
	if !zkfile.DirExists(e.config.ZkDataDir) {
		return fmt.Errorf("data directory does not exist: %s", e.config.ZkDataDir)
	}

	return nil
}

// createBackupDirs creates backup directory structure
func (e *BackupEngine) createBackupDirs(backupDir string) error {
	dirs := []string{
		filepath.Join(backupDir, "metadata"),
		filepath.Join(backupDir, "logs"),
		filepath.Join(backupDir, "txnlogs"),
		filepath.Join(backupDir, "snapshots"),
	}

	for _, dir := range dirs {
		if err := zkfile.EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// getCurrentZxid gets current ZXID from ZooKeeper
func (e *BackupEngine) getCurrentZxid() (zkfile.ZXID, string, error) {
	client, err := utils.NewZKClient(e.config.ZkHost, 5*time.Second)
	if err != nil {
		return 0, "", err
	}
	defer client.Close()

	version, err := client.GetVersion()
	if err != nil {
		version = "unknown"
	}

	zxid, err := client.GetCurrentZXID()
	if err != nil {
		return 0, "", err
	}

	return zxid, version, nil
}

// backupTxnLogs backs up all txnlog files
func (e *BackupEngine) backupTxnLogs(backupDir string, backupInfo *metadata.BackupInfo) error {
	txnlogs, err := zkfile.ListTxnLogFiles(e.config.ZkLogDir)
	if err != nil {
		return err
	}

	txnlogDir := filepath.Join(backupDir, "txnlogs")

	for _, txnlog := range txnlogs {
		e.logger.Debug("Copying txnlog", zap.String("file", txnlog))
		err = zkfile.CopyFile(txnlog, filepath.Join(txnlogDir, filepath.Base(txnlog)))
		if err != nil {
			return err
		}

		// Get txnlog info
		info, err := zkfile.GetTxnLogInfo(txnlog)
		if err != nil {
			e.logger.Warn("Failed to get txnlog info", zap.Error(err))
			continue
		}

		backupInfo.AddTxnLog(info)
	}

	e.logger.Info("TxnLog backup completed", zap.Int("count", len(txnlogs)))

	return nil
}

// backupSnapshots backs up all snapshot files
func (e *BackupEngine) backupSnapshots(backupDir string, backupInfo *metadata.BackupInfo) error {
	snapshots, err := zkfile.ListSnapshotFiles(e.config.ZkDataDir)
	if err != nil {
		return err
	}

	snapshotDir := filepath.Join(backupDir, "snapshots")

	for _, snapshot := range snapshots {
		e.logger.Debug("Copying snapshot", zap.String("file", snapshot))

		err = zkfile.CopyFile(snapshot, filepath.Join(snapshotDir, filepath.Base(snapshot)))
		if err != nil {
			return err
		}

		// Get snapshot info
		info, err := zkfile.GetSnapshotInfo(snapshot)
		if err != nil {
			e.logger.Warn("Failed to get snapshot info", zap.Error(err))
			continue
		}

		backupInfo.AddSnapshot(info)
	}

	e.logger.Info("Snapshot backup completed", zap.Int("count", len(snapshots)))

	return nil
}

// verifyBackup verifies the backup
func (e *BackupEngine) verifyBackup(backupDir string, backupInfo *metadata.BackupInfo) error {
	txnlogDir := filepath.Join(backupDir, "txnlogs")
	snapshotDir := filepath.Join(backupDir, "snapshots")

	results, err := zkfile.ValidateBackupFiles(snapshotDir, txnlogDir)
	if err != nil {
		return err
	}

	validFiles := 0
	repairedFiles := 0
	corruptedFiles := 0

	for path, result := range results {
		if result.IsValid {
			validFiles++
		} else {
			corruptedFiles++
			e.logger.Warn("File validation failed",
				zap.String("file", path),
				zap.String("corruption_type", result.CorruptionType))

			// Try to repair
			if zkfile.DetermineFileType(path) == zkfile.FileTypeTxnLog {
				e.logger.Info("Attempting to repair", zap.String("file", path))
				repairedPath := path + ".repaired"
				if _, err := zkfile.RepairTxnLog(path, repairedPath); err == nil {
					// Replace original with repaired
					zkfile.RemoveFile(path)
					zkfile.CopyFile(repairedPath, path)
					zkfile.RemoveFile(repairedPath)
					repairedFiles++
					e.logger.Info("File repaired successfully", zap.String("file", path))
				}
			}
		}
	}

	backupInfo.UpdateValidation(validFiles, corruptedFiles, repairedFiles)

	e.logger.Info("Verification completed", zap.Int("total", len(results)),
		zap.Int("valid", validFiles), zap.Int("corrupted", corruptedFiles), zap.Int("repaired", repairedFiles))

	return nil
}

// saveMetadata saves backup metadata
func (e *BackupEngine) saveMetadata(backupDir string, backupInfo *metadata.BackupInfo) error {
	metadataDir := filepath.Join(backupDir, "metadata")

	// Save backup_info.json
	infoPath := filepath.Join(metadataDir, "backup_info.json")
	if err := backupInfo.SaveToFile(infoPath); err != nil {
		return err
	}

	// Save MANIFEST.txt
	manifestPath := filepath.Join(metadataDir, "MANIFEST.txt")
	manifest := backupInfo.GenerateManifest()
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		e.logger.Warn("Failed to write manifest", zap.Error(err))
	}

	e.logger.Info("Metadata saved", zap.String("path", metadataDir))

	return nil
}
