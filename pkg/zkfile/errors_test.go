package zkfile

import (
	"errors"
	"testing"
)

func TestBackupError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *BackupError
		want string
	}{
		{
			name: "with cause",
			err: &BackupError{
				Category: ErrorCategoryIO,
				Message:  "failed to read",
				Cause:    errors.New("file not found"),
				Context:  make(map[string]interface{}),
			},
			want: "[IO] failed to read: file not found",
		},
		{
			name: "without cause",
			err: &BackupError{
				Category: ErrorCategoryValidation,
				Message:  "invalid format",
				Cause:    nil,
				Context:  make(map[string]interface{}),
			},
			want: "[Validation] invalid format",
		},
		{
			name: "with context",
			err: &BackupError{
				Category: ErrorCategoryIO,
				Message:  "failed to read",
				Cause:    nil,
				Context:  map[string]interface{}{"path": "/tmp/file"},
			},
			want: "[IO] failed to read {path=/tmp/file}",
		},
		{
			name: "with context and cause",
			err: &BackupError{
				Category: ErrorCategoryIO,
				Message:  "failed to read",
				Cause:    errors.New("file not found"),
				Context:  map[string]interface{}{"path": "/tmp/file"},
			},
			want: "[IO] failed to read {path=/tmp/file}: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			// Since map iteration order is not deterministic, for tests with context, check if key parts are present
			if tt.name == "with context" || tt.name == "with context and cause" {
				if !containsSubstring(got, "[IO] failed to read") {
					t.Errorf("BackupError.Error() = %v, should contain '[IO] failed to read'", got)
				}
				if !containsSubstring(got, "path=/tmp/file") {
					t.Errorf("BackupError.Error() = %v, should contain 'path=/tmp/file'", got)
				}
				if tt.name == "with context and cause" && !containsSubstring(got, "file not found") {
					t.Errorf("BackupError.Error() = %v, should contain 'file not found'", got)
				}
			} else {
				if got != tt.want {
					t.Errorf("BackupError.Error() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBackupError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := &BackupError{
		Category: ErrorCategoryIO,
		Message:  "wrapped",
		Cause:    cause,
	}

	if got := err.Unwrap(); got != cause {
		t.Errorf("BackupError.Unwrap() = %v, want %v", got, cause)
	}
}

func TestNewIOError(t *testing.T) {
	err := NewIOError("test message")

	if err.Category != ErrorCategoryIO {
		t.Errorf("Category = %v, want %v", err.Category, ErrorCategoryIO)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want %v", err.Message, "test message")
	}
	if err.Cause != nil {
		t.Errorf("Cause = %v, want nil", err.Cause)
	}
	if err.Context == nil {
		t.Error("Context should be initialized")
	}
}

func TestNewUserError(t *testing.T) {
	err := NewUserError("test message")

	if err.Category != ErrorCategoryUser {
		t.Errorf("Category = %v, want %v", err.Category, ErrorCategoryUser)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want %v", err.Message, "test message")
	}
}

func TestNewCorruptionError(t *testing.T) {
	err := NewCorruptionError("test message")

	if err.Category != ErrorCategoryCorruption {
		t.Errorf("Category = %v, want %v", err.Category, ErrorCategoryCorruption)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want %v", err.Message, "test message")
	}
}

func TestBackupError_WithError(t *testing.T) {
	cause := errors.New("test error")
	err := NewIOError("test message").WithError(cause)

	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestBackupError_WithContext(t *testing.T) {
	err := NewIOError("test").WithContext("file", "/path/to/file")

	if val, ok := err.Context["file"]; !ok || val != "/path/to/file" {
		t.Errorf("Context['file'] = %v, want %v", val, "/path/to/file")
	}
}

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		name string
		ec   ErrorCategory
		want string
	}{
		{
			name: "IO",
			ec:   ErrorCategoryIO,
			want: "IO",
		},
		{
			name: "Validation",
			ec:   ErrorCategoryValidation,
			want: "Validation",
		},
		{
			name: "Corruption",
			ec:   ErrorCategoryCorruption,
			want: "Corruption",
		},
		{
			name: "Configuration",
			ec:   ErrorCategoryConfiguration,
			want: "Configuration",
		},
		{
			name: "ZooKeeper",
			ec:   ErrorCategoryZooKeeper,
			want: "ZooKeeper",
		},
		{
			name: "User",
			ec:   ErrorCategoryUser,
			want: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ec.String(); got != tt.want {
				t.Errorf("ErrorCategory.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
