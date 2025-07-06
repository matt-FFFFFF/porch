// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/matt-FFFFFF/porch/tools/language-server/internal/parser"
	"github.com/zclconf/go-cty/cty/function"
)

// LSP Message types.
type Message struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// ResponseMessage represents a JSON-RPC response (always has result field).
type ResponseMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

// ErrorResponseMessage represents a JSON-RPC error response.
type ErrorResponseMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Error   interface{} `json:"error"`
}

// Position represents a position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier includes version.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents a text document item.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentContentChangeEvent represents a change to a text document.
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// TextDocumentPositionParams contains params for position-based requests.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionItem represents a completion suggestion.
type CompletionItem struct {
	Label            string `json:"label"`
	Kind             int    `json:"kind,omitempty"`
	Detail           string `json:"detail,omitempty"`
	Documentation    string `json:"documentation,omitempty"`
	InsertText       string `json:"insertText,omitempty"`
	InsertTextFormat int    `json:"insertTextFormat,omitempty"` // 1=PlainText, 2=Snippet
	SortText         string `json:"sortText,omitempty"`
}

// Hover represents hover information.
type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range,omitempty"`
}

// Diagnostic represents a diagnostic message.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// CompletionItemKind constants.
const (
	CompletionItemKindText          = 1
	CompletionItemKindMethod        = 2
	CompletionItemKindFunction      = 3
	CompletionItemKindConstructor   = 4
	CompletionItemKindField         = 5
	CompletionItemKindVariable      = 6
	CompletionItemKindClass         = 7
	CompletionItemKindInterface     = 8
	CompletionItemKindModule        = 9
	CompletionItemKindProperty      = 10
	CompletionItemKindUnit          = 11
	CompletionItemKindValue         = 12
	CompletionItemKindEnum          = 13
	CompletionItemKindKeyword       = 14
	CompletionItemKindSnippet       = 15
	CompletionItemKindColor         = 16
	CompletionItemKindFile          = 17
	CompletionItemKindReference     = 18
	CompletionItemKindFolder        = 19
	CompletionItemKindEnumMember    = 20
	CompletionItemKindConstant      = 21
	CompletionItemKindStruct        = 22
	CompletionItemKindEvent         = 23
	CompletionItemKindOperator      = 24
	CompletionItemKindTypeParameter = 25
)

// DiagnosticSeverity constants.
const (
	DiagnosticSeverityError       = 1
	DiagnosticSeverityWarning     = 2
	DiagnosticSeverityInformation = 3
	DiagnosticSeverityHint        = 4
)

// Server implements an LSP server for Porch HCL files.
type Server struct {
	mu        sync.RWMutex
	documents map[string]*parser.HCLDocument
	reader    *bufio.Reader
	writer    io.Writer
}

// NewServer creates a new LSP server instance.
func NewServer(reader io.Reader, writer io.Writer) *Server {
	// Ensure all logging goes to stderr to avoid interfering with LSP protocol
	log.SetOutput(os.Stderr)

	return &Server{
		documents: make(map[string]*parser.HCLDocument),
		reader:    bufio.NewReader(reader),
		writer:    writer,
	}
}

// Run starts the language server using stdio.
func (s *Server) Run(ctx context.Context) error {
	log.Println("Starting Porch HCL Language Server...")
	log.Printf("Server running with PID: %d", os.Getpid())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.handleMessage(ctx); err != nil {
				if err == io.EOF {
					log.Println("Client disconnected")
					return nil
				}

				log.Printf("Error handling message: %v", err)

				continue
			}
		}
	}
}

// handleMessage reads and processes a single LSP message.
func (s *Server) handleMessage(ctx context.Context) error {
	// Read headers
	var contentLength int

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return err
		}

		// Remove \r\n or \n
		line = strings.TrimRight(line, "\r\n")

		// Empty line means headers are done
		if line == "" {
			break
		}

		// Parse Content-Length header
		if strings.HasPrefix(line, "Content-Length: ") {
			lengthStr := strings.TrimPrefix(line, "Content-Length: ")
			if length, err := strconv.Atoi(lengthStr); err == nil {
				contentLength = length
				log.Printf("Content-Length: %d", contentLength)
			}
		}
	}

	if contentLength == 0 {
		return fmt.Errorf("no content length specified")
	}

	// Read the message body
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, body); err != nil {
		return err
	}

	log.Printf("Received message: %s", string(body))

	// Parse the JSON message
	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return err
	}

	log.Printf("Processing method: %s", msg.Method)

	// Handle the message
	return s.processMessage(ctx, &msg)
}

// processMessage processes an LSP message and sends a response if needed.
func (s *Server) processMessage(ctx context.Context, msg *Message) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return s.handleInitialized(msg)
	case "textDocument/didOpen":
		return s.handleTextDocumentDidOpen(ctx, msg)
	case "textDocument/didChange":
		return s.handleTextDocumentDidChange(ctx, msg)
	case "textDocument/didClose":
		return s.handleTextDocumentDidClose(msg)
	case "textDocument/completion":
		return s.handleCompletion(ctx, msg)
	case "textDocument/hover":
		return s.handleHover(ctx, msg)
	case "shutdown":
		return s.handleShutdown(msg)
	case "exit":
		return io.EOF
	default:
		log.Printf("Unhandled method: %s", msg.Method)
		return nil
	}
}

