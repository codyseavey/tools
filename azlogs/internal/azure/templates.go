package azure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TemplateEntry represents a saved query template
type TemplateEntry struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Query       string    `json:"query"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UseCount    int       `json:"use_count"`
}

// Templates manages query templates
type Templates struct {
	Entries  []TemplateEntry `json:"entries"`
	filePath string
}

// NewTemplates creates a new templates manager
func NewTemplates() *Templates {
	t := &Templates{
		Entries: []TemplateEntry{},
	}
	t.setDefaultPath()
	return t
}

// setDefaultPath sets the default templates file path
func (t *Templates) setDefaultPath() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".config", "azlogs")
	t.filePath = filepath.Join(configDir, "templates.json")
}

// Load reads templates from disk
func (t *Templates) Load() error {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No templates file yet
		}
		return err
	}

	return json.Unmarshal(data, t)
}

// Save writes templates to disk
func (t *Templates) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(t.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.filePath, data, 0644)
}

// Add adds a new template
func (t *Templates) Add(name, query, description string, tags []string) *TemplateEntry {
	entry := TemplateEntry{
		ID:          uuid.New().String(),
		Name:        name,
		Query:       query,
		Description: description,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		UseCount:    0,
	}

	t.Entries = append(t.Entries, entry)
	return &t.Entries[len(t.Entries)-1]
}

// Update updates an existing template
func (t *Templates) Update(id string, name, query, description string, tags []string) bool {
	for i := range t.Entries {
		if t.Entries[i].ID == id {
			t.Entries[i].Name = name
			t.Entries[i].Query = query
			t.Entries[i].Description = description
			t.Entries[i].Tags = tags
			t.Entries[i].UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// Delete removes a template by ID
func (t *Templates) Delete(id string) bool {
	for i, entry := range t.Entries {
		if entry.ID == id {
			t.Entries = append(t.Entries[:i], t.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// GetByID finds a template by ID
func (t *Templates) GetByID(id string) *TemplateEntry {
	for i := range t.Entries {
		if t.Entries[i].ID == id {
			return &t.Entries[i]
		}
	}
	return nil
}

// GetAll returns all templates
func (t *Templates) GetAll() []TemplateEntry {
	return t.Entries
}

// IncrementUseCount increments the use count for a template
func (t *Templates) IncrementUseCount(id string) {
	for i := range t.Entries {
		if t.Entries[i].ID == id {
			t.Entries[i].UseCount++
			t.Entries[i].UpdatedAt = time.Now()
			return
		}
	}
}

// Search searches templates by name, description, or tags
func (t *Templates) Search(query string) []TemplateEntry {
	if query == "" {
		return t.Entries
	}

	var results []TemplateEntry
	queryLower := strings.ToLower(query)

	for _, entry := range t.Entries {
		// Check name
		if strings.Contains(strings.ToLower(entry.Name), queryLower) {
			results = append(results, entry)
			continue
		}

		// Check description
		if strings.Contains(strings.ToLower(entry.Description), queryLower) {
			results = append(results, entry)
			continue
		}

		// Check tags
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, entry)
				break
			}
		}
	}

	return results
}

// Count returns the number of templates
func (t *Templates) Count() int {
	return len(t.Entries)
}
