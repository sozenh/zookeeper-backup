package zkfile

import (
	"bytes"
	"encoding/binary"
	"hash/adler32"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create a valid TxnLog file for testing
func createTestTxnLog(t *testing.T, path string, dbId uint64, transactions []testTransaction) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer f.Close()

	// Write header
	binary.Write(f, binary.BigEndian, uint32(MagicNumber))
	binary.Write(f, binary.BigEndian, uint32(LogVersion))
	binary.Write(f, binary.BigEndian, dbId)

	// Write transactions
	for _, txn := range transactions {
		writeTestTransaction(t, f, txn)
	}
}

type testTransaction struct {
	ClientId  int64
	Cxid      int32
	Zxid      ZXID
	Timestamp int64
	Type      int32
}

func writeTestTransaction(t *testing.T, w io.Writer, txn testTransaction) {
	t.Helper()

	// Create transaction body
	var body bytes.Buffer
	binary.Write(&body, binary.BigEndian, txn.ClientId)
	binary.Write(&body, binary.BigEndian, txn.Cxid)
	binary.Write(&body, binary.BigEndian, uint64(txn.Zxid))
	binary.Write(&body, binary.BigEndian, txn.Timestamp)
	binary.Write(&body, binary.BigEndian, txn.Type)

	bodyBytes := body.Bytes()

	// Calculate checksum
	checksum := int64(adler32.Checksum(bodyBytes))

	// Write transaction
	binary.Write(w, binary.BigEndian, checksum)
	binary.Write(w, binary.BigEndian, int32(len(bodyBytes)))
	w.Write(bodyBytes)
}

func TestOpenTxnLog(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "log.100000000")
		createTestTxnLog(t, path, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100000000), Timestamp: 1000, Type: 1},
		})

		reader, err := OpenTxnLog(path)
		if err != nil {
			t.Fatalf("OpenTxnLog() error = %v", err)
		}
		defer reader.Close()

		if reader.header.Magic != MagicNumber {
			t.Errorf("Magic = %x, want %x", reader.header.Magic, MagicNumber)
		}
		if reader.header.Version != LogVersion {
			t.Errorf("Version = %d, want %d", reader.header.Version, LogVersion)
		}
		if reader.header.DbId != 12345 {
			t.Errorf("DbId = %d, want %d", reader.header.DbId, 12345)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := OpenTxnLog("/nonexistent/path/log.100")
		if err == nil {
			t.Error("OpenTxnLog() should return error for nonexistent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.log")
		os.WriteFile(path, []byte{}, 0644)

		_, err := OpenTxnLog(path)
		if err == nil {
			t.Error("OpenTxnLog() should return error for empty file")
		}
	})

	t.Run("invalid magic", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid_magic.log")
		f, _ := os.Create(path)
		binary.Write(f, binary.BigEndian, uint32(0xDEADBEEF)) // Invalid magic
		f.Close()

		_, err := OpenTxnLog(path)
		if err == nil {
			t.Error("OpenTxnLog() should return error for invalid magic")
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid_version.log")
		f, _ := os.Create(path)
		binary.Write(f, binary.BigEndian, uint32(MagicNumber))
		binary.Write(f, binary.BigEndian, uint32(99)) // Invalid version
		f.Close()

		_, err := OpenTxnLog(path)
		if err == nil {
			t.Error("OpenTxnLog() should return error for invalid version")
		}
	})
}

