// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
)

// CommandStatus represents the current state of a command in the TUI.
type CommandStatus int

const (
	// StatusPending indicates the command is waiting to be executed.
	StatusPending CommandStatus = iota
	// StatusRunning indicates the command is currently executing.
	StatusRunning
	// StatusSuccess indicates the command completed successfully.
	StatusSuccess
	// StatusFailed indicates the command failed.
	StatusFailed
	// StatusSkipped indicates the command was skipped.
	StatusSkipped
)

const (
	minViewportWidth  = 40                     // Minimum width for the TUI viewport
	ellipsis          = "..."                  // Used for truncating long text
	teaTickInterval   = 100 * time.Millisecond // Interval for periodic updates
	tuiUpdateInterval = 1 * time.Second        // Interval for regular TUI updates (elapsed time, etc.)

	// Default viewport dimensions.
	defaultViewportWidth    = 80  // Default viewport width
	defaultViewportHeight   = 24  // Default viewport height
	defaultColumnSplitRatio = 0.6 // Default split ratio for left column (60% left, 40% right)
	statusBarPadding        = 2   // Padding for status bar (left + right)
)

// String returns a string representation of the command status.
func (s CommandStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// CommandNode represents a command in the execution tree.
type CommandNode struct {
	Path       []string       // Hierarchical path to this command
	Name       string         // Display name of the command
	Status     CommandStatus  // Current execution status
	StartTime  *time.Time     // When execution started
	EndTime    *time.Time     // When execution completed
	LastOutput string         // Last line of output from this command
	ErrorMsg   string         // Error message if failed
	Children   []*CommandNode // Child commands for hierarchical display
	mutex      sync.RWMutex   // Protects concurrent access to fields
}

// NewCommandNode creates a new command node.
func NewCommandNode(path []string, name string) *CommandNode {
	pathCopy := make([]string, len(path))
	copy(pathCopy, path)

	return &CommandNode{
		Path:     pathCopy,
		Name:     name,
		Status:   StatusPending,
		Children: make([]*CommandNode, 0),
	}
}

// UpdateStatus safely updates the command status.
func (cn *CommandNode) UpdateStatus(status CommandStatus) {
	cn.mutex.Lock()
	defer cn.mutex.Unlock()

	cn.Status = status
	now := time.Now()

	switch status {
	case StatusRunning:
		if cn.StartTime == nil {
			cn.StartTime = &now
		}
	case StatusSuccess, StatusFailed:
		if cn.EndTime == nil {
			cn.EndTime = &now
		}
	}
}

// UpdateOutput safely updates the last output line.
func (cn *CommandNode) UpdateOutput(output string) {
	cn.mutex.Lock()
	defer cn.mutex.Unlock()

	// Keep only the last line and trim whitespace
	if output != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 0 {
			cn.LastOutput = strings.TrimSpace(lines[len(lines)-1])
		}
	}
}

// UpdateError safely updates the error message.
func (cn *CommandNode) UpdateError(err string) {
	cn.mutex.Lock()
	defer cn.mutex.Unlock()

	cn.ErrorMsg = err
}

// UpdateErrorMsg safely updates the error message without changing the status.
func (cn *CommandNode) UpdateErrorMsg(msg string) {
	cn.mutex.Lock()
	defer cn.mutex.Unlock()

	cn.ErrorMsg = msg
}

// GetDisplayInfo safely retrieves display information.
func (cn *CommandNode) GetDisplayInfo() (CommandStatus, string, string, string, *time.Time, *time.Time) {
	cn.mutex.RLock()
	defer cn.mutex.RUnlock()

	return cn.Status, cn.Name, cn.LastOutput, cn.ErrorMsg, cn.StartTime, cn.EndTime
}

