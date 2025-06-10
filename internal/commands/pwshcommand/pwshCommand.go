package pwshcommand

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

const (
	// goOSWindows is the constant for Windows operating system.
	goOSWindows = "windows"
)

var (
	// 	ErrCannotFindPwsh is returned when the pwsh executable cannot be found in the system PATH.
	ErrCannotFindPwsh = errors.New("cannot find pwsh executable in PATH")
	// ErrBothScriptAndScriptFileSpecified is returned when both Script and ScriptFile are specified in the command definition.
	ErrBothScriptAndScriptFileSpecified = errors.New("cannot specify both script and scriptFile in the same command")
	// ErrCannotCreateTempFile is returned when a temporary file cannot be created for the script.
	ErrCannotCreateTempFile = errors.New("cannot create temporary file for script")
	// ErrCannotWriteTempFile is returned when the script cannot be written to the temporary file.
	ErrCannotWriteTempFile = errors.New("cannot write script to temporary file")
)

func New(ctx context.Context, def *Definition) (runbatch.Runnable, error) {
	// Check for conflicting script definitions first
	if def.Script != "" && def.ScriptFile != "" {
		return nil, ErrBothScriptAndScriptFileSpecified
	}

	execName := "pwsh"
	if runtime.GOOS == goOSWindows {
		execName = "pwsh.exe"
	}

	execPath, err := exec.LookPath(execName)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return nil, errors.Join(ErrCannotFindPwsh, err)
	}

	// If script is specified, write it to a temporary file and use that as the script file.
	if def.Script != "" {
		tmpFile, err := os.CreateTemp("", "script-*.ps1")

		if err != nil {
			return nil, errors.Join(ErrCannotCreateTempFile, err)
		}

		defer tmpFile.Close()

		if _, err := tmpFile.Write([]byte(def.Script)); err != nil {
			return nil, errors.Join(ErrCannotWriteTempFile, err)
		}

		def.ScriptFile = tmpFile.Name()
	}

	osCommmandArgs := make([]string, 4)

	osCommmandArgs[0] = "-NonInteractive"
	osCommmandArgs[1] = "-NoProfile"
	osCommmandArgs[2] = "-File"
	osCommmandArgs[3] = def.ScriptFile

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, commands.NewErrCommandCreate(commandType)
	}

	return &runbatch.OSCommand{
		BaseCommand:      base,
		Path:             execPath,
		Args:             osCommmandArgs,
		SuccessExitCodes: def.SuccessExitCodes,
		SkipExitCodes:    def.SkipExitCodes,
	}, nil
}
