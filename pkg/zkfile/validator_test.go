package zkfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBackupFiles(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotDir := filepath.Join(tmpDir, "snapshots")
	txnlogDir := filepath.Join(tmpDir, "txnlogs")

	os.MkdirAll(snapshotDir, 0755)
	os.MkdirAll(txnlogDir, 0755)

	t.Run("validate mixed files", func(t *testing.T) {
		// Create snapshots
		os.WriteFile(filepath.Join(snapshotDir, "snapshot.100"), []byte("data1"), 0644)
		os.WriteFile(filepath.Join(snapshotDir, "snapshot.200"), []byte("data2"), 0644)

		// Create txnlogs
		createTestTxnLog(t, filepath.Join(txnlogDir, "log.100"), 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
		})

		results, err := ValidateBackupFiles(snapshotDir, txnlogDir)
		if err != nil {
			t.Fatalf("ValidateBackupFiles() error = %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Check snapshot results
		for path, result := range results {
			if DetermineFileType(path) == FileTypeSnapshot {
				if !result.IsValid {
					t.Errorf("Snapshot %v should be valid", path)
				}
			}
		}
	})
}

func TestGetValidationSummary(t *testing.T) {
	results := map[string]*ValidationResult{
		"file1": {IsValid: true},
		"file2": {IsValid: true},
		"file3": {IsValid: false, CorruptionType: "ChecksumMismatch"},
	}

	summary := GetValidationSummary(results)

	if summary == "" {
		t.Error("Summary should not be empty")
	}

	// Summary should contain counts
	expected := "Total: 3, Valid: 2, Corrupted: 1"
	if summary != expected {
		t.Errorf("Summary = %v, want %v", summary, expected)
	}
}
