// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
)

// SchemaField represents a field in a command's YAML schema.
type SchemaField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// CommandSchema represents the complete schema for a command type.
type CommandSchema struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Fields      []SchemaField `json:"fields"`
	Examples    []string      `json:"examples,omitempty"`
}

// SchemaGenerator can generate schema documentation from command definitions.
type SchemaGenerator struct{}

// NewSchemaGenerator creates a new schema generator.
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{}
}

// GenerateSchema generates schema documentation from a definition struct.
func (g *SchemaGenerator) GenerateSchema(cmdType string, definition interface{}) (*CommandSchema, error) {
	schema := &CommandSchema{
		Type:   cmdType,
		Fields: []SchemaField{},
	}

	// Use reflection to analyze the struct
	value := reflect.ValueOf(definition)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("definition must be a struct, got %s", value.Kind())
	}

	structType := value.Type()

	// Process each field
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := value.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle inline structs (like BaseDefinition)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == ",inline" || strings.Contains(yamlTag, "inline") {
			// Recursively process fields from the inline struct
			if fieldValue.Kind() == reflect.Struct {
				inlineType := fieldValue.Type()
				for j := 0; j < inlineType.NumField(); j++ {
					inlineField := inlineType.Field(j)
					inlineFieldValue := fieldValue.Field(j)

					if !inlineField.IsExported() {
						continue
					}

					schemaField := g.processField(inlineField, inlineFieldValue)
					if schemaField != nil {
						schema.Fields = append(schema.Fields, *schemaField)
					}
				}
			}
			continue
		}

		schemaField := g.processField(field, fieldValue)
		if schemaField != nil {
			schema.Fields = append(schema.Fields, *schemaField)
		}
	}

	return schema, nil
}

// processField processes a single struct field and returns schema information.
func (g *SchemaGenerator) processField(field reflect.StructField, value reflect.Value) *SchemaField {
	yamlTag := field.Tag.Get("yaml")
	if yamlTag == "-" {
		return nil // Skip fields marked with yaml:"-"
	}

	// Parse yaml tag
	yamlName := g.parseYAMLTag(yamlTag)
	if yamlName == "" {
		yamlName = strings.ToLower(field.Name)
	}

	schemaField := &SchemaField{
		Name:        yamlName,
		Type:        g.getFieldType(field.Type),
		Required:    !strings.Contains(yamlTag, "omitempty"),
		Description: g.getFieldDescription(field),
	}

	// Set example values based on field type and name
	schemaField.Example = g.generateExampleValue(field, value)

	return schemaField
}

// parseYAMLTag extracts the field name from a YAML tag.
func (g *SchemaGenerator) parseYAMLTag(tag string) string {
	if tag == "" {
		return ""
	}

	parts := strings.Split(tag, ",")
	return parts[0]
}

// getFieldType returns a human-readable type description.
func (g *SchemaGenerator) getFieldType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		elementType := g.getFieldType(t.Elem())
		return fmt.Sprintf("array of %s", elementType)
	case reflect.Map:
		return "object"
	case reflect.Interface:
		return "any"
	default:
		return t.String()
	}
}

// getFieldDescription extracts description from docdesc struct tag or generates one.
func (g *SchemaGenerator) getFieldDescription(field reflect.StructField) string {
	// First try to get description from docdesc tag
	if desc := field.Tag.Get("docdesc"); desc != "" {
		return desc
	}

	return fmt.Sprintf("Configuration for %s", strings.ToLower(field.Name))
}

// generateExampleValue creates appropriate example values for different field types.
func (g *SchemaGenerator) generateExampleValue(field reflect.StructField, value reflect.Value) interface{} {
	name := strings.ToLower(field.Name)

	switch field.Type.Kind() {
	case reflect.String:
		switch name {
		case "type":
			return "" // Let the caller set this
		case "name":
			return "my-command"
		case "workingdirectory":
			return "/path/to/working/directory"
		case "runsoncondition":
			return "success"
		case "commandline":
			return "echo 'Hello, World!'"
		case "mode":
			return "parallel"
		case "workingdirectorystrategy":
			return "item_relative"
		default:
			return "example_value"
		}
	case reflect.Int:
		if name == "depth" {
			return 1
		}
		return 0
	case reflect.Bool:
		return false
	case reflect.Slice:
		if strings.Contains(name, "exitcodes") {
			return []int{0, 2, 99}
		}
		if name == "commands" {
			return []interface{}{
				map[string]interface{}{
					"type":         "shell",
					"name":         "Sub Command",
					"command_line": "echo 'sub command'",
				},
			}
		}
		return []interface{}{}
	case reflect.Map:
		if name == "env" {
			return map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			}
		}
		return map[string]interface{}{}
	default:
		return nil
	}
}

