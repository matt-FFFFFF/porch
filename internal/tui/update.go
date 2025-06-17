// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"os"
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
	leftColMinWith              = 0.2
	leftColMaxWith              = 0.8
)

// Init implements bubbletea.Model.Init.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseCellMotion, // Enable mouse support
		m.startTicker(),           // Start the regular update ticker
		m.listenForSignals(),      // Start listening for signals
	)
}

// startTicker creates a command that sends TickMsg on a regular interval.
func (m *Model) startTicker() tea.Cmd {
	return tea.Tick(tuiUpdateInterval, func(_ time.Time) tea.Msg {
		return TickMsg{}
	})
}

// listenForSignals creates a command that listens for OS signals.
func (m *Model) listenForSignals() tea.Cmd {
	return func() tea.Msg {
		// Block waiting for a signal
		signal := <-m.signalChan
		return SignalReceivedMsg{Signal: signal}
	}
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

	case TickMsg:
		// Handle regular ticker updates for time-dependent content
		// This keeps the UI responsive and updates things like elapsed time
		nextTick := m.startTicker()
		return m, tea.Batch(cmd, nextTick)

	case SignalReceivedMsg:
		// Handle signals received
		return m.handleSignal(msg.Signal)
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

// TickMsg is sent on a regular interval to update time-dependent UI elements.
type TickMsg struct{}

// SignalReceivedMsg indicates that a signal was received.
type SignalReceivedMsg struct {
	Signal os.Signal
}

// handleKeyPress processes keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch msg.String() {
	case "ctrl+c":
		// Send a real interrupt signal to the current process
		// This ensures both the main process watchdog and TUI receive the signal
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			break
		}

		_ = process.Signal(os.Interrupt)

		return m, nil

	case "q":
		m.quitting = true
		return m, tea.Quit

	case "r":
		// Refresh view
		return m, nil

	case "[":
		// Move split left (decrease left column width by 5%)
		m.columnSplitRatio -= 0.05
		if m.columnSplitRatio < leftColMinWith { // Minimum 20% for left column
			m.columnSplitRatio = leftColMinWith
		}

		return m, nil

	case "]":
		// Move split right (increase left column width by 5%)
		m.columnSplitRatio += 0.05
		if m.columnSplitRatio > leftColMaxWith { // Maximum 80% for left column
			m.columnSplitRatio = leftColMaxWith
		}

		return m, nil
	}

	// All other keys (scrolling) are handled by viewport
	return m, nil
}

// handleSignal processes a received signal and updates the model state.
func (m *Model) handleSignal(signal os.Signal) (tea.Model, tea.Cmd) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Update signal state
	now := time.Now()
	if !m.signalReceived {
		m.signalTime = &now
	}

	m.signalReceived = true
	m.signalCount++
	m.lastSignal = signal

	// Continue listening for more signals
	nextSignalCmd := m.listenForSignals()

	return m, nextSignalCmd
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

	// Add column headers
	m.renderColumnHeaders(&content)

	// Command tree
	m.renderCommandTree(&content, m.rootNode, "", true)

	// Set viewport content
	m.viewport.SetContent(content.String())

	// Build the final view
	var view strings.Builder

	// Title
	// Show completion status if commands are done
	title := m.styles.Title.Render("🏗️  Porch Command Orchestration")
	view.WriteString(title)
	view.WriteString("\n")

	// Signal status message (if any signals received)
	switch m.signalReceived {
	case true:
		signalMsg := fmt.Sprintf("⚠️  Signal received: %s (count: %d, last: %s)",
			m.lastSignal.String(),
			m.signalCount,
			m.signalTime.Format("15:04:05"))
		styledSignalMsg := m.styles.Failed.AlignHorizontal(lipgloss.Left).Render(signalMsg)
		view.WriteString(styledSignalMsg)
		view.WriteString("\n")
	}

	// Viewport with border
	viewportContent := m.viewport.View()
	borderedViewport := m.styles.Border.Render(viewportContent)
	view.WriteString(borderedViewport)

	view.WriteString("\n")

	var completionMsg string

	switch {
	case m.completed && m.results != nil && m.results.HasError():
		completionMsg = m.styles.Failed.Render("⚠️  Execution completed with errors, press 'q' to see full details")
	case m.completed && m.results != nil && !m.results.HasError():
		completionMsg = m.styles.Success.Render("✅  Execution completed successfully")
	case !m.completed:
		completionMsg = m.styles.Pending.Render("⏳  Execution in progress, please wait...")
	default:
		completionMsg = m.styles.Pending.Render("⏳  Please wait...")
	}

	view.WriteString(completionMsg)
	view.WriteString("\n")

	// Footer with status bar and help
	if m.height > minStatusBarAvailableHeight {
		view.WriteString("\n")

		// Status bar
		statusBar := m.renderStatusBar()
		view.WriteString(statusBar)
		view.WriteString("\n")

		// Help text
		helpText := "↑/↓ or j/k to scroll, PgUp/PgDn for pages, Home/End to jump, " +
			"[/] to adjust column split, 'q' to quit, 'r' to refresh"
		if m.completed {
			helpText = "↑/↓ or j/k to scroll, [/] to adjust column split, 'q' to quit and return to terminal"
		}

		help := m.styles.Help.Render(helpText)
		view.WriteString(help)
	}

	return view.String()
}

