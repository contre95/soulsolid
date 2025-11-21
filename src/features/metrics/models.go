package metrics

import "strconv"

// ApexChartData represents data for ApexCharts.
type ApexChartData struct {
	Labels []string  `json:"labels"`
	Series []float64 `json:"series"`
	Colors []string  `json:"colors,omitempty"`
}

// GenreChartData converts genre metrics to ApexCharts format.
func (m *MetricsData) GenreChartData() *ApexChartData {
	labels := make([]string, len(m.GenreCounts))
	series := make([]float64, len(m.GenreCounts))
	colors := make([]string, len(m.GenreCounts))

	colorPalette := []string{
		"#008FFB", "#00E396", "#FEB019", "#FF4560", "#775DD0",
		"#00D9FF", "#FF6B6B", "#4ECDC4", "#5470C6", "#91EE7A",
	}

	for i, metric := range m.GenreCounts {
		labels[i] = metric.Key
		series[i] = float64(metric.Value)
		colors[i] = colorPalette[i%len(colorPalette)]
	}

	return &ApexChartData{
		Labels: labels,
		Series: series,
		Colors: colors,
	}
}

// LyricsChartData converts lyrics stats to ApexCharts format.
func (m *MetricsData) LyricsChartData() *ApexChartData {
	labels := []string{"With Lyrics", "Without Lyrics"}
	series := []float64{0, 0}

	for _, metric := range m.LyricsStats {
		if metric.Key == "has_lyrics" {
			series[0] = float64(metric.Value)
		} else if metric.Key == "no_lyrics" {
			series[1] = float64(metric.Value)
		}
	}

	return &ApexChartData{
		Labels: labels,
		Series: series,
		Colors: []string{"#00E396", "#FF4560"},
	}
}

// MetadataChartData converts metadata completeness to ApexCharts format.
func (m *MetricsData) MetadataChartData() *ApexChartData {
	labels := make([]string, len(m.MetadataCompleteness))
	series := make([]float64, len(m.MetadataCompleteness))

	for i, metric := range m.MetadataCompleteness {
		switch metric.Key {
		case "complete":
			labels[i] = "Complete Metadata"
		case "missing_genre":
			labels[i] = "Missing Genre"
		case "missing_year":
			labels[i] = "Missing Year"
		case "missing_lyrics":
			labels[i] = "Missing Lyrics"
		default:
			labels[i] = metric.Key
		}
		series[i] = float64(metric.Value)
	}

	return &ApexChartData{
		Labels: labels,
		Series: series,
		Colors: []string{"#00E396", "#FEB019", "#FF4560", "#008FFB"},
	}
}

// YearBarData converts year distribution to ApexCharts format.
func (m *MetricsData) YearBarData() *ApexChartData {
	// Filter years from 1900 to current year
	var filtered []Metric
	for _, metric := range m.YearDistribution {
		if year, err := strconv.Atoi(metric.Key); err == nil && year >= 1900 {
			filtered = append(filtered, metric)
		}
	}

	// Sort by year
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			yi, _ := strconv.Atoi(filtered[i].Key)
			yj, _ := strconv.Atoi(filtered[j].Key)
			if yi > yj {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	labels := make([]string, len(filtered))
	series := make([]float64, len(filtered))

	for i, metric := range filtered {
		labels[i] = metric.Key
		series[i] = float64(metric.Value)
	}

	return &ApexChartData{
		Labels: labels,
		Series: series,
		Colors: []string{"#008FFB"},
	}
}

// FormatBarData converts format distribution to ApexCharts format.
func (m *MetricsData) FormatBarData() *ApexChartData {
	labels := make([]string, len(m.FormatDistribution))
	series := make([]float64, len(m.FormatDistribution))
	colors := make([]string, len(m.FormatDistribution))

	// Blue and green colors alternating
	colorPalette := []string{"#008FFB", "#00E396"} // Blue, Green

	for i, metric := range m.FormatDistribution {
		labels[i] = metric.Key
		series[i] = float64(metric.Value)
		colors[i] = colorPalette[i%2]
	}

	return &ApexChartData{
		Labels: labels,
		Series: series,
		Colors: colors,
	}
}
