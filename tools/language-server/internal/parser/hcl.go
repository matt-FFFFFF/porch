// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package parser

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/lonegunmanb/hclfuncs"
	"github.com/zclconf/go-cty/cty/function"
)

// HCLDocument represents a parsed HCL document.
type HCLDocument struct {
	URI       string
	Content   string
	File      *hcl.File
	Body      *hclsyntax.Body
	Functions map[string]function.Function
	Diags     hcl.Diagnostics
}

// Position represents a position in the document.
type Position struct {
	Line      int
	Character int
}

// Range represents a range in the document.
type Range struct {
	Start Position
	End   Position
}

// Diagnostic represents a diagnostic message.
type Diagnostic struct {
	Range    Range
	Severity int
	Message  string
	Source   string
}

// CompletionItem represents a completion suggestion.
type CompletionItem struct {
	Label         string
	Kind          int
	Detail        string
	Documentation string
	InsertText    string
}

// ParseDocument parses HCL content and returns a document representation.
func ParseDocument(ctx context.Context, uri, content string) (*HCLDocument, error) {
	file, diags := hclsyntax.ParseConfig([]byte(content), uri, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		log.Printf("HCL parsing errors for %s: %v", uri, diags)
		// Still continue with the document for basic functionality
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type")
	}

	// Get all available functions from hclfuncs
	functions := hclfuncs.Functions("")

	return &HCLDocument{
		URI:       uri,
		Content:   content,
		File:      file,
		Body:      body,
		Functions: functions,
		Diags:     diags,
	}, nil
}

// GetDiagnostics returns diagnostics for the document.
func (doc *HCLDocument) GetDiagnostics() []Diagnostic {
	var diagnostics []Diagnostic

	for _, diag := range doc.Diags {
		severity := 1 // Error
		if diag.Severity == hcl.DiagWarning {
			severity = 2 // Warning
		}

		start := Position{
			Line:      diag.Subject.Start.Line - 1, // LSP is 0-based
			Character: diag.Subject.Start.Column - 1,
		}
		end := Position{
			Line:      diag.Subject.End.Line - 1,
			Character: diag.Subject.End.Column - 1,
		}

		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: start,
				End:   end,
			},
			Severity: severity,
			Message:  diag.Summary + ": " + diag.Detail,
			Source:   "porch-hcl",
		})
	}

	return diagnostics
}

// GetCompletionItems returns completion items for the given position.
func (doc *HCLDocument) GetCompletionItems(ctx context.Context, pos Position) []CompletionItem {
	var items []CompletionItem

	// Convert 0-based LSP position to 1-based HCL position
	hclPos := hcl.Pos{
		Line:   pos.Line + 1,
		Column: pos.Character + 1,
	}

	// Get the current line content
	lines := strings.Split(doc.Content, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return items
	}

	currentLine := lines[pos.Line]
	beforeCursor := currentLine[:pos.Character]

	// Check if we're in a function call context
	if strings.Contains(beforeCursor, "(") && !strings.Contains(beforeCursor, ")") {
		// We're inside a function call, don't provide block completions
		return items
	}

	// Check if we're at the start of a line or after whitespace (block context)
	trimmed := strings.TrimSpace(beforeCursor)
	if trimmed == "" || !strings.Contains(trimmed, "=") {
		// Add block type completions
		items = append(items, CompletionItem{
			Label:         "command",
			Kind:          14, // Keyword
			Detail:        "Porch command block",
			Documentation: "Defines a single command to execute",
			InsertText:    "command \"${1:name}\" {\n\t${2:command} = \"${3:command to run}\"\n}",
		})

		items = append(items, CompletionItem{
			Label:         "command_chain",
			Kind:          14, // Keyword
			Detail:        "Porch command chain block",
			Documentation: "Defines a chain of commands to execute in sequence",
			InsertText:    "command_chain \"${1:name}\" {\n\t${2}\n}",
		})

		items = append(items, CompletionItem{
			Label:         "foreach_directory",
			Kind:          14, // Keyword
			Detail:        "Porch foreach directory block",
			Documentation: "Executes commands for each directory matching a pattern",
			InsertText:    "foreach_directory \"${1:name}\" {\n\tdirectory_pattern = \"${2:*}\"\n\t${3}\n}",
		})
	}

	// Add attribute completions if we're in a block
	if doc.isInBlock(hclPos) {
		commonAttrs := []string{
			"name", "description", "working_directory", "command",
			"directory_pattern", "depends_on", "condition", "timeout",
		}

		for _, attr := range commonAttrs {
			items = append(items, CompletionItem{
				Label:      attr,
				Kind:       10, // Property
				Detail:     "Porch attribute",
				InsertText: attr + " = \"${1}\"",
			})
		}
	}

	// Add HCL function completions
	for name := range doc.Functions {
		items = append(items, CompletionItem{
			Label:         name,
			Kind:          3, // Function
			Detail:        "HCL function",
			Documentation: "Built-in HCL function from hclfuncs",
			InsertText:    name + "(${1})",
		})
	}

	return items
}

