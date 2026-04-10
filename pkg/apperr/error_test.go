package apperr

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test_action", CodeInvalidArg, "invalid argument")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr := &Error{}
	if !errors.As(err, &appErr) {
		t.Fatal("expected Error type")
	}

	if appErr.Action != "test_action" {
		t.Errorf("expected action 'test_action', got '%s'", appErr.Action)
	}
	if appErr.Code != CodeInvalidArg {
		t.Errorf("expected code '%s', got '%s'", CodeInvalidArg, appErr.Code)
	}
	if appErr.Message != "invalid argument" {
		t.Errorf("expected message 'invalid argument', got '%s'", appErr.Message)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap("upload", CodeUpload, "upload failed", cause)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr := &Error{}
	if !errors.As(err, &appErr) {
		t.Fatal("expected Error type")
	}

	if appErr.Action != "upload" {
		t.Errorf("expected action 'upload', got '%s'", appErr.Action)
	}
	if appErr.Code != CodeUpload {
		t.Errorf("expected code '%s', got '%s'", CodeUpload, appErr.Code)
	}
	if appErr.Cause != cause {
		t.Error("cause should be preserved")
	}
}

func TestErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "normal error",
			err:      &Error{Action: "test", Code: CodeInvalidArg, Message: "test error"},
			expected: "test error",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "empty message",
			err:      &Error{Action: "test", Code: CodeInvalidArg, Message: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("original cause")
	err := &Error{
		Action:  "test",
		Code:    CodeInternal,
		Message: "test error",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("expected unwrap to return cause, got %v", unwrapped)
	}

	// Test nil case
	var nilErr *Error
	if nilErr.Unwrap() != nil {
		t.Error("nil error unwrap should return nil")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		defaultAction  string
		expectedAction string
		expectedCode   string
		expectedMsg    string
	}{
		{
			name:           "AppError with all fields",
			err:            New("upload", CodeUpload, "file too large"),
			defaultAction:  "default",
			expectedAction: "upload",
			expectedCode:   CodeUpload,
			expectedMsg:    "file too large",
		},
		{
			name:           "AppError with empty action",
			err:            &Error{Action: "", Code: CodeInvalidArg, Message: "test"},
			defaultAction:  "default",
			expectedAction: "default",
			expectedCode:   CodeInvalidArg,
			expectedMsg:    "test",
		},
		{
			name:           "AppError with empty code",
			err:            &Error{Action: "test", Code: "", Message: "test"},
			defaultAction:  "default",
			expectedAction: "test",
			expectedCode:   CodeInternal,
			expectedMsg:    "test",
		},
		{
			name:           "AppError with empty message",
			err:            &Error{Action: "test", Code: CodeInternal, Message: ""},
			defaultAction:  "default",
			expectedAction: "test",
			expectedCode:   CodeInternal,
			expectedMsg:    "",
		},
		{
			name:           "nil error",
			err:            nil,
			defaultAction:  "default",
			expectedAction: "default",
			expectedCode:   CodeInternal,
			expectedMsg:    "unknown error",
		},
		{
			name:           "standard error",
			err:            errors.New("standard error"),
			defaultAction:  "default",
			expectedAction: "default",
			expectedCode:   CodeInternal,
			expectedMsg:    "standard error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, code, msg := Parse(tt.err, tt.defaultAction)
			if action != tt.expectedAction {
				t.Errorf("expected action '%s', got '%s'", tt.expectedAction, action)
			}
			if code != tt.expectedCode {
				t.Errorf("expected code '%s', got '%s'", tt.expectedCode, code)
			}
			if msg != tt.expectedMsg {
				t.Errorf("expected msg '%s', got '%s'", tt.expectedMsg, msg)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// Verify error codes are defined
	if CodeInvalidArg == "" {
		t.Error("CodeInvalidArg should not be empty")
	}
	if CodePathTraversal == "" {
		t.Error("CodePathTraversal should not be empty")
	}
	if CodeNotFound == "" {
		t.Error("CodeNotFound should not be empty")
	}
	if CodeConflict == "" {
		t.Error("CodeConflict should not be empty")
	}
	if CodeProvider == "" {
		t.Error("CodeProvider should not be empty")
	}
	if CodeUpload == "" {
		t.Error("CodeUpload should not be empty")
	}
	if CodeDownload == "" {
		t.Error("CodeDownload should not be empty")
	}
	if CodeArchive == "" {
		t.Error("CodeArchive should not be empty")
	}
	if CodeConfig == "" {
		t.Error("CodeConfig should not be empty")
	}
	if CodeInternal == "" {
		t.Error("CodeInternal should not be empty")
	}
}