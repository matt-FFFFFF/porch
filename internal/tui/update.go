// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

const (
	minStatusBarAvailableHeight = 10
	commandDurationRounding     = 100 * time.Millisecond // Round durations to 100ms
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
	var cmd tea.Cmd

	// Update the viewport first
	m.viewport, cmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keys not handled by viewport
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.mutex.Lock()
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportSize()
		m.mutex.Unlock()

		return m, cmd

	case ProgressEventMsg:
		progressCmd := m.processProgressEvent(msg.Event)
		return m, tea.Batch(cmd, progressCmd)

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
	Event progress.Event
}

// CommandCompletedMsg indicates that all commands have finished executing.
type CommandCompletedMsg struct {
	Results runbatch.Results
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
	case "[":
		// Move split left (decrease left column width by 5%)
		m.columnSplitRatio -= 0.05
		if m.columnSplitRatio < 0.2 { // Minimum 20% for left column
			m.columnSplitRatio = 0.2
		}
		return m, nil
	case "]":
		// Move split right (increase left column width by 5%)
		m.columnSplitRatio += 0.05
		if m.columnSplitRatio > 0.8 { // Maximum 80% for left column
			m.columnSplitRatio = 0.8
		}
		return m, nil
	}

	// All other keys (scrolling) are handled by viewport
	return m, nil
}

// View implements bubbletea.Model.View.
func (m *Model) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Build content for the viewport
	var content strings.Builder

	// Command tree
	m.renderCommandTree(&content, m.rootNode, "", true)

	// Show completion status if commands are done
	if m.completed {
		content.WriteString("\n")

		if m.results != nil && m.results.HasError() {
			completionMsg := m.styles.Failed.Render("⚠️  Execution completed with errors")
			content.WriteString(completionMsg)
		} else {
			completionMsg := m.styles.Success.Render("✅ Execution completed successfully")
			content.WriteString(completionMsg)
		}

		content.WriteString("\n")
	}

	// Set viewport content
	m.viewport.SetContent(content.String())

	// Build the final view
	var view strings.Builder

	// Title
	title := m.styles.Title.Render("🏗️  Porch Command Orchestration")
	view.WriteString(title)
	view.WriteString("\n")

	// Viewport with border
	viewportContent := m.viewport.View()
	borderedViewport := m.styles.Border.Render(viewportContent)
	view.WriteString(borderedViewport)

	// Footer with status bar and help
	if m.height > minStatusBarAvailableHeight {
		view.WriteString("\n\n")

		// Status bar
		statusBar := m.renderStatusBar()
		view.WriteString(statusBar)
		view.WriteString("\n")

		// Help text
		helpText := "↑/↓ or j/k to scroll, PgUp/PgDn for pages, Home/End to jump, [/] to adjust column split, 'q' to quit, 'r' to refresh"
		if m.completed {
			helpText = "↑/↓ or j/k to scroll, '['/']' to adjust column split, 'q' to quit and return to terminal"
		}

		help := m.styles.Help.Render(helpText)
		view.WriteString(help)
	}

	return view.String()
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

// renderCommandNode renders a single command node using a table for consistent alignment.
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

	// Build the left column (command info)
	treePrefix := m.styles.TreeBranch.Render(prefix + connector)
	leftColumn := fmt.Sprintf("%s%s %s", treePrefix, statusIcon, styledName)

	// Add timing information if available
	if startTime != nil {
		elapsed := time.Since(*startTime)
		if endTime != nil {
			elapsed = endTime.Sub(*startTime)
		}
		leftColumn += m.styles.Output.Render(fmt.Sprintf(" (%v)", elapsed.Round(commandDurationRounding)))
	}

	// Build the right column (output or error)
	var rightColumn string
	if errorMsg != "" && status == StatusFailed {
		rightColumn = m.styles.Error.Render(fmt.Sprintf("Error: %s", errorMsg))
	} else if output != "" && status == StatusRunning {
		rightColumn = m.styles.Output.Render(output)
	}

	// Create a single-row table for this command with consistent alignment
	t := table.New().
		Width(m.viewport.Width).
		BorderStyle(m.styles.TreeBranch).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Column 0: Command tree (dynamic width based on split ratio)
			if col == 0 {
				leftWidth := int(float64(m.viewport.Width) * m.columnSplitRatio)
				return lipgloss.NewStyle().Width(leftWidth)
			}
			// Column 1: Output (remaining width)
			rightWidth := int(float64(m.viewport.Width) * (1.0 - m.columnSplitRatio))
			return lipgloss.NewStyle().Width(rightWidth)
		})

	// Add the row to the table
	t.Row(leftColumn, rightColumn)

	// Render the table and add to buffer
	rendered := t.Render()
	b.WriteString(rendered)
	b.WriteString("\n")
}
