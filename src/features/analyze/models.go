package analyze

// AnalysisType represents different types of analysis that can be performed
type AnalysisType string

const (
	AnalysisTypeAcoustID AnalysisType = "acoustid"
	AnalysisTypeLyrics   AnalysisType = "lyrics"
	// Future analysis types can be added here
	// AnalysisTypeMetadata  AnalysisType = "metadata"
)