// sendResponse sends a JSON-RPC response.
func (s *Server) sendResponse(id interface{}, result interface{}) error {
	response := ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result, // Allow explicit null values
	}

	return s.sendMessage(response)
}

// sendError sends a JSON-RPC error response.
func (s *Server) sendError(id interface{}, code int, message string) error {
	errorObj := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	response := ErrorResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error:   errorObj,
	}

	return s.sendMessage(response)
}

// sendNotification sends a JSON-RPC notification.
func (s *Server) sendNotification(method string, params interface{}) error {
	notification := Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return s.sendMessage(notification)
}

// sendMessage sends a JSON-RPC message.
func (s *Server) sendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	log.Printf("Sending message: %s%s", header, string(data))

	if _, err := s.writer.Write([]byte(header)); err != nil {
		log.Printf("Error writing header: %v", err)
		return err
	}

	if _, err := s.writer.Write(data); err != nil {
		log.Printf("Error writing body: %v", err)
		return err
	}

	return nil
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(msg *Message) error {
	capabilities := map[string]interface{}{
		"textDocumentSync": map[string]interface{}{
			"openClose": true,
			"change":    2, // Incremental
		},
		"completionProvider": map[string]interface{}{
			"triggerCharacters": []string{".", "(", "\"", " ", "="},
			"resolveProvider":   false,
		},
		"hoverProvider": true,
		"signatureHelpProvider": map[string]interface{}{
			"triggerCharacters": []string{"(", ","},
		},
	}

	result := map[string]interface{}{
		"capabilities": capabilities,
		"serverInfo": map[string]interface{}{
			"name":    "porch-hcl-lsp",
			"version": "0.1.0",
		},
	}

	return s.sendResponse(msg.ID, result)
}

// handleInitialized handles the initialized notification.
func (s *Server) handleInitialized(msg *Message) error {
	log.Println("Server initialized")
	return nil
}

// handleTextDocumentDidOpen handles document open notifications.
func (s *Server) handleTextDocumentDidOpen(ctx context.Context, msg *Message) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		log.Printf("Invalid params for textDocument/didOpen")
		return nil
	}

	textDocInterface, ok := params["textDocument"]
	if !ok {
		log.Printf("Missing textDocument in didOpen params")
		return nil
	}

	textDoc, ok := textDocInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid textDocument format in didOpen")
		return nil
	}

	uriInterface, ok := textDoc["uri"]
	if !ok {
		log.Printf("Missing uri in textDocument")
		return nil
	}

	uri, ok := uriInterface.(string)
	if !ok {
		log.Printf("Invalid uri format")
		return nil
	}

	textInterface, ok := textDoc["text"]
	if !ok {
		log.Printf("Missing text in textDocument")
		return nil
	}

	text, ok := textInterface.(string)
	if !ok {
		log.Printf("Invalid text format")
		return nil
	}

	log.Printf("Opening document: %s", uri)

	// Parse the document
	doc, err := parser.ParseDocument(ctx, uri, text)
	if err != nil {
		log.Printf("Error parsing document %s: %v", uri, err)
		return nil
	}

	// Store the document
	s.mu.Lock()
	s.documents[uri] = doc
	s.mu.Unlock()

	// Send diagnostics
	return s.publishDiagnostics(ctx, uri)
}

// handleTextDocumentDidChange handles document change notifications.
func (s *Server) handleTextDocumentDidChange(ctx context.Context, msg *Message) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		log.Printf("Invalid params for didChange")
		return nil
	}

	textDocInterface, ok := params["textDocument"]
	if !ok {
		log.Printf("Missing textDocument in didChange")
		return nil
	}

	textDoc, ok := textDocInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid textDocument format in didChange")
		return nil
	}

	changesInterface, ok := params["contentChanges"]
	if !ok {
		log.Printf("Missing contentChanges in didChange")
		return nil
	}

	changes, ok := changesInterface.([]interface{})
	if !ok {
		log.Printf("Invalid contentChanges format in didChange")
		return nil
	}

	uriInterface, ok := textDoc["uri"]
	if !ok {
		log.Printf("Missing uri in textDocument")
		return nil
	}

	uri, ok := uriInterface.(string)
	if !ok {
		log.Printf("Invalid uri format")
		return nil
	}

	log.Printf("Updating document: %s with %d changes", uri, len(changes))

	// Get the current document
	s.mu.RLock()
	currentDoc, exists := s.documents[uri]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Document not found for change: %s", uri)
		return nil
	}

	// Start with the current content
	updatedContent := currentDoc.Content

	// Apply changes in order
	for i, changeInterface := range changes {
		change, ok := changeInterface.(map[string]interface{})
		if !ok {
			log.Printf("Invalid change format at index %d", i)
			continue
		}

		textInterface, ok := change["text"]
		if !ok {
			log.Printf("Missing text in change at index %d", i)
			continue
		}

		text, ok := textInterface.(string)
		if !ok {
			log.Printf("Invalid text format in change at index %d", i)
			continue
		}

		// Check if this is a full document change (no range) or incremental
		if rangeInterface, hasRange := change["range"]; hasRange && rangeInterface != nil {
			// Incremental change - apply the change to the specific range
			rangeMap, ok := rangeInterface.(map[string]interface{})
			if !ok {
				log.Printf("Invalid range format in change at index %d", i)
				continue
			}

			updatedContent = s.applyIncrementalChange(updatedContent, rangeMap, text)
			log.Printf("Applied incremental change at index %d: %d chars", i, len(text))
		} else {
			// Full document change
			updatedContent = text
			log.Printf("Applied full document change at index %d: %d chars", i, len(text))
		}
	}

	log.Printf("Document updated: %d lines", strings.Count(updatedContent, "\n")+1)

	// Re-parse the document with updated content
	doc, err := parser.ParseDocument(ctx, uri, updatedContent)
	if err != nil {
		log.Printf("Error parsing updated document %s: %v", uri, err)
		return nil
	}

	// Update the stored document
	s.mu.Lock()
	s.documents[uri] = doc
	s.mu.Unlock()

	// Send updated diagnostics
	return s.publishDiagnostics(ctx, uri)
}

