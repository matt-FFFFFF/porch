// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package runbatch provides a framework for running batches of commands in parallel or serially.
// It allows for easy management of command execution, including handling of working directories,
// context cancellation, and error reporting.
//
// The main components of the package are:
//
// ### Runnable
//
// An interface for something that can be run as part of a batch (either a Command or a nested Batch).
//
// ### OSCommand
//
// A Runnable that runs a command in the OS.
//
// ### FunctionCommand
//
// A Runnable that runs a go function in a goroutine.
//
// ### SerialBatch
//
// A Runnable that runs a collection of commands or nested batches serially.
//
// ### ParallelBatch
//
// A Runnable that runs a collection of commands or nested batches in parallel.
//
// Command results are aggregated and returned as a Results type, which contains information about the execution.
// Results can be visualised using the WriteResults function and your favourite output io.Writer.
//
// Signal handling is also supported, allowing for graceful termination of running commands.
// This is achieved by listening for OS signals and forwarding them to the appropriate command.
// See the signalbroker package for the signals that are caught by default.
//
// Importantly, the first caught signal does not cancel the context but does get passed to any running OSCommands.
// This allows for graceful shutdown of the commands without terminating the entire process.
// The second caught signal (of the same type) will cancel the context and attempt to kill all running OSCommands.
// FunctionCommands must handle context cancellation themselves.
package runbatch
