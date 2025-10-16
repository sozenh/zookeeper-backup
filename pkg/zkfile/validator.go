package zkfile

import (
	"fmt"
)

// ValidationResult is the result of validation
type ValidationResult struct {
	IsValid               bool   // Whether it's completely valid
	ValidTransactionCount int    // Number of valid transactions
	LastValidPos          int64  // Position of last valid transaction
	LastValidZxid         ZXID   // ZXID of last valid transaction
	CorruptionType        string // Type of corruption
	Transactions          []ZXID // List of all transaction ZXIDs
}

// GetValidationSummary generates a validation summary
func GetValidationSummary(results map[string]*ValidationResult) string {
	totalFiles := len(results)
	validFiles := 0
	corruptedFiles := 0

	for _, result := range results {
		if result.IsValid {
			validFiles++
		} else {
			corruptedFiles++
		}
	}

	return fmt.Sprintf("Total: %d, Valid: %d, Corrupted: %d", totalFiles, validFiles, corruptedFiles)
}

// ValidateBackupFiles validates all files in the backup directories
func ValidateBackupFiles(snapshotDir, txnlogDir string) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	// Validate all txnlog files
	txnlogs, err := ListTxnLogFiles(txnlogDir)
	if err != nil {
		return nil, err
	}

	// Validate all snapshot files
	snapshots, err := ListSnapshotFiles(snapshotDir)
	if err != nil {
		return nil, err
	}

	for _, txnlog := range txnlogs {
		result, err := ValidateTxnLog(txnlog)
		if err != nil {
			return nil, err
		}
		results[txnlog] = result
	}

	for _, snapshot := range snapshots {
		err = ValidateSnapshot(snapshot)
		if err == nil {
			results[snapshot] = &ValidationResult{IsValid: true}
		} else {
			results[snapshot] = &ValidationResult{IsValid: false, CorruptionType: err.Error()}
		}
	}

	return results, nil
}
