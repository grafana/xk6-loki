package loki

import (
	"context"
	"testing"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/stats"
)

func BenchmarkGenerateEntries(b *testing.B) {
	samples := make(chan stats.SampleContainer)
	state := &lib.State{
		Samples: samples,
		VUID:    15,
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = lib.WithState(ctx, state)

	defer cancel()
	defer close(samples)
	go func() { // this is so that we read the send samples
		for range samples {
		}
	}()
	faker := gofakeit.New(12345)
	cardinalities := map[string]int{
		"app":       5,
		"namespace": 10,
		"pod":       100,
	}
	streams, minBatchSize, maxBatchSize := 5, 500, 1000
	labels := newLabelPool(faker, cardinalities)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = generateEntries(ctx, "12", labels, streams, minBatchSize, maxBatchSize)
	}
}
