package loki

import (
	"context"
	"testing"

	"go.k6.io/k6/js/modulestest"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/metrics"
)

func TestPushParameterizedInvalidArguments(t *testing.T) {
	samples := make(chan metrics.SampleContainer, 16)
	defer close(samples)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := &Client{
		vu: &modulestest.VU{
			CtxField:   ctx,
			StateField: &lib.State{Samples: samples, VUID: 1},
		},
	}

	tests := map[string]struct {
		streams, minBatchSize, maxBatchSize int
	}{
		"zero streams":         {0, 1000, 1000},
		"negative streams":     {-1, 1000, 1000},
		"negative min size":    {1, -1, 1000},
		"min greater than max": {1, 2000, 1000},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PushParameterized panicked: %v", r)
				}
			}()
			if _, err := c.PushParameterized(tc.streams, tc.minBatchSize, tc.maxBatchSize); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}
