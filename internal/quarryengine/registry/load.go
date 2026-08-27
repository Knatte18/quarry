// load.go implements LoadRegistry, the optional servers.yaml overlay loader.
// LoadRegistry is told a resolved absolute config file path; resolving that path from
// --config/$QUARRY_CONFIG/os.UserConfigDir() precedence happens in internal/cli, not here. An
// absent file falls back to builtins() with no error, and present entries whole-replace the
// corresponding built-in.

package registry

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadRegistry loads the optional servers.yaml overlay from path, replacing built-in entries
// whole.
// An absent file returns builtins();
// an empty file also returns builtins() unchanged.
func LoadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// servers.yaml is optional — no file means "use the built-ins",
			// not a failure.
			return builtins(), nil
		}
		return nil, fmt.Errorf("quarry: read %s: %w", path, err)
	}

	var fileEntries map[string]Entry
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&fileEntries); err != nil {
		// An empty or comments-only file yields io.EOF from Decode with no
		// fileEntries set — that is a valid "no entries" file, not malformed
		// YAML.
		if errors.Is(err, io.EOF) {
			return builtins(), nil
		}
		return nil, fmt.Errorf("quarry: parse %s: %w", path, err)
	}

	registry := builtins()
	for name, entry := range fileEntries {
		if err := validateEntry(name, entry); err != nil {
			// validateEntry's message already names the offending entry;
			// prepend the file path so the operator knows which
			// servers.yaml to fix.
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		// Whole-entry replacement: the file's entry for this language
		// replaces the built-in (or absent) entry outright — no field-level
		// merge, so an override can never leak a stale built-in default.
		registry[name] = entry
	}

	return registry, nil
}
