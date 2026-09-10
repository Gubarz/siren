package actions

import (
	"testing"
	"time"
)

// timeoutMs comes straight from script source and is scaled into a
// time.Duration, so a large value overflows into a nonsense deadline.
func TestClampHTTPTimeoutBoundsTheCallerValue(t *testing.T) {
	cases := []struct {
		in   int64
		want time.Duration
	}{
		{0, time.Millisecond},
		{-5, time.Millisecond},
		{1500, 1500 * time.Millisecond},
		{int64(scriptHTTPDefaultTimeout / time.Millisecond), scriptHTTPDefaultTimeout},
		{int64(1) << 62, scriptHTTPMaxTimeout},
	}
	for _, c := range cases {
		if got := clampHTTPTimeout(c.in); got != c.want {
			t.Fatalf("clampHTTPTimeout(%d) = %s, want %s", c.in, got, c.want)
		}
	}
}
