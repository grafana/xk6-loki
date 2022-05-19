package loki

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	fake "github.com/brianvoe/gofakeit/v6"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/grafana/loki/pkg/logproto"
	json "github.com/json-iterator/go"
	"github.com/mingrammer/flog/flog"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/metrics"
)

var (
	LabelValuesFormat = []string{"apache_common", "apache_combined", "apache_error", "rfc3164", "rfc5424", "json", "logfmt"}
	LabelValuesOS     = []string{"darwin", "linux", "windows"}
)

type FakeFunc func() string

type LabelPool map[model.LabelName][]string

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

type JSONStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type JSONPushRequest struct {
	Streams []JSONStream `json:"streams"`
}

func isValidLogFormat(format string) bool {
	for _, f := range LabelValuesFormat {
		if f == format {
			return true
		}
	}
	return false
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
	req, entriesCount := b.createJSONPushRequest()
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, 0, err
	}
	return buf, entriesCount, nil
}

// createJSONPushRequest creates a JSON push payload and returns it, together with
// number of entries
func (b *Batch) createJSONPushRequest() (*JSONPushRequest, int) {
	req := JSONPushRequest{
		Streams: make([]JSONStream, 0, len(b.Streams)),
	}

	entriesCount := 0
	for _, stream := range b.Streams {
		req.Streams = append(req.Streams, JSONStream{
			Stream: labelStringToMap(stream.Labels),
			Values: entriesToValues(stream.Entries),
		})
		entriesCount += len(stream.Entries)
	}
	return &req, entriesCount
}

// labelStringToMap converts a label string used by the `Batch` struct in
// format `{label_a="value_a",label_b="value_b"}` to a map that can be used in the
// JSON payload of push requests.
func labelStringToMap(labels string) map[string]string {
	kvList := strings.Trim(labels, "{}")
	kv := strings.Split(kvList, ",")
	labelMap := make(map[string]string, len(kv))
	for _, item := range kv {
		parts := strings.Split(item, "=")
		labelMap[parts[0]] = parts[1][1 : len(parts[1])-1]
	}
	return labelMap
}

// entriesToValues converts a slice of `Entry` to a slice of string tuples that
// can be used in the JSON payload of push requests.
func entriesToValues(entries []logproto.Entry) [][]string {
	lines := make([][]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, []string{
			strconv.FormatInt(entry.Timestamp.UnixNano(), 10),
			entry.Line,
		})
	}
	return lines
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

// newBatch creates a batch with randomly generated log streams
func (c *Client) newBatch(pool LabelPool, numStreams, minBatchSize, maxBatchSize int) *Batch {
	batch := &Batch{
		Streams:   make(map[string]*logproto.Stream, numStreams),
		CreatedAt: time.Now(),
	}
	state := c.vu.State()

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	maxSizePerStream := (minBatchSize + rand.Intn(maxBatchSize-minBatchSize)) / numStreams
	lines := 0

	for i := 0; i < numStreams; i++ {
		labels := labelsFromPool(pool)
		if _, ok := labels[model.InstanceLabel]; !ok {
			labels[model.InstanceLabel] = model.LabelValue(fmt.Sprintf("vu%d.%s", state.VUID, hostname))
		}
		stream := &logproto.Stream{Labels: labels.String()}
		batch.Streams[stream.Labels] = stream

		var now time.Time
		logFmt := string(labels[model.LabelName("format")])
		if !isValidLogFormat(logFmt) {
			panic(fmt.Sprintf("%s is not a valid log format", logFmt))
		}
		var line string
		for ; batch.Bytes < maxSizePerStream; batch.Bytes += len(line) {
			now = time.Now()
			line = flog.NewLog(logFmt, now)
			stream.Entries = append(stream.Entries, logproto.Entry{
				Timestamp: now,
				Line:      line,
			})
		}
		lines += len(stream.Entries)
	}

	now := time.Now() // TODO move this in the send
	ctx := c.vu.Context()
	metrics.PushIfNotDone(ctx, state.Samples, metrics.ConnectedSamples{
		Samples: []metrics.Sample{
			{
				Metric: c.metrics.ClientUncompressedBytes,
				Tags:   &metrics.SampleTags{},
				Value:  float64(batch.Bytes),
				Time:   now,
			},
			{
				Metric: c.metrics.ClientLines,
				Tags:   &metrics.SampleTags{},
				Value:  float64(lines),
				Time:   now,
			},
		},
	})

	return batch
}

// choice returns a single label value from a list of label values
func choice(values []string) string {
	return values[rand.Intn(len(values))]
}

// labelsFromPool creates a label set from the given label value pool `p`
func labelsFromPool(p LabelPool) model.LabelSet {
	ls := make(model.LabelSet, len(p))
	for k, v := range p {
		ls[k] = model.LabelValue(choice(v))
	}
	return ls
}

// generateValues returns `n` label values generated with the `ff` gofakeit function
func generateValues(ff FakeFunc, n int) []string {
	res := make([]string, n)
	for i := 0; i < n; i++ {
		res[i] = ff()
	}
	return res
}

func defaultOrCustom(pool LabelPool, m map[string]interface{}, key string, generatorFn func(int) []string) LabelPool {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int64:
			pool[model.LabelName(key)] = generatorFn(int(v))
		case string:
			pool[model.LabelName(key)] = []string{v}
		case []interface{}:
			res := make([]string, 0, len(v))
			for i := range v {
				switch w := v[i].(type) {
				case string:
					res = append(res, string(w))
				default:
					panic(fmt.Sprintf("invalid value type: %T\n", v))
				}
			}
			pool[model.LabelName(key)] = res
		default:
			panic(fmt.Sprintf("invalid value type: %T\n", v))
		}
	}
	return pool
}

// newLabelPool creates a "pool" of values for each label name
func newLabelPool(faker *fake.Faker, in map[string]interface{}, logger logrus.FieldLogger) LabelPool {
	pool := LabelPool{
		"format": LabelValuesFormat,
		"os":     LabelValuesOS,
	}
	// format and os must always be present
	pool = defaultOrCustom(pool, in, "format", func(n int) []string { return LabelValuesFormat[:n] })
	pool = defaultOrCustom(pool, in, "os", func(n int) []string { return LabelValuesOS[:n] })
	// namespace, app, pod, language, and word are "builtin" labels
	// they are kept for backwards comptibility
	pool = defaultOrCustom(pool, in, "namespace", func(n int) []string { return generateValues(faker.BS, n) })
	pool = defaultOrCustom(pool, in, "app", func(n int) []string { return generateValues(faker.AppName, n) })
	pool = defaultOrCustom(pool, in, "pod", func(n int) []string { return generateValues(faker.BS, n) })
	pool = defaultOrCustom(pool, in, "language", func(n int) []string { return generateValues(faker.LanguageAbbreviation, n) })
	pool = defaultOrCustom(pool, in, "word", func(n int) []string {
		logger.Warn(`label "word" is deprecated`)
		return generateValues(faker.Noun, n)
	})
	// instance label can be overwritten
	pool = defaultOrCustom(pool, in, "instance", func(n int) []string { panic("instance label must not be defined with int value") })
	return pool
}
