package zkfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SnapshotInfo contains Snapshot file information
type SnapshotInfo struct {
	Name     string `json:"name"`
	Zxid     ZXID   `json:"zxid"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"` // SHA256
}

// GetSnapshotInfo extracts information from a snapshot file
func GetSnapshotInfo(path string) (*SnapshotInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, NewIOError("failed to stat file").WithError(err).WithContext("path", path)
	}

	zxid, err := ParseZxidFromFileName(path)
	if err != nil {
		return nil, err
	}

	checksum, err := calculateFileChecksum(path)
	if err != nil {
		return nil, err
	}

	return &SnapshotInfo{Name: filepath.Base(path), Zxid: zxid, Size: info.Size(), Checksum: checksum}, nil
}

// ListSnapshotFiles lists all snapshot files in the given directory
func ListSnapshotFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, NewIOError("failed to read directory").WithError(err).WithContext("dir", dir)
	}

	var snapshots []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "snapshot.") {
			snapshots = append(snapshots, filepath.Join(dir, name))
		}
	}

	// Sort by ZXID (ascending)
	sort.Slice(snapshots, func(i, j int) bool {
		zxidI, _ := ParseZxidFromFileName(snapshots[i])
		zxidJ, _ := ParseZxidFromFileName(snapshots[j])
		return zxidI < zxidJ
	})

	return snapshots, nil
}

// GetLatestSnapshot returns the snapshot file with the highest ZXID
func GetLatestSnapshot(dir string) (string, ZXID, error) {
	snapshots, err := ListSnapshotFiles(dir)
	if err != nil {
		return "", 0, err
	}

	if len(snapshots) == 0 {
		return "", 0, NewIOError("no snapshot files found").WithContext("dir", dir)
	}

	// Last one has the highest ZXID
	latest := snapshots[len(snapshots)-1]
	zxid, err := ParseZxidFromFileName(latest)
	if err != nil {
		return "", 0, err
	}

	return latest, zxid, nil
}

// ValidateSnapshot validates the integrity of a Snapshot file
func ValidateSnapshot(path string) error {
	// Simple validation: check if file exists and size > 0
	info, err := GetFileInfo(path)
	if err != nil {
		return err
	}

	if info.Size == 0 {
		return NewCorruptionError("empty snapshot file").WithContext("path", path)
	}

	// TODO: Can add more detailed snapshot format validation
	// ZooKeeper snapshot files also have a specific format, can validate file header

	return nil
}