// Model represents the TUI application state.
type Model struct {
	ctx       context.Context
	reporter  progress.Reporter
	rootNode  *CommandNode
	nodeMap   map[string]*CommandNode // Maps path strings to nodes for quick lookup
	width     int
	height    int
	quitting  bool
	completed bool             // Track if all commands have completed
	results   runbatch.Results // Store final results
	mutex     sync.RWMutex

	// Viewport for content scrolling
	viewport viewport.Model

	// Status tracking
	startTime time.Time // When the execution started

	// UI configuration
	columnSplitRatio float64 // Ratio for left column (0.0-1.0), default 0.6

	// Style definitions
	styles *Styles

	// Signal handling state
	signalReceived bool           // Whether a signal has been received
	signalTime     *time.Time     // When the first signal was received
	signalCount    int            // Number of signals received
	lastSignal     os.Signal      // The most recent signal received
	signalChan     chan os.Signal // Signal channel from signalbroker
}

// Styles contains all the styling for the TUI.
type Styles struct {
	Title      lipgloss.Style
	Pending    lipgloss.Style
	Running    lipgloss.Style
	Success    lipgloss.Style
	Skipped    lipgloss.Style
	Failed     lipgloss.Style
	Output     lipgloss.Style
	Error      lipgloss.Style
	Help       lipgloss.Style
	TreeBranch lipgloss.Style
	StatusBar  lipgloss.Style
	Border     lipgloss.Style
}

// NewStyles creates the default styling for the TUI.
func NewStyles() *Styles {
	return &Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1),
		Pending: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		Running: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")),
		Skipped: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Italic(true),
		Failed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")),
		Output: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Italic(true),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1),
		TreeBranch: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("8")).
			Bold(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")),
	}
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context) *Model {
	model := &Model{
		ctx:      ctx,
		rootNode: NewCommandNode([]string{}, "Root"),
		nodeMap:  make(map[string]*CommandNode),
		viewport: viewport.New(
			defaultViewportWidth,
			defaultViewportHeight), // Default size, will be updated on window resize
		columnSplitRatio: defaultColumnSplitRatio, // Default 60% for left column, 40% for right
		styles:           NewStyles(),
		startTime:        time.Now(),
		signalChan:       signalbroker.New(ctx),
	}

	return model
}

// SetReporter sets the progress reporter for the model.
func (m *Model) SetReporter(reporter progress.Reporter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.reporter = reporter
}

// updateViewportSize updates the viewport dimensions when the window is resized.
func (m *Model) updateViewportSize() {
	// Reserve space for title (3 lines), completion message (2 lines),
	// status bar (1 line), completion message (1 line) help text (2 lines), and border (2 lines).
	reservedLines := 11

	viewportHeight := m.height - reservedLines
	if viewportHeight < 1 {
		viewportHeight = 1 // Minimum viewport height
	}

	// Account for border space (2 characters on each side for rounded border)
	m.viewport.Width = m.width - 4 // nolint:mnd
	if m.viewport.Width <= 0 {
		m.viewport.Width = minViewportWidth // Ensure minimum viewport width
	}

	m.viewport.Height = viewportHeight
}

// pathToString converts a command path to a string key.
func pathToString(path []string) string {
	return strings.Join(path, "/")
}

// getOrCreateNode safely gets or creates a command node and ensures the full hierarchy exists.
func (m *Model) getOrCreateNode(path []string, name string) *CommandNode {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	pathKey := pathToString(path)
	if node, exists := m.nodeMap[pathKey]; exists {
		return node
	}

	// Ensure all parent nodes exist
	m.ensureParentNodes(path)

	// Create new node
	node := NewCommandNode(path, name)
	m.nodeMap[pathKey] = node

	// Add to parent's children
	if len(path) > 1 {
		parentPath := path[:len(path)-1]

		parentKey := pathToString(parentPath)
		if parent, exists := m.nodeMap[parentKey]; exists {
			parent.Children = append(parent.Children, node)
		}

		return node
	}

	if len(path) == 1 {
		// Add to root
		m.rootNode.Children = append(m.rootNode.Children, node)
	}

	return node
}

