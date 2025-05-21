// Package runbatch provides a way to run a batch of commands, optionally in parallel.
// It will return the exit code of all of the commends and format a nice error message explaining which command failed.
// It is designed to be used in a command line application where you want to run a batch of commands and return the exit code all the ones that fail.
// Commands can be added as serial or parallel commands. They can also be nested.
package runbatch