func TestTxnLogReader_ReadTransaction(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("read single transaction", func(t *testing.T) {
		path := filepath.Join(tmpDir, "log.100")
		txns := []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
		}
		createTestTxnLog(t, path, 12345, txns)

		reader, err := OpenTxnLog(path)
		if err != nil {
			t.Fatalf("OpenTxnLog() error = %v", err)
		}
		defer reader.Close()

		txn, err := reader.ReadTransaction()
		if err != nil {
			t.Fatalf("ReadTransaction() error = %v", err)
		}

		if txn.Zxid != ZXID(0x100) {
			t.Errorf("Zxid = %v, want %v", txn.Zxid, ZXID(0x100))
		}
		if txn.ClientId != 1 {
			t.Errorf("ClientId = %v, want 1", txn.ClientId)
		}
		if txn.Cxid != 1 {
			t.Errorf("Cxid = %v, want 1", txn.Cxid)
		}
	})

	t.Run("read multiple transactions", func(t *testing.T) {
		path := filepath.Join(tmpDir, "log.200")
		txns := []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x200), Timestamp: 1000, Type: 1},
			{ClientId: 2, Cxid: 2, Zxid: ZXID(0x201), Timestamp: 2000, Type: 2},
			{ClientId: 3, Cxid: 3, Zxid: ZXID(0x202), Timestamp: 3000, Type: 3},
		}
		createTestTxnLog(t, path, 12345, txns)

		reader, err := OpenTxnLog(path)
		if err != nil {
			t.Fatalf("OpenTxnLog() error = %v", err)
		}
		defer reader.Close()

		for i, expected := range txns {
			txn, err := reader.ReadTransaction()
			if err != nil {
				t.Fatalf("ReadTransaction(%d) error = %v", i, err)
			}
			if txn.Zxid != expected.Zxid {
				t.Errorf("Transaction %d: Zxid = %v, want %v", i, txn.Zxid, expected.Zxid)
			}
		}

		// Should return EOF after all transactions
		_, err = reader.ReadTransaction()
		if err != io.EOF {
			t.Errorf("Expected EOF after all transactions, got %v", err)
		}
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		path := filepath.Join(tmpDir, "log.bad_checksum")
		f, _ := os.Create(path)

		// Write header
		binary.Write(f, binary.BigEndian, uint32(MagicNumber))
		binary.Write(f, binary.BigEndian, uint32(LogVersion))
		binary.Write(f, binary.BigEndian, uint64(12345))

		// Write transaction with wrong checksum
		binary.Write(f, binary.BigEndian, int64(99999)) // Wrong checksum
		binary.Write(f, binary.BigEndian, int32(32))
		body := make([]byte, 32)
		f.Write(body)
		f.Close()

		reader, _ := OpenTxnLog(path)
		defer reader.Close()

		_, err := reader.ReadTransaction()
		if err == nil {
			t.Error("ReadTransaction() should return error for checksum mismatch")
		}
	})
}

func TestTxnLogReader_CurrentPosition(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "log.300")
	createTestTxnLog(t, path, 12345, []testTransaction{
		{ClientId: 1, Cxid: 1, Zxid: ZXID(0x300), Timestamp: 1000, Type: 1},
	})

	reader, _ := OpenTxnLog(path)
	defer reader.Close()

	pos1, err := reader.CurrentPosition()
	if err != nil {
		t.Fatalf("CurrentPosition() error = %v", err)
	}

	reader.ReadTransaction()

	pos2, err := reader.CurrentPosition()
	if err != nil {
		t.Fatalf("CurrentPosition() error = %v", err)
	}

	if pos2 <= pos1 {
		t.Errorf("Position should advance after reading transaction: %d -> %d", pos1, pos2)
	}
}

func TestTxnLogReader_Seek(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "log.400")
	createTestTxnLog(t, path, 12345, []testTransaction{
		{ClientId: 1, Cxid: 1, Zxid: ZXID(0x400), Timestamp: 1000, Type: 1},
	})

	reader, _ := OpenTxnLog(path)
	defer reader.Close()

	// Seek to beginning
	pos, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if pos != 0 {
		t.Errorf("Seek(0, SeekStart) = %d, want 0", pos)
	}

	// Seek to header end
	pos, err = reader.Seek(HeaderSize, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if pos != HeaderSize {
		t.Errorf("Seek(HeaderSize, SeekStart) = %d, want %d", pos, HeaderSize)
	}
}

func TestCreateTxnLog(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("create and write", func(t *testing.T) {
		path := filepath.Join(tmpDir, "new_log.100")
		header := &TxnLogHeader{
			Magic:   MagicNumber,
			Version: LogVersion,
			DbId:    54321,
		}

		writer, err := CreateTxnLog(path, header)
		if err != nil {
			t.Fatalf("CreateTxnLog() error = %v", err)
		}
		defer writer.Close()

		// Write a transaction
		txn := &Transaction{
			Checksum:  12345,
			Length:    32,
			ClientId:  1,
			Cxid:      1,
			Zxid:      ZXID(0x100),
			Timestamp: 1000,
			Type:      1,
			Data:      make([]byte, 32),
		}

		err = writer.WriteTransaction(txn)
		if err != nil {
			t.Fatalf("WriteTransaction() error = %v", err)
		}

		writer.Sync()
		writer.Close()

		// Verify file was created
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("File should exist after CreateTxnLog")
		}

		// Verify we can read it back
		reader, err := OpenTxnLog(path)
		if err != nil {
			t.Fatalf("OpenTxnLog() error = %v", err)
		}
		defer reader.Close()

		if reader.header.DbId != 54321 {
			t.Errorf("DbId = %d, want 54321", reader.header.DbId)
		}
	})

	t.Run("create in nonexistent directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nonexistent", "log.100")
		header := &TxnLogHeader{
			Magic:   MagicNumber,
			Version: LogVersion,
			DbId:    12345,
		}

		_, err := CreateTxnLog(path, header)
		if err == nil {
			t.Error("CreateTxnLog() should return error for nonexistent directory")
		}
	})
}