// GenerateYAMLExample generates a complete YAML example for a command.
func (g *SchemaGenerator) GenerateYAMLExample(schema *CommandSchema) (string, error) {
	example := make(map[string]interface{})

	// Set the type first
	example["type"] = schema.Type

	// Add all fields with their example values
	for _, field := range schema.Fields {
		if field.Example != nil {
			example[field.Name] = field.Example
		}
	}

	yamlBytes, err := yaml.Marshal(example)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML example: %w", err)
	}

	return string(yamlBytes), nil
}

// GenerateMarkdownDoc generates markdown documentation for a command schema.
func (g *SchemaGenerator) GenerateMarkdownDoc(schema *CommandSchema) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s Command\n\n", strings.ToUpper(string(schema.Type[0]))+strings.ToLower(schema.Type[1:])))

	if schema.Description != "" {
		builder.WriteString(fmt.Sprintf("%s\n\n", schema.Description))
	}

	builder.WriteString("## Fields\n\n")
	builder.WriteString("| Field | Type | Required | Description |\n")
	builder.WriteString("|-------|------|----------|-------------|\n")

	for _, field := range schema.Fields {
		required := "No"
		if field.Required {
			required = "Yes"
		}
		builder.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			field.Name, field.Type, required, field.Description))
	}

	builder.WriteString("\n## Example\n\n```yaml\n")

	if yamlExample, err := g.GenerateYAMLExample(schema); err == nil {
		builder.WriteString(yamlExample)
	}

	builder.WriteString("```\n")

	return builder.String()
}

// JSONSchemaProperty represents a property in JSON Schema format.
type JSONSchemaProperty struct {
	Type        interface{}                    `json:"type,omitempty"`
	Description string                         `json:"description,omitempty"`
	Items       *JSONSchemaProperty            `json:"items,omitempty"`
	Properties  map[string]*JSONSchemaProperty `json:"properties,omitempty"`
	Required    []string                       `json:"required,omitempty"`
	OneOf       []*JSONSchemaProperty          `json:"oneOf,omitempty"`
	Enum        []interface{}                  `json:"enum,omitempty"`
	Default     interface{}                    `json:"default,omitempty"`
	Examples    []interface{}                  `json:"examples,omitempty"`
	Ref         string                         `json:"$ref,omitempty"`
}

// JSONSchema represents a complete JSON Schema.
type JSONSchema struct {
	Schema      string                         `json:"$schema"`
	Title       string                         `json:"title"`
	Type        string                         `json:"type"`
	Description string                         `json:"description,omitempty"`
	Properties  map[string]*JSONSchemaProperty `json:"properties"`
	Required    []string                       `json:"required"`
	Definitions map[string]*JSONSchemaProperty `json:"definitions,omitempty"`
}

// ConfigSchema represents the full configuration schema with all command types.
type ConfigSchema struct {
	RootSchema     *JSONSchema                    `json:"root_schema"`
	CommandSchemas map[string]*JSONSchemaProperty `json:"command_schemas"`
}

