// buildtags_test.go table-drives NormalizeBuildTags against the normalization vocabulary its doc
// comment promises: order-independence, whitespace/empty-element handling, deduplication, the
// multi-argument form, and idempotence.

package registry

import "testing"

func TestNormalizeBuildTags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "b,a normalizes",
			args: []string{"b,a"},
			want: []string{"a", "b"},
		},
		{
			name: "a,b normalizes identically to b,a",
			args: []string{"a,b"},
			want: []string{"a", "b"},
		},
		{
			name: "empty string normalizes to nil",
			args: []string{""},
			want: nil,
		},
		{
			name: "lone comma normalizes to nil",
			args: []string{","},
			want: nil,
		},
		{
			name: "whitespace-padded comma normalizes to nil",
			args: []string{" , "},
			want: nil,
		},
		{
			name: "empty elements and padding are dropped",
			args: []string{"a,,b, a "},
			want: []string{"a", "b"},
		},
		{
			name: "multi-argument form matches single-string form",
			args: []string{"b", "a"},
			want: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBuildTags(tt.args...)
			if !stringSlicesEqual(got, tt.want) {
				t.Errorf("NormalizeBuildTags(%v) = %v; want %v", tt.args, got, tt.want)
			}

			// Idempotence: re-normalizing an already-normalized result must
			// be a no-op, since the engine re-normalizes defensively what
			// the CLI already normalized.
			again := NormalizeBuildTags(got...)
			if !stringSlicesEqual(again, got) {
				t.Errorf("NormalizeBuildTags(NormalizeBuildTags(%v)...) = %v; want %v (idempotence)", tt.args, again, got)
			}
		})
	}
}

func TestNormalizeBuildTags_MultiArgumentMatchesSingleString(t *testing.T) {
	multi := NormalizeBuildTags("b", "a")
	single := NormalizeBuildTags("b,a")
	if !stringSlicesEqual(multi, single) {
		t.Errorf("NormalizeBuildTags(\"b\", \"a\") = %v; want %v (equal to NormalizeBuildTags(\"b,a\"))", multi, single)
	}
}
