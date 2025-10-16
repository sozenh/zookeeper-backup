package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zookeeper-backup/pkg/engine"
)

// NewBackupCmd creates the backup command
func NewBackupCmd() *cobra.Command {
	var config engine.BackupConfig

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup ZooKeeper data",
		Long: `Create a full backup of ZooKeeper data including snapshots and transaction logs.

Example:
  zkbackup backup \
    --zk-data-dir /zookeeper/data/version-2 \
    --zk-log-dir /zookeeper/datalog/version-2 \
    --output-dir /backup/zookeeper \
    --zk-host localhost:2181`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Verbose = verbose

			backupEngine := engine.NewBackupEngine(&config)
			return backupEngine.Run()
		},
	}

	// Flags
	cmd.Flags().StringVar(&config.ZkDataDir, "zk-data-dir", "", "ZooKeeper dataDir path (required)")
	cmd.Flags().StringVar(&config.ZkLogDir, "zk-log-dir", "", "ZooKeeper dataLogDir path (required)")
	cmd.Flags().StringVar(&config.OutputDir, "output-dir", "", "Backup output directory (required)")
	cmd.Flags().StringVar(&config.ZkHost, "zk-host", "localhost:2181", "ZooKeeper host address")
	cmd.Flags().StringVar(&config.BackupID, "backup-id", "", "Backup ID (optional, auto-generated if not set)")
	cmd.Flags().BoolVar(&config.Verify, "verify", true, "Verify backup after completion")
	cmd.Flags().StringVar(&config.Compression, "compression", "none", "Compression: none|gzip|zstd")

	// Required flags
	cmd.MarkFlagRequired("zk-data-dir")
	cmd.MarkFlagRequired("zk-log-dir")
	cmd.MarkFlagRequired("output-dir")

	return cmd
}
