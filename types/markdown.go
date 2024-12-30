// types/markdown.go
package types

// DocSection represents a parsed section of the markdown documentation
type DocSection struct {
	Title    string
	Content  string
	Sections map[string]*DocSection
}

// MarkdownFormatter interface defines methods for markdown generation and parsing
type MarkdownFormatter interface {
	MarkDownFromConfig(config interface{}) (string, error)
	ParseSections(markdown string) *DocSection
	GetSection(path ...string) (*DocSection, error)
	ListSections() [][]string
}
