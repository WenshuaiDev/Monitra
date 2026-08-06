package runtimeconfig_test

import (
	"bytes"
	"testing"

	"monitra/internal/foundation/runtimeconfig"
)

func TestProjectWritesStrictBrowserSafeConfiguration(t *testing.T) {
	var output bytes.Buffer

	err := runtimeconfig.Project(&output, runtimeconfig.Configuration{
		ExpectedAPIMajor:        1,
		ExpectedReleaseIdentity: "2026.08.06-production",
	})
	if err != nil {
		t.Fatalf("project runtime config: %v", err)
	}

	const expected = `{"expected_api_major":1,"expected_release_identity":"2026.08.06-production"}` + "\n"
	if output.String() != expected {
		t.Fatalf("runtime config = %q, want %q", output.String(), expected)
	}
}

func TestProjectRejectsInvalidPublicConfigurationWithoutWriting(t *testing.T) {
	tests := []struct {
		name          string
		configuration runtimeconfig.Configuration
	}{
		{name: "missing API major", configuration: runtimeconfig.Configuration{ExpectedReleaseIdentity: "release"}},
		{name: "missing release identity", configuration: runtimeconfig.Configuration{ExpectedAPIMajor: 1}},
		{name: "whitespace around release identity", configuration: runtimeconfig.Configuration{ExpectedAPIMajor: 1, ExpectedReleaseIdentity: " release "}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := runtimeconfig.Project(&output, test.configuration); err == nil {
				t.Fatal("project runtime config succeeded with invalid public configuration")
			}
			if output.Len() != 0 {
				t.Fatalf("invalid projection wrote %q", output.String())
			}
		})
	}
}
