package music

import (
	"encoding/json"
	"fmt"
)

// StartTypedJob starts a background job from a typed parameter struct instead of
// a hand-built map, so a misspelled or wrongly-typed field is a compile error at
// the call site rather than a validation failure once the job runs.
//
// It is a free function rather than a method on JobService because Go does not
// allow type parameters on methods, and JobService has to stay a plain interface
// so feature packages never import the jobs engine directly.
//
// P must be the same struct the job's Task declares. The engine decodes
// job.Metadata back into that struct before running the task, so the type is
// enforced on both ends of the round trip.
func StartTypedJob[P any](js JobService, jobType string, name string, params P) (string, error) {
	metadata, err := paramsToMetadata(params)
	if err != nil {
		return "", err
	}
	return js.StartJob(jobType, name, metadata)
}

// paramsToMetadata converts a job parameter struct into the untyped map the
// engine stores, serializes and renders. Encoding through JSON keeps the keys
// identical to the struct's json tags, which is exactly what the engine uses to
// decode them again.
func paramsToMetadata[P any](params P) (map[string]any, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("invalid job parameters: %w", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("invalid job parameters: %w", err)
	}
	return metadata, nil
}