// handleTextDocumentDidClose handles document close notifications.
func (s *Server) handleTextDocumentDidClose(msg *Message) error {
	params := msg.Params.(map[string]interface{})
	textDoc := params["textDocument"].(map[string]interface{})

	uri := textDoc["uri"].(string)

	log.Printf("Closing document: %s", uri)

	// Remove the document from memory
	s.mu.Lock()
	delete(s.documents, uri)
	s.mu.Unlock()

	return nil
}

// handleCompletion handles completion requests.
func (s *Server) handleCompletion(ctx context.Context, msg *Message) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		log.Printf("Invalid params for completion")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	textDocInterface, ok := params["textDocument"]
	if !ok {
		log.Printf("Missing textDocument in completion params")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	textDoc, ok := textDocInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid textDocument format in completion")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	positionInterface, ok := params["position"]
	if !ok {
		log.Printf("Missing position in completion params")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	position, ok := positionInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid position format in completion")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	uriInterface, ok := textDoc["uri"]
	if !ok {
		log.Printf("Missing uri in textDocument")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	uri, ok := uriInterface.(string)
	if !ok {
		log.Printf("Invalid uri format")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	lineInterface, ok := position["line"]
	if !ok {
		log.Printf("Missing line in position")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	lineFloat, ok := lineInterface.(float64)
	if !ok {
		log.Printf("Invalid line format")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	characterInterface, ok := position["character"]
	if !ok {
		log.Printf("Missing character in position")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	characterFloat, ok := characterInterface.(float64)
	if !ok {
		log.Printf("Invalid character format")
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	line := int(lineFloat)
	character := int(characterFloat)

	log.Printf("Completion request for %s at line %d, character %d", uri, line, character)

	s.mu.RLock()
	doc, exists := s.documents[uri]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Document not found: %s", uri)
		return s.sendResponse(msg.ID, []CompletionItem{})
	}

	items := s.getCompletionItems(ctx, doc, Position{Line: line, Character: character})
	log.Printf("Returning %d completion items", len(items))

	return s.sendResponse(msg.ID, items)
}

// handleHover handles hover requests.
func (s *Server) handleHover(ctx context.Context, msg *Message) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		log.Printf("Invalid params for hover")
		return s.sendResponse(msg.ID, nil)
	}

	textDocInterface, ok := params["textDocument"]
	if !ok {
		log.Printf("Missing textDocument in hover params")
		return s.sendResponse(msg.ID, nil)
	}

	textDoc, ok := textDocInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid textDocument format in hover")
		return s.sendResponse(msg.ID, nil)
	}

	positionInterface, ok := params["position"]
	if !ok {
		log.Printf("Missing position in hover params")
		return s.sendResponse(msg.ID, nil)
	}

	position, ok := positionInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid position format in hover")
		return s.sendResponse(msg.ID, nil)
	}

	uriInterface, ok := textDoc["uri"]
	if !ok {
		log.Printf("Missing uri in textDocument")
		return s.sendResponse(msg.ID, nil)
	}

	uri, ok := uriInterface.(string)
	if !ok {
		log.Printf("Invalid uri format")
		return s.sendResponse(msg.ID, nil)
	}

	lineInterface, ok := position["line"]
	if !ok {
		log.Printf("Missing line in position")
		return s.sendResponse(msg.ID, nil)
	}

	lineFloat, ok := lineInterface.(float64)
	if !ok {
		log.Printf("Invalid line format")
		return s.sendResponse(msg.ID, nil)
	}

	characterInterface, ok := position["character"]
	if !ok {
		log.Printf("Missing character in position")
		return s.sendResponse(msg.ID, nil)
	}

	characterFloat, ok := characterInterface.(float64)
	if !ok {
		log.Printf("Invalid character format")
		return s.sendResponse(msg.ID, nil)
	}

	line := int(lineFloat)
	character := int(characterFloat)

	log.Printf("Hover request for %s at line %d, character %d", uri, line, character)

	s.mu.RLock()
	doc, exists := s.documents[uri]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Document not found for hover: %s", uri)
		return s.sendResponse(msg.ID, nil)
	}

	hover := s.getHoverInfo(ctx, doc, Position{Line: line, Character: character})
	if hover != nil {
		log.Printf("Returning hover info")
	} else {
		log.Printf("No hover info found")
	}

	return s.sendResponse(msg.ID, hover)
}

// handleShutdown handles shutdown requests.
func (s *Server) handleShutdown(msg *Message) error {
	log.Println("Server shutdown requested")
	return s.sendResponse(msg.ID, nil)
}

// publishDiagnostics sends diagnostic notifications for a document.
func (s *Server) publishDiagnostics(ctx context.Context, uri string) error {
	s.mu.RLock()
	doc, exists := s.documents[uri]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	diagnostics := s.getDiagnostics(ctx, doc)

	params := map[string]interface{}{
		"uri":         uri,
		"diagnostics": diagnostics,
	}

	return s.sendNotification("textDocument/publishDiagnostics", params)
}

// getCompletionItems generates completion items for the given position.
func (s *Server) getCompletionItems(ctx context.Context, doc *parser.HCLDocument, pos Position) []CompletionItem {
	// Always return a non-nil slice to ensure JSON marshaling produces [] instead of null
	items := make([]CompletionItem, 0)

	// Get the line content to understand context
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		log.Printf("Position line %d >= total lines %d", pos.Line, len(lines))
		return items
	}

	currentLine := lines[pos.Line]
	lineBeforeCursor := currentLine[:min(pos.Character, len(currentLine))]

	log.Printf("Completion context: line='%s', before cursor='%s'", currentLine, lineBeforeCursor)

	// Check context flags
	inBlock := s.isInBlock(lines, pos.Line, pos.Character)
	atBlockLevel := s.isAtBlockLevel(lines, pos.Line)
	atRootLevel := s.isAtRootLevel(lines, pos.Line)
	inExpression := s.isInExpression(lineBeforeCursor)

	log.Printf("Context flags: inBlock=%t, atBlockLevel=%t, atRootLevel=%t, inExpression=%t", inBlock, atBlockLevel, atRootLevel, inExpression)

	// Check if we're in a function call context - lower priority than attributes
	if inExpression {
		log.Printf("Adding function completions")
		// Add HCL function completions with medium priority
		for name, fn := range doc.Functions {
			// Create snippet with parameter placeholders
			params := ""

			fnParams := fn.Params()
			if len(fnParams) > 0 {
				paramPlaceholders := make([]string, len(fnParams))

				for i, param := range fnParams {
					placeholder := param.Name
					if placeholder == "" {
						placeholder = fmt.Sprintf("arg%d", i+1)
					}

					paramPlaceholders[i] = fmt.Sprintf("${%d:%s}", i+1, placeholder)
				}

				params = strings.Join(paramPlaceholders, ", ")
			}

			items = append(items, CompletionItem{
				Label:            name,
				Kind:             CompletionItemKindFunction,
				Detail:           "HCL Function",
				Documentation:    s.getFunctionDocumentation(name, fn),
				InsertText:       name + "(" + params + ")",
				InsertTextFormat: 2,            // Snippet format
				SortText:         "100" + name, // Medium priority
			})
		}
	}

	// Check if we're in a block context (for attributes)
	if inBlock {
		log.Printf("Adding attribute completions")

		// Determine the current block type to provide relevant completions
		blockType := s.getCurrentBlockType(lines, pos.Line)
		log.Printf("Detected block type: %s", blockType)

		var porchAttrs map[string]string

		switch blockType {
		case "workflow":
			porchAttrs = map[string]string{
				"name":        "Name of the workflow",
				"description": "Description of the workflow",
			}
		case "command":
			// Determine command type for more specific completions
			commandType := s.getCommandType(lines, pos.Line)
			log.Printf("Detected command type: %s", commandType)

			porchAttrs = map[string]string{
				"type":              "Type of command (shell, pwsh, parallel, serial, etc.)",
				"name":              "Name of the command",
				"working_directory": "Working directory for command execution",
				"condition":         "Condition for command execution",
			}

			// Add command-type specific attributes
			switch commandType {
			case "shell":
				porchAttrs["command_line"] = "Shell command to execute"
				porchAttrs["environment"] = "Environment variables"
			case "pwsh":
				porchAttrs["script"] = "PowerShell script content"
				porchAttrs["environment"] = "Environment variables"
			case "foreachdirectory":
				porchAttrs["pattern"] = "Pattern for foreachdirectory commands"
				porchAttrs["mode"] = "Mode for foreachdirectory (files, directories, both)"
				porchAttrs["depth"] = "Depth for directory traversal"
			case "copycwdtotemp":
				// No additional attributes for copycwdtotemp
			case "parallel", "serial":
				// These contain nested commands, no additional attributes
			default:
				// If no type is set yet, suggest common command attributes
				porchAttrs["command_line"] = "Shell command to execute"
				porchAttrs["script"] = "PowerShell script content"
				porchAttrs["pattern"] = "Pattern for foreachdirectory commands"
				porchAttrs["mode"] = "Mode for foreachdirectory (files, directories, both)"
				porchAttrs["depth"] = "Depth for directory traversal"
				porchAttrs["environment"] = "Environment variables"
			}
		case "variable":
			porchAttrs = map[string]string{
				"description": "Description of the variable",
				"type":        "Type of variable (string, number, bool, list, map, etc.)",
				"default":     "Default value for variable",
				"validation":  "Validation block for variable",
			}
		case "locals":
			// Locals blocks contain assignments, not attributes
			porchAttrs = map[string]string{}
		case "validation":
			porchAttrs = map[string]string{
				"condition":     "Validation condition expression",
				"error_message": "Error message to display when validation fails",
			}
		default:
			// Unknown or top-level context - provide common attributes
			porchAttrs = map[string]string{
				"name":        "Name of the block",
				"description": "Description of the block",
			}
		}

		for attr, desc := range porchAttrs {
			// Skip if attribute is already present in the current block
			if !s.isAttributeAlreadySet(lines, pos.Line, attr) {
				insertText := attr + " = \"${1}\""
				insertTextFormat := 2 // Snippet format

				// Special handling for different attribute types
				switch attr {
				case "type":
					// Provide type-specific snippets based on context
					switch blockType {
					case "variable":
						insertText = attr + " = ${1|string,number,bool,list,map,object|}"
					case "command":
						insertText = attr + " = \"${1|shell,pwsh,parallel,serial,foreachdirectory,copycwdtotemp|}\""
					default:
						insertText = attr + " = \"${1}\""
					}
				case "default":
					insertText = attr + " = ${1}"
				case "condition":
					insertText = attr + " = ${1:true}"
				case "environment":
					insertText = attr + " = {\n  ${1:KEY} = \"${2:value}\"\n}"
				case "validation":
					insertText = attr + " {\n  condition     = ${1:true}\n  error_message = \"${2:Error message}\"\n}"
					insertTextFormat = 2 // This is definitely a snippet
				case "command_line":
					insertText = attr + " = \"${1:command}\""
				case "script":
					insertText = attr + " = <<EOF\n${1:# PowerShell script}\nEOF"
				case "pattern":
					insertText = attr + " = \"${1:**/*}\""
				case "mode":
					insertText = attr + " = \"${1|files,directories,both|}\""
				case "depth":
					insertText = attr + " = ${1:1}"
				case "working_directory":
					insertText = attr + " = \"${1:.}\""
				default:
					insertText = attr + " = \"${1}\""
				}

				items = append(items, CompletionItem{
					Label:            attr,
					Kind:             CompletionItemKindProperty,
					Detail:           "Porch attribute",
					Documentation:    desc,
					InsertText:       insertText,
					InsertTextFormat: insertTextFormat,
					SortText:         "000" + attr, // Highest priority with leading zeros
				})
			}
		}
	}

	// Add Porch-specific block completions
	if atBlockLevel {
		log.Printf("Adding block completions")

		// Root-level blocks (workflow, variable, locals) - only at root level
		if atRootLevel {
			log.Printf("Adding root-level block completions")

			rootBlocks := map[string]string{
				"variable": "variable \"${1:name}\" {\n  description = \"${2:Variable description}\"\n  type        = ${3|string,number,bool,list,map,object|}\n  default     = \"${4:default_value}\"\n}",
				"locals":   "locals {\n  ${1:key} = \"${2:value}\"\n}",
				"workflow": "workflow \"${1:name}\" {\n  name = \"${2:Workflow Name}\"\n  description = \"${3:Workflow description}\"\n  \n  command {\n    type = \"${4|shell,pwsh,parallel,serial|}\"\n    name = \"${5:Command name}\"\n    ${6}\n  }\n}",
			}

			for block, template := range rootBlocks {
				items = append(items, CompletionItem{
					Label:            block,
					Kind:             CompletionItemKindSnippet,
					Detail:           "Porch root-level block",
					Documentation:    "Create a " + block + " block (root level only)",
					InsertText:       template,
					InsertTextFormat: 2,             // Snippet format
					SortText:         "200" + block, // Lower priority than attributes and functions
				})
			}
		}

		// Nested blocks (command) - can be inside workflows
		nestedBlocks := map[string]string{
			"command": "command {\n  type = \"${1|shell,pwsh,parallel,serial,foreachdirectory,copycwdtotemp|}\"\n  name = \"${2:Command name}\"\n  ${3}\n}",
		}

		for block, template := range nestedBlocks {
			items = append(items, CompletionItem{
				Label:            block,
				Kind:             CompletionItemKindSnippet,
				Detail:           "Porch nested block",
				Documentation:    "Create a " + block + " block",
				InsertText:       template,
				InsertTextFormat: 2,             // Snippet format
				SortText:         "200" + block, // Lower priority than attributes and functions
			})
		}
	}

	return items
}

