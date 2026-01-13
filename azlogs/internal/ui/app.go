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
	"github.com/codyseavey/tools/azlogs/internal/azure"
)

// View represents different application views
type View int

const (
	ViewQuery View = iota
	ViewResults
	ViewHistory
	ViewHelp
	ViewWorkspace
	ViewRowDetail
	ViewTemplates
)

// Model is the main application model
type Model struct {
	// Core components
	editor         QueryEditor
	table          ResultsTable
	spinner        spinner.Model
	workspaceInput textinput.Model

	// Azure clients
	client       *azure.LogAnalyticsClient
	openaiClient *azure.OpenAIClient
	auth         *azure.Authenticator
	authMethod   azure.AuthMethod
	config       *azure.Config
	history      *azure.History

	// State
	currentView      View
	width            int
	height           int
	loading          bool
	lastQuery        string
	lastError        string
	lastDuration     time.Duration
	rowCount         int
	styles           *Styles
	connected        bool
	connecting       bool
	workspaceID      string
	historyIndex     int
	historyList      []azure.HistoryEntry
	detailScrollPos  int
	hideEmptyFields  bool // Hide empty/null fields in row detail view

	// Autocomplete state
	suggestion            string
	suggestLoading        bool
	suggestionDebounceTag int
	availableTables       []string
	schemaCache           map[string][]azure.Column // Cache of table schemas

	// Local autocomplete
	autocompleteEngine *AutocompleteEngine
	suggestionPopup    *SuggestionPopup

	// Templates state
	templates      *azure.Templates
	templateList   []azure.TemplateEntry
	templateIndex  int
	templateInput  textinput.Model
	savingTemplate bool
}

// Messages
type queryResultMsg struct {
	result *azure.QueryResult
	err    error
}

type connectMsg struct {
	err          error
	auth         *azure.Authenticator
	client       *azure.LogAnalyticsClient
	openaiClient *azure.OpenAIClient
}

type suggestionMsg struct {
	suggestion string
	err        error
	tag        int
}

type debounceMsg struct {
	tag int
}

type tablesMsg struct {
	tables []string
	err    error
}

type schemaMsg struct {
	tableName string
	columns   []azure.Column
	err       error
}

// waitForDebounce waits for a short period before triggering autocomplete
func waitForDebounce(tag int) tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
		return debounceMsg{tag: tag}
	})
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

	templates := azure.NewTemplates()
	templates.Load()

	ti := textinput.New()
	ti.Placeholder = "Enter template name"
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		editor:             NewQueryEditor(),
		table:              NewResultsTable(),
		spinner:            s,
		workspaceInput:     wi,
		config:             config,
		history:            history,
		authMethod:         authMethod,
		currentView:        ViewQuery,
		styles:             DefaultStyles(),
		workspaceID:        workspaceID,
		connecting:         workspaceID != "", // Start connecting if workspace provided
		schemaCache:        make(map[string][]azure.Column),
		hideEmptyFields:    true, // Hide empty fields by default
		autocompleteEngine: NewAutocompleteEngine(),
		suggestionPopup:    NewSuggestionPopup(),
		templates:          templates,
		templateInput:      ti,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.editor.Init(),
		m.spinner.Tick,
	}

	// Auto-connect if workspace is provided
	if m.workspaceID != "" {
		cmds = append(cmds, m.Connect(m.authMethod))
	}

	return tea.Batch(cmds...)
}

