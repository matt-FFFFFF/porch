// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config"
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

var (
	// ErrNotAStruct is returned when a non-struct type is passed to schema generation.
	ErrNotAStruct = fmt.Errorf("expected a struct type, got a different type")

	// ErrCreatingSchema is returned when there is an error creating the schema.
	ErrCreatingSchema = errors.New("failed to create schema") //nolint:stylecheck
)

// NewErrNotAStruct creates a new error indicating that the provided type is not a struct.
func NewErrNotAStruct(kind string) error {
	return fmt.Errorf("%w: %s", ErrNotAStruct, kind)
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
	Title                string           `json:"title,omitempty"`
	Description          string           `json:"description,omitempty"`
	Properties           map[string]Field `json:"properties"`
	Required             []string         `json:"required,omitempty"`
	AdditionalProperties bool             `json:"additionalProperties,omitempty"`
	Fields               []Field          `json:"-"` // For internal use
	CommandType          string           `json:"-"` // For internal use
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

	// 2. Add "title" field if present
	if s.Title != "" {
		structFields = append(structFields, reflect.StructField{
			Name: "Title",
			Type: reflect.TypeOf(""),
			Tag:  `json:"title,omitempty"`,
		})
	}

	// 3. Add "description" field if present
	if s.Description != "" {
		structFields = append(structFields, reflect.StructField{
			Name: "Description",
			Type: reflect.TypeOf(""),
			Tag:  `json:"description,omitempty"`,
		})
	}

	// 4. Add "properties" field
	structFields = append(structFields, reflect.StructField{
		Name: "Properties",
		Type: reflect.TypeOf(orderedProperties).Elem(), // Get the underlying type
		Tag:  `json:"properties"`,
	})

	// 5. Add "required" field if present
	if len(s.Required) > 0 {
		structFields = append(structFields, reflect.StructField{
			Name: "Required",
			Type: reflect.TypeOf([]string{}),
			Tag:  `json:"required,omitempty"`,
		})
	}

	// 6. Add "additionalProperties" field
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

	if s.Title != "" {
		structValue.FieldByName("Title").SetString(s.Title)
	}

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

// createOrderedRootPropertiesStruct creates an ordered struct for root properties:
// name, description, command_groups, commands.
func (g *Generator) createOrderedRootPropertiesStruct(f commands.CommanderFactory) interface{} {
	// Extract field information from config.Definition struct
	definitionType := reflect.TypeOf(config.Definition{})
	rootFields := make(map[string]Field)

	// Extract fields from config.Definition using reflection and struct tags
	for i := 0; i < definitionType.NumField(); i++ {
		field := definitionType.Field(i)
		if field.IsExported() {
			schemaField := g.fieldToSchemaField(field)
			if schemaField == nil {
				continue // Skip fields that can't be processed
			}

			rootFields[schemaField.Name] = *schemaField
		}
	}

	// Create struct fields in the desired order: name, description, command_groups, commands
	var structFields []reflect.StructField

	// 1. Add "name" field
	structFields = append(structFields, reflect.StructField{
		Name: "Name",
		Type: reflect.TypeOf(map[string]interface{}{}),
		Tag:  `json:"name"`,
	})

	// 2. Add "description" field
	structFields = append(structFields, reflect.StructField{
		Name: "Description",
		Type: reflect.TypeOf(map[string]interface{}{}),
		Tag:  `json:"description"`,
	})

	// 3. Add "command_groups" field
	structFields = append(structFields, reflect.StructField{
		Name: "CommandGroups",
		Type: reflect.TypeOf(map[string]interface{}{}),
		Tag:  `json:"command_groups"`,
	})

	// 4. Add "commands" field
	structFields = append(structFields, reflect.StructField{
		Name: "Commands",
		Type: reflect.TypeOf(map[string]interface{}{}),
		Tag:  `json:"commands"`,
	})

	// Create the struct type
	structType := reflect.StructOf(structFields)
	structValue := reflect.New(structType).Elem()

	// Set the values using information from the config.Definition struct
	if nameField, exists := rootFields["name"]; exists {
		nameProperty := map[string]interface{}{
			"type":        nameField.Type,
			"description": nameField.Description,
		}
		structValue.FieldByName("Name").Set(reflect.ValueOf(nameProperty))
	}

	if descField, exists := rootFields["description"]; exists {
		descriptionProperty := map[string]interface{}{
			"type":        descField.Type,
			"description": descField.Description,
		}
		structValue.FieldByName("Description").Set(reflect.ValueOf(descriptionProperty))
	}

	if commandGroupsField, exists := rootFields["command_groups"]; exists {
		commandGroupsProperty := map[string]interface{}{
			"type":        "array",
			"description": commandGroupsField.Description,
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the command group",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the command group",
					},
					"commands": map[string]interface{}{
						"type":        "array",
						"description": "List of commands in this group",
						"items": map[string]interface{}{
							"anyOf": g.generateCommandSchemas(f),
						},
					},
				},
				"required":             []string{"name", "commands"},
				"additionalProperties": false,
			},
		}
		structValue.FieldByName("CommandGroups").Set(reflect.ValueOf(commandGroupsProperty))
	}

	if commandsField, exists := rootFields["commands"]; exists {
		commandsProperty := map[string]interface{}{
			"type":        "array",
			"description": commandsField.Description,
			"items": map[string]interface{}{
				"anyOf": g.generateCommandSchemas(f),
			},
		}
		structValue.FieldByName("Commands").Set(reflect.ValueOf(commandsProperty))
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

	for i, field := range sortedFields {
		// Special handling for type field to add enum constraint and default value
		if field.Name == "type" {
			field.Enum = []string{commandType}
			field.Default = commandType
			sortedFields[i] = field // Update the field in the slice
		}

		properties[field.Name] = field

		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema := &Schema{
		Type:                 "object",
		Title:                strings.ToUpper(string(commandType[0])) + commandType[1:] + " Command",
		Properties:           properties,
		Required:             required,
		Fields:               sortedFields,
		AdditionalProperties: false,
		CommandType:          commandType,
	}

	// Set description if provided
	if len(description) > 0 && description[0] != "" {
		schema.Description = description[0]
	}

	return schema, nil
}

// GenerateJSONSchemaString generates a complete JSON schema for the entire configuration.
func (g *Generator) GenerateJSONSchemaString(f commands.CommanderFactory) (string, error) {
	// Create the final schema with metadata
	finalSchema := g.createFinalRootSchema(f)

	bytes, err := json.MarshalIndent(finalSchema, "", "  ")
	if err != nil {
		return "", errors.Join(ErrCreatingSchema, err)
	}

	return string(bytes), nil
}

// createFinalRootSchema creates the final ordered root schema with proper metadata and command schemas.
func (g *Generator) createFinalRootSchema(f commands.CommanderFactory) interface{} {
	// Create struct fields in the desired order
	var structFields []reflect.StructField

	// 1. Add "$schema" field
	structFields = append(structFields, reflect.StructField{
		Name: "Schema",
		Type: reflect.TypeOf(""),
		Tag:  `json:"$schema"`,
	})

	// 2. Add "type" field
	structFields = append(structFields, reflect.StructField{
		Name: "Type",
		Type: reflect.TypeOf(""),
		Tag:  `json:"type"`,
	})

	// 3. Add "title" field
	structFields = append(structFields, reflect.StructField{
		Name: "Title",
		Type: reflect.TypeOf(""),
		Tag:  `json:"title"`,
	})

	// 4. Add "description" field
	structFields = append(structFields, reflect.StructField{
		Name: "Description",
		Type: reflect.TypeOf(""),
		Tag:  `json:"description"`,
	})

	// 5. Add "properties" field using the reordered properties
	propertiesStruct := g.createOrderedRootPropertiesStruct(f)
	structFields = append(structFields, reflect.StructField{
		Name: "Properties",
		Type: reflect.TypeOf(propertiesStruct).Elem(),
		Tag:  `json:"properties"`,
	})

	// 6. Add "required" field
	structFields = append(structFields, reflect.StructField{
		Name: "Required",
		Type: reflect.TypeOf([]string{}),
		Tag:  `json:"required"`,
	})

	// Create the struct type
	structType := reflect.StructOf(structFields)
	structValue := reflect.New(structType).Elem()

	// Set the values
	structValue.FieldByName("Schema").SetString("https://json-schema.org/draft/2020-12/schema")
	structValue.FieldByName("Type").SetString("object")
	structValue.FieldByName("Title").SetString("Porch Configuration Schema")
	structValue.FieldByName("Description").
		SetString("Schema for porch process orchestration framework configuration files")
	structValue.FieldByName("Properties").Set(reflect.ValueOf(propertiesStruct).Elem())
	structValue.FieldByName("Required").Set(reflect.ValueOf([]string{"commands"}))

	return structValue.Interface()
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
					"type":  "object",
					"title": strings.ToUpper(string(commandType[0])) + commandType[1:] + " Command",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type":    "string",
							"enum":    []string{commandType},
							"default": commandType,
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
		return nil, NewErrNotAStruct(t.Kind().String())
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

		schemaField := g.fieldToSchemaField(field)

		if schemaField != nil {
			fields = append(fields, *schemaField)
		}
	}

	return fields, nil
}

