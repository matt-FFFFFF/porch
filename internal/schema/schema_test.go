// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package schema

import (
	"encoding/json"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateJSONSchemaString_IncludesCommandGroups(t *testing.T) {
	// Create a simple registry without importing command packages (to avoid import cycles)
	registry := commandregistry.New()

	// Generate the schema
	generator := NewGenerator()
	schemaJSON, err := generator.GenerateJSONSchemaString(registry)
	require.NoError(t, err)
	require.NotEmpty(t, schemaJSON)

	// Parse the JSON to verify structure
	var schema map[string]interface{}
	err = json.Unmarshal([]byte(schemaJSON), &schema)
	require.NoError(t, err)

	// Check that properties exist
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "Schema should have properties")

	// Check that command_groups is included
	commandGroups, ok := properties["command_groups"]
	require.True(t, ok, "Schema should include command_groups property")

	// Verify command_groups is an array
	commandGroupsSchema, ok := commandGroups.(map[string]interface{})
	require.True(t, ok, "command_groups should be an object")

	assert.Equal(t, "array", commandGroupsSchema["type"], "command_groups should be an array")
	assert.Contains(
		t,
		commandGroupsSchema["description"],
		"command group",
		"command_groups should have appropriate description",
	)

	// Check that command_groups has items schema
	items, ok := commandGroupsSchema["items"].(map[string]interface{})
	require.True(t, ok, "command_groups should have items schema")

	// Verify items schema has the expected properties
	itemsProperties, ok := items["properties"].(map[string]interface{})
	require.True(t, ok, "command_groups items should have properties")

	// Check that the command group properties exist
	assert.Contains(t, itemsProperties, "name", "command group should have name property")
	assert.Contains(t, itemsProperties, "description", "command group should have description property")
	assert.Contains(t, itemsProperties, "commands", "command group should have commands property")

	// Verify the schema is valid JSON and contains expected elements
	assert.Contains(t, schemaJSON, "command_groups")
	assert.Contains(t, schemaJSON, "List of command groups")

	// Check that both commands and command_groups are present
	assert.Contains(t, properties, "commands", "Schema should still include commands property")
	assert.Contains(t, properties, "command_groups", "Schema should include command_groups property")
}

func TestGenerateJSONSchemaString_ValidJSON(t *testing.T) {
	// Create a simple registry
	registry := commandregistry.New()

	// Generate the schema
	generator := NewGenerator()
	schemaJSON, err := generator.GenerateJSONSchemaString(registry)
	require.NoError(t, err)
	require.NotEmpty(t, schemaJSON)

	// Verify it's valid JSON
	var schema interface{}
	err = json.Unmarshal([]byte(schemaJSON), &schema)
	require.NoError(t, err, "Generated schema should be valid JSON")

	// Verify it's not just an empty object
	assert.Greater(t, len(schemaJSON), 100, "Schema should be substantial")

	// Should contain the schema meta-info
	assert.Contains(t, schemaJSON, "$schema")
	assert.Contains(t, schemaJSON, "Porch Configuration Schema")
}
