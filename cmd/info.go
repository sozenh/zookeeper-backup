package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewInfoCmd creates the info command
func NewInfoCmd() *cobra.Command {
	var (
		backupBaseDir string
		format        string
	)

	cmd := &cobra.Command{
		Use:   "info <backup-id>",
		Short: "Show backup details",
		Long: `Show detailed information about a specific backup.

Example:
  zkbackup info backup-20250115-103000 --backup-base-dir /backup/zookeeper`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupID := args[0]

			fmt.Printf("Backup ID: %s\n", backupID)
			fmt.Printf("Base Dir: %s\n\n", backupBaseDir)

			// TODO: Implement info logic
			fmt.Println("Backup not found")
			return nil
		},
	}

	cmd.Flags().StringVar(&backupBaseDir, "backup-base-dir", "/backup/zookeeper", "Backup base directory")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text|json")

	return cmd
}