// getHoverInfo generates hover information for the given position.
func (s *Server) getHoverInfo(ctx context.Context, doc *parser.HCLDocument, pos Position) *Hover {
	// Get the line content to understand what we're hovering over
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	currentLine := lines[pos.Line]
	if pos.Character >= len(currentLine) {
		return nil
	}

	// Find the word at the cursor position
	word := s.getWordAtPosition(currentLine, pos.Character)
	if word == "" {
		return nil
	}

	log.Printf("Hover request for word: '%s'", word)

	// Check if it's an HCL function
	if fn, exists := doc.Functions[word]; exists {
		documentation := s.getFunctionDocumentation(word, fn)

		return &Hover{
			Contents: map[string]interface{}{
				"kind":  "markdown",
				"value": fmt.Sprintf("**%s** (HCL Function)\n\n%s", word, documentation),
			},
		}
	}

	// Check if it's a Porch-specific keyword
	porchKeywords := map[string]string{
		"variable":          "Declares an input variable",
		"locals":            "Defines local values",
		"workflow":          "Defines a Porch workflow",
		"command":           "Defines a command within a workflow",
		"name":              "Specifies the name of a block or command",
		"description":       "Provides a description for documentation",
		"working_directory": "Sets the working directory for command execution",
		"command_line":      "Specifies the shell command to execute",
		"type":              "Specifies the type (for variables or commands)",
		"default":           "Sets the default value for a variable",
		"condition":         "Defines a condition for command execution",
		"environment":       "Sets environment variables",
		"script":            "PowerShell script content",
		"shell":             "Execute a shell command",
		"pwsh":              "Execute a PowerShell command",
		"parallel":          "Execute commands in parallel",
		"serial":            "Execute commands sequentially",
		"foreachdirectory":  "Execute command for each directory",
		"copycwdtotemp":     "Copy current directory to temp and execute",
	}

	if desc, exists := porchKeywords[word]; exists {
		return &Hover{
			Contents: map[string]interface{}{
				"kind":  "markdown",
				"value": fmt.Sprintf("**%s** (Porch)\n\n%s", word, desc),
			},
		}
	}

	// Generic hover for unrecognized words
	return &Hover{
		Contents: map[string]interface{}{
			"kind":  "markdown",
			"value": fmt.Sprintf("**%s**\n\nPorch HCL element", word),
		},
	}
}

