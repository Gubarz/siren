package beacons

import (
	"context"
	"errors"
	"testing"
)

func TestShouldCancelPendingBeaconTask(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "local cancellation", err: context.Canceled, want: false},
		{name: "wrapped local cancellation", err: errors.Join(errors.New("wait failed"), context.Canceled), want: false},
		{name: "timeout", err: context.DeadlineExceeded, want: true},
		{name: "rpc error", err: errors.New("rpc failed"), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldCancelPendingBeaconTask(tc.err); got != tc.want {
				t.Fatalf("shouldCancelPendingBeaconTask(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
