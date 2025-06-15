// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

// Init implements bubbletea.Model.Init.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseCellMotion, // Enable mouse support
	)
}

// Update implements bubbletea.Model.Update.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)

	case tea.WindowSizeMsg:
		m.mutex.Lock()
		m.width = msg.Width
		m.height = msg.Height
		m.mutex.Unlock()

		return m, nil

	case ProgressEventMsg:
		cmd := m.processProgressEvent(msg.Event)
		return m, cmd

	case CommandCompletedMsg:
		m.mutex.Lock()
		m.completed = true
		m.results = msg.Results
		// Update error messages from final results to get specific errors
		m.updateErrorsFromResults()
		m.mutex.Unlock()

		return m, nil

	case tea.QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// ProgressEventMsg wraps a progress event for the tea framework.
type ProgressEventMsg struct {
	Event progress.ProgressEvent
}

// CommandCompletedMsg indicates that all commands have finished executing.
type CommandCompletedMsg struct {
	Results runbatch.Results
}

// handleMouseEvent processes mouse input for scrolling.
func (m *Model) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		// Scroll up
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}

		return m, nil
	case tea.MouseButtonWheelDown:
		// Scroll down
		maxScroll := m.calculateMaxScrollOffset()
		if m.scrollOffset < maxScroll {
			m.scrollOffset++
		}

		return m, nil
	}

	return m, nil
}

// handleKeyPress processes keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "r":
		// Refresh view
		return m, nil
	case "up", "k":
		// Scroll up
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}

		return m, nil
	case "down", "j":
		// Scroll down
		maxScroll := m.calculateMaxScrollOffset()
		if m.scrollOffset < maxScroll {
			m.scrollOffset++
		}

		return m, nil
	case "pgup":
		// Page up (scroll up by viewport height)
		scrollAmount := m.getViewportHeight()

		m.scrollOffset -= scrollAmount
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}

		return m, nil
	case "pgdown":
		// Page down (scroll down by viewport height)
		scrollAmount := m.getViewportHeight()
		maxScroll := m.calculateMaxScrollOffset()
		m.scrollOffset += scrollAmount

		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}

		return m, nil
	case "home":
		// Jump to top
		m.scrollOffset = 0
		return m, nil
	case "end":
		// Jump to bottom
		m.scrollOffset = m.calculateMaxScrollOffset()
		return m, nil
	}

	return m, nil
}

// View implements bubbletea.Model.View.
func (m *Model) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var contentBuilder strings.Builder

	var lines []string

	// Title
	title := m.styles.Title.Render("🏗️  Porch Command Orchestration")
	lines = append(lines, title, "")

	// Command tree - build all content first to count lines
	var treeBuilder strings.Builder

	m.renderCommandTree(&treeBuilder, m.rootNode, "", true)
	treeContent := treeBuilder.String()
	treeLines := strings.Split(strings.TrimSuffix(treeContent, "\n"), "\n")

	// Filter out empty lines at the end
	for len(treeLines) > 0 && strings.TrimSpace(treeLines[len(treeLines)-1]) == "" {
		treeLines = treeLines[:len(treeLines)-1]
	}

	lines = append(lines, treeLines...)

	// Show completion status if commands are done
	completed := m.completed
	results := m.results

	if completed {
		lines = append(lines, "")

		if results != nil && results.HasError() {
			completionMsg := m.styles.Failed.Render("⚠️  Execution completed with errors")
			lines = append(lines, completionMsg)
		} else {
			completionMsg := m.styles.Success.Render("✅ Execution completed successfully")
			lines = append(lines, completionMsg)
		}
	}

	// Store total lines for scrolling calculations
	m.totalLines = len(lines)
	m.resetScrollIfNeeded()

	// Calculate visible content with scrolling
	viewportHeight := m.getViewportHeight()
	startLine := m.scrollOffset
	endLine := startLine + viewportHeight

	// Build visible content
	for i := startLine; i < endLine && i < len(lines); i++ {
		contentBuilder.WriteString(lines[i])

		if i < endLine-1 && i < len(lines)-1 {
			contentBuilder.WriteString("\n")
		}
	}

	// Add scroll indicators and help text
	if m.height > 10 { // Only show help if we have enough space
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString("\n") // Extra line gap before status bar

		// Status bar
		statusBar := m.renderStatusBar()
		contentBuilder.WriteString(statusBar)
		contentBuilder.WriteString("\n")

		// Scroll indicator
		if m.totalLines > viewportHeight {
			scrollInfo := fmt.Sprintf("Lines %d-%d of %d",
				startLine+1,
				min(endLine, m.totalLines),
				m.totalLines)
			scrollIndicator := m.styles.Help.Render(scrollInfo)
			contentBuilder.WriteString(scrollIndicator)
			contentBuilder.WriteString("\n")
		}

		// Help text
		helpText := "↑/↓ or j/k to scroll, PgUp/PgDn for pages, Home/End to jump, 'q' to quit, 'r' to refresh"
		if completed {
			helpText = "↑/↓ or j/k to scroll, 'q' to quit and return to terminal"
		}

		help := m.styles.Help.Render(helpText)
		contentBuilder.WriteString(help)
	}

	return contentBuilder.String()
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// renderCommandTree recursively renders the command tree.
func (m *Model) renderCommandTree(b *strings.Builder, node *CommandNode, prefix string, isLast bool) {
	if node == nil {
		return
	}

	// Skip rendering the root node itself
	if len(node.Path) == 0 {
		for i, child := range node.Children {
			m.renderCommandTree(b, child, "", i == len(node.Children)-1)
		}

		return
	}

	// Render the current node
	m.renderCommandNode(b, node, prefix, isLast)

	// Render children
	if len(node.Children) > 0 {
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		for i, child := range node.Children {
			m.renderCommandTree(b, child, childPrefix, i == len(node.Children)-1)
		}
	}
}

