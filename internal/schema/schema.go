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

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
)

// Writer provides methods to write JSON Schema to an io.Writer.
type Writer interface {
	// WriteJSONSchema writes JSON Schema to the writer
	WriteJSONSchema(w io.Writer, f commands.CommanderFactory) error
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
	Type                 string           `json:"type"`
	Description          string           `json:"description,omitempty"`
	Properties           map[string]Field `json:"properties"`
	Required             []string         `json:"required,omitempty"`
	AdditionalProperties bool             `json:"additionalProperties,omitempty"`
	Fields               []Field          `json:"-"` // For internal use
}

// ToMap converts a Schema struct to a map[string]interface{} for JSON serialization.
func (s *Schema) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":                 s.Type,
		"additionalProperties": s.AdditionalProperties,
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if len(s.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, field := range s.Properties {
			generator := &Generator{}
			properties[name] = generator.schemaFieldToProperty(field)
		}
		result["properties"] = properties
	}

	if len(s.Required) > 0 {
		result["required"] = s.Required
	}

	return result
}

// ToOrderedStruct creates a dynamically ordered struct for JSON marshaling with proper field ordering.
func (s *Schema) ToOrderedStruct() interface{} {
	// Create ordered properties using reflection to maintain field order
	orderedProperties := s.createOrderedPropertiesStruct()

	// Create struct fields in the desired order
	var structFields []reflect.StructField

	// 1. Add "type" field
	structFields = append(structFields, reflect.StructField{
		Name: "Type",
		Type: reflect.TypeOf(""),
		Tag:  `json:"type"`,
	})

	// 2. Add "description" field if present
	if s.Description != "" {
		structFields = append(structFields, reflect.StructField{
			Name: "Description",
			Type: reflect.TypeOf(""),
			Tag:  `json:"description,omitempty"`,
		})
	}

	// 3. Add "properties" field
	structFields = append(structFields, reflect.StructField{
		Name: "Properties",
		Type: reflect.TypeOf(orderedProperties).Elem(), // Get the underlying type
		Tag:  `json:"properties"`,
	})

	// 4. Add "required" field if present
	if len(s.Required) > 0 {
		structFields = append(structFields, reflect.StructField{
			Name: "Required",
			Type: reflect.TypeOf([]string{}),
			Tag:  `json:"required,omitempty"`,
		})
	}

	// 5. Add "additionalProperties" field
	structFields = append(structFields, reflect.StructField{
		Name: "AdditionalProperties",
		Type: reflect.TypeOf(false),
		Tag:  `json:"additionalProperties,omitempty"`,
	})

	// Create the struct type
	structType := reflect.StructOf(structFields)
	structValue := reflect.New(structType).Elem()

	// Set the values
	structValue.FieldByName("Type").SetString(s.Type)

	if s.Description != "" {
		structValue.FieldByName("Description").SetString(s.Description)
	}

	structValue.FieldByName("Properties").Set(reflect.ValueOf(orderedProperties).Elem())

	if len(s.Required) > 0 {
		structValue.FieldByName("Required").Set(reflect.ValueOf(s.Required))
	}

	structValue.FieldByName("AdditionalProperties").SetBool(s.AdditionalProperties)

	return structValue.Interface()
}

// createOrderedPropertiesStruct creates an ordered struct for properties to maintain field ordering in JSON.
func (s *Schema) createOrderedPropertiesStruct() interface{} {
	var structFields []reflect.StructField
	generator := &Generator{}

	// Add fields in the sorted order
	for _, field := range s.Fields {
		// Convert field name to a valid Go identifier (capitalize first letter)
		fieldName := strings.ToUpper(string(field.Name[0])) + field.Name[1:]

		structFields = append(structFields, reflect.StructField{
			Name: fieldName,
			Type: reflect.TypeOf(map[string]interface{}{}),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field.Name)),
		})
	}

	if len(structFields) == 0 {
		// Return empty struct if no fields
		return struct{}{}
	}

	// Create the struct type
	structType := reflect.StructOf(structFields)
	structValue := reflect.New(structType).Elem()

	// Set the values for each field
	for _, field := range s.Fields {
		fieldName := strings.ToUpper(string(field.Name[0])) + field.Name[1:]
		property := generator.schemaFieldToProperty(field)
		structValue.FieldByName(fieldName).Set(reflect.ValueOf(property))
	}

	return structValue.Addr().Interface()
}