// ensureParentNodes recursively creates all parent nodes if they don't exist.
func (m *Model) ensureParentNodes(path []string) {
	if len(path) <= 1 {
		return // No parents to create
	}

	// Check each parent level
	for i := 1; i < len(path); i++ {
		parentPath := path[:i]
		parentKey := pathToString(parentPath)

		if _, exists := m.nodeMap[parentKey]; !exists {
			// Create parent node
			parentName := parentPath[len(parentPath)-1]
			parentNode := NewCommandNode(parentPath, parentName)
			m.nodeMap[parentKey] = parentNode

			// Add to its parent
			if len(parentPath) > 1 {
				grandParentPath := parentPath[:len(parentPath)-1]

				grandParentKey := pathToString(grandParentPath)
				if grandParent, exists := m.nodeMap[grandParentKey]; exists {
					grandParent.Children = append(grandParent.Children, parentNode)
				}

				continue
			}
			// Add to root
			m.rootNode.Children = append(m.rootNode.Children, parentNode)

			continue
		}
	}
}

// processProgressEvent handles incoming progress events.
func (m *Model) processProgressEvent(event progress.Event) tea.Cmd {
	// Extract command name from the last element of the path
	commandName := "Unknown"
	if len(event.CommandPath) > 0 {
		commandName = event.CommandPath[len(event.CommandPath)-1]
	}

	switch event.Type {
	case progress.EventStarted:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateStatus(StatusRunning)

	case progress.EventCompleted:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateStatus(StatusSuccess)

	case progress.EventFailed:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateStatus(StatusFailed)

		// Set error message from either stderr output or error message
		if event.Data.OutputLine != "" {
			// Prefer stderr output line if available
			node.UpdateError(event.Data.OutputLine)
		} else if event.Data.Error != nil {
			// Fall back to error message if no stderr output
			node.UpdateError(event.Data.Error.Error())
		}

	case progress.EventProgress:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		// Update the node with the latest output
		if event.Data.OutputLine != "" {
			node.UpdateOutput(event.Data.OutputLine)
		}

	case progress.EventSkipped:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateStatus(StatusSkipped)

		if event.Data.OutputLine != "" {
			node.UpdateOutput(event.Data.OutputLine)
		}

		// Set error message from either stderr output or error message (for skip reasons)
		if event.Data.OutputLine != "" {
			// Prefer stderr output line if available
			node.UpdateError(event.Data.OutputLine)
		} else if event.Data.Error != nil {
			// Fall back to error message if no stderr output
			node.UpdateError(event.Data.Error.Error())
		}
	}

	// Trigger immediate UI update
	return tea.Tick(teaTickInterval, func(_ time.Time) tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	})
}

// getCommandStats recursively counts command statuses in the tree.
func (m *Model) getCommandStats() (completed, running, pending, failed int) {
	m.visitNodes(m.rootNode, func(node *CommandNode) {
		// Skip the root node
		if len(node.Path) == 0 {
			return
		}

		status, _, _, _, _, _ := node.GetDisplayInfo()
		switch status {
		case StatusSuccess:
			completed++
		case StatusRunning:
			running++
		case StatusPending:
			pending++
		case StatusFailed:
			failed++
		}
	})

	return
}

// visitNodes recursively visits all nodes in the tree.
func (m *Model) visitNodes(node *CommandNode, visitor func(*CommandNode)) {
	if node == nil {
		return
	}

	visitor(node)

	for _, child := range node.Children {
		m.visitNodes(child, visitor)
	}
}

// getMemoryUsage returns the current memory usage of this process.
// It attempts to get more accurate memory usage on Unix systems,
// falling back to Go runtime statistics on other platforms.
func (m *Model) getMemoryUsage() string {
	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	// Get the current process memory usage in MB from Go runtime
	processMemMB := float64(memStats.Alloc) / (1024 * 1024) //nolint:mnd

	// Try to get more accurate memory usage on Unix systems
	if unixMemMB := m.getUnixMemoryUsage(); unixMemMB > 0 {
		return fmt.Sprintf("%.1fMB", unixMemMB)
	}

	// Fallback to Go runtime statistics
	return fmt.Sprintf("%.1fMB", processMemMB)
}