// fieldToSchemaField converts a reflect.StructField to a SchemaField.
func (g *Generator) fieldToSchemaField(field reflect.StructField) *Field {
	// Check for yaml tag to get the field name
	yamlTag := field.Tag.Get("yaml")
	if yamlTag == "-" {
		return nil // Skip this field
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
	required := yamlTag == "" || !strings.Contains(yamlTag, "omitempty")

	// Determine field type
	fieldType := g.getSchemaType(field.Type)

	return &Field{
		Name:        fieldName,
		Type:        fieldType,
		Description: description,
		Required:    required,
	}
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
		return err //nolint:wrapcheck
	}

	_, err = w.Write([]byte(schema))

	if err != nil {
		return errors.Join(ErrCreatingSchema, err)
	}

	return nil
}

// WriteYAMLExample writes a YAML example to the writer using the provided definition.
func (b *BaseSchemaGenerator) WriteYAMLExample(w io.Writer, ex interface{}) error {
	// Import yaml package for marshaling
	yamlBytes, err := yaml.Marshal(ex)
	if err != nil {
		return errors.Join(ErrCreatingSchema, err)
	}

	_, err = w.Write(yamlBytes)

	if err != nil {
		return errors.Join(ErrCreatingSchema, err)
	}

	return nil
}

// WriteMarkdownExample writes Markdown documentation to the writer using the provided definition
// and description.
func (b *BaseSchemaGenerator) WriteMarkdownExample(
	w io.Writer,
	commandType string,
	ex interface{},
	description string,
) error {
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

	if err != nil {
		return errors.Join(ErrCreatingSchema, err)
	}

	return nil
}
