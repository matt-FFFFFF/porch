// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"errors"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/avmtool/internal/registry"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

const (
	TypeCommand  = "command"
	TypeSerial   = "serial"
	TypeParallel = "parallel"
	TypeForeach  = "foreach"
)

var (
	ErrTooManyRootCommands = errors.New("too many root commands")
	ErrInvalidYaml         = errors.New("invalid YAML")
	ErrUnknownCommand      = errors.New("unknown command")
	ErrUnknownType         = errors.New("unknown type")
	ErrRunnableCreate      = errors.New("failed to create runnable")
	ErrUnknownItemProvider = errors.New("unknown item provider")
	ErrInvalidForEachMode  = errors.New("invalid foreach mode")
)

type Definition struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Commands    []Command `yaml:"commands"`
}

// CommandDefinition represents a command in the YAML.
type Command struct {
	Type       string    `yaml:"type"`
	Name       string    `yaml:"name"`
	Command    string    `yaml:"command,omitempty"`
	Executable string    `yaml:"executable,omitempty"`
	Cwd        string    `yaml:"cwd,omitempty"`
	Args       []string  `yaml:"args,omitempty"`
	Commands   []Command `yaml:"commands,omitempty"`
	// ForEach specific fields
	ItemProvider string `yaml:"itemProvider,omitempty"`
	ForEachMode  string `yaml:"mode,omitempty"` // "serial" or "parallel"
}

func BuildFromYAML(yamlData []byte) (runbatch.Runnable, error) {
	var def Definition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, errors.Join(ErrInvalidYaml, err)
	}

	if len(def.Commands) > 1 {
		return nil, ErrTooManyRootCommands
	}

	rbls, err := build(def.Commands)
	if err != nil {
		return nil, err //nolint:err113
	}

	res := runbatch.SerialBatch{
		Label:    def.Name,
		Commands: rbls,
	}

	return &res, nil
}

func build(cmds []Command) ([]runbatch.Runnable, error) {
	var runnables []runbatch.Runnable

	for _, cmd := range cmds {
		switch cmd.Type {
		case TypeCommand:
			c, ok := registry.DefaultRegistry[cmd.Command]
			if !ok {
				return nil, ErrUnknownCommand
			}

			rble, err := c.Create(
				cmd.Name, cmd.Executable, cmd.Cwd, cmd.Args...)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			runnables = append(runnables, rble)
		case TypeSerial:
			rble, err := build(cmd.Commands)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			serialBatch := &runbatch.SerialBatch{
				Label:    cmd.Name,
				Commands: rble,
			}
			runnables = append(runnables, serialBatch)
		case TypeParallel:
			rble, err := build(cmd.Commands)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			parallelBatch := &runbatch.ParallelBatch{
				Label:    cmd.Name,
				Commands: rble,
			}
			runnables = append(runnables, parallelBatch)
		case TypeForeach:
			// Check if the ItemProvider exists
			provider, ok := registry.DefaultItemProviderRegistry[cmd.ItemProvider]
			if !ok {
				return nil, ErrUnknownItemProvider
			}

			if len(cmd.Commands) > 1 {
				return nil, ErrTooManyRootCommands
			}

			// Parse the foreach mode
			mode := runbatch.ForEachSerial
			if cmd.ForEachMode == "parallel" {
				mode = runbatch.ForEachParallel
			} else if cmd.ForEachMode != "" && cmd.ForEachMode != "serial" {
				return nil, ErrInvalidForEachMode
			}

			// Build the child commands
			childCommands, err := build(cmd.Commands)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			// Create the foreach command
			foreachCmd := &runbatch.ForEachCommand{
				Label:         cmd.Name,
				Cwd:           cmd.Cwd,
				ItemsProvider: provider,
				Commands:      childCommands,
				Mode:          mode,
			}

			runnables = append(runnables, foreachCmd)
		default:
			return nil, ErrUnknownType
		}
	}

	return runnables, nil
}