// getDiagnostics generates diagnostics for the document.
func (s *Server) getDiagnostics(ctx context.Context, doc *parser.HCLDocument) []Diagnostic {
	// Always return a non-nil slice to ensure JSON marshaling produces [] instead of null
	diagnostics := make([]Diagnostic, 0)

	// For now, return empty diagnostics
	// In a real implementation, we'd analyze the HCL for errors and warnings

	return diagnostics
}

// Helper function for min.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// isInBlock checks if the cursor is inside a block (for attribute completion).
func (s *Server) isInBlock(lines []string, currentLineIndex, character int) bool {
	// Check for indentation on current line
	if currentLineIndex < len(lines) {
		currentLine := lines[currentLineIndex]
		if len(currentLine) > 0 && (currentLine[0] == ' ' || currentLine[0] == '\t') {
			return true
		}
	}

	// Look backwards for an opening brace to see if we're inside a block
	braceCount := 0

	for i := currentLineIndex; i >= 0; i-- {
		line := lines[i]

		// If we're on the current line, only look at content before cursor
		searchLine := line
		if i == currentLineIndex {
			searchLine = line[:min(character, len(line))]
		}

		// Count braces
		for _, char := range searchLine {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
			}
		}

		// If we have a positive brace count, we're inside a block
		if braceCount > 0 {
			return true
		}

		// If we hit a negative count, we're outside blocks
		if braceCount < 0 {
			return false
		}
	}

	return false
}

