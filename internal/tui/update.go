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
		m.listenForProgressEvents(),
	)
}

// listenForProgressEvents returns a command that listens for progress events.
func (m *Model) listenForProgressEvents() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// This will be implemented when we connect the progress system
		return nil
	})
}

// Update implements bubbletea.Model.Update.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

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

// handleKeyPress processes keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "r":
		// Refresh view
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

	var b strings.Builder

	// Title
	title := m.styles.Title.Render("🏗️  Porch Command Orchestration")
	b.WriteString(title)
	b.WriteString("\n\n")
	// Command tree
	m.renderCommandTree(&b, m.rootNode, "", true)

	// Show completion status if commands are done
	m.mutex.RLock()
	completed := m.completed
	results := m.results
	m.mutex.RUnlock()

	if completed {
		b.WriteString("\n")
		if results != nil && results.HasError() {
			completionMsg := m.styles.Failed.Render("⚠️  Execution completed with errors")
			b.WriteString(completionMsg)
		} else {
			completionMsg := m.styles.Success.Render("✅ Execution completed successfully")
			b.WriteString(completionMsg)
		}
		b.WriteString("\n")
	}

	// Help text
	if m.height > 10 { // Only show help if we have enough space
		helpText := "Press 'q' to quit, 'r' to refresh"
		if completed {
			helpText = "Press 'q' to quit and return to terminal"
		}
		help := m.styles.Help.Render(helpText)
		b.WriteString("\n")
		b.WriteString(help)
	}

	return b.String()
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

// renderCommandNode renders a single command node.
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

	// Build the main line
	treePrefix := m.styles.TreeBranch.Render(prefix + connector)
	statusText := fmt.Sprintf("%s %s", statusIcon, styledName)

	// Add timing information if available
	if startTime != nil {
		elapsed := time.Since(*startTime)
		if endTime != nil {
			elapsed = endTime.Sub(*startTime)
		}
		statusText += m.styles.Output.Render(fmt.Sprintf(" (%v)", elapsed.Round(time.Millisecond)))
	}

	b.WriteString(treePrefix)
	b.WriteString(statusText)
	b.WriteString("\n")

	// Add output line if available
	if output != "" && status == StatusRunning {
		outputPrefix := prefix
		if isLast {
			outputPrefix += "    "
		} else {
			outputPrefix += "│   "
		}
		outputLine := m.styles.Output.Render(fmt.Sprintf("  → %s", output))
		b.WriteString(outputPrefix)
		b.WriteString(outputLine)
		b.WriteString("\n")
	}

	// Add error message if failed
	if errorMsg != "" && status == StatusFailed {
		errorPrefix := prefix
		if isLast {
			errorPrefix += "    "
		} else {
			errorPrefix += "│   "
		}
		errorLine := m.styles.Error.Render(fmt.Sprintf("  ✗ %s", errorMsg))
		b.WriteString(errorPrefix)
		b.WriteString(errorLine)
		b.WriteString("\n")
	}
}