// GetHoverInfo returns hover information for the given position.
func (doc *HCLDocument) GetHoverInfo(ctx context.Context, pos Position) string {
	// Get the word at the position
	word := doc.getWordAtPosition(pos)
	if word == "" {
		return ""
	}

	// Check if it's a function
	if _, exists := doc.Functions[word]; exists {
		return fmt.Sprintf("**%s** - HCL function\n\nBuilt-in function from hclfuncs package", word)
	}

	// Check if it's a Porch block type
	switch word {
	case "command":
		return "**command** - Porch block type\n\nDefines a single command to execute"
	case "command_chain":
		return "**command_chain** - Porch block type\n\nDefines a chain of commands to execute in sequence"
	case "foreach_directory":
		return "**foreach_directory** - Porch block type\n\nExecutes commands for each directory matching a pattern"
	}

	// Check if it's a common attribute
	attrDocs := map[string]string{
		"name":              "The name identifier for this block",
		"description":       "A human-readable description",
		"working_directory": "The directory to execute commands in",
		"command":           "The command to execute",
		"directory_pattern": "Pattern to match directories",
		"depends_on":        "List of dependencies",
		"condition":         "Conditional execution expression",
		"timeout":           "Execution timeout",
	}

	if doc, exists := attrDocs[word]; exists {
		return fmt.Sprintf("**%s** - Porch attribute\n\n%s", word, doc)
	}

	return ""
}

// isInBlock checks if the position is inside a block.
func (doc *HCLDocument) isInBlock(pos hcl.Pos) bool {
	// Simple heuristic: check if we're inside braces
	// This is a simplified implementation
	content := doc.Content
	lines := strings.Split(content, "\n")

	braceCount := 0

	for i := 0; i < pos.Line-1; i++ {
		if i < len(lines) {
			line := lines[i]
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		}
	}

	// Check current line up to the position
	if pos.Line-1 < len(lines) {
		currentLine := lines[pos.Line-1]
		if pos.Column-1 < len(currentLine) {
			partialLine := currentLine[:pos.Column-1]
			braceCount += strings.Count(partialLine, "{") - strings.Count(partialLine, "}")
		}
	}

	return braceCount > 0
}

// getWordAtPosition extracts the word at the given position.
func (doc *HCLDocument) getWordAtPosition(pos Position) string {
	lines := strings.Split(doc.Content, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}

	line := lines[pos.Line]
	if pos.Character < 0 || pos.Character >= len(line) {
		return ""
	}

	// Find word boundaries
	start := pos.Character
	end := pos.Character

	// Move start backwards to find word start
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Move end forwards to find word end
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	return line[start:end]
}

// isWordChar checks if a character is part of a word.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