// isAtBlockLevel checks if we're at the top level where blocks can be defined.
func (s *Server) isAtBlockLevel(lines []string, currentLineIndex int) bool {
	// If we're not inside any block, we're at block level
	return !s.isInBlock(lines, currentLineIndex, 0)
}

// isAtRootLevel checks if we're at the root level where top-level blocks can be defined
// This is different from isAtBlockLevel which checks if we can define ANY blocks.
func (s *Server) isAtRootLevel(lines []string, currentLineIndex int) bool {
	// Check if we're inside any block at all
	if s.isInBlock(lines, currentLineIndex, 0) {
		return false
	}

	// Additional check: scan backwards to see if we're inside any block structure
	braceCount := 0

	for i := currentLineIndex; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Count braces on each line
		for _, char := range line {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
			}
		}

		// If we have any open braces, we're not at root level
		if braceCount > 0 {
			return false
		}
	}

	return true
}

// getFunctionDocumentation returns documentation for HCL functions.
func (s *Server) getFunctionDocumentation(name string, fn function.Function) string {
	// Basic documentation for common HCL functions
	docs := map[string]string{
		"upper":        "Converts a string to uppercase",
		"lower":        "Converts a string to lowercase",
		"title":        "Converts a string to title case",
		"trim":         "Removes whitespace from both ends of a string",
		"trimspace":    "Removes whitespace from both ends of a string",
		"split":        "Splits a string into a list using a delimiter",
		"join":         "Joins a list of strings with a delimiter",
		"replace":      "Replaces occurrences of a substring",
		"substr":       "Extracts a substring",
		"format":       "Formats a string using printf-style formatting",
		"base64encode": "Encodes a string to base64",
		"base64decode": "Decodes a base64 string",
		"jsonencode":   "Encodes a value as JSON",
		"jsondecode":   "Decodes a JSON string",
		"timestamp":    "Returns the current timestamp in RFC3339 format",
		"formatdate":   "Formats a timestamp using a layout string",
		"timeadd":      "Adds a duration to a timestamp",
		"md5":          "Calculates the MD5 hash of a string",
		"sha1":         "Calculates the SHA1 hash of a string",
		"sha256":       "Calculates the SHA256 hash of a string",
		"uuid":         "Generates a random UUID",
		"length":       "Returns the length of a list, map, or string",
		"keys":         "Returns the keys of a map",
		"values":       "Returns the values of a map",
		"lookup":       "Looks up a value in a map",
		"element":      "Returns an element from a list by index",
		"contains":     "Checks if a list contains a value",
		"distinct":     "Returns unique elements from a list",
		"sort":         "Sorts a list",
		"reverse":      "Reverses a list",
		"merge":        "Merges maps together",
		"concat":       "Concatenates lists together",
		"flatten":      "Flattens nested lists",
		"zipmap":       "Creates a map from two lists",
		"abs":          "Returns the absolute value of a number",
		"ceil":         "Returns the smallest integer >= the input",
		"floor":        "Returns the largest integer <= the input",
		"max":          "Returns the maximum value from arguments",
		"min":          "Returns the minimum value from arguments",
		"pow":          "Returns base raised to the power of exponent",
		"sqrt":         "Returns the square root of a number",
		"type":         "Returns the type of a value",
		"can":          "Tests if an expression can be evaluated",
		"try":          "Tries to evaluate an expression with fallback",
		"tonumber":     "Converts a value to a number",
		"tostring":     "Converts a value to a string",
		"tobool":       "Converts a value to a boolean",
		"tolist":       "Converts a value to a list",
		"toset":        "Converts a value to a set",
		"tomap":        "Converts a value to a map",
		"file":         "Reads the contents of a file",
		"fileexists":   "Checks if a file exists",
		"dirname":      "Returns the directory portion of a path",
		"basename":     "Returns the filename portion of a path",
		"pathexpand":   "Expands ~ in file paths",
		"cidrhost":     "Calculates a host IP address within a CIDR block",
		"cidrnetmask":  "Calculates the netmask for a CIDR block",
		"cidrsubnet":   "Calculates a subnet within a CIDR block",
	}

	if doc, exists := docs[name]; exists {
		return doc
	}

	return "HCL function"
}

