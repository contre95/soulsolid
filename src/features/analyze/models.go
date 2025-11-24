package analyze

// AnalysisType represents different types of analysis that can be performed
type AnalysisType string

const (
	AnalysisTypeAcoustID AnalysisType = "acoustid"
	// Future analysis types can be added here
	// AnalysisTypeLyrics    AnalysisType = "lyrics"
	// AnalysisTypeMetadata  AnalysisType = "metadata"
)