// GenerateFullJSONSchema generates a complete JSON Schema for the entire porch configuration.
func (g *SchemaGenerator) GenerateFullJSONSchema() (*JSONSchema, error) {
	// Create the root schema for the configuration file
	schema := &JSONSchema{
		Schema:      "http://json-schema.org/draft-07/schema#",
		Title:       "Porch Configuration Schema",
		Type:        "object",
		Description: "Schema for porch process orchestration framework configuration files",
		Properties: map[string]*JSONSchemaProperty{
			"name": {
				Type:        "string",
				Description: "Name of the configuration",
				Examples:    []interface{}{"My Build Pipeline"},
			},
			"description": {
				Type:        "string",
				Description: "Description of what this configuration does",
				Examples:    []interface{}{"Builds and deploys the application"},
			},
			"commands": {
				Type:        "array",
				Description: "List of commands to execute",
				Items: &JSONSchemaProperty{
					OneOf: g.generateCommandOneOfSchema(),
				},
			},
		},
		Required:    []string{"commands"},
		Definitions: make(map[string]*JSONSchemaProperty),
	}

	// Generate definitions for each command type
	commandTypes := []string{"shell", "serial", "parallel", "foreachdirectory", "copycwdtotemp"}
	for _, cmdType := range commandTypes {
		def, err := g.generateCommandJSONSchema(cmdType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for %s: %w", cmdType, err)
		}
		schema.Definitions[cmdType+"Command"] = def
	}

	return schema, nil
}

// generateCommandOneOfSchema creates a oneOf schema for all command types.
func (g *SchemaGenerator) generateCommandOneOfSchema() []*JSONSchemaProperty {
	commandTypes := []string{"shell", "serial", "parallel", "foreachdirectory", "copycwdtotemp"}
	oneOf := make([]*JSONSchemaProperty, len(commandTypes))

	for i, cmdType := range commandTypes {
		oneOf[i] = &JSONSchemaProperty{
			Ref: fmt.Sprintf("#/definitions/%sCommand", cmdType),
		}
	}

	return oneOf
}

// generateCommandJSONSchema generates JSON Schema for a specific command type.
func (g *SchemaGenerator) generateCommandJSONSchema(cmdType string) (*JSONSchemaProperty, error) {
	// Use simplified struct definitions that only include the command-specific fields
	// BaseDefinition fields are handled via inline embedding
	var definition interface{}

	switch cmdType {
	case "shell":
		definition = &struct {
			BaseDefinition   `yaml:",inline"`
			CommandLine      string `yaml:"command_line" docdesc:"The command to execute, can be a path or a command name"`
			SuccessExitCodes []int  `yaml:"success_exit_codes,omitempty" docdesc:"Exit codes that indicate success, defaults to 0"`
			SkipExitCodes    []int  `yaml:"skip_exit_codes,omitempty" docdesc:"Exit codes that indicate skip remaining tasks, defaults to empty"`
		}{}
	case "serial", "parallel":
		definition = &struct {
			BaseDefinition `yaml:",inline"`
			Commands       []interface{} `yaml:"commands" docdesc:"List of commands to execute"`
		}{}
	case "foreachdirectory":
		definition = &struct {
			BaseDefinition           `yaml:",inline"`
			Mode                     string        `yaml:"mode" docdesc:"Execution mode: 'parallel' or 'serial'"`
			Depth                    int           `yaml:"depth" docdesc:"Directory traversal depth (0 for unlimited)"`
			IncludeHidden            bool          `yaml:"include_hidden" docdesc:"Whether to include hidden directories in traversal"`
			WorkingDirectoryStrategy string        `yaml:"working_directory_strategy" docdesc:"Strategy for setting working directory: 'none', 'item_relative', or 'item_absolute'"`
			Commands                 []interface{} `yaml:"commands" docdesc:"List of commands to execute in each directory"`
		}{}
	case "copycwdtotemp":
		// copycwdtotemp only uses BaseDefinition fields (working_directory serves as the source directory)
		definition = &struct {
			BaseDefinition `yaml:",inline"`
		}{}
	default:
		return nil, fmt.Errorf("unknown command type: %s", cmdType)
	}

	return g.convertToJSONSchemaProperty(definition, cmdType)
}

// GenerateCommandJSONSchema generates JSON Schema for a specific command type (public method).
func (g *SchemaGenerator) GenerateCommandJSONSchema(cmdType string) (*JSONSchemaProperty, error) {
	return g.generateCommandJSONSchema(cmdType)
}

// MarshalJSONSchema marshals a JSON schema property to a formatted JSON string.
func (g *SchemaGenerator) MarshalJSONSchema(schema *JSONSchemaProperty) (string, error) {
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}
	return string(jsonBytes), nil
}

