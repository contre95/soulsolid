package providers

import (
	"net/http"
	"time"
)

// httpClient is shared by all providers. The timeout matters because jobs call
// providers with context.Background(), so a hung external API would otherwise
// block a job goroutine forever.
var httpClient = &http.Client{Timeout: 15 * time.Second}
