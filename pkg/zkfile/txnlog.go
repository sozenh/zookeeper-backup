package zkfile

import (
	"encoding/binary"
	"hash/adler32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// LogVersion is the supported log version
	LogVersion = 2

	// MagicNumber is the magic number for TxnLog files "ZKLG"
	MagicNumber = 0x5a4b4c47

	// MaxRecordSize is the maximum size of a single record (10MB)
	MaxRecordSize = 10 * 1024 * 1024

	// HeaderSize is the size of the file header (bytes)
	HeaderSize = 16 // 4(MagicNumber) + 4(LogVersion) + 8(DbID)
)

// TxnLogInfo contains TxnLog file information
type TxnLogInfo struct {
	Name             string `json:"name"`
	StartZxid        ZXID   `json:"start_zxid"`
	EndZxid          ZXID   `json:"end_zxid"`
	Size             int64  `json:"size"`
	Status           string `json:"status"` // valid, truncated, corrupted
	TransactionCount int    `json:"transaction_count"`
	Note             string `json:"note,omitempty"`
}

// GetTxnLogInfo extracts information from a txnlog file
func GetTxnLogInfo(path string) (*TxnLogInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, NewIOError("failed to stat file").WithError(err).WithContext("path", path)
	}

	startZxid, err := ParseZxidFromFileName(path)
	if err != nil {
		return nil, err
	}

	// Validate the file and get end ZXID
	result, err := ValidateTxnLog(path)
	if err != nil {
		return nil, err
	}

	status := "valid"
	if !result.IsValid {
		status = "corrupted"
	}

	var endZxid ZXID
	if len(result.Transactions) > 0 {
		endZxid = result.Transactions[len(result.Transactions)-1]
	} else {
		endZxid = startZxid
	}

	return &TxnLogInfo{
		Name:             filepath.Base(path),
		StartZxid:        startZxid,
		EndZxid:          endZxid,
		Size:             info.Size(),
		Status:           status,
		TransactionCount: result.ValidTransactionCount,
	}, nil
}

// TxnLogHeader is the header of a TxnLog file
type TxnLogHeader struct {
	Magic   uint32 // 0x5a4b4c47 ("ZKLG")
	Version uint32 // Version number, usually 2
	DbId    uint64 // Cluster Database ID
}

// Transaction represents a transaction record
type Transaction struct {
	Length   int32  // Record body length
	Data     []byte // Transaction data
	Checksum int64  // Adler32 or CRC32 checksum

	ClientId  int64 // Client ID
	Cxid      int32 // Client transaction ID
	Zxid      ZXID  // ZooKeeper's transaction ID
	Timestamp int64 // Timestamp
	Type      int32 // Transaction type

}

// parse parses transaction data
func (t *Transaction) parse() error {
	if len(t.Data) < 32 {
		return NewCorruptionError("invalid data length").WithContext("length", len(t.Data))
	}

	// Parse fields (all fields are BigEndian)
	t.ClientId = int64(binary.BigEndian.Uint64(t.Data[0:8]))
	t.Cxid = int32(binary.BigEndian.Uint32(t.Data[8:12]))
	t.Zxid = ZXID(binary.BigEndian.Uint64(t.Data[12:20]))
	t.Timestamp = int64(binary.BigEndian.Uint64(t.Data[20:28]))
	t.Type = int32(binary.BigEndian.Uint32(t.Data[28:32]))

	return nil
}

// TxnLogWriter is a writer for TxnLog files
type TxnLogWriter struct {
	path string
	file *os.File
}

// TxnLogReader is a reader for TxnLog files
type TxnLogReader struct {
	path   string
	file   *os.File
	header *TxnLogHeader
}

// CreateTxnLog creates a new TxnLog file
func CreateTxnLog(path string, header *TxnLogHeader) (*TxnLogWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, NewIOError("failed to create txnlog").WithError(err).WithContext("path", path)
	}

	writer := &TxnLogWriter{
		file: f,
		path: path,
	}

	err = writer.writeHeader(header)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}

	return writer, nil
}

// Sync syncs the file to disk
func (w *TxnLogWriter) Sync() error {
	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}