// convertToJSONSchemaProperty converts a Go struct to a JSON Schema property.
func (g *SchemaGenerator) convertToJSONSchemaProperty(definition interface{}, cmdType string) (*JSONSchemaProperty, error) {
	value := reflect.ValueOf(definition)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("definition must be a struct, got %s", value.Kind())
	}

	structType := value.Type()

	property := &JSONSchemaProperty{
		Type:        "object",
		Description: g.getCommandTypeDescription(cmdType),
		Properties:  make(map[string]*JSONSchemaProperty),
		Required:    []string{"type"},
	}

	// Add type constraint
	property.Properties["type"] = &JSONSchemaProperty{
		Type:        "string",
		Description: "The type of command",
		Enum:        []interface{}{cmdType},
	}

	// Process each field
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := value.Field(i)

		if !field.IsExported() {
			continue
		}

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "-" {
			continue
		}

		// Handle inline structs (like BaseDefinition)
		if yamlTag == ",inline" || strings.Contains(yamlTag, "inline") {
			// Recursively process fields from the inline struct
			if fieldValue.Kind() == reflect.Struct {
				inlineType := fieldValue.Type()
				for j := 0; j < inlineType.NumField(); j++ {
					inlineField := inlineType.Field(j)
					inlineFieldValue := fieldValue.Field(j)

					if !inlineField.IsExported() {
						continue
					}

					inlineYamlTag := inlineField.Tag.Get("yaml")
					if inlineYamlTag == "-" {
						continue
					}

					inlineYamlName := g.parseYAMLTag(inlineYamlTag)
					if inlineYamlName == "" {
						inlineYamlName = strings.ToLower(inlineField.Name)
					}

					if inlineYamlName == "type" {
						continue // Already handled above
					}

					inlineFieldProperty := g.convertFieldToJSONSchemaProperty(inlineField, inlineFieldValue)
					if inlineFieldProperty != nil {
						property.Properties[inlineYamlName] = inlineFieldProperty

						// Add to required if not omitempty
						if !strings.Contains(inlineYamlTag, "omitempty") && inlineYamlName != "type" {
							property.Required = append(property.Required, inlineYamlName)
						}
					}
				}
			}
			continue
		}

		yamlName := g.parseYAMLTag(yamlTag)
		if yamlName == "" {
			yamlName = strings.ToLower(field.Name)
		}

		if yamlName == "type" {
			continue // Already handled above
		}

		fieldProperty := g.convertFieldToJSONSchemaProperty(field, fieldValue)
		if fieldProperty != nil {
			property.Properties[yamlName] = fieldProperty

			// Add to required if not omitempty
			if !strings.Contains(yamlTag, "omitempty") && yamlName != "type" {
				property.Required = append(property.Required, yamlName)
			}
		}
	}

	return property, nil
}

// convertFieldToJSONSchemaProperty converts a single struct field to JSON Schema property.
func (g *SchemaGenerator) convertFieldToJSONSchemaProperty(field reflect.StructField, value reflect.Value) *JSONSchemaProperty {
	property := &JSONSchemaProperty{
		Description: g.getFieldDescription(field),
	}

	switch field.Type.Kind() {
	case reflect.String:
		property.Type = "string"
		if example := g.generateExampleValue(field, value); example != nil {
			property.Examples = []interface{}{example}
		}

		// Add enum constraints for specific fields
		fieldName := strings.ToLower(field.Name)
		if fieldName == "runsoncondition" {
			property.Enum = []interface{}{"success", "error", "always", "exit-codes"}
		} else if fieldName == "mode" {
			property.Enum = []interface{}{"parallel", "serial"}
		} else if fieldName == "workingdirectorystrategy" {
			property.Enum = []interface{}{"none", "item_relative", "item_absolute"}
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		property.Type = "integer"
		if example := g.generateExampleValue(field, value); example != nil {
			property.Examples = []interface{}{example}
		}

	case reflect.Bool:
		property.Type = "boolean"
		if example := g.generateExampleValue(field, value); example != nil {
			property.Examples = []interface{}{example}
		}

	case reflect.Slice:
		property.Type = "array"
		elementType := field.Type.Elem()

		if elementType.Kind() == reflect.Int {
			property.Items = &JSONSchemaProperty{
				Type: "integer",
			}
		} else if elementType.Kind() == reflect.Interface {
			// For commands array - reference back to command oneOf
			property.Items = &JSONSchemaProperty{
				OneOf: g.generateCommandOneOfSchema(),
			}
		} else {
			property.Items = &JSONSchemaProperty{
				Type: g.getJSONSchemaType(elementType.Kind()),
			}
		}

	case reflect.Map:
		property.Type = "object"
		if field.Type.Key().Kind() == reflect.String && field.Type.Elem().Kind() == reflect.String {
			// For string->string maps like env variables
			property.Properties = nil
		}

	default:
		property.Type = "string" // fallback
	}

	return property
}

// getJSONSchemaType converts Go reflect.Kind to JSON Schema type.
func (g *SchemaGenerator) getJSONSchemaType(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "string"
	}
}

