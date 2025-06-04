// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
)

// Writer provides methods to write JSON Schema to an io.Writer.
type Writer interface {
	// WriteJSONSchema writes JSON Schema to the writer
	WriteJSONSchema(w io.Writer) error
	// WriteYAMLExample writes YAML example schema to the writer
	WriteYAMLExample(w io.Writer) error
	// WriteMarkdownDoc writes Markdown documentation to the writer
	WriteMarkdownDoc(w io.Writer) error
}

// Provider provides methods to get schema information for commands.
type Provider interface {
	// GetSchemaFields returns the schema fields for this command type
	GetSchemaFields() []Field
	// GetCommandType returns the command type string
	GetCommandType() string
	// GetCommandDescription returns a description of what this command does
	GetCommandDescription() string
	// GetExampleDefinition returns an example definition for YAML generation
	GetExampleDefinition() interface{}
}

// Field represents a field in a JSON schema.
type Field struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Properties  map[string]Field `json:"properties,omitempty"`
	Items       *Field           `json:"items,omitempty"`
	Enum        []string         `json:"enum,omitempty"`
	Default     interface{}      `json:"default,omitempty"`
}

// Schema represents a complete JSON schema for a command type.
type Schema struct {
	Type        string           `json:"type"`
	Properties  map[string]Field `json:"properties"`
	Required    []string         `json:"required,omitempty"`
	Fields      []Field          `json:"-"` // For internal use
	Description string           `json:"description,omitempty"`
}

// Generator provides methods to generate schemas from struct definitions.
type Generator struct{}

// NewGenerator creates a new SchemaGenerator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates a schema from a command type and definition struct.
func (g *Generator) Generate(commandType string, def interface{}) (*Schema, error) {
	fields, err := g.extractFields(reflect.TypeOf(def))
	if err != nil {
		return nil, err
	}

	// Sort fields according to the required order: type, name, others (lexically), commands
	sortedFields := g.sortFields(fields)

	properties := make(map[string]Field)

	var required []string

	for _, field := range sortedFields {
		properties[field.Name] = field

		if field.Required {
			required = append(required, field.Name)
		}
	}

	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
		Fields:     sortedFields,
	}, nil
}

// GenerateFromDefinition generates a JSON schema object from a command definition struct.
func (g *Generator) GenerateFromDefinition(commandType string, def interface{}, description string) (map[string]interface{}, error) {
	fields, err := g.extractFields(reflect.TypeOf(def))
	if err != nil {
		return nil, err
	}

	// Sort fields according to the required order
	sortedFields := g.sortFields(fields)

	properties := make(map[string]interface{})

	var required []string

	for _, field := range sortedFields {
		prop := g.schemaFieldToProperty(field)

		// Special handling for type field to add enum constraint
		if field.Name == "type" {
			prop["enum"] = []string{commandType}
		}

		properties[field.Name] = prop

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema := map[string]interface{}{
		"type":                 "object",
		"description":          description,
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}

	return schema, nil
}

// GenerateJSONSchemaString generates a complete JSON schema for the entire configuration.
func (g *Generator) GenerateJSONSchemaString() (string, error) {
	// Create root schema for the entire configuration
	rootSchema := map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"type":        "object",
		"title":       "Porch Configuration Schema",
		"description": "Schema for porch process orchestration framework configuration files",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the configuration",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Description of what this configuration does",
			},
			"commands": map[string]interface{}{
				"type":        "array",
				"description": "List of commands to execute",
				"items": map[string]interface{}{
					"anyOf": g.generateCommandSchemas(),
				},
			},
		},
		"required": []string{"commands"},
	}

	bytes, err := json.MarshalIndent(rootSchema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// generateCommandSchemas generates schemas for all available command types by using the registry.
func (g *Generator) generateCommandSchemas() []map[string]interface{} {
	var schemas []map[string]interface{}

	// Access the default registry from commandregistry package
	// This will contain all registered command types and their commanders
	for commandType, commander := range commandregistry.DefaultRegistry {
		// Check if the commander implements SchemaProvider
		if provider, ok := commander.(Provider); ok {
			// Generate schema for this command type using its definition
			schema, err := g.GenerateFromDefinition(
				commandType,
				provider.GetExampleDefinition(),
				provider.GetCommandDescription(),
			)
			if err != nil {
				// If we can't generate the schema, create a basic one
				schema = map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type": "string",
							"enum": []string{commandType},
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the command",
						},
					},
					"required": []string{"type", "name"},
				}
			}

			schemas = append(schemas, schema)
		}
	}

	return schemas
}

// extractFields extracts schema fields from a struct type using reflection.
func (g *Generator) extractFields(t reflect.Type) ([]Field, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", t.Kind())
	}

	var fields []Field

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs (like BaseDefinition)
		if field.Anonymous {
			embeddedFields, err := g.extractFields(field.Type)
			if err != nil {
				return nil, err
			}

			fields = append(fields, embeddedFields...)

			continue
		}

		schemaField, err := g.fieldToSchemaField(field)
		if err != nil {
			return nil, err
		}

		if schemaField != nil {
			fields = append(fields, *schemaField)
		}
	}

	return fields, nil
}