// Close closes the file
func (w *TxnLogWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// writeHeader writes the file header
func (w *TxnLogWriter) writeHeader(header *TxnLogHeader) error {
	err := binary.Write(w.file, binary.BigEndian, header.Magic)
	if err != nil {
		return NewIOError("failed to write magic").WithError(err).WithContext("path", w.path)
	}

	err = binary.Write(w.file, binary.BigEndian, header.Version)
	if err != nil {
		return NewIOError("failed to write version").WithError(err).WithContext("path", w.path)
	}

	err = binary.Write(w.file, binary.BigEndian, header.DbId)
	if err != nil {
		return NewIOError("failed to write dbid").WithError(err).WithContext("path", w.path)
	}

	return nil
}

// WriteTransaction writes a transaction
func (w *TxnLogWriter) WriteTransaction(txn *Transaction) error {
	// Write checksum
	err := binary.Write(w.file, binary.BigEndian, txn.Checksum)
	if err != nil {
		return NewIOError("failed to write checksum").WithError(err).WithContext("path", w.path)
	}

	// Write length
	err = binary.Write(w.file, binary.BigEndian, txn.Length)
	if err != nil {
		return NewIOError("failed to write length").WithError(err).WithContext("path", w.path)
	}

	// Write data
	_, err = w.file.Write(txn.Data)
	if err != nil {
		return NewIOError("failed to write data").WithError(err).WithContext("path", w.path)
	}

	return nil
}

// OpenTxnLog opens a TxnLog file
func OpenTxnLog(path string) (*TxnLogReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, NewIOError("failed to open txnlog").WithError(err).WithContext("path", path)
	}

	reader := &TxnLogReader{
		file: f,
		path: path,
	}

	err = reader.readHeader()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return reader, nil
}

// Path returns the file path
func (r *TxnLogReader) Path() string {
	return r.path
}

// Header returns the file header
func (r *TxnLogReader) Header() *TxnLogHeader {
	return r.header
}

// Seek seeks to the specified position
func (r *TxnLogReader) Seek(offset int64, whence int) (int64, error) {
	return r.file.Seek(offset, whence)
}

// Close closes the file
func (r *TxnLogReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// CurrentPosition returns the current file position
func (r *TxnLogReader) CurrentPosition() (int64, error) {
	return r.file.Seek(0, io.SeekCurrent)
}

// readHeader reads the file header
func (r *TxnLogReader) readHeader() error {
	r.header = &TxnLogHeader{}

	// Read Magic Number (4 bytes)
	err := binary.Read(r.file, binary.BigEndian, &r.header.Magic)
	if err != nil {
		if err == io.EOF {
			return NewCorruptionError("empty file").WithContext("path", r.path)
		}
		return NewIOError("failed to read magic").WithError(err).WithContext("path", r.path)
	}
	if r.header.Magic != MagicNumber {
		return NewCorruptionError("invalid magic number").
			WithContext("path", r.path).WithContext("magic", r.header.Magic).WithContext("expected", MagicNumber)
	}

	// Read Log Version (4 bytes)
	err = binary.Read(r.file, binary.BigEndian, &r.header.Version)
	if err != nil {
		return NewIOError("failed to read version").WithError(err).WithContext("path", r.path)
	}
	if r.header.Version != LogVersion {
		return NewCorruptionError("unsupported version").
			WithContext("path", r.path).WithContext("version", r.header.Version).WithContext("expected", LogVersion)
	}

	// Read DbId (8 bytes)
	err = binary.Read(r.file, binary.BigEndian, &r.header.DbId)
	if err != nil {
		return NewIOError("failed to read dbid").WithError(err).WithContext("path", r.path)
	}

	return nil
}

// ReadTransaction reads the next transaction
func (r *TxnLogReader) ReadTransaction() (*Transaction, error) {
	txn := &Transaction{}

	// Read checksum (8 bytes)
	err := binary.Read(r.file, binary.BigEndian, &txn.Checksum)
	if err != nil {
		return nil, err
	}

	// Read length (4 bytes)
	err = binary.Read(r.file, binary.BigEndian, &txn.Length)
	if err != nil {
		return nil, NewCorruptionError("failed to read length").WithContext("path", r.path)
	}

	if txn.Length <= 0 || txn.Length > MaxRecordSize {
		return nil, NewCorruptionError("invalid record length").
			WithContext("path", r.path).WithContext("length", txn.Length).WithContext("max", MaxRecordSize)
	}

	// Read record body
	txn.Data = make([]byte, txn.Length)
	_, err = io.ReadFull(r.file, txn.Data)
	if err != nil {
		return nil, NewCorruptionError("failed to read body").WithContext("path", r.path).WithContext("length", txn.Length)
	}

	// Verify checksum
	calculated := int64(adler32.Checksum(txn.Data))
	if calculated != txn.Checksum {
		return nil, NewCorruptionError("checksum mismatch").
			WithContext("path", r.path).WithContext("expected", txn.Checksum).WithContext("calculated", calculated)
	}

	// Parse transaction fields
	if err = txn.parse(); err != nil {
		return nil, NewCorruptionError("failed to parse transaction").WithError(err).WithContext("path", r.path)
	}

	return txn, nil
}

// ListTxnLogFiles lists all txnlog files in the given directory
func ListTxnLogFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, NewIOError("failed to read directory").WithError(err).WithContext("dir", dir)
	}

	var txnlogs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "log.") {
			txnlogs = append(txnlogs, filepath.Join(dir, name))
		}
	}

	// Sort by ZXID (ascending)
	sort.Slice(txnlogs, func(i, j int) bool {
		zxidI, _ := ParseZxidFromFileName(txnlogs[i])
		zxidJ, _ := ParseZxidFromFileName(txnlogs[j])
		return zxidI < zxidJ
	})

	return txnlogs, nil
}