// getUnixMemoryUsage attempts to read memory usage from /proc/self/status (Linux/Unix).
// Returns 0 if unable to read or parse the information.
func (m *Model) getUnixMemoryUsage() float64 {
	// This is a best-effort attempt to get more accurate memory usage
	// on Unix-like systems. It gracefully fails on other platforms.
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0 // Not available on this platform or permission denied
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			// Extract memory in kB and convert to MB
			parts := strings.Fields(line)
			if len(parts) >= 2 { //nolint:mnd
				var memKB float64
				if _, err := fmt.Sscanf(parts[1], "%f", &memKB); err == nil {
					return memKB / 1024 //nolint:mnd // Convert kB to MB
				}
			}
		}
	}

	return 0
}

// formatDuration formats a duration in HH:MM:SS format.
func (m *Model) formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60 //nolint:mnd
	seconds := int(d.Seconds()) % 60 //nolint:mnd

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// renderStatusBar creates the status bar with fixed columns for running, completed, execution time, and memory usage.
func (m *Model) renderStatusBar() string {
	t := table.New().
		Width(m.width - statusBarPadding).
		BorderStyle(m.styles.StatusBar).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		BorderColumn(true).
		BorderRow(false).
		StyleFunc(func(_, _ int) lipgloss.Style {
			return m.styles.StatusBar
		})

	completed, running, pending, failed := m.getCommandStats()

	// Calculate runtime
	runtime := time.Since(m.startTime)
	runtimeStr := m.formatDuration(runtime)

	// Get memory usage
	memoryStr := m.getMemoryUsage()

	// Create the four columns with equal width
	runningCol := fmt.Sprintf("⚡ %d", running)
	completedCol := fmt.Sprintf("✅ %d", completed)
	pendingCol := fmt.Sprintf("⏳ %d", pending)
	failedCol := fmt.Sprintf("❌ %d", failed)
	runtimeCol := fmt.Sprintf("Runtime: %s", runtimeStr)
	memoryCol := fmt.Sprintf("Memory: %s", memoryStr)

	t.Row(runningCol, completedCol, pendingCol, failedCol, runtimeCol, memoryCol)

	return t.Render()
}

// formatColumn formats a string to fit within the specified column width.
func formatColumn(text string, width int) string {
	if width < 1 {
		return ""
	}

	// If text is longer than width, truncate it
	if len(text) > width {
		if width > len(ellipsis) { //nolint:mnd
			return text[:width-len(ellipsis)] + ellipsis
		}

		return text[:width]
	}

	return text
}

// updateErrorsFromResults updates command node error messages using the final results
// after execution is complete. This ensures we get the actual detailed error messages
// rather than generic "result has children with errors" messages.
func (m *Model) updateErrorsFromResults() {
	if m.results == nil {
		return
	}

	// Recursively update errors from the results tree
	m.updateNodeErrorsFromResults([]string{}, m.results)
}

// updateNodeErrorsFromResults recursively updates node errors from results.
func (m *Model) updateNodeErrorsFromResults(basePath []string, results runbatch.Results) {
	for _, result := range results {
		// Build the command path for this result
		var commandPath []string
		if len(basePath) == 0 {
			// Root level - use just the label
			commandPath = []string{result.Label}
		} else {
			// Nested level - append to base path
			commandPath = append(basePath, result.Label)
		}

		// Find the corresponding node
		pathKey := pathToString(commandPath)
		if node, exists := m.nodeMap[pathKey]; exists {
			// Update error message if this result has a specific error
			if result.Error != nil && (result.Status == runbatch.ResultStatusError) {
				// Only update if we have a more specific error than the generic one
				if !errors.Is(result.Error, runbatch.ErrResultChildrenHasError) {
					node.UpdateError(result.Error.Error())
				}
			}
		}

		// Recursively process children
		if len(result.Children) > 0 {
			m.updateNodeErrorsFromResults(commandPath, result.Children)
		}
	}
}
