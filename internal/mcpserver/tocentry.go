// tocentry.go implements the toc tools' own whole-call validation and per-entry stat
// classification: cli.ValidateTOCLang and cli.ParseDocSentences run once, up front, exactly as the
// CLI runs them before any argument is processed (see internal/cli/toc.go's tocFileCommand and
// tocDirCommand RunE closures), and the stat step below applies toc's own status rule rather than
// the LSP predicates classifyLSPError/classifySymbolError use — toc consults no language server and
// resolves no symbol, so "ambiguous" never applies here and a missing path is "not_found" rather
// than "error".

package mcpserver

import (
	"fmt"
	"os"

	"github.com/Knatte18/quarry/internal/cli"
)

// tocPreflight runs the toc tools' whole-call validations in the CLI's own order: cli.ValidateTOCLang(lang)
// first, then, when doc carries a value, cli.ParseDocSentences on that value's string form. The
// parsed int is discarded — each entry re-resolves DocSentences against its own directory via
// cli.ResolveDocSentences, since the setting is per-directory and a batch may span directories —
// and only the string form is returned, empty when doc was not supplied. Any validation error here
// fails the whole call, exactly as tocFileCommand/tocDirCommand fail before processing any
// argument.
func tocPreflight(lang string, doc docSentences) (string, error) {
	if err := cli.ValidateTOCLang(lang); err != nil {
		return "", err
	}

	docString, ok := doc.value()
	if !ok {
		return "", nil
	}

	if _, err := cli.ParseDocSentences(docString); err != nil {
		return "", err
	}
	return docString, nil
}

// tocStat classifies the stat step for one toc entry at abs, applying toc's own status rule rather
// than the LSP predicates classifyLSPError uses: applying those here instead would report "error"
// where the CLI reports "not_found" for a missing file, which is a divergence rather than a
// mapping. os.IsNotExist on the stat error yields statusNotFound carrying the stat error's own
// message. A directory found where wantDir is false (a "toc_file" entry) yields statusError worded
// as tocFileOne words it. A non-directory found where wantDir is true (a "toc_dir" entry) yields
// statusError worded as tocDirOne words it. Any other stat error yields statusError carrying its
// own message.
//
// A nil returned error means the stat succeeded and the entry's type matches wantDir, so the
// caller should proceed to its facade call; a non-nil error means status and message are already
// this entry's final outcome and the caller must not proceed.
func tocStat(abs string, wantDir bool) (status string, message string, err error) {
	info, statErr := os.Stat(abs)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return statusNotFound, statErr.Error(), statErr
		}
		return statusError, statErr.Error(), statErr
	}

	if info.IsDir() && !wantDir {
		wrongType := fmt.Errorf("toc: %s is a directory; use %q for a directory listing", abs, "quarry toc dir")
		return statusError, wrongType.Error(), wrongType
	}
	if !info.IsDir() && wantDir {
		wrongType := fmt.Errorf("toc: %s is a file; use %q for a single file", abs, "quarry toc file")
		return statusError, wrongType.Error(), wrongType
	}

	return "", "", nil
}