// getCommandTypeDescription returns a description for each command type.
func (g *SchemaGenerator) getCommandTypeDescription(cmdType string) string {
	switch cmdType {
	case "shell":
		return "Execute shell commands with environment control and exit code handling"
	case "serial":
		return "Execute a list of commands sequentially, one after another"
	case "parallel":
		return "Execute a list of commands in parallel for improved performance"
	case "foreachdirectory":
		return "Execute commands for each directory found in the file system"
	case "copycwdtotemp":
		return "Copy current working directory to a temporary location before executing commands"
	default:
		return fmt.Sprintf("Configuration for %s command", cmdType)
	}
}

// GenerateJSONSchemaString generates the complete JSON Schema as a formatted string.
func (g *SchemaGenerator) GenerateJSONSchemaString() (string, error) {
	schema, err := g.GenerateFullJSONSchema()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	return string(jsonBytes), nil
}

// SchemaProvider interface allows commands to provide their own schema information.
type SchemaProvider interface {
	// GetSchemaFields returns the schema fields for this command type
	GetSchemaFields() []SchemaField
	// GetCommandType returns the command type string
	GetCommandType() string
	// GetCommandDescription returns a description of what this command does
	GetCommandDescription() string
	// GetExampleDefinition returns an example definition for YAML generation
	GetExampleDefinition() interface{}
}

// SchemaWriter provides methods to write schemas in different formats to an io.Writer.
type SchemaWriter interface {
	// WriteYAMLSchema writes YAML schema documentation to the writer
	WriteYAMLSchema(w io.Writer) error
	// WriteMarkdownSchema writes Markdown schema documentation to the writer
	WriteMarkdownSchema(w io.Writer) error
	// WriteJSONSchema writes JSON Schema to the writer
	WriteJSONSchema(w io.Writer) error
}

// BaseSchemaGenerator provides common schema generation functionality.
type BaseSchemaGenerator struct {
	provider SchemaProvider
}

// NewBaseSchemaGenerator creates a new base schema generator.
func NewBaseSchemaGenerator(provider SchemaProvider) *BaseSchemaGenerator {
	return &BaseSchemaGenerator{provider: provider}
}