// Generator provides methods to generate schemas from struct definitions.
type Generator struct{}

// NewGenerator creates a new SchemaGenerator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates a schema from a command type and definition struct.
func (g *Generator) Generate(commandType string, def interface{}, description ...string) (*Schema, error) {
	fields, err := g.extractFields(reflect.TypeOf(def))
	if err != nil {
		return nil, err
	}

	// Sort fields according to the required order: type, name, others (lexically), commands
	sortedFields := g.sortFields(fields)

	properties := make(map[string]Field)

	var required []string

	for _, field := range sortedFields {
		// Special handling for type field to add enum constraint
		if field.Name == "type" {
			field.Enum = []string{commandType}
		}

		properties[field.Name] = field

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema := &Schema{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		Fields:               sortedFields,
		AdditionalProperties: false,
	}

	// Set description if provided
	if len(description) > 0 && description[0] != "" {
		schema.Description = description[0]
	}

	return schema, nil
}

// GenerateJSONSchemaString generates a complete JSON schema for the entire configuration.
func (g *Generator) GenerateJSONSchemaString(f commands.CommanderFactory) (string, error) {
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
					"anyOf": g.generateCommandSchemas(f),
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
func (g *Generator) generateCommandSchemas(f commands.CommanderFactory) []interface{} {
	var schemas []interface{}

	// Access the default registry from commandregistry package
	// This will contain all registered command types and their commanders
	for commandType, commander := range f.Iter() {
		// Check if the commander implements SchemaProvider
		if provider, ok := commander.(Provider); ok {
			// Generate schema for this command type using its definition
			schemaStruct, err := g.Generate(
				commandType,
				provider.GetExampleDefinition(),
				provider.GetCommandDescription(),
			)
			if err != nil {
				// If we can't generate the schema, create a basic one
				schema := map[string]interface{}{
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
				schemas = append(schemas, schema)
			} else {
				// Convert the Schema struct to an ordered struct for proper JSON field ordering
				schemas = append(schemas, schemaStruct.ToOrderedStruct())
			}
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
func (b *BaseSchemaGenerator) WriteJSONSchema(w io.Writer, f commands.CommanderFactory) error {
	schema, err := b.generator.GenerateJSONSchemaString(f)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(schema))

	return err
}

// WriteYAMLSchema writes a YAML example to the writer using the provided definition.
func (b *BaseSchemaGenerator) WriteYAMLExample(w io.Writer, ex interface{}) error {
	// Import yaml package for marshaling
	yamlBytes, err := yaml.Marshal(ex)
	if err != nil {
		return fmt.Errorf("failed to marshal example definition to YAML: %w", err)
	}

	_, err = w.Write(yamlBytes)
	return err
}

// WriteMarkdownExample writes Markdown documentation to the writer using the provided definition and description.
func (b *BaseSchemaGenerator) WriteMarkdownExample(w io.Writer, commandType string, ex interface{}, description string) error {
	// Generate YAML example from definition
	yamlBytes, err := yaml.Marshal(ex)
	if err != nil {
		return fmt.Errorf("failed to marshal example definition to YAML: %w", err)
	}

	// Create markdown documentation
	title := strings.ToUpper(string(commandType[0])) + commandType[1:]
	doc := fmt.Sprintf(`# %s Command

%s

## Example Usage

`+"```yaml"+`
%s`+"```"+`

## Fields

`, title, description, string(yamlBytes))

	// Add field documentation if available
	if provider, ok := ex.(interface{ GetSchemaFields() []Field }); ok {
		fields := provider.GetSchemaFields()
		for _, field := range fields {
			required := ""
			if field.Required {
				required = " (required)"
			}
			doc += fmt.Sprintf("- **%s** (%s)%s", field.Name, field.Type, required)
			if field.Description != "" {
				doc += fmt.Sprintf(": %s", field.Description)
			}
			doc += "\n"
		}
	}

	_, err = w.Write([]byte(doc))
	return err
}
