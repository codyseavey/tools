package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/codyseavey/linux-clipboard/internal/azure"
)

// View represents different application views
type View int

const (
	ViewQuery View = iota
	ViewResults
	ViewHistory
	ViewHelp
	ViewWorkspace
)

// Model is the main application model
type Model struct {
	// Core components
	editor       QueryEditor
	table        ResultsTable
	spinner      spinner.Model
	workspaceInput textinput.Model

	// Azure client
	client   *azure.LogAnalyticsClient
	auth     *azure.Authenticator
	config   *azure.Config
	history  *azure.History

	// State
	currentView   View
	width         int
	height        int
	loading       bool
	lastQuery     string
	lastError     string
	lastDuration  time.Duration
	rowCount      int
	styles        *Styles
	connected     bool
	workspaceID   string
	historyIndex  int
	historyList   []azure.HistoryEntry
}

// Messages
type queryResultMsg struct {
	result *azure.QueryResult
	err    error
}

type connectMsg struct {
	err error
}

// NewModel creates a new application model
func NewModel(workspaceID string, authMethod azure.AuthMethod) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	wi := textinput.New()
	wi.Placeholder = "Enter Workspace ID"
	wi.CharLimit = 100
	wi.Width = 50

	if workspaceID != "" {
		wi.SetValue(workspaceID)
	}

	config := azure.NewConfig()
	config.Load()

	history := azure.NewHistory(1000)
	history.Load()

	return Model{
		editor:         NewQueryEditor(),
		table:          NewResultsTable(),
		spinner:        s,
		workspaceInput: wi,
		config:         config,
		history:        history,
		currentView:    ViewQuery,
		styles:         DefaultStyles(),
		workspaceID:    workspaceID,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.editor.Init(),
		m.spinner.Tick,
	)
}

// Connect connects to Azure
func (m *Model) Connect(authMethod azure.AuthMethod) tea.Cmd {
	return func() tea.Msg {
		auth, err := azure.NewAuthenticator(authMethod)
		if err != nil {
			return connectMsg{err: err}
		}

		client, err := azure.NewLogAnalyticsClient(auth.GetCredential(), m.workspaceID)
		if err != nil {
			return connectMsg{err: err}
		}

		m.auth = auth
		m.client = client
		m.connected = true

		return connectMsg{err: nil}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetSize(msg.Width-4, 8)
		m.table.SetSize(msg.Width-4, msg.Height-20)
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			m.saveState()
			return m, tea.Quit

		case "f1":
			m.currentView = ViewHelp
			return m, nil

		case "f2":
			m.showHistory()
			return m, nil

		case "f3":
			m.currentView = ViewWorkspace
			m.workspaceInput.Focus()
			return m, nil

		case "esc":
			if m.currentView != ViewQuery {
				m.currentView = ViewQuery
				m.editor.Focus()
				m.table.Blur()
			}
			return m, nil
		}

		// View-specific handling
		switch m.currentView {
		case ViewQuery:
			return m.updateQueryView(msg)
		case ViewResults:
			return m.updateResultsView(msg)
		case ViewHistory:
			return m.updateHistoryView(msg)
		case ViewHelp:
			return m.updateHelpView(msg)
		case ViewWorkspace:
			return m.updateWorkspaceView(msg)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case queryResultMsg:
		m.loading = false
		if msg.err != nil {
			m.lastError = msg.err.Error()
			m.addToHistory(false, msg.err.Error())
		} else {
			m.lastError = ""
			m.processResults(msg.result)
			m.addToHistory(true, "")
		}
		return m, nil

	case connectMsg:
		if msg.err != nil {
			m.lastError = fmt.Sprintf("Connection failed: %v", msg.err)
			m.connected = false
		} else {
			m.connected = true
			m.lastError = ""
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateQueryView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+enter", "f5":
		if !m.connected {
			m.lastError = "Not connected. Press F3 to set workspace."
			return m, nil
		}
		return m.executeQuery()

	case "tab":
		m.currentView = ViewResults
		m.editor.Blur()
		m.table.Focus()
		return m, nil

	case "ctrl+l":
		m.editor.Reset()
		return m, nil

	case "up":
		// Navigate history when editor is empty or at top
		if m.editor.Value() == "" || m.historyIndex > 0 {
			return m.navigateHistory(-1)
		}
	case "down":
		if m.historyIndex < len(m.historyList)-1 {
			return m.navigateHistory(1)
		}
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m Model) updateResultsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.currentView = ViewQuery
		m.table.Blur()
		m.editor.Focus()
		return m, nil

	case "y":
		// Copy selected row (would need clipboard integration)
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) updateHistoryView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.historyIndex >= 0 && m.historyIndex < len(m.historyList) {
			m.editor.SetValue(m.historyList[m.historyIndex].Query)
			m.currentView = ViewQuery
			m.editor.Focus()
		}
		return m, nil

	case "up", "k":
		if m.historyIndex > 0 {
			m.historyIndex--
		}
		return m, nil

	case "down", "j":
		if m.historyIndex < len(m.historyList)-1 {
			m.historyIndex++
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateHelpView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "q":
		m.currentView = ViewQuery
		return m, nil
	}
	return m, nil
}

