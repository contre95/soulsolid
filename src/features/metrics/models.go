package metrics

import "strconv"

// ChartData represents data for Chart.js charts.
type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset represents a Chart.js dataset.
type Dataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor []string  `json:"backgroundColor,omitempty"`
	BorderColor     []string  `json:"borderColor,omitempty"`
}

// GenreChartData converts genre metrics to chart format.
func (m *MetricsData) GenreChartData() *ChartData {
	labels := make([]string, len(m.GenreCounts))
	data := make([]float64, len(m.GenreCounts))
	colors := make([]string, len(m.GenreCounts))

	colorPalette := []string{
		"#FF6384", "#36A2EB", "#FFCE56", "#4BC0C0", "#9966FF",
		"#FF9F40", "#FF6384", "#C9CBCF", "#4BC0C0", "#FF6384",
	}

	for i, metric := range m.GenreCounts {
		labels[i] = metric.Key
		data[i] = float64(metric.Value)
		colors[i] = colorPalette[i%len(colorPalette)]
	}

	return &ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Tracks by Genre",
			Data:            data,
			BackgroundColor: colors,
		}},
	}
}

// LyricsChartData converts lyrics stats to chart format.
func (m *MetricsData) LyricsChartData() *ChartData {
	labels := []string{"With Lyrics", "Without Lyrics"}
	data := []float64{0, 0}

	for _, metric := range m.LyricsStats {
		if metric.Key == "has_lyrics" {
			data[0] = float64(metric.Value)
		} else if metric.Key == "no_lyrics" {
			data[1] = float64(metric.Value)
		}
	}

	return &ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Lyrics Presence",
			Data:            data,
			BackgroundColor: []string{"#4BC0C0", "#FF6384"},
		}},
	}
}

// MetadataChartData converts metadata completeness to chart format.
func (m *MetricsData) MetadataChartData() *ChartData {
	labels := make([]string, len(m.MetadataCompleteness))
	data := make([]float64, len(m.MetadataCompleteness))

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
		data[i] = float64(metric.Value)
	}

	return &ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Metadata Completeness",
			Data:            data,
			BackgroundColor: []string{"#4BC0C0", "#FFCE56", "#FF6384", "#36A2EB"},
		}},
	}
}

// YearBarData converts year distribution to vertical bar chart format.
func (m *MetricsData) YearBarData() *ChartData {
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
	data := make([]float64, len(filtered))

	for i, metric := range filtered {
		labels[i] = metric.Key
		data[i] = float64(metric.Value)
	}

	return &ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Tracks by Year",
			Data:            data,
			BackgroundColor: []string{"#36A2EB"},
		}},
	}
}

// FormatBarData converts format distribution to pie chart format.
func (m *MetricsData) FormatBarData() *ChartData {
	labels := make([]string, len(m.FormatDistribution))
	data := make([]float64, len(m.FormatDistribution))
	colors := make([]string, len(m.FormatDistribution))

	// Orange and blue colors alternating
	colorPalette := []string{"#FFCE56", "#36A2EB"} // Orange, Blue

	for i, metric := range m.FormatDistribution {
		labels[i] = metric.Key
		data[i] = float64(metric.Value)
		colors[i] = colorPalette[i%2]
	}

	return &ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Tracks by Format",
			Data:            data,
			BackgroundColor: colors,
		}},
	}
}
