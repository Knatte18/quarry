// main.go is the ladderbench binary's entry point: build and run the cobra command tree Command()
// assembles, exiting non-zero when execution returns an error.

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Command().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