// ValidateTxnLog validates the integrity of a TxnLog file
func ValidateTxnLog(path string) (*ValidationResult, error) {
	reader, err := OpenTxnLog(path)
	if err != nil {
		return nil, NewIOError("failed to open log").WithError(err).WithContext("path", path)
	}
	defer func() { _ = reader.Close() }()

	result := &ValidationResult{
		IsValid:      true,
		Transactions: make([]ZXID, 0),
	}

	txnCount := 0
	for {
		// Record current position
		pos, err := reader.CurrentPosition()
		if err != nil {
			return nil, NewIOError("failed to get position").WithError(err).WithContext("path", path)
		}

		// Try to read next transaction
		txn, err := reader.ReadTransaction()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Found corruption
			result.IsValid = false
			result.LastValidPos = pos
			result.CorruptionType = err.Error()

			// If at least one transaction was read, record the last valid ZXID
			if txnCount > 0 {
				result.LastValidZxid = result.Transactions[txnCount-1]
			}

			break
		}

		// Record transaction
		txnCount++
		result.ValidTransactionCount = txnCount
		result.LastValidPos = pos
		result.LastValidZxid = txn.Zxid
		result.Transactions = append(result.Transactions, txn.Zxid)
	}

	return result, nil
}

// RepairTxnLog repairs a corrupted TxnLog file
// Repairs by truncating to the last valid transaction
func RepairTxnLog(inputPath, outputPath string) (*ValidationResult, error) {
	// First validate the file
	result, err := ValidateTxnLog(inputPath)
	if err != nil {
		return nil, err
	}

	if result.IsValid {
		// File is intact, no repair needed
		return result, nil
	}

	// File is corrupted, needs truncation
	if result.ValidTransactionCount == 0 {
		// No valid transactions, cannot repair
		return result, NewCorruptionError("no valid transactions found").WithContext("path", inputPath)
	}

	// Truncate to the last valid ZXID
	_, err = CopyTxnLogUntilZxid(inputPath, outputPath, result.LastValidZxid)
	if err != nil {
		return result, err
	}

	// Validate repaired file
	repairedResult, err := ValidateTxnLog(outputPath)
	if err != nil {
		return result, NewIOError("repaired file validation failed").WithError(err).WithContext("output_path", outputPath)
	}

	if !repairedResult.IsValid {
		return result, NewCorruptionError("repair file validation failed, file still corrupted").WithContext("output_path", outputPath)
	}

	return repairedResult, nil
}

// CopyTxnLogUntilZxid copies a TxnLog file until the specified ZXID
func CopyTxnLogUntilZxid(inputPath, outputPath string, maxZxid ZXID) (int, error) {
	return copyTxnLogWithFilter(inputPath, outputPath, func(zxid ZXID) bool {
		return zxid <= maxZxid
	})
}

// CopyTxnLogFromZxid copies a TxnLog file from the specified ZXID onward
func CopyTxnLogFromZxid(inputPath, outputPath string, minZxid ZXID) (int, error) {
	return copyTxnLogWithFilter(inputPath, outputPath, func(zxid ZXID) bool {
		return zxid >= minZxid
	})
}

// copyTxnLogWithFilter copies transactions based on a filter
func copyTxnLogWithFilter(inputPath, outputPath string, filter func(ZXID) bool) (int, error) {
	reader, err := OpenTxnLog(inputPath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = reader.Close() }()

	writer, err := CreateTxnLog(outputPath, reader.Header())
	if err != nil {
		return 0, err
	}
	defer func() { _ = writer.Close() }()

	copiedCount := 0

	// Process transactions one by one
	for {
		txn, err := reader.ReadTransaction()
		if err != nil {
			break // Encountered corrupted transaction, stop
		}

		// Apply filter
		if !filter(txn.Zxid) {
			continue
		}

		// Write transaction
		if err = writer.WriteTransaction(txn); err != nil {
			return 0, err
		}

		copiedCount++
	}

	if copiedCount == 0 {
		return 0, NewUserError("no transactions to copy").WithContext("input_path", inputPath)
	}

	if err = writer.Sync(); err != nil {
		return 0, NewIOError("failed to sync").WithError(err).WithContext("output_path", outputPath)
	}

	return copiedCount, nil
}
