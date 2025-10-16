package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zookeeper-backup/pkg/engine"
)

// NewRestoreCmd creates the restore command
func NewRestoreCmd() *cobra.Command {
	var config engine.RestoreConfig

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore ZooKeeper data from backup",
		Long: `Restore ZooKeeper data from a backup directory.

Example:
  zkbackup restore \
    --backup-dir /backup/zookeeper/backup-20250115-103000 \
    --zk-data-dir /zookeeper/data/version-2 \
    --zk-log-dir /zookeeper/datalog/version-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Verbose = verbose

			restoreEngine := engine.NewRestoreEngine(&config)
			return restoreEngine.Run()
		},
	}

	// Flags
	cmd.Flags().StringVar(&config.BackupDir, "backup-dir", "", "Backup directory path (required)")
	cmd.Flags().StringVar(&config.ZkDataDir, "zk-data-dir", "", "ZooKeeper dataDir path (required)")
	cmd.Flags().StringVar(&config.ZkLogDir, "zk-log-dir", "", "ZooKeeper dataLogDir path (required)")
	cmd.Flags().BoolVar(&config.Force, "force", false, "Force restore without confirmation")
	cmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Simulate restore without making changes")
	cmd.Flags().BoolVar(&config.SkipVerify, "skip-verify", false, "Skip backup verification before restore")
	cmd.Flags().StringVar(&config.TruncateToZxid, "truncate-to-zxid", "", "Restore to specific ZXID (optional)")

	// Required flags
	cmd.MarkFlagRequired("backup-dir")
	cmd.MarkFlagRequired("zk-data-dir")
	cmd.MarkFlagRequired("zk-log-dir")

	return cmd
}