// renderCommandNode renders a single command node with inline output display.
func (m *Model) renderCommandNode(b *strings.Builder, node *CommandNode, prefix string, isLast bool) {
	status, name, output, errorMsg, startTime, endTime := node.GetDisplayInfo()

	// Tree structure characters
	var connector string
	if isLast {
		connector = "└── "
	} else {
		connector = "├── "
	}

	// Status icon and styling
	var statusIcon string

	var styledName string

	switch status {
	case StatusPending:
		statusIcon = "⏳"
		styledName = m.styles.Pending.Render(name)
	case StatusRunning:
		statusIcon = "⚡"
		styledName = m.styles.Running.Render(name)
	case StatusSuccess:
		statusIcon = "✅"
		styledName = m.styles.Success.Render(name)
	case StatusFailed:
		statusIcon = "❌"
		styledName = m.styles.Failed.Render(name)
	default:
		statusIcon = "❓"
		styledName = m.styles.Pending.Render(name)
	}

	// Build the left side (command info)
	treePrefix := m.styles.TreeBranch.Render(prefix + connector)
	leftSide := fmt.Sprintf("%s %s", statusIcon, styledName)

	// Add timing information if available
	if startTime != nil {
		elapsed := time.Since(*startTime)
		if endTime != nil {
			elapsed = endTime.Sub(*startTime)
		}

		leftSide += m.styles.Output.Render(fmt.Sprintf(" (%v)", elapsed.Round(time.Millisecond)))
	}

	// Build the right side (output or error)
	var rightSide string
	if errorMsg != "" && status == StatusFailed {
		rightSide = m.styles.Error.Render(fmt.Sprintf("Error: %s", errorMsg))
	} else if output != "" && status == StatusRunning {
		rightSide = m.styles.Output.Render(output)
	}

	// Calculate available width for layout
	availableWidth := m.width - len(treePrefix) - 2 // Account for prefix and some padding
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Split available width: 50% for left (command), 50% for right (output)
	// This gives more space to the output and reduces truncation
	leftWidth := availableWidth / 2
	rightWidth := availableWidth - leftWidth

	// Truncate left side if too long
	if len(leftSide) > leftWidth {
		if leftWidth > 3 {
			leftSide = leftSide[:leftWidth-3] + "..."
		} else {
			leftSide = leftSide[:leftWidth]
		}
	}

	// Truncate right side if too long
	if len(rightSide) > rightWidth {
		if rightWidth > 3 {
			rightSide = rightSide[:rightWidth-3] + "..."
		} else {
			rightSide = rightSide[:rightWidth]
		}
	}

	// Pad left side to align right side
	paddedLeftSide := leftSide + strings.Repeat(" ", leftWidth-len(leftSide))

	// Build the complete line
	b.WriteString(treePrefix)
	b.WriteString(paddedLeftSide)

	if rightSide != "" {
		b.WriteString(rightSide)
	}

	b.WriteString("\n")
}
