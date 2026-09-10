// Package mapslice clones maps of slices for the localstate stores. Callers
// receive independent slices and cannot mutate stored state.
package mapslice

// Clone deep-copies src. Empty and nil source slices become empty non-nil
// slices, matching the JSON shape callers previously produced by hand.
func Clone[K comparable, V any](src map[K][]V) map[K][]V {
	out := make(map[K][]V, len(src))
	for k, list := range src {
		clone := make([]V, len(list))
		copy(clone, list)
		out[k] = clone
	}
	return out
}
