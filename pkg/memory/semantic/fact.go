package semantic

import "time"

type FactType string

const (
	FactTypeTriple    FactType = "triple"
	FactTypeStatement FactType = "statement"
)

type Fact interface {
	isFact()
	GetID() string
	GetType() FactType
	GetSources() []Source
	GetTags() []string
	GetMetadata() FactMetadata
}

type TripleFact struct {
	ID        string
	Entity    string
	Attribute string
	Value     string
	Sources   []Source
	Tags      []string
	Metadata  FactMetadata
}

func (f TripleFact) isFact()                   {}
func (f TripleFact) GetID() string             { return f.ID }
func (f TripleFact) GetType() FactType         { return FactTypeTriple }
func (f TripleFact) GetSources() []Source      { return f.Sources }
func (f TripleFact) GetTags() []string         { return f.Tags }
func (f TripleFact) GetMetadata() FactMetadata { return f.Metadata }

type StatementFact struct {
	ID        string
	Statement string
	Sources   []Source
	Tags      []string
	Metadata  FactMetadata
}

func (f StatementFact) isFact()                   {}
func (f StatementFact) GetID() string             { return f.ID }
func (f StatementFact) GetType() FactType         { return FactTypeStatement }
func (f StatementFact) GetSources() []Source      { return f.Sources }
func (f StatementFact) GetTags() []string         { return f.Tags }
func (f StatementFact) GetMetadata() FactMetadata { return f.Metadata }

type Source struct {
	URI         string
	RetrievedAt time.Time
}

type FactMetadata struct {
	CreatedAt    time.Time
	ValidatedAt  time.Time
	LastAccessed time.Time
	AccessCount  int
	AccessScore  float64
	Constraints  map[string]string
}