// WriteYAMLSchema writes YAML schema documentation to the writer
func (g *BaseSchemaGenerator) WriteYAMLSchema(w io.Writer) error {
	schema := &CommandSchema{
		Type:        g.provider.GetCommandType(),
		Description: g.provider.GetCommandDescription(),
		Fields:      g.provider.GetSchemaFields(),
	}

	fmt.Fprintf(w, "# YAML Schema for '%s' command\n", schema.Type)
	fmt.Fprintf(w, "# %s\n\n", schema.Description)

	// Generate comprehensive YAML schema showing all fields
	for _, field := range schema.Fields {
		comment := field.Description
		if field.Required {
			comment += " (required)"
		} else {
			comment += " (optional)"
		}

		// Generate appropriate example value based on field type and name
		var exampleValue string
		switch field.Type {
		case "string":
			if field.Example != nil && fmt.Sprintf("%v", field.Example) != "" {
				exampleValue = fmt.Sprintf("%v", field.Example)
			} else {
				exampleValue = g.getExampleStringValue(field.Name)
			}
		case "[]string", "array of string":
			if field.Example != nil {
				exampleValue = fmt.Sprintf("\n%s", g.formatArrayExample(field.Example))
			} else {
				exampleValue = fmt.Sprintf("\n%s", g.getExampleArrayValue(field.Name))
			}
		case "[]int", "array of integer":
			if field.Example != nil {
				exampleValue = fmt.Sprintf("\n%s", g.formatArrayExample(field.Example))
			} else {
				exampleValue = fmt.Sprintf("\n%s", g.getExampleIntArrayValue(field.Name))
			}
		case "map[string]string", "object":
			// Check if it's an env field specifically to avoid formatting other objects
			if field.Example != nil && field.Name == "env" {
				// Format as YAML map properly
				if m, ok := field.Example.(map[string]string); ok {
					var lines []string
					for k, v := range m {
						lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
					}
					exampleValue = fmt.Sprintf("\n%s", strings.Join(lines, "\n"))
				} else {
					exampleValue = fmt.Sprintf("\n%s", g.getExampleMapValue(field.Name))
				}
			} else if len(strings.TrimSpace(g.getExampleMapValue(field.Name))) > 0 {
				exampleValue = fmt.Sprintf("\n%s", g.getExampleMapValue(field.Name))
			} else {
				exampleValue = "{}"
			}
		case "[]any", "[]interface{}", "array of any":
			if field.Example != nil {
				exampleValue = fmt.Sprintf("\n%s", g.formatArrayExample(field.Example))
			} else {
				exampleValue = fmt.Sprintf("\n%s", g.getExampleAnyArrayValue(field.Name))
			}
		case "bool":
			exampleValue = "true"
		case "int":
			if field.Name == "depth" {
				exampleValue = "2"
			} else {
				exampleValue = "0"
			}
		default:
			if field.Example != nil {
				exampleValue = fmt.Sprintf("%v", field.Example)
			} else {
				exampleValue = ""
			}
		}

		if exampleValue != "" {
			fmt.Fprintf(w, "%s: %s  # %s\n", field.Name, exampleValue, comment)
		} else {
			fmt.Fprintf(w, "# %s: <value>  # %s\n", field.Name, comment)
		}
	}

	return nil
}

// getExampleStringValue returns an example string value for a field name
func (g *BaseSchemaGenerator) getExampleStringValue(fieldName string) string {
	switch fieldName {
	case "type":
		return g.provider.GetCommandType()
	case "name":
		return "my-command-name"
	case "working_directory":
		return "/path/to/directory"
	case "runs_on_condition":
		return "success"
	case "command_line":
		return "echo 'example command'"
	case "mode":
		return "parallel"
	case "working_directory_strategy":
		return "item_relative"
	default:
		return "example-value"
	}
}

// formatArrayExample formats an array example for YAML output
func (g *BaseSchemaGenerator) formatArrayExample(example interface{}) string {
	switch v := example.(type) {
	case []string:
		result := ""
		for _, item := range v {
			result += fmt.Sprintf("- %s\n", item)
		}
		return strings.TrimSuffix(result, "\n")
	case []int:
		result := ""
		for _, item := range v {
			result += fmt.Sprintf("- %d\n", item)
		}
		return strings.TrimSuffix(result, "\n")
	case []interface{}:
		result := ""
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				// Format map as YAML
				result += "- "
				first := true
				for k, val := range m {
					if !first {
						result += "  "
					}
					result += fmt.Sprintf("%s: %v\n", k, val)
					first = false
				}
			} else {
				result += fmt.Sprintf("- %v\n", item)
			}
		}
		return strings.TrimSuffix(result, "\n")
	default:
		return fmt.Sprintf("- %v", example)
	}
}

// getExampleArrayValue returns an example string array value for a field name
func (g *BaseSchemaGenerator) getExampleArrayValue(fieldName string) string {
	switch fieldName {
	case "command_line":
		return "- echo\n- 'Hello World'"
	default:
		return "- example-item"
	}
}

// getExampleIntArrayValue returns an example int array value for a field name
func (g *BaseSchemaGenerator) getExampleIntArrayValue(fieldName string) string {
	switch fieldName {
	case "success_exit_codes":
		return "- 0"
	case "skip_exit_codes":
		return "- 1"
	case "runs_on_exit_codes":
		return "- 1\n- 2"
	default:
		return "- 0"
	}
}

// getExampleMapValue returns an example map value for a field name
func (g *BaseSchemaGenerator) getExampleMapValue(fieldName string) string {
	switch fieldName {
	case "env":
		return "  ENV_VAR: value\n  ANOTHER_VAR: another-value"
	default:
		return "" // Return empty string for optional maps
	}
}

