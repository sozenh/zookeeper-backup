package zkfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		os.WriteFile(path, content, 0644)

		info, err := GetFileInfo(path)
		if err != nil {
			t.Fatalf("GetFileInfo() error = %v", err)
		}

		if info.Name != "test.txt" {
			t.Errorf("Name = %v, want test.txt", info.Name)
		}
		if info.Size != int64(len(content)) {
			t.Errorf("Size = %v, want %v", info.Size, len(content))
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := GetFileInfo("/nonexistent/file.txt")
		if err == nil {
			t.Error("GetFileInfo() should return error for nonexistent file")
		}
	})
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy file successfully", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source.txt")
		dst := filepath.Join(tmpDir, "dest.txt")
		content := []byte("test content for copy")

		os.WriteFile(src, content, 0644)

		err := CopyFile(src, dst)
		if err != nil {
			t.Fatalf("CopyFile() error = %v", err)
		}

		// Verify destination file exists
		if !FileExists(dst) {
			t.Error("Destination file should exist after copy")
		}

		// Verify content matches
		dstContent, _ := os.ReadFile(dst)
		if string(dstContent) != string(content) {
			t.Errorf("Content mismatch: got %v, want %v", string(dstContent), string(content))
		}
	})

	t.Run("copy to subdirectory", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source2.txt")
		dst := filepath.Join(tmpDir, "subdir", "dest2.txt")
		content := []byte("test content")

		os.WriteFile(src, content, 0644)

		err := CopyFile(src, dst)
		if err != nil {
			t.Fatalf("CopyFile() error = %v", err)
		}

		if !FileExists(dst) {
			t.Error("Destination file should exist")
		}
	})

	t.Run("source file not found", func(t *testing.T) {
		src := "/nonexistent/source.txt"
		dst := filepath.Join(tmpDir, "dest.txt")

		err := CopyFile(src, dst)
		if err == nil {
			t.Error("CopyFile() should return error for nonexistent source")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("create new directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "newdir")

		err := EnsureDir(dir)
		if err != nil {
			t.Fatalf("EnsureDir() error = %v", err)
		}

		if !DirExists(dir) {
			t.Error("Directory should exist after EnsureDir")
		}
	})

	t.Run("directory already exists", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "existing")
		os.Mkdir(dir, 0755)

		err := EnsureDir(dir)
		if err != nil {
			t.Errorf("EnsureDir() should not error for existing directory: %v", err)
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "a", "b", "c")

		err := EnsureDir(dir)
		if err != nil {
			t.Fatalf("EnsureDir() error = %v", err)
		}

		if !DirExists(dir) {
			t.Error("Nested directory should exist")
		}
	})
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("file exists", func(t *testing.T) {
		path := filepath.Join(tmpDir, "exists.txt")
		os.WriteFile(path, []byte("test"), 0644)

		if !FileExists(path) {
			t.Error("FileExists() should return true for existing file")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		path := filepath.Join(tmpDir, "notexists.txt")

		if FileExists(path) {
			t.Error("FileExists() should return false for nonexistent file")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "dir")
		os.Mkdir(dir, 0755)

		if !FileExists(dir) {
			t.Error("FileExists() should return true for directory")
		}
	})
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("directory exists", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "testdir")
		os.Mkdir(dir, 0755)

		if !DirExists(dir) {
			t.Error("DirExists() should return true for existing directory")
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "notexists")

		if DirExists(dir) {
			t.Error("DirExists() should return false for nonexistent directory")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		file := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(file, []byte("test"), 0644)

		if DirExists(file) {
			t.Error("DirExists() should return false for file")
		}
	})
}

func TestRemoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("remove existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "remove.txt")
		os.WriteFile(path, []byte("test"), 0644)

		err := RemoveFile(path)
		if err != nil {
			t.Fatalf("RemoveFile() error = %v", err)
		}

		if FileExists(path) {
			t.Error("File should not exist after RemoveFile")
		}
	})

	t.Run("remove nonexistent file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "notexists.txt")

		err := RemoveFile(path)
		if err == nil {
			t.Error("RemoveFile() should return error for nonexistent file")
		}
	})
}

func TestRemoveDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("remove empty directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "emptydir")
		os.Mkdir(dir, 0755)

		err := RemoveDir(dir)
		if err != nil {
			t.Fatalf("RemoveDir() error = %v", err)
		}

		if DirExists(dir) {
			t.Error("Directory should not exist after RemoveDir")
		}
	})

	t.Run("remove directory with contents", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "fulldir")
		os.Mkdir(dir, 0755)
		os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)

		err := RemoveDir(dir)
		if err != nil {
			t.Fatalf("RemoveDir() error = %v", err)
		}

		if DirExists(dir) {
			t.Error("Directory should not exist after RemoveDir")
		}
	})
}

func TestGetDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "emptysize")
		os.Mkdir(dir, 0755)

		size, err := GetDirSize(dir)
		if err != nil {
			t.Fatalf("GetDirSize() error = %v", err)
		}

		if size != 0 {
			t.Errorf("Empty directory size = %d, want 0", size)
		}
	})

	t.Run("directory with files", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "withfiles")
		os.Mkdir(dir, 0755)

		content1 := []byte("file1 content")
		content2 := []byte("file2 content longer")

		os.WriteFile(filepath.Join(dir, "file1.txt"), content1, 0644)
		os.WriteFile(filepath.Join(dir, "file2.txt"), content2, 0644)

		size, err := GetDirSize(dir)
		if err != nil {
			t.Fatalf("GetDirSize() error = %v", err)
		}

		expectedSize := int64(len(content1) + len(content2))
		if size != expectedSize {
			t.Errorf("Directory size = %d, want %d", size, expectedSize)
		}
	})

	t.Run("directory with subdirectories", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "withsub")
		subdir := filepath.Join(dir, "subdir")
		os.MkdirAll(subdir, 0755)

		content1 := []byte("root file")
		content2 := []byte("sub file")

		os.WriteFile(filepath.Join(dir, "root.txt"), content1, 0644)
		os.WriteFile(filepath.Join(subdir, "sub.txt"), content2, 0644)

		size, err := GetDirSize(dir)
		if err != nil {
			t.Fatalf("GetDirSize() error = %v", err)
		}

		expectedSize := int64(len(content1) + len(content2))
		if size != expectedSize {
			t.Errorf("Directory size = %d, want %d", size, expectedSize)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := GetDirSize("/nonexistent/dir")
		if err == nil {
			t.Error("GetDirSize() should return error for nonexistent directory")
		}
	})
}

func TestListTxnLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("list txnlog files", func(t *testing.T) {
		// Create some test files
		os.WriteFile(filepath.Join(tmpDir, "log.100000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "log.200000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "snapshot.100000000"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0644)

		files, err := ListTxnLogFiles(tmpDir)
		if err != nil {
			t.Fatalf("ListTxnLogFiles() error = %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 txnlog files, got %d", len(files))
		}

		// Verify they are sorted by ZXID
		if len(files) == 2 {
			zxid1, _ := ParseZxidFromFileName(files[0])
			zxid2, _ := ParseZxidFromFileName(files[1])
			if zxid1 > zxid2 {
				t.Error("Files should be sorted by ZXID ascending")
			}
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.Mkdir(emptyDir, 0755)

		files, err := ListTxnLogFiles(emptyDir)
		if err != nil {
			t.Fatalf("ListTxnLogFiles() error = %v", err)
		}

		if len(files) != 0 {
			t.Errorf("Expected 0 files, got %d", len(files))
		}
	})
}
