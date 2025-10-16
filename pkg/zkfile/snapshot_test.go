package zkfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSnapshotFiles(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("list snapshot files", func(t *testing.T) {
		// Create test files
		os.WriteFile(filepath.Join(tmpDir, "snapshot.100000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "snapshot.200000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "snapshot.300000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "log.100000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0644)

		files, err := ListSnapshotFiles(tmpDir)
		if err != nil {
			t.Fatalf("ListSnapshotFiles() error = %v", err)
		}

		if len(files) != 3 {
			t.Errorf("Expected 3 snapshot files, got %d", len(files))
		}

		// Verify sorted by ZXID ascending
		if len(files) == 3 {
			zxid1, _ := ParseZxidFromFileName(files[0])
			zxid2, _ := ParseZxidFromFileName(files[1])
			zxid3, _ := ParseZxidFromFileName(files[2])

			if !(zxid1 < zxid2 && zxid2 < zxid3) {
				t.Error("Files should be sorted by ZXID ascending")
			}
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.Mkdir(emptyDir, 0755)

		files, err := ListSnapshotFiles(emptyDir)
		if err != nil {
			t.Fatalf("ListSnapshotFiles() error = %v", err)
		}

		if len(files) != 0 {
			t.Errorf("Expected 0 files, got %d", len(files))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := ListSnapshotFiles("/nonexistent/dir")
		if err == nil {
			t.Error("ListSnapshotFiles() should return error for nonexistent directory")
		}
	})
}

func TestGetLatestSnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("get latest snapshot", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpDir, "snapshot.100000000"), []byte("test1"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "snapshot.200000000"), []byte("test2"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "snapshot.300000000"), []byte("test3"), 0644)

		path, zxid, err := GetLatestSnapshot(tmpDir)
		if err != nil {
			t.Fatalf("GetLatestSnapshot() error = %v", err)
		}

		if zxid != ZXID(0x300000000) {
			t.Errorf("Latest ZXID = %v, want 0x300000000", zxid)
		}

		if filepath.Base(path) != "snapshot.300000000" {
			t.Errorf("Latest file = %v, want snapshot.300000000", filepath.Base(path))
		}
	})

	t.Run("no snapshot files", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "nosnapshots")
		os.Mkdir(emptyDir, 0755)

		_, _, err := GetLatestSnapshot(emptyDir)
		if err == nil {
			t.Error("GetLatestSnapshot() should return error when no snapshots found")
		}
	})
}

func TestGetSnapshotInfo(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid snapshot", func(t *testing.T) {
		path := filepath.Join(tmpDir, "snapshot.100000000")
		content := []byte("test snapshot content")
		os.WriteFile(path, content, 0644)

		info, err := GetSnapshotInfo(path)
		if err != nil {
			t.Fatalf("GetSnapshotInfo() error = %v", err)
		}

		if info.Name != "snapshot.100000000" {
			t.Errorf("Name = %v, want snapshot.100000000", info.Name)
		}
		if info.Zxid != ZXID(0x100000000) {
			t.Errorf("Zxid = %v, want 0x100000000", info.Zxid)
		}
		if info.Size != int64(len(content)) {
			t.Errorf("Size = %v, want %v", info.Size, len(content))
		}
		if info.Checksum == "" {
			t.Error("Checksum should not be empty")
		}
		if len(info.Checksum) < 10 {
			t.Errorf("Checksum seems too short: %v", info.Checksum)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := GetSnapshotInfo("/nonexistent/snapshot.100")
		if err == nil {
			t.Error("GetSnapshotInfo() should return error for nonexistent file")
		}
	})

	t.Run("invalid filename", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.txt")
		os.WriteFile(path, []byte("test"), 0644)

		_, err := GetSnapshotInfo(path)
		if err == nil {
			t.Error("GetSnapshotInfo() should return error for invalid filename")
		}
	})
}

func TestCopySnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy snapshot successfully", func(t *testing.T) {
		src := filepath.Join(tmpDir, "snapshot.100000000")
		dst := filepath.Join(tmpDir, "copy", "snapshot.100000000")
		content := []byte("snapshot data")

		os.WriteFile(src, content, 0644)

		err := CopySnapshot(src, dst)
		if err != nil {
			t.Fatalf("CopySnapshot() error = %v", err)
		}

		if !FileExists(dst) {
			t.Error("Destination snapshot should exist")
		}

		dstContent, _ := os.ReadFile(dst)
		if string(dstContent) != string(content) {
			t.Error("Snapshot content should match after copy")
		}
	})

	t.Run("source not found", func(t *testing.T) {
		err := CopySnapshot("/nonexistent/snapshot.100", filepath.Join(tmpDir, "dest"))
		if err == nil {
			t.Error("CopySnapshot() should return error for nonexistent source")
		}
	})
}

func TestValidateSnapshot(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid snapshot", func(t *testing.T) {
		path := filepath.Join(tmpDir, "snapshot.100")
		os.WriteFile(path, []byte("snapshot content"), 0644)

		err := ValidateSnapshot(path)
		if err != nil {
			t.Errorf("ValidateSnapshot() error = %v, want nil", err)
		}
	})

	t.Run("empty snapshot", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty_snapshot.100")
		os.WriteFile(path, []byte{}, 0644)

		err := ValidateSnapshot(path)
		if err == nil {
			t.Error("ValidateSnapshot() should return error for empty file")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		err := ValidateSnapshot("/nonexistent/snapshot.100")
		if err == nil {
			t.Error("ValidateSnapshot() should return error for nonexistent file")
		}
	})
}
