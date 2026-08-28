// buildtags.go implements NormalizeBuildTags, the single tag-set normalization function shared by
// internal/cli (which calls it with one raw --build-tags/$QUARRY_BUILD_TAGS string) and the engine
// (which calls it again, defensively, on an already-split []string).

package registry

import (
	"sort"
	"strings"
)

// NormalizeBuildTags splits every element of tags on ",", trims surrounding whitespace from each
// resulting piece, drops empty pieces, removes duplicates, and returns the remainder sorted
// lexicographically.
// A call whose arguments all normalize away returns nil, not an empty non-nil slice.
//
// The variadic shape lets both callers share one function without a second entry point:
// internal/cli calls NormalizeBuildTags(rawFlagOrEnvValue) with the single raw
// --build-tags/$QUARRY_BUILD_TAGS string, and the engine calls
// NormalizeBuildTags(alreadySplitTags...) with an already-split []string expanded via "...". The
// engine's call re-normalizes defensively what the CLI already normalized, so NormalizeBuildTags
// must be idempotent: NormalizeBuildTags(NormalizeBuildTags("b,a")...) equals
// NormalizeBuildTags("b,a").
func NormalizeBuildTags(tags ...string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, arg := range tags {
		for _, piece := range strings.Split(arg, ",") {
			tag := strings.TrimSpace(piece)
			if tag == "" || seen[tag] {
				continue
			}
			seen[tag] = true
			result = append(result, tag)
		}
	}
	sort.Strings(result)
	return result
}
