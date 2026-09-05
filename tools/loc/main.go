// Command loc counts lines of Go source in the repo, split into test and
// non-test files.
package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type counts struct {
	files int
	lines int
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	perDir := map[string]*[2]counts{} // index 0 = non-test, 1 = test
	var total [2]counts

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		n, err := countLines(path)
		if err != nil {
			return err
		}

		isTest := strings.HasSuffix(path, "_test.go")
		idx := 0
		if isTest {
			idx = 1
		}

		dir := filepath.Dir(path)
		c, ok := perDir[dir]
		if !ok {
			c = &[2]counts{}
			perDir[dir] = c
		}
		c[idx].files++
		c[idx].lines += n
		total[idx].files++
		total[idx].lines += n

		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "loc:", err)
		os.Exit(1)
	}

	dirs := make([]string, 0, len(perDir))
	for dir := range perDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	fmt.Printf("%-60s %10s %10s %10s %10s\n", "dir", "non-test", "files", "test", "files")
	for _, dir := range dirs {
		c := perDir[dir]
		fmt.Printf("%-60s %10d %10d %10d %10d\n", dir, c[0].lines, c[0].files, c[1].lines, c[1].files)
	}

	fmt.Println(strings.Repeat("-", 104))
	fmt.Printf("%-60s %10d %10d %10d %10d\n", "TOTAL", total[0].lines, total[0].files, total[1].lines, total[1].files)
	fmt.Printf("\nnon-test: %d lines across %d files\n", total[0].lines, total[0].files)
	fmt.Printf("test:     %d lines across %d files\n", total[1].lines, total[1].files)
	fmt.Printf("all:      %d lines across %d files\n", total[0].lines+total[1].lines, total[0].files+total[1].files)
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}
