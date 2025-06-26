package hcl

import (
	"testing"

	"github.com/prashantv/gostub"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_workflowDecode(t *testing.T) {
	content := `
locals {
  build_env = {
    GO_ENV     = var.environment
    BUILD_TIME = timestamp()
    VERSION    = "1.0.0"
  }

  test_commands = [
    "go test ./...",
    "go test -race ./...",
    "go test -bench=. ./..."
  ]
}

variable "environment" {
	default = "development"
}

workflow "enhanced_build" {
  name        = "Enhanced Build Pipeline"
  description = "Build pipeline with variables and locals"

  # Shell command
  command {
    type = "shell"
    name = "Environment Setup"
    env  = local.build_env
    command_line = <<-EOT
      echo "Building for environment: ${var.environment}"
      echo "Build started at: ${local.build_env.BUILD_TIME}"
    EOT
  }
}
	`
	fs := afero.NewMemMapFs()
	dummyFsWithFiles(fs, []string{"test.porch.hcl", "/example/testfile"}, []string{content, "world"})
	gostub.Stub(&FsFactory, func() afero.Fs {
		return fs
	})

	config, err := BuildPorchConfig("/", "", nil, nil)
	require.NoError(t, err)

	plan, err := RunPorchPlan(config)
	require.NoError(t, err)
	assert.Len(t, plan.Workflows, 1)
	assert.Len(t, plan.Workflows[0].Command, 1)
	assert.Contains(t, plan.Workflows[0].Command[0].CommandLine, `echo "Building for environment: development"`)
}

func Test_workflowWithDynamicCommand(t *testing.T) {
	content := `
# variables.porch.hcl
variable "environment" {
  description = "Target environment"
  type        = string
  default     = "development"
}

variable "parallel_jobs" {
  description = "Number of parallel jobs"
  type        = number
  default     = 4
}

# workflow.porch.hcl
locals {
  build_env = {
    GO_ENV     = var.environment
    BUILD_TIME = timestamp()
    VERSION    = "1.0.0"
  }

  test_commands = [
    "go test ./...",
    "go test -race ./...",
    "go test -bench=. ./..."
  ]
}

workflow "enhanced_build" {
  name        = "Enhanced Build Pipeline"
  description = "Build pipeline with variables and locals"

  # Dynamic test generation
  dynamic "command" {
    for_each = local.test_commands
    content {
      type         = "shell"
      name         = "Test: ${command.value}"
      command_line = command.value
      env = {
        GOMAXPROCS = var.parallel_jobs
      }
    }
  }
}
	`
	fs := afero.NewMemMapFs()
	dummyFsWithFiles(fs, []string{"test.porch.hcl", "/example/testfile"}, []string{content, "world"})
	gostub.Stub(&FsFactory, func() afero.Fs {
		return fs
	})

	config, err := BuildPorchConfig("/", "", nil, nil)
	require.NoError(t, err)

	plan, err := RunPorchPlan(config)
	require.NoError(t, err)
	assert.Len(t, plan.Workflows, 1)
	assert.Len(t, plan.Workflows[0].Command, 3)
	expectedCommands := []string{
		"go test ./...",
		"go test -race ./...",
		"go test -bench=. ./...",
	}
	for i, expectedCmd := range expectedCommands {
		assert.Equal(t, expectedCmd, plan.Workflows[0].Command[i].CommandLine)
	}
}

func dummyFsWithFiles(fs afero.Fs, fileNames []string, contents []string) {
	for i := range fileNames {
		_ = afero.WriteFile(fs, fileNames[i], []byte(contents[i]), 0644)
	}
}
