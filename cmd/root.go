package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/zookeeper-backup/pkg/utils"
)

var (
	cfgFile string
	verbose bool
)

// NewRootCmd creates the root command
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "github.com/zookeeper-backup",
		Short: "ZooKeeper Backup and Restore Tool",
		Long: `zkbackup is a reliable backup and restore tool for ZooKeeper.

It provides:
- Full backup of ZooKeeper data
- Reliable restore with validation
- TxnLog verification and repair
- Easy integration with existing systems`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize logger
			level := "info"
			if verbose {
				level = "debug"
			}
			utils.InitLogger(level, "text")
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./zkbackup.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add subcommands
	rootCmd.AddCommand(NewBackupCmd())
	rootCmd.AddCommand(NewRestoreCmd())
	rootCmd.AddCommand(NewVerifyCmd())
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewInfoCmd())
	rootCmd.AddCommand(NewPruneCmd())

	return rootCmd
}

// Execute runs the root command
func Execute(version, commit, date string) error {
	defer utils.Sync()

	rootCmd := NewRootCmd(version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	return nil
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
		viper.AddConfigPath("/etc/zkbackup")
		viper.SetConfigName("github.com/zookeeper-backup")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
