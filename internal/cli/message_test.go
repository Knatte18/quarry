// message_test.go pins rootUsageMessage's two root-resolution sentences as exact strings. Before
// this task the two sentences were asserted only by substring — via cli_test.go's Run-level
// fixtures — and carried no golden, so the internal/repopath extraction could have drifted them
// silently. This test is what makes the "same messages" claim checkable.

package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Knatte18/quarry/internal/repopath"
)

func TestRootUsageMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		flagRoot string
		cwd      string
		want     string
		wantOK   bool
	}{
		{
			name:   "no-repository-root",
			err:    fmt.Errorf("resolve: %w", repopath.ErrNoRepositoryRoot),
			cwd:    "/some/cwd",
			want:   "no repository root found above /some/cwd; pass --root",
			wantOK: true,
		},
		{
			name:     "root-not-a-directory",
			err:      fmt.Errorf("resolve: %w", repopath.ErrRootNotDirectory),
			flagRoot: "../given",
			want:     "--root is not a directory: ../given",
			wantOK:   true,
		},
		{
			name:   "unrelated-error",
			err:    errors.New("boom"),
			want:   "",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rootUsageMessage(tt.err, tt.flagRoot, tt.cwd)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("rootUsageMessage(%v, %q, %q) = %q, %v; want %q, %v",
					tt.err, tt.flagRoot, tt.cwd, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
