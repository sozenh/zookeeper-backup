package zkfile

import (
	"fmt"
)

// ErrorCategory represents the category of an error
type ErrorCategory int

const (
	ErrorCategoryIO ErrorCategory = iota
	ErrorCategoryUser
	ErrorCategoryZooKeeper
	ErrorCategoryValidation
	ErrorCategoryCorruption
	ErrorCategoryConfiguration
)

func (ec ErrorCategory) String() string {
	switch ec {
	default:
		return "Unknown"
	case ErrorCategoryIO:
		return "IO"
	case ErrorCategoryUser:
		return "User"
	case ErrorCategoryZooKeeper:
		return "ZooKeeper"
	case ErrorCategoryValidation:
		return "Validation"
	case ErrorCategoryCorruption:
		return "Corruption"
	case ErrorCategoryConfiguration:
		return "Configuration"
	}
}

// BackupError is a structured error with category and context
type BackupError struct {
	Cause    error
	Message  string
	Category ErrorCategory
	Context  map[string]interface{}
}

// NewIOError creates an IO error
func NewIOError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryIO,
		Context:  make(map[string]interface{}),
	}
}

// NewUserError creates a user error
func NewUserError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryUser,
		Context:  make(map[string]interface{}),
	}
}

// NewZooKeeperError creates a ZooKeeper error
func NewZooKeeperError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryZooKeeper,
		Context:  make(map[string]interface{}),
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryValidation,
		Context:  make(map[string]interface{}),
	}
}

// NewCorruptionError creates a corruption error
func NewCorruptionError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryCorruption,
		Context:  make(map[string]interface{}),
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string) *BackupError {
	return &BackupError{
		Message:  message,
		Category: ErrorCategoryConfiguration,
		Context:  make(map[string]interface{}),
	}
}

func (e *BackupError) Unwrap() error {
	return e.Cause
}

func (e *BackupError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Category, e.Message)

	// Add context information
	if len(e.Context) > 0 {
		msg += " {"
		first := true
		for k, v := range e.Context {
			if !first {
				msg += ", "
			}
			msg += fmt.Sprintf("%s=%v", k, v)
			first = false
		}
		msg += "}"
	}

	// Add underlying error
	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}

	return msg
}

// WithError adds the underlying error cause
func (e *BackupError) WithError(cause error) *BackupError {
	e.Cause = cause
	return e
}

// WithContext adds context information
func (e *BackupError) WithContext(key string, value interface{}) *BackupError {
	e.Context[key] = value
	return e
}
