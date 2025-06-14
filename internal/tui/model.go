// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

// CommandStatus represents the current state of a command in the TUI.
type CommandStatus int

const (
	StatusPending CommandStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
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

// GetDisplayInfo safely retrieves display information.
func (cn *CommandNode) GetDisplayInfo() (CommandStatus, string, string, string, *time.Time, *time.Time) {
	cn.mutex.RLock()
	defer cn.mutex.RUnlock()

	return cn.Status, cn.Name, cn.LastOutput, cn.ErrorMsg, cn.StartTime, cn.EndTime
}

// Model represents the TUI application state.
type Model struct {
	ctx       context.Context
	reporter  progress.ProgressReporter
	rootNode  *CommandNode
	nodeMap   map[string]*CommandNode // Maps path strings to nodes for quick lookup
	width     int
	height    int
	quitting  bool
	completed bool             // Track if all commands have completed
	results   runbatch.Results // Store final results
	mutex     sync.RWMutex

	// Scrolling support
	scrollOffset int // Number of lines scrolled from top
	totalLines   int // Total number of lines in the rendered content

	// Status tracking
	startTime time.Time // When the execution started

	// Style definitions
	styles *Styles
}

// Styles contains all the styling for the TUI.
type Styles struct {
	Title      lipgloss.Style
	Pending    lipgloss.Style
	Running    lipgloss.Style
	Success    lipgloss.Style
	Failed     lipgloss.Style
	Output     lipgloss.Style
	Error      lipgloss.Style
	Help       lipgloss.Style
	TreeBranch lipgloss.Style
	StatusBar  lipgloss.Style
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
			Bold(true).
			Padding(0, 1),
	}
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context) *Model {
	return &Model{
		ctx:       ctx,
		rootNode:  NewCommandNode([]string{}, "Root"),
		nodeMap:   make(map[string]*CommandNode),
		styles:    NewStyles(),
		startTime: time.Now(),
	}
}

// SetReporter sets the progress reporter for the model.
func (m *Model) SetReporter(reporter progress.ProgressReporter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.reporter = reporter
}

// getViewportHeight returns the available height for content display.
func (m *Model) getViewportHeight() int {
	// Reserve space for title (3 lines), completion message (2 lines), status bar (1 line), and help text (2 lines)
	reservedLines := 8
	if m.height <= reservedLines {
		return 1 // Minimum viewport height
	}
	return m.height - reservedLines
}

// calculateMaxScrollOffset returns the maximum scroll offset based on content height.
func (m *Model) calculateMaxScrollOffset() int {
	viewportHeight := m.getViewportHeight()
	if m.totalLines <= viewportHeight {
		return 0 // No scrolling needed
	}
	return m.totalLines - viewportHeight
}

// resetScrollIfNeeded resets scroll position if content shrinks.
func (m *Model) resetScrollIfNeeded() {
	maxScroll := m.calculateMaxScrollOffset()
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
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
	} else if len(path) == 1 {
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
			} else {
				// Add to root
				m.rootNode.Children = append(m.rootNode.Children, parentNode)
			}
		}
	}
}

// processProgressEvent handles incoming progress events.
func (m *Model) processProgressEvent(event progress.ProgressEvent) tea.Cmd {
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
		if event.Data.Error != nil {
			node.UpdateError(event.Data.Error.Error())
		}

	case progress.EventOutput:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateOutput(event.Data.OutputLine)

	case progress.EventSkipped:
		node := m.getOrCreateNode(event.CommandPath, commandName)
		node.UpdateStatus(StatusPending) // Keep as pending for skipped commands
	}

	return nil
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

// getMemoryUsage returns the current memory usage of this process and its children.
func (m *Model) getMemoryUsage() string {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get the current process memory usage in MB
	processMemMB := float64(memStats.Alloc) / (1024 * 1024)

	// Try to get total memory usage including child processes
	totalMemMB := processMemMB

	// On Unix systems, we can try to read /proc/self/status for more accurate memory info
	if pidMemMB := m.getProcMemoryUsage(); pidMemMB > 0 {
		totalMemMB = pidMemMB
	}

	return fmt.Sprintf("%.1fMB", totalMemMB)
}

// getProcMemoryUsage attempts to read memory usage from /proc/self/status (Linux/Unix).
func (m *Model) getProcMemoryUsage() float64 {
	// This is a best-effort attempt to get more accurate memory usage
	// including child processes where possible
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			// Extract memory in kB and convert to MB
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var memKB float64
				if _, err := fmt.Sscanf(parts[1], "%f", &memKB); err == nil {
					return memKB / 1024 // Convert kB to MB
				}
			}
		}
	}

	return 0
}

// formatDuration formats a duration in HH:MM:SS format.
func (m *Model) formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// renderStatusBar creates the status bar with fixed columns for running, completed, execution time, and memory usage.
func (m *Model) renderStatusBar() string {
	completed, running, _, _ := m.getCommandStats()

	// Calculate runtime
	runtime := time.Since(m.startTime)
	runtimeStr := m.formatDuration(runtime)

	// Get memory usage
	memoryStr := m.getMemoryUsage()

	// Calculate column width (divide available width by 4 columns, accounting for borders)
	availableWidth := m.width
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Account for column separators (3 separators = 3 characters)
	contentWidth := availableWidth - 3
	colWidth := contentWidth / 4

	// Create the four columns with equal width
	runningCol := m.formatColumn(fmt.Sprintf("Running: %d", running), colWidth)
	completedCol := m.formatColumn(fmt.Sprintf("Completed: %d", completed), colWidth)
	runtimeCol := m.formatColumn(fmt.Sprintf("Runtime: %s", runtimeStr), colWidth)
	memoryCol := m.formatColumn(fmt.Sprintf("Memory: %s", memoryStr), colWidth)

	// Combine columns with separators
	statusLine := fmt.Sprintf("%s│%s│%s│%s",
		runningCol, completedCol, runtimeCol, memoryCol)

	return m.styles.StatusBar.Render(statusLine)
}

// formatColumn formats a string to fit within the specified column width.
func (m *Model) formatColumn(text string, width int) string {
	if width < 1 {
		return ""
	}

	// If text is longer than width, truncate it
	if len(text) > width {
		if width > 3 {
			return text[:width-3] + "..."
		}
		return text[:width]
	}

	// Center the text within the column
	padding := width - len(text)
	leftPad := padding / 2
	rightPad := padding - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}