// getWordAtPosition extracts the word at the given character position in a line.
func (s *Server) getWordAtPosition(line string, character int) string {
	if character >= len(line) {
		return ""
	}

	// Find the start of the word
	start := character
	for start > 0 && (isAlphaNumeric(line[start-1]) || line[start-1] == '_') {
		start--
	}

	// Find the end of the word
	end := character
	for end < len(line) && (isAlphaNumeric(line[end]) || line[end] == '_') {
		end++
	}

	if start >= end {
		return ""
	}

	return line[start:end]
}

// isAlphaNumeric checks if a character is alphanumeric.
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// isInExpression checks if the cursor is in an expression context (like inside =, (), etc.)
func (s *Server) isInExpression(lineBeforeCursor string) bool {
	return strings.Contains(lineBeforeCursor, "=") ||
		strings.Contains(lineBeforeCursor, "(") ||
		strings.Contains(lineBeforeCursor, ",")
}

// getCurrentBlockType determines what type of block the cursor is currently in.
func (s *Server) getCurrentBlockType(lines []string, currentLineIndex int) string {
	// Look backwards to find the current block type
	for i := currentLineIndex; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for block definitions
		if strings.HasPrefix(line, "variable ") {
			return "variable"
		}

		if strings.HasPrefix(line, "locals ") || line == "locals {" {
			return "locals"
		}

		if strings.HasPrefix(line, "workflow ") {
			return "workflow"
		}

		if strings.HasPrefix(line, "command ") || line == "command {" {
			return "command"
		}

		if strings.HasPrefix(line, "validation ") || line == "validation {" {
			return "validation"
		}

		// If we hit a closing brace, we might be outside any block
		if line == "}" {
			return "top-level"
		}
	}

	return "top-level"
}

// getCommandType determines the type of command block we're in.
func (s *Server) getCommandType(lines []string, currentLineIndex int) string {
	// Look for a "type" attribute in the current command block
	blockStart := s.findBlockStart(lines, currentLineIndex, "command")
	if blockStart == -1 {
		return ""
	}

	// Look for type = "..." within this command block
	braceCount := 0

	for i := blockStart; i <= currentLineIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Count braces to stay within the current block
		for _, char := range line {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					// We've exited the command block
					return ""
				}
			}
		}

		// Look for type attribute
		if strings.Contains(line, "type") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"'")

				return value
			}
		}
	}

	return ""
}

// findBlockStart finds the start line of a specific block type containing the current line.
func (s *Server) findBlockStart(lines []string, currentLineIndex int, blockType string) int {
	braceCount := 0

	// Go backwards from current line
	for i := currentLineIndex; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Count braces (in reverse)
		for j := len(line) - 1; j >= 0; j-- {
			char := line[j]
			switch char {
			case '}':
				braceCount++
			case '{':
				braceCount--
			}
		}

		// If we're at brace level 0 and found the block type, this is the start
		if braceCount == 0 && strings.HasPrefix(line, blockType+" ") {
			return i
		}

		// If we've gone past the block (brace count > 0), we've gone too far
		if braceCount > 0 {
			return -1
		}
	}

	return -1
}

// isAttributeAlreadySet checks if an attribute is already defined in the current block.
func (s *Server) isAttributeAlreadySet(lines []string, currentLineIndex int, attribute string) bool {
	blockStart := s.findCurrentBlockStart(lines, currentLineIndex)
	if blockStart == -1 {
		return false
	}

	// Look for the attribute within the current block
	braceCount := 0

	for i := blockStart; i <= currentLineIndex && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Count braces to stay within the current block
		for _, char := range line {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					// We've exited the block
					return false
				}
			}
		}

		// Check if this line defines the attribute
		if strings.HasPrefix(line, attribute+" ") && strings.Contains(line, "=") {
			return true
		}
	}

	return false
}

