// Package loki is the k6 extension module.
package loki

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"github.com/dop251/goja"
	"github.com/prometheus/common/model"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

var (
	DefaultProtobufRatio = 0.9
	DefaultPushTimeout   = 10000
	DefaultUserAgent     = "xk6-loki/0.0.1"
)

// init registers the Go module as Javascript module for k6
// The module can be imported like so:
// ```js
// import remote from 'k6/x/loki';
// ```
//
// See examples/simple.js for a full example how to use the xk6-loki extension.
func init() {
	modules.Register("k6/x/loki", new(LokiRoot))
}

var _ modules.Module = &LokiRoot{}

type lokiMetrics struct {
	ClientUncompressedBytes  *metrics.Metric
	ClientLines              *metrics.Metric
	BytesProcessedTotal      *metrics.Metric
	BytesProcessedPerSeconds *metrics.Metric
	LinesProcessedTotal      *metrics.Metric
	LinesProcessedPerSeconds *metrics.Metric
}

// LokiRoot is the root module
type LokiRoot struct{}

func (*LokiRoot) NewModuleInstance(vu modules.VU) modules.Instance {
	m, err := registerMetrics(vu)
	if err != nil {
		common.Throw(vu.Runtime(), err)
	}

	return &Loki{vu: vu, metrics: m}
}

func registerMetrics(vu modules.VU) (lokiMetrics, error) {
	var err error
	registry := vu.InitEnv().Registry
	m := lokiMetrics{}

	m.ClientUncompressedBytes, err = registry.NewMetric("loki_client_uncompressed_bytes", metrics.Counter, metrics.Data)
	if err != nil {
		return m, err
	}

	m.ClientLines, err = registry.NewMetric("loki_client_lines", metrics.Counter, metrics.Default)
	if err != nil {
		return m, err
	}

	m.BytesProcessedTotal, err = registry.NewMetric("loki_bytes_processed_total", metrics.Counter, metrics.Data)
	if err != nil {
		return m, err
	}

	m.BytesProcessedPerSeconds, err = registry.NewMetric("loki_bytes_processed_per_second", metrics.Trend, metrics.Data)
	if err != nil {
		return m, err
	}

	m.LinesProcessedTotal, err = registry.NewMetric("loki_lines_processed_total", metrics.Counter, metrics.Default)
	if err != nil {
		return m, err
	}

	m.LinesProcessedPerSeconds, err = registry.NewMetric("loki_lines_processed_per_second", metrics.Trend, metrics.Default)
	if err != nil {
		return m, err
	}

	return m, nil
}

// Loki is the k6 extension that can be imported in the Javascript test file.
type Loki struct {
	vu      modules.VU
	metrics lokiMetrics
}

func (r *Loki) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Config":    r.config,
			"Client":    r.client,
			"Labels":    r.createLabels,
			"getLables": r.getLabels,
		},
	}
}

// config provides a constructor interface for the Config for the Javascript runtime
// ```js
// const cfg = new loki.Config(url);
// ```
func (r *Loki) config(c goja.ConstructorCall) *goja.Object {
	initEnv := r.vu.InitEnv()
	rt := r.vu.Runtime()

	if initEnv == nil {
		common.Throw(rt, errors.New("Client constructor needs to be called in the init context"))
	}
	urlString := c.Argument(0).String()

	timeoutMs := int(c.Argument(1).ToInteger())
	if timeoutMs == 0 {
		timeoutMs = DefaultPushTimeout
	}

	protobufRatio := c.Argument(2).ToFloat()
	if protobufRatio == 0 {
		protobufRatio = DefaultProtobufRatio
	}

	var cardinalities map[string]int
	if err := rt.ExportTo(c.Argument(3), &cardinalities); err != nil {
		common.Throw(rt, fmt.Errorf("Config constructor expects map of string to integers as forth argument"))
	}

	var labels LabelPool
	if err := rt.ExportTo(c.Argument(4), &labels); err != nil {
		common.Throw(rt, fmt.Errorf("Config constructor expects Labels as fifth argument"))
	}

	var tenantIds []string
	if err := rt.ExportTo(c.Argument(5), &tenantIds); err != nil {
		common.Throw(rt, fmt.Errorf("Config constructor expects an array of tenant IDs as the sixth argument"))
	}

	initEnv.Logger.Debug(fmt.Sprintf("url=%s timeoutMs=%d protobufRatio=%f cardinalities=%v tenantIds=%v", urlString, timeoutMs, protobufRatio, cardinalities, tenantIds))

	faker := gofakeit.New(12345)

	u, err := url.Parse(urlString)
	if err != nil {
		panic(err)
	}

	if len(tenantIds) == 0 {
		if u.User.Username() != "" {
			tenantIds = []string{u.User.Username()}
		} else {
			initEnv.Logger.Warn("Running in multi-tenant-mode. Each VU has its own X-Scope-OrgID")
		}
	}

	if len(labels) == 0 {
		if len(cardinalities) == 0 {
			cardinalities = map[string]int{
				"app":       5,
				"namespace": 10,
				"pod":       50,
			}
		}
		labels = newLabelPool(faker, cardinalities)
	}

	config := &Config{
		URL:           *u,
		UserAgent:     DefaultUserAgent,
		TenantIDs:     tenantIds,
		Timeout:       time.Duration(timeoutMs) * time.Millisecond,
		Labels:        labels,
		ProtobufRatio: protobufRatio,
	}

	return rt.ToValue(config).ToObject(rt)
}

// client provides a constructor interface for the Config for the Javascript runtime
// ```js
// const client = new loki.Client(cfg);
// ```
func (r *Loki) client(c goja.ConstructorCall) *goja.Object {
	rt := r.vu.Runtime()
	config, ok := c.Argument(0).Export().(*Config)
	if !ok {
		common.Throw(rt, fmt.Errorf("Client constructor expect Config as it's argument"))
	}
	return rt.ToValue(&Client{
		client:  &http.Client{},
		cfg:     config,
		vu:      r.vu,
		metrics: r.metrics,
	}).ToObject(rt)
}

func (r *Loki) createLabels(c goja.ConstructorCall) *goja.Object {
	rt := r.vu.Runtime()
	var labels map[string][]string
	if err := rt.ExportTo(c.Argument(0), &labels); err != nil {
		common.Throw(rt, fmt.Errorf("Labels constructor expects map of string to string array argument"))
	}
	pool := make(LabelPool, len(labels))
	for k, v := range labels {
		pool[model.LabelName(k)] = v
	}
	return rt.ToValue(&pool).ToObject(rt)
}

func (r *Loki) getLabels(config Config) interface{} {
	return &config.Labels
}