// fieldToSchemaField converts a reflect.StructField to a SchemaField.
func (g *Generator) fieldToSchemaField(field reflect.StructField) (*Field, error) {
	// Check for yaml tag to get the field name
	yamlTag := field.Tag.Get("yaml")
	if yamlTag == "-" {
		return nil, nil // Skip this field
	}

	// Parse yaml tag for field name
	fieldName := field.Name

	if yamlTag != "" {
		// Extract field name from yaml tag (before any options like omitempty)
		parts := strings.Split(yamlTag, ",")
		if parts[0] != "" {
			fieldName = parts[0]
		}
	}

	// Convert field name to lowercase
	fieldName = strings.ToLower(fieldName)

	// Get description from docdesc tag
	description := field.Tag.Get("docdesc")

	// Determine if field is required
	required := !(yamlTag != "" && strings.Contains(yamlTag, "omitempty"))

	// Determine field type
	fieldType := g.getSchemaType(field.Type)

	return &Field{
		Name:        fieldName,
		Type:        fieldType,
		Description: description,
		Required:    required,
	}, nil
}

// getSchemaType converts a Go type to a JSON schema type.
func (g *Generator) getSchemaType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Ptr:
		return g.getSchemaType(t.Elem())
	default:
		return "string" // Default fallback
	}
}

// sortFields sorts fields according to the required order: type, name, others (lexically), commands.
func (g *Generator) sortFields(fields []Field) []Field {
	var typeField, nameField *Field

	var commandsField *Field

	var otherFields []Field

	for i := range fields {
		field := &fields[i]
		switch field.Name {
		case "type":
			typeField = field
		case "name":
			nameField = field
		case "commands":
			commandsField = field
		default:
			otherFields = append(otherFields, *field)
		}
	}

	// Sort other fields lexically
	sort.Slice(otherFields, func(i, j int) bool {
		return otherFields[i].Name < otherFields[j].Name
	})

	// Build the final ordered list
	var result []Field

	if typeField != nil {
		result = append(result, *typeField)
	}

	if nameField != nil {
		result = append(result, *nameField)
	}

	result = append(result, otherFields...)
	if commandsField != nil {
		result = append(result, *commandsField)
	}

	return result
}

// schemaFieldToProperty converts a SchemaField to a JSON schema property.
func (g *Generator) schemaFieldToProperty(field Field) map[string]interface{} {
	prop := map[string]interface{}{
		"type": field.Type,
	}

	if field.Description != "" {
		prop["description"] = field.Description
	}

	if field.Default != nil {
		prop["default"] = field.Default
	}

	if len(field.Enum) > 0 {
		prop["enum"] = field.Enum
	}

	if field.Type == "array" && field.Items != nil {
		prop["items"] = g.schemaFieldToProperty(*field.Items)
	}

	if field.Type == "object" && len(field.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, subField := range field.Properties {
			properties[name] = g.schemaFieldToProperty(subField)
		}

		prop["properties"] = properties
	}

	return prop
}

// BaseSchemaGenerator provides basic schema generation functionality.
type BaseSchemaGenerator struct {
	generator *Generator
}

// NewBaseSchemaGenerator creates a new BaseSchemaGenerator.
func NewBaseSchemaGenerator() *BaseSchemaGenerator {
	return &BaseSchemaGenerator{
		generator: NewGenerator(),
	}
}

// GetSchemaFields returns the schema fields for a given struct type.
func (b *BaseSchemaGenerator) GetSchemaFields(def interface{}) ([]Field, error) {
	return b.generator.extractFields(reflect.TypeOf(def))
}

// WriteJSONSchema writes the complete JSON schema to the writer.
func (b *BaseSchemaGenerator) WriteJSONSchema(w io.Writer) error {
	schema, err := b.generator.GenerateJSONSchemaString()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(schema))

	return err
}

// WriteYAMLSchema writes a YAML example to the writer.
func (b *BaseSchemaGenerator) WriteYAMLSchema(w io.Writer) error {
	// This would generate a YAML example based on the schema
	// For now, just write a basic example
	example := `name: "Example Configuration"
description: "An example porch configuration"
commands:
  - type: "shell"
    name: "hello"
    command: "echo 'Hello, World!'"
`
	_, err := w.Write([]byte(example))

	return err
}

// WriteMarkdownSchema writes Markdown documentation to the writer.
func (b *BaseSchemaGenerator) WriteMarkdownSchema(w io.Writer) error {
	// This would generate markdown documentation based on the schema
	// For now, just write basic documentation
	doc := `# Porch Configuration Schema

This document describes the schema for porch configuration files.

## Root Structure

- **name** (string): Name of the configuration
- **description** (string, optional): Description of what this configuration does
- **commands** (array): List of commands to execute

## Command Types

`
	_, err := w.Write([]byte(doc))

	return err
}
