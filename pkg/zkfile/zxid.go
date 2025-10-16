package zkfile

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ZXID is a ZooKeeper Transaction ID (64-bit)
type ZXID uint64

// FileType represents the type of file
type FileType int

const (
	FileTypeTxnLog FileType = iota
	FileTypeSnapshot
	FileTypeUnknown
)

func (ft FileType) String() string {
	switch ft {
	case FileTypeTxnLog:
		return "txnlog"
	case FileTypeSnapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}

// String returns the hexadecimal representation of ZXID
func (z ZXID) String() string {
	return fmt.Sprintf("0x%x", uint64(z))
}

// Hex returns hexadecimal string (without 0x prefix)
func (z ZXID) Hex() string {
	return fmt.Sprintf("%x", uint64(z))
}

// Compare compares two ZXIDs
// Returns: -1 (z < other), 0 (z == other), 1 (z > other)
func (z ZXID) Compare(other ZXID) int {
	if z > other {
		return 1
	}
	if z < other {
		return -1
	}
	return 0
}

// MaxZXID returns the larger of two ZXIDs
func MaxZXID(a, b ZXID) ZXID {
	if a > b {
		return a
	}
	return b
}

// MinZXID returns the smaller of two ZXIDs
func MinZXID(a, b ZXID) ZXID {
	if a < b {
		return a
	}
	return b
}

// DetermineFileType determines the file type
func DetermineFileType(filename string) FileType {
	base := filepath.Base(filename)

	if strings.HasPrefix(base, "log.") {
		return FileTypeTxnLog
	}
	if strings.HasPrefix(base, "snapshot.") {
		return FileTypeSnapshot
	}

	return FileTypeUnknown
}

// FormatZxidFileName formats a ZXID as a filename
func FormatZxidFileName(fileType FileType, zxid ZXID) string {
	switch fileType {
	default:
		return ""
	case FileTypeTxnLog:
		return "log." + zxid.Hex()
	case FileTypeSnapshot:
		return "snapshot." + zxid.Hex()
	}
}

// ParseZxidFromFileName parses ZXID from filename
// Supported formats:
// - log.100000000
// - snapshot.100000000
func ParseZxidFromFileName(filename string) (ZXID, error) {
	base := filepath.Base(filename)

	// Split by dot
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return 0, NewUserError("invalid file name").WithContext("filename", filename)
	}

	// Last part should be hex ZXID
	zxidStr := parts[len(parts)-1]

	// Parse hexadecimal
	zxid, err := strconv.ParseUint(zxidStr, 16, 64)
	if err != nil {
		return 0, NewUserError("failed to parse zxid").
			WithError(err).WithContext("filename", filename).WithContext("zxid_str", zxidStr)
	}

	return ZXID(zxid), nil
}