// Connect connects to Azure
func (m *Model) Connect(authMethod azure.AuthMethod) tea.Cmd {
	workspaceID := m.workspaceID
	return func() tea.Msg {
		auth, err := azure.NewAuthenticator(authMethod)
		if err != nil {
			return connectMsg{err: err, auth: nil, client: nil, openaiClient: nil}
		}

		client, err := azure.NewLogAnalyticsClient(auth.GetCredential(), workspaceID)
		if err != nil {
			return connectMsg{err: err, auth: nil, client: nil, openaiClient: nil}
		}

		// Create OpenAI client for autocomplete
		openaiClient := azure.NewOpenAIClientWithDefaults(auth.GetCredential())

		return connectMsg{err: nil, auth: auth, client: client, openaiClient: openaiClient}
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
			m.historyList = m.history.GetRecent(50)
			m.historyIndex = 0
			m.currentView = ViewHistory
			return m, nil

		case "f3":
			m.currentView = ViewWorkspace
			m.workspaceInput.Focus()
			return m, nil

		case "f4":
			m.templateList = m.templates.GetAll()
			m.templateIndex = 0
			m.currentView = ViewTemplates
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
		case ViewRowDetail:
			return m.updateRowDetailView(msg)
		case ViewTemplates:
			return m.updateTemplatesView(msg)
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
		m.connecting = false
		if msg.err != nil {
			m.lastError = fmt.Sprintf("Connection failed: %v", msg.err)
			m.connected = false
		} else {
			m.auth = msg.auth
			m.client = msg.client
			m.openaiClient = msg.openaiClient
			m.connected = true
			m.lastError = ""
			// Load available tables for autocomplete context
			return m, m.loadAvailableTables()
		}
		return m, nil

	case suggestionMsg:
		if msg.tag == m.suggestionDebounceTag {
			m.suggestLoading = false
			if msg.err != nil {
				// Silently ignore suggestion errors
				m.suggestion = ""
			} else {
				m.suggestion = msg.suggestion
			}
		}
		return m, nil

	case debounceMsg:
		if msg.tag == m.suggestionDebounceTag {
			if !m.connected || m.openaiClient == nil {
				return m, nil
			}
			m.suggestLoading = true
			return m, m.getSuggestion(m.suggestionDebounceTag)
		}
		return m, nil

	case tablesMsg:
		if msg.err == nil {
			m.availableTables = msg.tables
			m.autocompleteEngine.SetTables(msg.tables)
			return m, m.fetchInitialSchemas(msg.tables)
		}
		return m, nil

	case schemaMsg:
		if msg.err == nil && msg.tableName != "" {
			if m.schemaCache == nil {
				m.schemaCache = make(map[string][]azure.Column)
			}
			m.schemaCache[msg.tableName] = msg.columns
			m.autocompleteEngine.SetSchemas(m.schemaCache)
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateQueryView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle popup navigation first if popup is visible
	if m.suggestionPopup.IsVisible() {
		switch msg.String() {
		case "up", "ctrl+p":
			m.suggestionPopup.Previous()
			return m, nil
		case "down", "ctrl+n":
			m.suggestionPopup.Next()
			return m, nil
		case "tab", "enter":
			// Accept selected suggestion
			if selected := m.suggestionPopup.GetSelectedText(); selected != "" {
				m.acceptLocalSuggestion(selected)
			}
			m.suggestionPopup.Hide()
			return m, nil
		case "esc":
			m.suggestionPopup.Hide()
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+enter", "f5":
		if !m.connected {
			m.lastError = "Not connected. Press F3 to set workspace."
			return m, nil
		}
		m.suggestion = "" // Clear any pending suggestion
		m.suggestionPopup.Hide()
		return m.executeQuery()

	case "tab":
		// Accept AI suggestion if available, otherwise switch to results
		if m.suggestion != "" {
			m.editor.SetValue(m.suggestion)
			m.suggestion = ""
			return m, nil
		}
		m.currentView = ViewResults
		m.editor.Blur()
		m.table.Focus()
		return m, nil

	case "ctrl+@", "ctrl+ ", "alt+s": // Ctrl+Space or Alt+S to manually trigger AI autocomplete
		if !m.connected || m.openaiClient == nil {
			m.lastError = "Connect to workspace first for AI suggestions"
			return m, nil
		}
		m.suggestionDebounceTag++
		tag := m.suggestionDebounceTag
		m.suggestLoading = true
		m.suggestion = ""
		m.suggestionPopup.Hide()
		return m, m.getSuggestion(tag)

	case "ctrl+l":
		m.editor.Reset()
		m.suggestion = ""
		m.suggestionPopup.Hide()
		return m, nil

	case "ctrl+s", "f6":
		// Save current query as template
		if m.editor.Value() != "" {
			m.savingTemplate = true
			m.templateInput.SetValue("")
			m.templateInput.Focus()
			return m, nil
		}

	case "esc":
		// Clear AI suggestion if present
		if m.suggestion != "" {
			m.suggestion = ""
			return m, nil
		}

	case "ctrl+up":
		// Navigate history
		m.suggestion = "" // Clear suggestion when navigating history
		m.suggestionPopup.Hide()
		return m.navigateHistory(-1)

	case "ctrl+down":
		// Navigate history
		m.suggestion = "" // Clear suggestion when navigating history
		m.suggestionPopup.Hide()
		return m.navigateHistory(1)
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)

	// Trigger local autocomplete on typing
	if len(msg.String()) == 1 || msg.String() == "backspace" || msg.String() == "delete" {
		m.suggestion = ""
		m.suggestionDebounceTag++

		// Update local autocomplete immediately
		m.updateLocalSuggestions()

		return m, tea.Batch(cmd, waitForDebounce(m.suggestionDebounceTag))
	}

	return m, cmd
}

func (m Model) updateResultsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.currentView = ViewQuery
		m.table.Blur()
		m.editor.Focus()
		return m, nil

	case "enter":
		// Open row detail view
		if m.table.RowCount() > 0 {
			m.detailScrollPos = 0
			m.currentView = ViewRowDetail
		}
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
		m.connecting = true
		m.connected = false
		return m, m.Connect(m.authMethod)
	}

	var cmd tea.Cmd
	m.workspaceInput, cmd = m.workspaceInput.Update(msg)
	return m, cmd
}

func (m Model) updateRowDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	row := m.table.GetSelectedRow()
	columns := m.table.GetColumns()
	maxScroll := len(columns) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch msg.String() {
	case "esc", "q", "enter":
		m.currentView = ViewResults
		return m, nil

	case "up", "k":
		if m.detailScrollPos > 0 {
			m.detailScrollPos--
		}
		return m, nil

	case "down", "j":
		if m.detailScrollPos < maxScroll {
			m.detailScrollPos++
		}
		return m, nil

	case "home", "g":
		m.detailScrollPos = 0
		return m, nil

	case "end", "G":
		m.detailScrollPos = maxScroll
		return m, nil

	case "pgup":
		m.detailScrollPos -= 10
		if m.detailScrollPos < 0 {
			m.detailScrollPos = 0
		}
		return m, nil

	case "pgdown":
		m.detailScrollPos += 10
		if m.detailScrollPos > maxScroll {
			m.detailScrollPos = maxScroll
		}
		return m, nil

	case "h":
		// Toggle hiding empty fields
		m.hideEmptyFields = !m.hideEmptyFields
		m.detailScrollPos = 0 // Reset scroll when toggling
		return m, nil
	}

	_ = row // Suppress unused warning
	return m, nil
}