func (m Model) updateWorkspaceView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.workspaceID = m.workspaceInput.Value()
		if m.client != nil {
			m.client.SetWorkspace(m.workspaceID)
		}
		m.currentView = ViewQuery
		m.editor.Focus()
		return m, m.Connect(azure.AuthDefault)
	}

	var cmd tea.Cmd
	m.workspaceInput, cmd = m.workspaceInput.Update(msg)
	return m, cmd
}

func (m Model) executeQuery() (tea.Model, tea.Cmd) {
	query := strings.TrimSpace(m.editor.Value())
	if query == "" {
		m.lastError = "Query cannot be empty"
		return m, nil
	}

	m.loading = true
	m.lastQuery = query
	m.lastError = ""

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.config.QueryTimeout)*time.Second)
			defer cancel()

			result, err := m.client.Query(ctx, query, nil)
			return queryResultMsg{result: result, err: err}
		},
	)
}

func (m *Model) processResults(result *azure.QueryResult) {
	if len(result.Tables) == 0 {
		return
	}

	table := result.Tables[0]
	columns := make([]string, len(table.Columns))
	columnTypes := make([]string, len(table.Columns))

	for i, col := range table.Columns {
		columns[i] = col.Name
		columnTypes[i] = col.Type
	}

	rows := make([][]string, len(table.Rows))
	for i, row := range table.Rows {
		rows[i] = make([]string, len(row))
		for j, cell := range row {
			rows[i][j] = formatCell(cell)
		}
	}

	m.table.SetData(columns, columnTypes, rows)
	m.rowCount = result.RowCount
	m.lastDuration = result.Duration
	m.currentView = ViewResults
	m.editor.Blur()
	m.table.Focus()
}

func (m Model) showHistory() {
	m.historyList = m.history.GetRecent(50)
	m.historyIndex = 0
	m.currentView = ViewHistory
}

func (m Model) navigateHistory(delta int) (tea.Model, tea.Cmd) {
	if len(m.historyList) == 0 {
		m.historyList = m.history.GetRecent(50)
	}

	m.historyIndex += delta
	if m.historyIndex < 0 {
		m.historyIndex = 0
	}
	if m.historyIndex >= len(m.historyList) {
		m.historyIndex = len(m.historyList) - 1
	}

	if m.historyIndex >= 0 && m.historyIndex < len(m.historyList) {
		m.editor.SetValue(m.historyList[m.historyIndex].Query)
	}

	return m, nil
}

func (m *Model) addToHistory(success bool, errMsg string) {
	entry := azure.HistoryEntry{
		Query:      m.lastQuery,
		Workspace:  m.workspaceID,
		ExecutedAt: time.Now(),
		Duration:   m.lastDuration.String(),
		RowCount:   m.rowCount,
		WasSuccess: success,
		ErrorMsg:   errMsg,
	}
	m.history.Add(entry)
	m.historyList = nil // Reset to force reload
}

func (m *Model) saveState() {
	m.history.Save()
	m.config.Save()
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Status bar
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n\n")

	// Main content based on view
	switch m.currentView {
	case ViewQuery, ViewResults:
		b.WriteString(m.renderMainView())
	case ViewHistory:
		b.WriteString(m.renderHistoryView())
	case ViewHelp:
		b.WriteString(m.renderHelpView())
	case ViewWorkspace:
		b.WriteString(m.renderWorkspaceView())
	}

	// Error message
	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("Error: " + m.lastError))
	}

	// Footer/Help
	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderHeader() string {
	title := m.styles.Title.Render("Azure Log Analytics CLI")
	return title
}

func (m Model) renderStatusBar() string {
	var parts []string

	// Connection status
	if m.connected {
		parts = append(parts, m.styles.Success.Render("● Connected"))
	} else {
		parts = append(parts, m.styles.Error.Render("○ Disconnected"))
	}

	// Workspace
	if m.workspaceID != "" {
		ws := m.workspaceID
		if len(ws) > 20 {
			ws = ws[:17] + "..."
		}
		parts = append(parts, m.styles.StatusBarKey.Render("Workspace: ")+m.styles.Muted.Render(ws))
	}

	// Loading indicator
	if m.loading {
		parts = append(parts, m.spinner.View()+" Querying...")
	}

	// Last query stats
	if m.rowCount > 0 && !m.loading {
		stats := fmt.Sprintf("%d rows in %s", m.rowCount, m.lastDuration.Round(time.Millisecond))
		parts = append(parts, m.styles.Muted.Render(stats))
	}

	return strings.Join(parts, "  │  ")
}

