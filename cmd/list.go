package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	var (
		backupBaseDir string
		format        string
		sortBy        string
		limit         int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all backups",
		Long: `List all backups in the backup base directory.

Example:
  zkbackup list --backup-base-dir /backup/zookeeper`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Listing backups in: %s\n\n", backupBaseDir)

			// TODO: Implement list logic
			fmt.Println("No backups found")
			return nil
		},
	}

	cmd.Flags().StringVar(&backupBaseDir, "backup-base-dir", "/backup/zookeeper", "Backup base directory")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table|json|simple")
	cmd.Flags().StringVar(&sortBy, "sort-by", "time", "Sort by: time|size|zxid")
	cmd.Flags().IntVar(&limit, "limit", 20, "Limit number of results")

	return cmd
}