func (m Model) updateTemplatesView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle save template dialog
	if m.savingTemplate {
		switch msg.String() {
		case "enter":
			name := m.templateInput.Value()
			if name != "" {
				m.templates.Add(name, m.editor.Value(), "", nil)
				m.templates.Save()
			}
			m.savingTemplate = false
			return m, nil
		case "esc":
			m.savingTemplate = false
			return m, nil
		}
		var cmd tea.Cmd
		m.templateInput, cmd = m.templateInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		if m.templateIndex >= 0 && m.templateIndex < len(m.templateList) {
			m.editor.SetValue(m.templateList[m.templateIndex].Query)
			m.templates.IncrementUseCount(m.templateList[m.templateIndex].ID)
			m.templates.Save()
			m.currentView = ViewQuery
			m.editor.Focus()
		}
		return m, nil

	case "d":
		if len(m.templateList) > 0 && m.templateIndex < len(m.templateList) {
			m.templates.Delete(m.templateList[m.templateIndex].ID)
			m.templates.Save()
			m.templateList = m.templates.GetAll()
			if m.templateIndex >= len(m.templateList) && m.templateIndex > 0 {
				m.templateIndex--
			}
		}
		return m, nil

	case "up", "k":
		if m.templateIndex > 0 {
			m.templateIndex--
		}
		return m, nil

	case "down", "j":
		if m.templateIndex < len(m.templateList)-1 {
			m.templateIndex++
		}
		return m, nil

	case "n":
		// Create new template from current query (if any)
		if m.editor.Value() != "" {
			m.savingTemplate = true
			m.templateInput.SetValue("")
			m.templateInput.Focus()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) executeQuery() (tea.Model, tea.Cmd) {
	query := strings.TrimSpace(m.editor.Value())
	if query == "" {
		m.lastError = "Query cannot be empty"
		return m, nil
	}

	// Add default limit if query doesn't specify one
	query = ensureQueryLimit(query, 100)

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

// ensureQueryLimit adds a limit to the query if one isn't already specified
func ensureQueryLimit(query string, defaultLimit int) string {
	queryLower := strings.ToLower(query)

	// Check if query already has a limit (take, limit, or top)
	limitKeywords := []string{"| take ", "|take ", "| limit ", "|limit ", "| top ", "|top "}
	for _, kw := range limitKeywords {
		if strings.Contains(queryLower, kw) {
			return query // Already has a limit
		}
	}

	// Also check for limit at the very end without space after pipe
	if strings.HasSuffix(queryLower, "| take") || strings.HasSuffix(queryLower, "|take") ||
		strings.HasSuffix(queryLower, "| limit") || strings.HasSuffix(queryLower, "|limit") ||
		strings.HasSuffix(queryLower, "| top") || strings.HasSuffix(queryLower, "|top") {
		return query // User is typing a limit
	}

	// Add default limit
	return fmt.Sprintf("%s | take %d", query, defaultLimit)
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
	m.templates.Save()
}

// loadAvailableTables fetches available tables for autocomplete context
func (m *Model) loadAvailableTables() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return tablesMsg{err: fmt.Errorf("not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tables, err := m.client.GetAvailableTables(ctx)
		return tablesMsg{tables: tables, err: err}
	}
}

// fetchInitialSchemas fetches schemas for the top available tables
func (m *Model) fetchInitialSchemas(tables []string) tea.Cmd {
	var cmds []tea.Cmd
	// Fetch schema for up to 10 common tables
	limit := 10
	if len(tables) < limit {
		limit = len(tables)
	}

	for i := 0; i < limit; i++ {
		table := tables[i]
		cmds = append(cmds, func() tea.Msg {
			if m.client == nil {
				return nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			columns, err := m.client.GetTableSchema(ctx, table)
			return schemaMsg{tableName: table, columns: columns, err: err}
		})
	}
	return tea.Batch(cmds...)
}

// getSuggestion fetches a query suggestion from OpenAI
func (m *Model) getSuggestion(tag int) tea.Cmd {
	return func() tea.Msg {
		if m.openaiClient == nil {
			return suggestionMsg{err: fmt.Errorf("OpenAI not available"), tag: tag}
		}
		query := m.editor.Value()
		if query == "" {
			return suggestionMsg{err: fmt.Errorf("empty query"), tag: tag}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Parse tables from the query and fetch their schemas
		referencedTables := m.parseTablesFromQuery(query)
		schemas := m.fetchSchemasForTables(ctx, referencedTables)

		suggestion, err := m.openaiClient.SuggestKQLQuery(ctx, query, m.availableTables, schemas)
		return suggestionMsg{suggestion: suggestion, err: err, tag: tag}
	}
}

// updateLocalSuggestions updates the popup with local autocomplete suggestions
func (m *Model) updateLocalSuggestions() {
	query := m.editor.Value()
	cursorPos := m.editor.CursorPosition()

	// Parse context and get suggestions
	ctx := m.autocompleteEngine.ParseContext(query, cursorPos)
	suggestions := m.autocompleteEngine.GetSuggestions(ctx, 10)

	// Filter out exact matches (already typed)
	var filtered []Suggestion
	for _, s := range suggestions {
		if s.Text != ctx.CurrentWord {
			filtered = append(filtered, s)
		}
	}

	m.suggestionPopup.SetSuggestions(filtered)
}

// acceptLocalSuggestion accepts a suggestion from the popup
func (m *Model) acceptLocalSuggestion(text string) {
	query := m.editor.Value()
	cursorPos := m.editor.CursorPosition()

	// Parse context to find what to replace
	ctx := m.autocompleteEngine.ParseContext(query, cursorPos)

	// Build new query: text before current word + suggestion + text after cursor
	beforeWord := query[:ctx.WordStartPos]
	afterCursor := ""
	if cursorPos < len(query) {
		afterCursor = query[cursorPos:]
	}

	newQuery := beforeWord + text + afterCursor
	m.editor.SetValue(newQuery)
}

// parseTablesFromQuery extracts table names from a KQL query
func (m *Model) parseTablesFromQuery(query string) []string {
	var tables []string
	seen := make(map[string]bool)

	// Check if any available table name appears at the start of the query
	// or after a pipe, union, or join
	queryLower := strings.ToLower(query)

	for _, table := range m.availableTables {
		tableLower := strings.ToLower(table)

		// Check various positions where a table name might appear
		patterns := []string{
			tableLower,                    // At the start or anywhere
			"| " + tableLower,             // After pipe
			"|" + tableLower,              // After pipe (no space)
			"union " + tableLower,         // In union
			"join " + tableLower,          // In join
			"join (" + tableLower,         // In join with paren
		}

		for _, pattern := range patterns {
			if strings.Contains(queryLower, pattern) {
				if !seen[table] {
					tables = append(tables, table)
					seen[table] = true
				}
				break
			}
		}
	}

	return tables
}

// fetchSchemasForTables fetches schemas for the given tables, using cache when available
func (m *Model) fetchSchemasForTables(ctx context.Context, tables []string) map[string][]azure.Column {
	schemas := make(map[string][]azure.Column)

	for _, table := range tables {
		// Check cache first
		if cached, ok := m.schemaCache[table]; ok {
			schemas[table] = cached
			continue
		}

		// Fetch from Log Analytics if not cached
		if m.client != nil {
			columns, err := m.client.GetTableSchema(ctx, table)
			if err == nil && len(columns) > 0 {
				schemas[table] = columns
				// Note: Can't update schemaCache here as this runs in a goroutine
				// The cache will be updated via schemaMsg if needed
			}
		}
	}

	return schemas
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
	case ViewRowDetail:
		b.WriteString(m.renderRowDetailView())
	case ViewTemplates:
		b.WriteString(m.renderTemplatesView())
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
	} else if m.connecting {
		parts = append(parts, m.spinner.View()+" "+m.styles.Warning.Render("Connecting..."))
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

	// Local autocomplete popup (takes priority)
	if m.suggestionPopup.IsVisible() {
		b.WriteString("\n")
		b.WriteString(m.suggestionPopup.View())
	} else if m.suggestLoading {
		// Show AI suggestion loading indicator
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(" Getting AI suggestion..."))
	} else if m.suggestion != "" {
		// Show AI suggestion ghost text
		b.WriteString("\n")

		// Calculate ghost text
		current := m.editor.Value()
		suggestion := m.suggestion
		var preview string

		if strings.HasPrefix(suggestion, current) && len(suggestion) > len(current) {
			// Suggestion extends current input - Ghost text style
			ghost := suggestion[len(current):]
			preview = m.styles.Bold.Render(current) + m.styles.Muted.Render(ghost)
		} else {
			// Suggestion is different or shorter - Show full suggestion muted
			preview = m.styles.Muted.Render(suggestion)
		}

		// Simple instructions
		b.WriteString(preview)
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(" [Tab] to accept · [Esc] to dismiss"))
	}

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

func (m Model) renderTemplatesView() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Query Templates"))
	b.WriteString("\n\n")

	// Handle save template dialog overlay
	if m.savingTemplate {
		b.WriteString("Save Current Query as Template\n\n")
		b.WriteString("Name: ")
		b.WriteString(m.templateInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.styles.Muted.Render("Press Enter to save, Esc to cancel"))
		return b.String()
	}

	if len(m.templateList) == 0 {
		b.WriteString(m.styles.Muted.Render("No templates saved yet."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Muted.Render("Press Ctrl+S or F6 in query view to save current query as template."))
		return b.String()
	}

	for i, tmpl := range m.templateList {
		prefix := "  "
		style := m.styles.Muted
		if i == m.templateIndex {
			prefix = "▶ "
			style = m.styles.Bold
		}

		name := tmpl.Name
		query := truncateString(tmpl.Query, 50)
		uses := ""
		if tmpl.UseCount > 0 {
			uses = fmt.Sprintf(" (%d uses)", tmpl.UseCount)
		}

		line := fmt.Sprintf("%s%s: %s%s", prefix, name, query, uses)
		b.WriteString(style.Render(line))
		b.WriteString("\n")

		if i >= 20 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ... and %d more", len(m.templateList)-20)))
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
  F4            Show saved templates
  Esc           Return to query view / Dismiss suggestion
  Ctrl+Q        Quit

QUERY EDITOR
  F5, Ctrl+Enter   Execute query
  Ctrl+Space       AI query suggestion (Azure OpenAI)
  Ctrl+S, F6       Save query as template
  Tab              Accept AI suggestion (when shown)
  Ctrl+L           Clear editor
  Ctrl+Up/Down     Navigate query history

RESULTS TABLE
  j/k, Up/Down     Navigate rows
  h/l, Left/Right  Scroll columns
  Enter            View row details (full content)
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

func (m Model) renderRowDetailView() string {
	var b strings.Builder

	row := m.table.GetSelectedRow()
	columns := m.table.GetColumns()
	rowIdx := m.table.GetSelectedRowIndex()

	b.WriteString(m.styles.Header.Render(fmt.Sprintf("Row Detail (Row %d/%d)", rowIdx+1, m.table.RowCount())))
	b.WriteString("\n\n")

	if row == nil || len(columns) == 0 {
		b.WriteString(m.styles.Muted.Render("No row selected"))
		return b.String()
	}

	// Build list of fields to display (filter empty if enabled)
	type fieldInfo struct {
		name  string
		value string
	}
	var fields []fieldInfo
	totalFields := len(columns)

	for i, col := range columns {
		if i >= len(row) {
			break
		}
		value := row[i]
		// Skip empty fields if hiding is enabled
		if m.hideEmptyFields && value == "" {
			continue
		}
		fields = append(fields, fieldInfo{name: col, value: value})
	}

	// Calculate visible rows based on height
	visibleRows := m.height - 12
	if visibleRows < 5 {
		visibleRows = 5
	}

	// Adjust scroll position if it exceeds filtered list
	maxScroll := len(fields) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	scrollPos := m.detailScrollPos
	if scrollPos > maxScroll {
		scrollPos = maxScroll
	}

	// Show fields with scroll
	endIdx := scrollPos + visibleRows
	if endIdx > len(fields) {
		endIdx = len(fields)
	}

	// Calculate max column name width for alignment
	maxNameWidth := 0
	for _, f := range fields {
		if len(f.name) > maxNameWidth {
			maxNameWidth = len(f.name)
		}
	}
	if maxNameWidth > 30 {
		maxNameWidth = 30
	}

	for i := scrollPos; i < endIdx; i++ {
		f := fields[i]

		// Highlight current scroll position
		prefix := "  "
		if i == scrollPos {
			prefix = "▶ "
		}

		// Format column name with padding
		paddedName := f.name
		if len(paddedName) > maxNameWidth {
			paddedName = paddedName[:maxNameWidth-3] + "..."
		}
		for len(paddedName) < maxNameWidth {
			paddedName += " "
		}

		// Format value
		valueStr := f.value
		if valueStr == "" {
			valueStr = m.styles.Muted.Render("(empty)")
		}

		line := fmt.Sprintf("%s%s: %s",
			prefix,
			m.styles.Bold.Foreground(ColorSecondary).Render(paddedName),
			valueStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator with filter info
	b.WriteString("\n")
	if m.hideEmptyFields {
		scrollInfo := fmt.Sprintf("Showing %d/%d fields (hiding %d empty) · h to show all",
			len(fields), totalFields, totalFields-len(fields))
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	} else {
		scrollInfo := fmt.Sprintf("Showing all %d fields · h to hide empty", totalFields)
		b.WriteString(m.styles.Muted.Render(scrollInfo))
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.Muted.Render("j/k to scroll · Esc to return"))

	return b.String()
}

func (m Model) renderFooter() string {
	var keys []string

	switch m.currentView {
	case ViewQuery:
		keys = []string{
			m.styles.HelpKey.Render("F5") + " Execute",
			m.styles.HelpKey.Render("Ctrl+Space") + " AI Suggest",
			m.styles.HelpKey.Render("Tab") + " Results",
			m.styles.HelpKey.Render("F2") + " History",
			m.styles.HelpKey.Render("F4") + " Templates",
			m.styles.HelpKey.Render("Ctrl+Q") + " Quit",
		}
	case ViewResults:
		keys = []string{
			m.styles.HelpKey.Render("Enter") + " Details",
			m.styles.HelpKey.Render("Tab") + " Editor",
			m.styles.HelpKey.Render("j/k") + " Navigate",
			m.styles.HelpKey.Render("h/l") + " Scroll",
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	case ViewRowDetail:
		keys = []string{
			m.styles.HelpKey.Render("j/k") + " Scroll",
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	case ViewHistory:
		keys = []string{
			m.styles.HelpKey.Render("Enter") + " Select",
			m.styles.HelpKey.Render("j/k") + " Navigate",
			m.styles.HelpKey.Render("Esc") + " Back",
		}
	case ViewTemplates:
		keys = []string{
			m.styles.HelpKey.Render("Enter") + " Load",
			m.styles.HelpKey.Render("d") + " Delete",
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
