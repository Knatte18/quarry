// main.go implements the mechanical half of the loomyard-to-quarry port: given a source
// directory, a destination directory, and an explicit list of filenames, it copies each named
// file and rewrites exactly two closed categories of token in the copied bytes — the loomyard
// import paths this task's engine package depends on, and the two package clauses being renamed.
// It is deliberately dependency-free (stdlib only) and touches nothing else: no string literal,
// no other identifier, no formatting change. Hand editing everything the rewrite is not allowed
// to touch is the rest of this batch's job, not this program's.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// importRewrites is the closed set of import path prefixes this program rewrites, applied in
// this order so the more specific loomyard paths are matched before any shorter prefix could
// shadow them. Every replacement is a literal, whole-import-path substring rewrite — no
// wildcarding beyond what is listed here.
var importRewrites = []struct {
	from string
	to   string
}{
	{"github.com/Knatte18/loomyard/internal/scoutengine", "github.com/Knatte18/quarry/quarry"},
	{"github.com/Knatte18/loomyard/internal/scoutcli", "github.com/Knatte18/quarry/internal/cli"},
	{"github.com/Knatte18/loomyard/internal/lock", "github.com/Knatte18/quarry/internal/lock"},
	{"github.com/Knatte18/loomyard/internal/proc", "github.com/Knatte18/quarry/internal/proc"},
	{"github.com/Knatte18/loomyard/internal/output", "github.com/Knatte18/quarry/internal/output"},
}

// packageClauseRewrites is the closed set of package clause renames this program applies. Each
// pattern is anchored at the start of a line so a "package scoutengine" mention inside a comment
// or a string literal is left alone.
var packageClauseRewrites = []struct {
	pattern *regexp.Regexp
	to      string
}{
	{regexp.MustCompile(`(?m)^package scoutengine$`), "package quarry"},
	{regexp.MustCompile(`(?m)^package scoutcli$`), "package cli"},
}

// scoutenginePrefixPattern counts occurrences of the "scoutengine: " error-message prefix so the
// operator can confirm the port left every string literal untouched.
var scoutenginePrefixPattern = regexp.MustCompile(`"scoutengine: `)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run parses flags, copies and rewrites every named file from src to dst, and prints the
// post-copy "scoutengine: " literal count.
func run(args []string) error {
	fs := flag.NewFlagSet("port", flag.ContinueOnError)
	src := fs.String("src", "", "source directory")
	dst := fs.String("dst", "", "destination directory")
	overwrite := fs.Bool("overwrite", false, "overwrite existing destination files")
	filesFlag := fs.String("files", "", "comma-separated list of \"srcName[:dstName]\" pairs, relative to -src/-dst")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *dst == "" || *filesFlag == "" {
		return fmt.Errorf("port: -src, -dst, and -files are all required")
	}

	pairs, err := parseFileList(*filesFlag)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*dst, 0o755); err != nil {
		return fmt.Errorf("port: create destination directory %s: %w", *dst, err)
	}

	for _, p := range pairs {
		if err := portFile(*src, *dst, p.from, p.to, *overwrite); err != nil {
			return err
		}
	}

	count, err := countScoutenginePrefix(*dst)
	if err != nil {
		return err
	}
	fmt.Printf("scoutengine: literal count in %s: %d\n", *dst, count)
	return nil
}

// fileMapping is one source-filename-to-destination-filename pair.
type fileMapping struct {
	from string
	to   string
}

// parseFileList parses -files's "srcName[:dstName]" comma-separated syntax. A pair with no colon
// keeps the same filename at the destination.
func parseFileList(spec string) ([]fileMapping, error) {
	entries := strings.Split(spec, ",")
	pairs := make([]fileMapping, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		from, to, ok := strings.Cut(entry, ":")
		if !ok {
			from, to = entry, entry
		}
		pairs = append(pairs, fileMapping{from: from, to: to})
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("port: -files listed no filenames")
	}
	return pairs, nil
}

// portFile copies one named file from srcDir to dstDir under its (possibly renamed) destination
// filename, rewriting import paths and package clauses in the copied bytes. It refuses to
// overwrite an existing destination file unless overwrite is true.
func portFile(srcDir, dstDir, fromName, toName string, overwrite bool) error {
	srcPath := filepath.Join(srcDir, fromName)
	dstPath := filepath.Join(dstDir, toName)

	if !overwrite {
		if _, err := os.Stat(dstPath); err == nil {
			return fmt.Errorf("port: destination %s already exists; pass -overwrite to replace it", dstPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("port: stat %s: %w", dstPath, err)
		}
	}

	body, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("port: read %s: %w", srcPath, err)
	}

	body = rewriteImports(body)
	body = rewritePackageClauses(body)

	if err := os.WriteFile(dstPath, body, 0o644); err != nil {
		return fmt.Errorf("port: write %s: %w", dstPath, err)
	}
	return nil
}

// rewriteImports applies every importRewrites entry as a literal substring replacement.
func rewriteImports(body []byte) []byte {
	s := string(body)
	for _, r := range importRewrites {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	return []byte(s)
}

// rewritePackageClauses applies every packageClauseRewrites entry, each anchored at the start of
// a line so a package-name mention inside a comment or string literal is left alone.
func rewritePackageClauses(body []byte) []byte {
	for _, r := range packageClauseRewrites {
		body = r.pattern.ReplaceAll(body, []byte(r.to))
	}
	return body
}

// countScoutenginePrefix walks dir and totals the occurrences of the `"scoutengine: ` literal
// prefix across every regular file found, matching the operator's grep -rc convention: each
// matching line contributes at most one to the count regardless of how many times the prefix
// appears on that line.
func countScoutenginePrefix(dir string) (int, error) {
	total := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("port: open %s: %w", path, err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			if scoutenginePrefixPattern.Match(scanner.Bytes()) {
				total++
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("port: scan %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
