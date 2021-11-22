package loki

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	fake "github.com/brianvoe/gofakeit/v6"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/loki/pkg/logproto"
	json "github.com/json-iterator/go"
	"github.com/mingrammer/flog/flog"
	"github.com/prometheus/common/model"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/stats"
)

type FakeFunc func() string

type LabelPool struct {
	Format    model.LabelValues
	Namespace model.LabelValues
	App       model.LabelValues
	Pod       model.LabelValues
	Instance  model.LabelValue
}

type Batch struct {
	Streams   map[string]*logproto.Stream
	Bytes     int
	CreatedAt time.Time
}

type Entry struct {
	logproto.Entry
	TenantID string
	Labels   model.LabelSet
}

func newBatch(entries ...Entry) *Batch {
	b := &Batch{
		Streams:   map[string]*logproto.Stream{},
		Bytes:     0,
		CreatedAt: time.Now(),
	}

	// Add entries to the batch
	for _, entry := range entries {
		b.add(entry)
	}

	return b
}

// add an entry to the batch
func (b *Batch) add(entry Entry) {
	b.Bytes += len(entry.Line)

	// Append the entry to an already existing stream (if any)
	labels := entry.Labels.String()
	if stream, ok := b.Streams[labels]; ok {
		stream.Entries = append(stream.Entries, entry.Entry)
		return
	}

	// Add the entry as a new stream
	b.Streams[labels] = &logproto.Stream{
		Labels:  labels,
		Entries: []logproto.Entry{entry.Entry},
	}
}

// sizeBytes returns the current batch size in bytes
func (b *Batch) sizeBytes() int {
	return b.Bytes
}

// sizeBytesAfter returns the size of the batch after the input entry
// will be added to the batch itself
func (b *Batch) sizeBytesAfter(entry Entry) int {
	return b.Bytes + len(entry.Line)
}

// age of the batch since its creation
func (b *Batch) age() time.Duration {
	return time.Since(b.CreatedAt)
}

// encodeSnappy encodes the batch as snappy-compressed push request, and
// returns the encoded bytes and the number of encoded entries
func (b *Batch) encodeSnappy() ([]byte, int, error) {
	req, entriesCount := b.createPushRequest()
	buf, err := proto.Marshal(req)
	if err != nil {
		return nil, 0, err
	}
	buf = snappy.Encode(nil, buf)
	return buf, entriesCount, nil
}

// encodeJSON encodes the batch as JSON push request, and returns the encoded
// bytes and the number of encoded entries
func (b *Batch) encodeJSON() ([]byte, int, error) {
	req, entriesCount := b.createPushRequest()
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, 0, err
	}
	return buf, entriesCount, nil
}

// createPushRequest creates a push request and returns it, together with
// number of entries
func (b *Batch) createPushRequest() (*logproto.PushRequest, int) {
	req := logproto.PushRequest{
		Streams: make([]logproto.Stream, 0, len(b.Streams)),
	}

	entriesCount := 0
	for _, stream := range b.Streams {
		req.Streams = append(req.Streams, *stream)
		entriesCount += len(stream.Entries)
	}
	return &req, entriesCount
}

// generateEntries creates a list of log entries
func generateEntries(ctx context.Context, tenantID string, pool LabelPool, numStreams, minBatchSize, maxBatchSize int) []Entry {
	state := lib.GetState(ctx)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	entries := make([]Entry, 0)
	var labels model.LabelSet
	maxSizePerStream := (minBatchSize + rand.Intn(maxBatchSize-minBatchSize)) / numStreams
	batchSize := 0
	lines := 0

	for i := 0; i < numStreams; i++ {
		labels = labelsFromPool(pool)
		labels[model.InstanceLabel] = model.LabelValue(fmt.Sprintf("vu%d.%s", state.VUID, hostname))

		streamSize := 0
		for streamSize < maxSizePerStream {
			line := flog.NewLog(string(labels[model.LabelName("format")]), time.Now())
			entry := Entry{
				Entry: logproto.Entry{
					Timestamp: time.Now(),
					Line:      line,
				},
				TenantID: tenantID,
				Labels:   labels,
			}
			entries = append(entries, entry)
			streamSize += len(line)
			lines++
		}
		batchSize += streamSize
	}

	now := time.Now()
	stats.PushIfNotDone(ctx, state.Samples, stats.ConnectedSamples{
		Samples: []stats.Sample{
			{
				Metric: ClientUncompressedBytes,
				Tags:   &stats.SampleTags{},
				Value:  float64(batchSize),
				Time:   now,
			},
			{
				Metric: ClientLines,
				Tags:   &stats.SampleTags{},
				Value:  float64(lines),
				Time:   now,
			},
		},
	})
	return entries
}

// choice returns a single label value from a list of label values
func choice(values model.LabelValues) model.LabelValue {
	return values[rand.Intn(values.Len())]
}

// labelsFromPool creates a label set from the given label value pool `p`
func labelsFromPool(p LabelPool) model.LabelSet {
	return model.LabelSet{
		"app":       choice(p.App),
		"format":    choice(p.Format),
		"namespace": choice(p.Namespace),
		"pod":       choice(p.Pod),
	}
}

// generateValues returns `n` label values generated with the `ff` gofakeit function
func generateValues(ff FakeFunc, n int) model.LabelValues {
	res := make(model.LabelValues, n)
	for i := 0; i < n; i++ {
		res[i] = model.LabelValue(ff())
	}
	return res
}

// newLabelPool creates a "pool" of values for each label name
// The amount of different values per label name is not configurable yet.
func newLabelPool(faker *fake.Faker) LabelPool {
	return LabelPool{
		Format:    model.LabelValues{"apache_common", "json"}, // needs to match the available flog formats
		Namespace: generateValues(faker.BS, 10),
		App:       generateValues(faker.BS, 5),
		Pod:       generateValues(faker.BS, 100),
	}
}