// getExampleAnyArrayValue returns an example any array value for a field name
func (g *BaseSchemaGenerator) getExampleAnyArrayValue(fieldName string) string {
	switch fieldName {
	case "commands":
		return "- type: shell\n  name: sub-command\n  command_line: echo 'sub command'"
	default:
		return "- item1\n- item2"
	}
}

// WriteMarkdownSchema writes Markdown schema documentation to the writer
func (g *BaseSchemaGenerator) WriteMarkdownSchema(w io.Writer) error {
	schema := &CommandSchema{
		Type:        g.provider.GetCommandType(),
		Description: g.provider.GetCommandDescription(),
		Fields:      g.provider.GetSchemaFields(),
	}

	fmt.Fprintf(w, "# %s Command\n\n", strings.ToUpper(string(schema.Type[0]))+strings.ToLower(schema.Type[1:]))

	if schema.Description != "" {
		fmt.Fprintf(w, "%s\n\n", schema.Description)
	}

	fmt.Fprintf(w, "## Fields\n\n")
	fmt.Fprintf(w, "| Field | Type | Required | Description |\n")
	fmt.Fprintf(w, "|-------|------|----------|-------------|\n")

	for _, field := range schema.Fields {
		required := "No"
		if field.Required {
			required = "Yes"
		}
		fmt.Fprintf(w, "| `%s` | %s | %s | %s |\n",
			field.Name, field.Type, required, field.Description)
	}

	fmt.Fprintf(w, "\n## Example\n\n```yaml\n")

	exampleDef := g.provider.GetExampleDefinition()
	yamlBytes, err := yaml.Marshal(exampleDef)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML example: %w", err)
	}

	fmt.Fprint(w, string(yamlBytes))
	fmt.Fprintf(w, "```\n")

	return nil
}

// WriteJSONSchema writes JSON Schema to the writer
func (g *BaseSchemaGenerator) WriteJSONSchema(w io.Writer) error {
	schema := &CommandSchema{
		Type:        g.provider.GetCommandType(),
		Description: g.provider.GetCommandDescription(),
		Fields:      g.provider.GetSchemaFields(),
	}

	fmt.Fprintf(w, "# JSON Schema for '%s' command\n", schema.Type)
	fmt.Fprintf(w, "# %s\n\n", schema.Description)

	// Convert to JSON Schema format
	jsonSchema := g.convertSchemaToJSONSchema(schema)

	jsonBytes, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	fmt.Fprint(w, string(jsonBytes))
	return nil
}

// convertSchemaToJSONSchema converts a CommandSchema to JSON Schema format
func (g *BaseSchemaGenerator) convertSchemaToJSONSchema(schema *CommandSchema) *JSONSchemaProperty {
	properties := make(map[string]*JSONSchemaProperty)
	var required []string

	// Add type constraint
	properties["type"] = &JSONSchemaProperty{
		Type:        "string",
		Description: "The type of command",
		Enum:        []interface{}{schema.Type},
	}
	required = append(required, "type")

	// Convert each field
	for _, field := range schema.Fields {
		if field.Name == "type" {
			continue // Already handled above
		}

		property := &JSONSchemaProperty{
			Description: field.Description,
		}

		// Set type and other properties based on field type
		switch field.Type {
		case "string":
			property.Type = "string"
			if field.Example != nil {
				property.Examples = []interface{}{field.Example}
			}
		case "integer":
			property.Type = "integer"
			if field.Example != nil {
				property.Examples = []interface{}{field.Example}
			}
		case "boolean":
			property.Type = "boolean"
			if field.Example != nil {
				property.Examples = []interface{}{field.Example}
			}
		case "object":
			property.Type = "object"
		default:
			if strings.HasPrefix(field.Type, "array of") {
				property.Type = "array"
				elementType := strings.TrimPrefix(field.Type, "array of ")
				property.Items = &JSONSchemaProperty{
					Type: elementType,
				}
			} else {
				property.Type = "string" // fallback
			}
		}

		properties[field.Name] = property

		if field.Required {
			required = append(required, field.Name)
		}
	}

	return &JSONSchemaProperty{
		Type:        "object",
		Description: g.provider.GetCommandDescription(),
		Properties:  properties,
		Required:    required,
	}
}
