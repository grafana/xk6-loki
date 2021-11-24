package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/stats"
)

var (
	DefaultProtobufRatio = 0.9
	DefaultPushTimeout   = 10000
	DefaultUserAgent     = "xk6-loki/0.0.1"

	ClientUncompressedBytes = stats.New("loki_client_uncompressed_bytes", stats.Counter, stats.Data)
	ClientLines             = stats.New("loki_client_lines", stats.Counter, stats.Default)
)

// init registers the Go module as Javascript module for k6
// The module can be imported like so:
// ```js
// import remote from 'k6/x/loki';
// ```
//
// See examples/simple.js for a full example how to use the xk6-loki extension.
func init() {
	modules.Register("k6/x/loki", new(Loki))
}

// Loki is the k6 extension that can be imported in the Javascript test file.
type Loki struct{}

// XConfig provides a constructor interface for the Config for the Javascript runtime
// ```js
// const cfg = new loki.Config(url);
// ```
func (r *Loki) XConfig(ctxPtr *context.Context, urlString string, timeoutMs int, protobufRatio float64, cardinalities map[string]int) interface{} {
	if timeoutMs == 0 {
		timeoutMs = DefaultPushTimeout
	}
	if protobufRatio == 0 {
		protobufRatio = DefaultProtobufRatio
	}
	if len(cardinalities) == 0 {
		cardinalities = map[string]int{
			"app":       5,
			"namespace": 10,
			"pod":       100,
		}
	}

	logger := common.GetInitEnv(*ctxPtr).Logger
	logger.Debug(fmt.Sprintf("url=%s timeoutMs=%d protobufRatio=%f cardinalities=%v", urlString, timeoutMs, protobufRatio, cardinalities))

	faker := gofakeit.New(12345)

	u, err := url.Parse(urlString)
	if err != nil {
		panic(err)
	}
	rt := common.GetRuntime(*ctxPtr)

	if timeoutMs == 0 {
		timeoutMs = DefaultPushTimeout
	}
	if protobufRatio == 0.0 {
		protobufRatio = DefaultProtobufRatio
	}

	if u.User.Username() == "" {
		logger := common.GetInitEnv(*ctxPtr).Logger
		logger.Warn("Running in multi-tenant-mode. Each VU has its own X-Scope-OrgID")
	}

	return common.Bind(
		rt,
		&Config{
			URL:           *u,
			UserAgent:     DefaultUserAgent,
			TenantID:      u.User.Username(),
			Timeout:       time.Duration(timeoutMs) * time.Millisecond,
			Labels:        newLabelPool(faker, cardinalities),
			ProtobufRatio: protobufRatio,
		},
		ctxPtr)
}

// XClient provides a constructor interface for the Config for the Javascript runtime
// ```js
// const client = new loki.Client(cfg);
// ```
func (r *Loki) XClient(ctxPtr *context.Context, config Config) interface{} {
	rt := common.GetRuntime(*ctxPtr)
	return common.Bind(rt, &Client{
		client: &http.Client{},
		cfg:    &config,
	}, ctxPtr)
}

func (r *Loki) GetLabels(ctxPtr *context.Context, config Config) interface{} {
	rt := common.GetRuntime(*ctxPtr)
	return common.Bind(rt, &config.Labels, ctxPtr)
}
