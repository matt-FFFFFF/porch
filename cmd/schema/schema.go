// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package schema provides the schema command for displaying YAML schema help.
package schema

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/urfave/cli/v3"
)

const (
	commandTypeArg = "command-type"
	formatFlag     = "format"
)

// SchemaCmd is the command that displays YAML schema help for command types.
var SchemaCmd = &cli.Command{
	Name:        "schema",
	Description: "Display YAML schema documentation for command types",
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: commandTypeArg,
		},
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        formatFlag,
			Aliases:     []string{"f"},
			Usage:       "Output format: yaml, markdown, or json",
			DefaultText: "yaml",
			Value:       "yaml",
		},
	},
	Action: actionFunc,
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	commandType := cmd.StringArg(commandTypeArg)
	format := cmd.String(formatFlag)

	// If no command type specified, list all available types
	if commandType == "" {
		return listCommandTypes()
	}

	// Validate format
	if !isValidFormat(format) {
		return cli.Exit(fmt.Sprintf("Invalid format: %s. Valid formats: yaml, markdown, json", format), 1)
	}

	// Generate schema for the specified command type
	return generateSchemaHelp(commandType, format)
}

func listCommandTypes() error {
	fmt.Println("Available command types:")
	fmt.Println()

	// Add special config option
	fmt.Printf("  %-15s - %s\n", "config", "Full configuration file schema")
	fmt.Println()

	// Get all registered command types
	var types []string
	for cmdType := range commandregistry.DefaultRegistry {
		types = append(types, cmdType)
	}
	sort.Strings(types)

	for _, cmdType := range types {
		fmt.Printf("  %-15s - %s\n", cmdType, getCommandDescription(cmdType))
	}

	fmt.Println()
	fmt.Println("Use 'porch schema <command-type>' to see detailed schema documentation.")
	fmt.Println("Use 'porch schema <command-type> --format markdown' for markdown documentation.")
	fmt.Println("Use 'porch schema config --format json' for full configuration JSON Schema.")

	return nil
}

func getCommandDescription(cmdType string) string {
	// Check if the command is registered and has a SchemaProvider
	if commander, exists := commandregistry.DefaultRegistry[cmdType]; exists {
		if provider, ok := commander.(commands.SchemaProvider); ok {
			return provider.GetCommandDescription()
		}
	}
	return "Description not available"
}

func generateSchemaHelp(commandType, format string) error {
	// Special case for full config schema
	if commandType == "config" {
		return generateFullConfigSchema(format)
	}

	// Get the commander from the registry
	commander, exists := commandregistry.DefaultRegistry[commandType]
	if !exists {
		return cli.Exit(fmt.Sprintf("Unknown command type: %s", commandType), 1)
	}

	// Check if the commander implements SchemaWriter
	schemaWriter, ok := commander.(commands.SchemaWriter)
	if !ok {
		return cli.Exit(fmt.Sprintf("Command type %s does not support schema generation", commandType), 1)
	}

	// Output in requested format using the commander's schema methods
	switch strings.ToLower(format) {
	case "yaml":
		return schemaWriter.WriteYAMLSchema(os.Stdout)
	case "markdown", "md":
		return schemaWriter.WriteMarkdownSchema(os.Stdout)
	case "json":
		return schemaWriter.WriteJSONSchema(os.Stdout)
	default:
		return cli.Exit(fmt.Sprintf("Invalid format: %s", format), 1)
	}
}

func generateFullConfigSchema(format string) error {
	generator := commands.NewSchemaGenerator()

	switch strings.ToLower(format) {
	case "json":
		// Generate full JSON schema for the entire config
		jsonStr, err := generator.GenerateJSONSchemaString()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to generate full JSON schema: %v", err), 1)
		}

		fmt.Print(jsonStr)
		return nil
	case "yaml":
		fmt.Println("# Full Porch Configuration Schema")
		fmt.Println("# This shows the complete structure of a porch configuration file")
		fmt.Println()
		fmt.Println("name: \"My Build Pipeline\"  # Name of the configuration")
		fmt.Println("description: \"Builds and deploys the application\"  # Description of what this configuration does")
		fmt.Println("commands:  # List of commands to execute")
		fmt.Println("  - type: shell")
		fmt.Println("    name: \"Example Shell Command\"")
		fmt.Println("    command_line: \"echo 'Hello, World!'\"")
		fmt.Println("  - type: serial")
		fmt.Println("    name: \"Sequential Commands\"")
		fmt.Println("    commands:")
		fmt.Println("      - type: shell")
		fmt.Println("        name: \"First Command\"")
		fmt.Println("        command_line: \"echo 'first'\"")
		fmt.Println("      - type: shell")
		fmt.Println("        name: \"Second Command\"")
		fmt.Println("        command_line: \"echo 'second'\"")
		fmt.Println()
		fmt.Println("# Use 'porch schema <command-type>' for specific command schemas")
		return nil
	case "markdown", "md":
		fmt.Println("# Porch Configuration Schema")
		fmt.Println()
		fmt.Println("Complete schema documentation for porch process orchestration framework configuration files.")
		fmt.Println()
		fmt.Println("## Root Configuration")
		fmt.Println()
		fmt.Println("| Field | Type | Required | Description |")
		fmt.Println("|-------|------|----------|-------------|")
		fmt.Println("| `name` | string | No | Name of the configuration |")
		fmt.Println("| `description` | string | No | Description of what this configuration does |")
		fmt.Println("| `commands` | array | Yes | List of commands to execute |")
		fmt.Println()
		fmt.Println("## Available Command Types")
		fmt.Println()
		fmt.Println("Commands can be one of the following types:")
		fmt.Println()
		fmt.Println("- **shell**: Execute shell commands with environment control and exit code handling")
		fmt.Println("- **serial**: Execute a list of commands sequentially, one after another")
		fmt.Println("- **parallel**: Execute a list of commands in parallel for improved performance")
		fmt.Println("- **foreachdirectory**: Execute commands for each directory found in the file system")
		fmt.Println("- **copycwdtotemp**: Copy current working directory to a temporary location before executing commands")
		fmt.Println()
		fmt.Println("Use `porch schema <command-type>` for detailed documentation of each command type.")
		return nil
	default:
		return cli.Exit(fmt.Sprintf("Unsupported format for config schema: %s", format), 1)
	}
}

func isValidFormat(format string) bool {
	validFormats := []string{"yaml", "markdown", "md", "json"}
	format = strings.ToLower(format)

	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}
