// tocconfig.go holds the optional toc config file's shape and loader.
// It deliberately does not reuse the registry package's LoadRegistry: that loader and this one
// have nothing in common but their YAML encoding, and sharing a loader would couple toc to the
// language-server registry it exists without.

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/Knatte18/quarry/quarry"
)

// tocConfigFile is the decoded shape of the optional toc config file: a single "toc" mapping.
// TOC is a pointer so an absent "toc" section is distinguishable from a present, empty one.
type tocConfigFile struct {
	TOC *tocSection `yaml:"toc"`
}

// tocSection is the "toc" mapping inside the toc config file.
// DocSentences is a *string rather than a *int because the setting's value is a union of a
// non-negative integer and the word "all": yaml.v3 decodes an unquoted integer scalar into a
// string field as its literal text, which is what makes one field serve both forms. Do not
// "fix" this to *int — that silently breaks "all". The pointer distinguishes an absent key
// (nil) from a present zero ("0").
type tocSection struct {
	DocSentences *string `yaml:"doc_sentences"`
}

// loadTOCConfig reads and decodes the toc config file at path, returning the raw doc_sentences
// value the file supplied, or nil when the file supplied none.
//
// A file that does not exist returns (nil, nil): an absent file is not an error, and the
// built-in default applies. Any other read error is returned wrapped. Decoding uses
// yaml.NewDecoder with decoder.KnownFields(true), exactly as registry.LoadRegistry does, so a
// misspelled key is a loud error naming the file rather than a silent no-op. An empty or
// comments-only file yields io.EOF from Decode with nothing set, which is a valid "no settings"
// file and returns (nil, nil) rather than an error — LoadRegistry already handles that case the
// same way. Any other decode error is returned wrapped with the file path prepended.
func loadTOCConfig(path string) (*string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("cli: read %s: %w", path, err)
	}

	var file tocConfigFile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		// An empty or comments-only file yields io.EOF from Decode with
		// nothing set — that is a valid "no settings" file, not malformed
		// YAML.
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	if file.TOC == nil {
		return nil, nil
	}
	return file.TOC.DocSentences, nil
}

// ParseDocSentences parses raw — the --doc-sentences flag value or the toc config file's
// doc_sentences value — into the int quarry.TOCOptions.DocSentences expects. It is the one place
// the value's grammar is defined, so both the flag and the config file reject the same values
// with the same message.
//
// "all" (case-sensitive, matching the documented form) yields quarry.TOCAllSentences. A
// non-negative integer yields itself. A negative integer, or any other string, is an error
// naming the valid forms.
func ParseDocSentences(raw string) (int, error) {
	if raw == "all" {
		return quarry.TOCAllSentences, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("toc: invalid doc-sentences value %q; want a non-negative integer, or %q", raw, "all")
	}
	return n, nil
}

// ResolveDocSentences resolves the effective DocSentences value for one "toc file" argument,
// highest precedence first:
//
//  1. a non-empty flagValue — parsed through ParseDocSentences;
//  2. the toc config file at resolveTOCConfigPath(targetDir), loaded through loadTOCConfig; when
//     it supplied a value, parsed through ParseDocSentences;
//  3. the built-in default, 1.
//
// This reads as three steps rather than the design's four-tier chain because tiers 2 and 3 of
// that chain — $QUARRY_TOC_CONFIG and the target directory's own .quarry.yaml — are both already
// resolved inside resolveTOCConfigPath; the two descriptions are not disagreeing.
func ResolveDocSentences(flagValue, targetDir string) (int, error) {
	if flagValue != "" {
		return ParseDocSentences(flagValue)
	}

	raw, err := loadTOCConfig(resolveTOCConfigPath(targetDir))
	if err != nil {
		return 0, err
	}
	if raw != nil {
		return ParseDocSentences(*raw)
	}

	return 1, nil
}
