// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"slices"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/avmtool/internal/registry"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

const (
	TypeCommand  = "command"
	TypeSerial   = "serial"
	TypeParallel = "parallel"
)

var (
	ErrTooManyRootCommands = errors.New("too many root commands")
	ErrInvalidYaml         = errors.New("invalid YAML")
	ErrUnknownCommand      = errors.New("unknown command")
	ErrUnknownType         = errors.New("unknown type")
	ErrRunnableCreate      = errors.New("failed to create runnable")
)

type Definition struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Commands    []Command `yaml:"commands"`
}

// CommandDefinition represents a command in the YAML.
type Command struct {
	Type     string    `yaml:"type"`
	Name     string    `yaml:"name"`
	Command  string    `yaml:"command,omitempty"`
	Cwd      string    `yaml:"cwd,omitempty"`
	Args     []string  `yaml:"args,omitempty"`
	Commands []Command `yaml:"commands,omitempty"`
}

func BuildFromYAML(yamlData []byte) (runbatch.Runnable, error) {
	var def Definition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, errors.Join(ErrInvalidYaml, err)
	}

	if len(def.Commands) > 1 {
		return nil, ErrTooManyRootCommands
	}

	res := runbatch.SerialBatch{
		Label:    def.Name,
		Commands: nil,
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
				cmd.Name, cmd.Command, cmd.Cwd, slices.Concat([]string{cmd.Command}, cmd.Args)...)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			runnables = append(runnables, rble)
		case TypeSerial:
			rble, err := build(cmd.Commands)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			runnables = append(runnables, rble...)
		case TypeParallel:
			rble, err := build(cmd.Commands)
			if err != nil {
				return nil, errors.Join(ErrRunnableCreate, err)
			}

			runnables = append(runnables, rble...)
		default:
			return nil, ErrUnknownType
		}
	}

	return runnables, nil
}
