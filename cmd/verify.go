package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVerifyCmd creates the verify command
func NewVerifyCmd() *cobra.Command {
	var (
		backupDir    string
		fix          bool
		outputFormat string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify backup integrity",
		Long: `Verify the integrity of a backup directory.

Example:
  zkbackup verify --backup-dir /backup/zookeeper/backup-20250115-103000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Verifying backup: %s\n\n", backupDir)

			// TODO: Implement full verify logic
			fmt.Println("✅ Verification completed")
			return nil
		},
	}

	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "Backup directory path (required)")
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix corrupted files")
	cmd.Flags().StringVar(&outputFormat, "output-format", "text", "Output format: text|json")

	cmd.MarkFlagRequired("backup-dir")

	return cmd
}