// renderColumnHeaders renders the column headers for Command and Output columns.
func (m *Model) renderColumnHeaders(b *strings.Builder) {
	// Create header row manually for better control
	leftWidth := int(float64(m.viewport.Width) * m.columnSplitRatio)
	rightWidth := int(float64(m.viewport.Width) * (1.0 - m.columnSplitRatio))

	// Adjust for the column separator (1 character)
	leftWidth--

	// Create header content with consistent styling - ensure single line height
	headerStyle := m.styles.Title.Bold(true).Align(lipgloss.Left).Height(1).Margin(0).Padding(0)
	leftHeader := headerStyle.Width(leftWidth).Render("Command")
	rightHeader := headerStyle.Width(rightWidth).Render("Output Sample / Error")

	// Render header row with column separator
	b.WriteString(leftHeader)
	b.WriteString("  ") // Account for the column separator
	b.WriteString(rightHeader)
	b.WriteString("\n")

	// Create horizontal border line
	borderWidth := m.viewport.Width
	if borderWidth <= 0 {
		borderWidth = 1 // Minimum border width to prevent panic
	}

	border := strings.Repeat("─", borderWidth)
	b.WriteString(border)
	b.WriteString("\n")
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

	leftWidth := int(float64(m.viewport.Width) * m.columnSplitRatio)
	rightWidth := int(float64(m.viewport.Width) * (1.0 - m.columnSplitRatio))

	// Build the left column (command info)
	treePrefix := m.styles.TreeBranch.Render(prefix + connector)
	leftColumn := fmt.Sprintf("%s%s %s", treePrefix, statusIcon, styledName)

	// Add timing information if available
	if startTime != nil {
		elapsed := time.Since(*startTime)
		if endTime != nil {
			elapsed = endTime.Sub(*startTime)
		}

		durStr := "(" + elapsed.Round(commandDurationRounding).String() + ")"
		leftColumn += m.styles.Output.Render(durStr)
	}

	// Create a single-row table for this command with consistent alignment
	t := table.New().
		Width(m.viewport.Width).
		BorderStyle(m.styles.TreeBranch).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(true). // Enable column border to show the split
		BorderRow(false).
		StyleFunc(func(_, col int) lipgloss.Style {
			// Column 0: Command tree (dynamic width based on split ratio)
			if col == 0 {
				return lipgloss.NewStyle().Width(leftWidth)
			}

			return lipgloss.NewStyle().Width(rightWidth)
		})

	// Build the right column (output or error)
	var rightColumn string

	if errorMsg != "" && status == StatusFailed {
		rightColumn = m.styles.Error.Render(fmt.Sprintf("Error: %s", errorMsg))
	} else {
		switch status {
		case StatusFailed:
			rightColumn = m.styles.Error.Render(
				formatColumn(output, rightWidth),
			)
		case StatusRunning:
			rightColumn = m.styles.Output.Render(
				formatColumn(output, rightWidth),
			)
		}
	}

	// Add the row to the table
	t.Row(
		leftColumn,
		rightColumn,
	)

	// Render the table and add to buffer
	rendered := t.Render()
	b.WriteString(rendered)
	b.WriteString("\n")
}