func TestTransaction_parse(t *testing.T) {
	t.Run("valid transaction data", func(t *testing.T) {
		txn := &Transaction{
			Data: make([]byte, 32),
		}

		binary.BigEndian.PutUint64(txn.Data[0:8], 12345)   // ClientId
		binary.BigEndian.PutUint32(txn.Data[8:12], 1)      // Cxid
		binary.BigEndian.PutUint64(txn.Data[12:20], 0x100) // Zxid
		binary.BigEndian.PutUint64(txn.Data[20:28], 1000)  // Timestamp
		binary.BigEndian.PutUint32(txn.Data[28:32], 2)     // Type

		err := txn.parse()
		if err != nil {
			t.Fatalf("parse() error = %v", err)
		}

		if txn.ClientId != 12345 {
			t.Errorf("ClientId = %d, want 12345", txn.ClientId)
		}
		if txn.Cxid != 1 {
			t.Errorf("Cxid = %d, want 1", txn.Cxid)
		}
		if txn.Zxid != ZXID(0x100) {
			t.Errorf("Zxid = %v, want 0x100", txn.Zxid)
		}
		if txn.Timestamp != 1000 {
			t.Errorf("Timestamp = %d, want 1000", txn.Timestamp)
		}
		if txn.Type != 2 {
			t.Errorf("Type = %d, want 2", txn.Type)
		}
	})

	t.Run("truncated data", func(t *testing.T) {
		txn := &Transaction{
			Data: make([]byte, 10), // Too short
		}

		err := txn.parse()
		if err == nil {
			t.Error("parse() should return error for truncated data")
		}
	})
}

func TestTxnLogReader_Header(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "log.500")
	createTestTxnLog(t, path, 99999, nil)

	reader, _ := OpenTxnLog(path)
	defer reader.Close()

	header := reader.Header()
	if header.DbId != 99999 {
		t.Errorf("Header().DbId = %d, want 99999", header.DbId)
	}
}

func TestTxnLogReader_Path(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "log.600")
	createTestTxnLog(t, path, 12345, nil)

	reader, _ := OpenTxnLog(path)
	defer reader.Close()

	if reader.Path() != path {
		t.Errorf("Path() = %v, want %v", reader.Path(), path)
	}
}

func TestValidateTxnLog(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid txnlog", func(t *testing.T) {
		path := filepath.Join(tmpDir, "log.100")
		createTestTxnLog(t, path, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
			{ClientId: 2, Cxid: 2, Zxid: ZXID(0x101), Timestamp: 2000, Type: 2},
		})

		result, err := ValidateTxnLog(path)
		if err != nil {
			t.Fatalf("ValidateTxnLog() error = %v", err)
		}

		if !result.IsValid {
			t.Error("Result should be valid")
		}
		if result.ValidTransactionCount != 2 {
			t.Errorf("ValidTransactionCount = %d, want 2", result.ValidTransactionCount)
		}
		if len(result.Transactions) != 2 {
			t.Errorf("Transactions count = %d, want 2", len(result.Transactions))
		}
		if result.LastValidZxid != ZXID(0x101) {
			t.Errorf("LastValidZxid = %v, want 0x101", result.LastValidZxid)
		}
	})

	t.Run("empty txnlog", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty_log.100")
		os.WriteFile(path, []byte{}, 0644)

		_, err := ValidateTxnLog(path)
		if err == nil {
			t.Error("ValidateTxnLog() should return error for empty file")
		}
	})

	t.Run("corrupted txnlog", func(t *testing.T) {
		path := filepath.Join(tmpDir, "corrupted_log.100")
		createTestTxnLog(t, path, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
		})

		// Append garbage data
		f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		f.Write([]byte("corrupted data"))
		f.Close()

		result, err := ValidateTxnLog(path)
		if err != nil {
			t.Fatalf("ValidateTxnLog() error = %v", err)
		}

		if result.IsValid {
			t.Error("Result should be invalid for corrupted file")
		}
		if result.CorruptionType == "" {
			t.Error("CorruptionType should be set for corrupted file")
		}
		if result.ValidTransactionCount != 1 {
			t.Errorf("ValidTransactionCount = %d, want 1", result.ValidTransactionCount)
		}
	})
}