func (m Model) renderMainView() string {
	var b strings.Builder

	// Query editor
	b.WriteString(m.editor.View())
	b.WriteString("\n\n")

	// Results table
	if m.table.RowCount() > 0 {
		b.WriteString(m.styles.Prompt.Render("Results"))
		b.WriteString("\n")
		b.WriteString(m.table.View())
	} else if !m.loading {
		b.WriteString(m.styles.Muted.Render("No results yet. Enter a query and press F5 or Ctrl+Enter to execute."))
	}

	return b.String()
}

func (m Model) renderHistoryView() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Query History"))
	b.WriteString("\n\n")

	if len(m.historyList) == 0 {
		b.WriteString(m.styles.Muted.Render("No history yet."))
		return b.String()
	}

	for i, entry := range m.historyList {
		prefix := "  "
		style := m.styles.Muted
		if i == m.historyIndex {
			prefix = "▶ "
			style = m.styles.Bold
		}

		query := truncateString(entry.Query, 60)
		status := m.styles.Success.Render("✓")
		if !entry.WasSuccess {
			status = m.styles.Error.Render("✗")
		}

		line := fmt.Sprintf("%s%s %s (%s, %d rows)",
			prefix, status, query, entry.ExecutedAt.Format("15:04:05"), entry.RowCount)
		b.WriteString(style.Render(line))
		b.WriteString("\n")

		if i >= 20 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ... and %d more", len(m.historyList)-20)))
			break
		}
	}

	return b.String()
}

func (m Model) renderHelpView() string {
	help := `
AZURE LOG ANALYTICS CLI - HELP

NAVIGATION
  Tab           Switch between query editor and results
  F1            Show this help
  F2            Show query history
  F3            Change workspace
  Esc           Return to query view
  Ctrl+Q        Quit

QUERY EDITOR
  F5, Ctrl+Enter   Execute query
  Ctrl+L           Clear editor
  Up/Down          Navigate query history

RESULTS TABLE
  j/k, Up/Down     Navigate rows
  h/l, Left/Right  Scroll columns
  PgUp/PgDown      Page navigation
  Home/End, g/G    Jump to start/end

KQL QUICK REFERENCE
  TableName | take 10              Fetch 10 rows
  TableName | where Column == "x"  Filter rows
  TableName | project Col1, Col2   Select columns
  TableName | summarize count()    Aggregate data
  TableName | order by Time desc   Sort results

Press Enter or Q to close help.
`
	return m.styles.Box.Render(help)
}

func (m Model) renderWorkspaceView() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Configure Workspace"))
	b.WriteString("\n\n")

	b.WriteString("Enter your Azure Log Analytics Workspace ID:\n\n")
	b.WriteString(m.workspaceInput.View())
	b.WriteString("\n\n")

	b.WriteString(m.styles.Muted.Render("Press Enter to connect, Esc to cancel"))

	// Show saved workspaces
	if len(m.config.SavedWorkspaces) > 0 {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Subtitle.Render("Saved Workspaces:"))
		b.WriteString("\n")
		for _, ws := range m.config.SavedWorkspaces {
			b.WriteString(fmt.Sprintf("  • %s: %s\n", ws.Name, ws.WorkspaceID))
		}
	}

	return b.String()
}

func (m Model) renderFooter() string {
	var keys []string

	switch m.currentView {
	case ViewQuery:
		keys = []string{
			m.styles.HelpKey.Render("F5") + " Execute",
			m.styles.HelpKey.Render("Tab") + " Results",
			m.styles.HelpKey.Render("F1") + " Help",
			m.styles.HelpKey.Render("F2") + " History",
			m.styles.HelpKey.Render("F3") + " Workspace",
			m.styles.HelpKey.Render("Ctrl+Q") + " Quit",
		}
	case ViewResults:
		keys = []string{
			m.styles.HelpKey.Render("Tab") + " Editor",
			m.styles.HelpKey.Render("j/k") + " Navigate",
			m.styles.HelpKey.Render("h/l") + " Scroll",
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	case ViewHistory:
		keys = []string{
			m.styles.HelpKey.Render("Enter") + " Select",
			m.styles.HelpKey.Render("j/k") + " Navigate",
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	default:
		keys = []string{
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	}

	return m.styles.Help.Render(strings.Join(keys, "  •  "))
}

func formatCell(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", val)
	}
}
