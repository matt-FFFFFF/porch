package commandinpath

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Create temp directory for test commands
	tempDir, err := os.MkdirTemp("", "commandinpath_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock command file in tempDir
	mockCommandName := "mockcommand"
	if os.Getenv("GOOS") == "windows" {
		mockCommandName += ".exe"
	}
	mockCommandPath := filepath.Join(tempDir, mockCommandName)

	// Create an executable file
	f, err := os.Create(mockCommandPath)
	if err != nil {
		t.Fatalf("Failed to create mock command: %v", err)
	}
	defer f.Close()

	// Make it executable (for Unix systems)
	if runtime.GOOS != "windows" {
		err = os.Chmod(mockCommandPath, 0755)
		if err != nil {
			t.Fatalf("Failed to make mock command executable: %v", err)
		}
	}

	// Test cases
	tests := []struct {
		name         string
		label        string
		command      string
		cwd          string
		args         []string
		path         string // PATH environment variable to set
		expectedNil  bool
		expectedPath string
		fileMode     os.FileMode
	}{
		{
			name:         "Command found",
			label:        "test-label",
			command:      mockCommandName,
			cwd:          "/test/cwd",
			args:         []string{},
			path:         tempDir,
			expectedNil:  false,
			expectedPath: mockCommandPath,
			fileMode:     0755,
		},
		{
			name:        "Command not found",
			label:       "test-label",
			command:     "nonexistentcommand",
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0755,
		},
		{
			name:         "Multiple paths in PATH - command found",
			label:        "test-label",
			command:      mockCommandName,
			cwd:          "/test/cwd",
			args:         []string{},
			path:         "/non/existent/path" + string(os.PathListSeparator) + tempDir,
			expectedNil:  false,
			expectedPath: mockCommandPath,
			fileMode:     0755,
		},
		{
			name:        "Empty PATH",
			label:       "test-label",
			command:     mockCommandName,
			cwd:         "/test/cwd",
			args:        []string{},
			path:        "",
			expectedNil: true,
			fileMode:    0755,
		},
		{
			name:        "File not executable",
			label:       "test-label",
			command:     mockCommandName,
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0644,
		},
		{
			name:        "Empty command",
			label:       "test-label",
			command:     "",
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0755,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set PATH for this test
			t.Setenv("PATH", tc.path)

			require.NoError(t, f.Chmod(tc.fileMode))

			// Call the function under test
			result := New(tc.label, tc.command, tc.cwd, tc.args)

			// Check if result is nil
			if tc.expectedNil && result != nil {
				t.Errorf("Expected nil result, got %+v", result)
				return
			}

			if !tc.expectedNil && result == nil {
				t.Errorf("Expected non-nil result, got nil")
				return
			}

			// If we expect a non-nil result, check the fields
			if !tc.expectedNil {
				if result.Label != tc.label {
					t.Errorf("Expected label %q, got %q", tc.label, result.Label)
				}

				if result.Path != tc.expectedPath {
					t.Errorf("Expected path %q, got %q", tc.expectedPath, result.Path)
				}

				if result.Cwd != tc.cwd {
					t.Errorf("Expected cwd %q, got %q", tc.cwd, result.Cwd)
				}

				if len(result.Args) != len(tc.args) {
					t.Errorf("Expected %d args, got %d", len(tc.args), len(result.Args))
				} else {
					for i, arg := range tc.args {
						if result.Args[i] != arg {
							t.Errorf("Expected arg %d to be %q, got %q", i, arg, result.Args[i])
						}
					}
				}
			}
		})
	}
}