func TestRepairTxnLog(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("repair corrupted txnlog", func(t *testing.T) {
		input := filepath.Join(tmpDir, "corrupted.log")
		output := filepath.Join(tmpDir, "repaired.log")

		// Create file with valid transaction + corrupted data
		createTestTxnLog(t, input, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
		})

		// Append garbage
		f, _ := os.OpenFile(input, os.O_APPEND|os.O_WRONLY, 0644)
		f.Write([]byte("garbage data"))
		f.Close()

		// Repair
		result, err := RepairTxnLog(input, output)
		if err != nil {
			t.Fatalf("RepairTxnLog() error = %v", err)
		}

		if !result.IsValid {
			t.Error("Repaired result should be valid")
		}
		if result.ValidTransactionCount != 1 {
			t.Errorf("ValidTransactionCount = %d, want 1", result.ValidTransactionCount)
		}

		// Verify output file is valid
		validResult, err := ValidateTxnLog(output)
		if err != nil {
			t.Fatalf("Validating repaired file error = %v", err)
		}
		if !validResult.IsValid {
			t.Error("Repaired file should be valid")
		}
	})

	t.Run("repair already valid file", func(t *testing.T) {
		input := filepath.Join(tmpDir, "valid.log")
		output := filepath.Join(tmpDir, "output.log")

		createTestTxnLog(t, input, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
		})

		result, err := RepairTxnLog(input, output)
		if err != nil {
			t.Fatalf("RepairTxnLog() error = %v", err)
		}

		if !result.IsValid {
			t.Error("Result should be valid for already valid file")
		}
	})

	t.Run("no valid transactions", func(t *testing.T) {
		input := filepath.Join(tmpDir, "no_valid.log")
		output := filepath.Join(tmpDir, "no_valid_output.log")

		// Create file with only header (no transactions)
		createTestTxnLog(t, input, 12345, nil)

		result, err := RepairTxnLog(input, output)
		// File with no transactions is technically valid, just empty
		// So we check that result shows 0 transactions
		if err != nil && result.ValidTransactionCount == 0 {
			// This is expected - empty file
		} else if result.ValidTransactionCount > 0 {
			t.Error("Should have 0 valid transactions")
		}
	})
}

func TestCopyTxnLogUntilZxid(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy until specific ZXID", func(t *testing.T) {
		input := filepath.Join(tmpDir, "copy_input.log")
		output := filepath.Join(tmpDir, "copy_output.log")

		createTestTxnLog(t, input, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
			{ClientId: 2, Cxid: 2, Zxid: ZXID(0x101), Timestamp: 2000, Type: 2},
			{ClientId: 3, Cxid: 3, Zxid: ZXID(0x102), Timestamp: 3000, Type: 3},
			{ClientId: 4, Cxid: 4, Zxid: ZXID(0x103), Timestamp: 4000, Type: 4},
			{ClientId: 5, Cxid: 5, Zxid: ZXID(0x104), Timestamp: 5000, Type: 5},
		})

		count, err := CopyTxnLogUntilZxid(input, output, ZXID(0x101))
		if err != nil {
			t.Fatalf("CopyTxnLogUntilZxid() error = %v", err)
		}

		if count != 2 {
			t.Errorf("Copied count = %d, want 2", count)
		}

		// Verify output
		result, err := ValidateTxnLog(output)
		if err != nil {
			t.Fatalf("ValidateTxnLog() error = %v", err)
		}

		if result.ValidTransactionCount != 2 {
			t.Errorf("ValidTransactionCount = %d, want 2", result.ValidTransactionCount)
		}
	})
}

func TestCopyTxnLogFromZxid(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy from specific ZXID", func(t *testing.T) {
		input := filepath.Join(tmpDir, "copy_from_input.log")
		output := filepath.Join(tmpDir, "copy_from_output.log")

		createTestTxnLog(t, input, 12345, []testTransaction{
			{ClientId: 1, Cxid: 1, Zxid: ZXID(0x100), Timestamp: 1000, Type: 1},
			{ClientId: 2, Cxid: 2, Zxid: ZXID(0x101), Timestamp: 2000, Type: 2},
			{ClientId: 3, Cxid: 3, Zxid: ZXID(0x102), Timestamp: 3000, Type: 3},
		})

		count, err := CopyTxnLogFromZxid(input, output, ZXID(0x101))
		if err != nil {
			t.Fatalf("CopyTxnLogFromZxid() error = %v", err)
		}

		if count != 2 {
			t.Errorf("Copied count = %d, want 2", count)
		}

		// Verify output
		result, err := ValidateTxnLog(output)
		if err != nil {
			t.Fatalf("ValidateTxnLog() error = %v", err)
		}

		if result.ValidTransactionCount != 2 {
			t.Errorf("ValidTransactionCount = %d, want 2", result.ValidTransactionCount)
		}

		// Verify starts from correct ZXID
		reader, _ := OpenTxnLog(output)
		defer reader.Close()
		txn, _ := reader.ReadTransaction()
		if txn.Zxid != ZXID(0x101) {
			t.Errorf("First transaction ZXID = %v, want 0x101", txn.Zxid)
		}
	})
}
