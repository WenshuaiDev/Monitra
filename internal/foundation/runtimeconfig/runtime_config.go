// Package runtimeconfig projects the browser-safe deployment configuration
// consumed before the application network boundary is created.
package runtimeconfig

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Configuration is the complete public configuration projected for browser
// bootstrap. It deliberately cannot carry database or other secret material.
type Configuration struct {
	ExpectedAPIMajor        int    `json:"expected_api_major"`
	ExpectedReleaseIdentity string `json:"expected_release_identity"`
}

// Project validates and writes the strict browser runtime-config document.
func Project(output io.Writer, configuration Configuration) error {
	if configuration.ExpectedAPIMajor < 1 {
		return errors.New("expected API major must be positive")
	}
	if configuration.ExpectedReleaseIdentity == "" ||
		strings.TrimSpace(configuration.ExpectedReleaseIdentity) != configuration.ExpectedReleaseIdentity {
		return errors.New("expected release identity must be non-empty and trimmed")
	}
	return json.NewEncoder(output).Encode(configuration)
}
