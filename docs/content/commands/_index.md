+++
title = "Commands"
weight = 2
+++

Porch provides six built-in command types for different execution patterns. Each command type serves a specific purpose and has its own configuration options.

## Overview

| Command Type                           | Purpose                                      | Execution Mode |
| -------------------------------------- | -------------------------------------------- | -------------- |
| [Shell](shell/)                        | Execute shell commands                       | Single         |
| [PowerShell](pwsh/)                    | Execute PowerShell scripts                   | Single         |
| [Serial](serial/)                      | Run commands sequentially                    | Container      |
| [Parallel](parallel/)                  | Run commands concurrently                    | Container      |
| [ForEach Directory](foreachdirectory/) | Execute commands in multiple directories     | Container      |
| [Copy to Temp](copycwdtotemp/)         | Copy working directory to temporary location | Utility        |

## Single Commands

Single commands execute a single task:

- **[Shell](shell/)**: Execute any shell command or script
- **[PowerShell](pwsh/)**: Execute PowerShell scripts (Windows, Linux, macOS)

## Container Commands

Container commands group and control the execution of other commands:

- **[Serial](serial/)**: Execute commands one after another
- **[Parallel](parallel/)**: Execute commands simultaneously
- **[ForEach Directory](foreachdirectory/)**: Execute commands for each directory found

## Utility Commands

Utility commands provide special functionality:

- **[Copy to Temp](copycwdtotemp/)**: Create isolated temporary environments

## Common Attributes

All commands share these common attributes:

- **`type`** (required): The command type
- **`name`** (required): Descriptive name for the command
- **`working_directory`** (optional): Working directory for execution
- **`env`** (optional): Environment variables as key-value pairs
- **`runs_on_condition`** (optional): When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`** (optional): Specific exit codes that trigger execution

See individual command pages for type-specific attributes and detailed examples.