// findCurrentBlockStart finds the start of the current block (any type).
func (s *Server) findCurrentBlockStart(lines []string, currentLineIndex int) int {
	braceCount := 0

	// Go backwards from current line
	for i := currentLineIndex; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Count braces (in reverse)
		for j := len(line) - 1; j >= 0; j-- {
			char := line[j]
			switch char {
			case '}':
				braceCount++
			case '{':
				braceCount--
			}
		}

		// If we're at brace level 0 and found a line that could start a block
		if braceCount == 0 && (strings.Contains(line, "{") ||
			strings.HasPrefix(line, "variable ") ||
			strings.HasPrefix(line, "locals") ||
			strings.HasPrefix(line, "workflow ") ||
			strings.HasPrefix(line, "command ") ||
			strings.HasPrefix(line, "validation ")) {
			return i
		}
	}

	return -1
}

// applyIncrementalChange applies an incremental change to the document content.
func (s *Server) applyIncrementalChange(content string, rangeMap map[string]interface{}, newText string) string {
	// Extract start and end positions from the range
	startInterface, ok := rangeMap["start"]
	if !ok {
		log.Printf("Missing start in range")
		return content
	}

	endInterface, ok := rangeMap["end"]
	if !ok {
		log.Printf("Missing end in range")
		return content
	}

	startMap, ok := startInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid start format in range")
		return content
	}

	endMap, ok := endInterface.(map[string]interface{})
	if !ok {
		log.Printf("Invalid end format in range")
		return content
	}

	// Extract line and character positions
	startLineInterface, ok := startMap["line"]
	if !ok {
		log.Printf("Missing line in start position")
		return content
	}

	startCharInterface, ok := startMap["character"]
	if !ok {
		log.Printf("Missing character in start position")
		return content
	}

	endLineInterface, ok := endMap["line"]
	if !ok {
		log.Printf("Missing line in end position")
		return content
	}

	endCharInterface, ok := endMap["character"]
	if !ok {
		log.Printf("Missing character in end position")
		return content
	}

	startLine, ok := startLineInterface.(float64)
	if !ok {
		log.Printf("Invalid start line format")
		return content
	}

	startChar, ok := startCharInterface.(float64)
	if !ok {
		log.Printf("Invalid start character format")
		return content
	}

	endLine, ok := endLineInterface.(float64)
	if !ok {
		log.Printf("Invalid end line format")
		return content
	}

	endChar, ok := endCharInterface.(float64)
	if !ok {
		log.Printf("Invalid end character format")
		return content
	}

	log.Printf("Applying change: [%d,%d] -> [%d,%d]: %s", int(startLine), int(startChar), int(endLine), int(endChar), newText)

	// Apply the incremental change properly
	lines := strings.Split(content, "\n")

	startLineInt := int(startLine)
	startCharInt := int(startChar)
	endLineInt := int(endLine)
	endCharInt := int(endChar)

	// Bounds checking
	if startLineInt < 0 || startLineInt >= len(lines) {
		log.Printf("Start line %d out of bounds (0-%d)", startLineInt, len(lines)-1)
		return content
	}

	if endLineInt < 0 || endLineInt >= len(lines) {
		log.Printf("End line %d out of bounds (0-%d)", endLineInt, len(lines)-1)
		return content
	}

	// Handle single line change
	if startLineInt == endLineInt {
		line := lines[startLineInt]
		if startCharInt < 0 || startCharInt > len(line) {
			log.Printf("Start character %d out of bounds for line %d (0-%d)", startCharInt, startLineInt, len(line))
			return content
		}

		if endCharInt < 0 || endCharInt > len(line) {
			log.Printf("End character %d out of bounds for line %d (0-%d)", endCharInt, startLineInt, len(line))
			return content
		}

		// Replace the range within the single line
		before := line[:startCharInt]
		after := line[endCharInt:]
		lines[startLineInt] = before + newText + after
	} else {
		// Multi-line change
		startLine := lines[startLineInt]
		endLine := lines[endLineInt]

		if startCharInt < 0 || startCharInt > len(startLine) {
			log.Printf("Start character %d out of bounds for line %d (0-%d)", startCharInt, startLineInt, len(startLine))
			return content
		}

		if endCharInt < 0 || endCharInt > len(endLine) {
			log.Printf("End character %d out of bounds for line %d (0-%d)", endCharInt, endLineInt, len(endLine))
			return content
		}

		// Keep part before the change and part after the change
		before := startLine[:startCharInt]
		after := endLine[endCharInt:]

		// Split newText into lines
		newLines := strings.Split(newText, "\n")

		// Build the new lines array
		var newLinesArray []string

		// Add lines before the change
		newLinesArray = append(newLinesArray, lines[:startLineInt]...)

		// Add the modified content
		if len(newLines) == 1 {
			// Single line replacement
			newLinesArray = append(newLinesArray, before+newLines[0]+after)
		} else {
			// Multi-line replacement
			newLinesArray = append(newLinesArray, before+newLines[0])
			if len(newLines) > 2 {
				newLinesArray = append(newLinesArray, newLines[1:len(newLines)-1]...)
			}

			newLinesArray = append(newLinesArray, newLines[len(newLines)-1]+after)
		}

		// Add lines after the change
		newLinesArray = append(newLinesArray, lines[endLineInt+1:]...)

		lines = newLinesArray
	}

	return strings.Join(lines, "\n")
}
