// Package loki is the k6 extension module.
package loki

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/brianvoe/gofakeit/v6"
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
		common.Throw(rt, errors.New("loki.Config() needs to be called in the init context"))
	}

	// The default config, which we might overwrite below
	config := &Config{
		Timeout:       time.Duration(DefaultPushTimeout) * time.Millisecond,
		ProtobufRatio: DefaultProtobufRatio,
		UserAgent:     DefaultUserAgent,
		Cardinalities: map[string]int{
			"app":       5,
			"namespace": 10,
			"pod":       50,
		},
		RandSeed: 12345, // TODO: actually use something random?
	}
	if len(c.Arguments) > 1 || c.Argument(0).ExportType().Kind() == reflect.String {
		if err := r.parsePositionalConfig(c, config); err != nil {
			common.Throw(rt, fmt.Errorf("could not parse positional loki config: %w", err))
		}
	} else {
		if err := r.parseConfigObject(c.Argument(0).ToObject(rt), config); err != nil {
			common.Throw(rt, fmt.Errorf("could not parse loki config object: %w", err))
		}
	}

	initEnv.Logger.Debug(fmt.Sprintf(
		"url=%s timeout=%s protobufRatio=%f cardinalities=%v",
		&config.URL, config.Timeout, config.ProtobufRatio, config.Cardinalities,
	))

	if config.TenantID == "" {
		initEnv.Logger.Warn("Running in multi-tenant-mode. Each VU has its own X-Scope-OrgID")
	}

	return rt.ToValue(config).ToObject(rt)
}

func (r *Loki) parsePositionalConfig(c goja.ConstructorCall, config *Config) error {
	rt := r.vu.Runtime()

	urlString := c.Argument(0).String()
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid loki URL: %w", err)
	}
	config.URL = *u

	if user := u.User.Username(); user != "" {
		config.TenantID = user
	}

	if len(c.Arguments) > 1 {
		config.Timeout = time.Duration(c.Argument(1).ToInteger()) * time.Millisecond
	}

	if len(c.Arguments) > 2 {
		config.ProtobufRatio = c.Argument(2).ToFloat()
	}

	if len(c.Arguments) > 3 {
		if err := rt.ExportTo(c.Argument(3), &config.Cardinalities); err != nil {
			return fmt.Errorf("Config constructor expects map of string to integers as forth argument")
		}
	}

	if len(c.Arguments) > 4 {
		if err := rt.ExportTo(c.Argument(4), &config.Labels); err != nil {
			return fmt.Errorf("Config constructor expects Labels as fifth argument")
		}
	}

	return nil
}

func isNully(v goja.Value) bool {
	return v == nil || goja.IsUndefined(v) || goja.IsNull(v)
}

func (r *Loki) parseConfigObject(c *goja.Object, config *Config) error {
	rt := r.vu.Runtime()
	if v := c.Get("url"); !isNully(v) {
		u, err := url.Parse(v.String())
		if err != nil {
			return fmt.Errorf("invalid loki URL: %w", err)
		}
		config.URL = *u

		if user := u.User.Username(); user != "" {
			config.TenantID = user
		}
	}

	if v := c.Get("userAgent"); !isNully(v) {
		config.UserAgent = v.String()
	}

	if v := c.Get("timeout"); !isNully(v) {
		config.Timeout = time.Duration(v.ToInteger()) * time.Millisecond
	}

	if v := c.Get("tenantID"); !isNully(v) {
		// This can overwrite the TenantID, even if we set it via the URL
		config.TenantID = v.String()
	}

	if v := c.Get("cardinalities"); !isNully(v) {
		if err := rt.ExportTo(v, &config.Cardinalities); err != nil {
			return fmt.Errorf("cardinatities should be a map of string to integers: %w", err)
		}
	}

	if v := c.Get("labels"); !isNully(v) {
		if err := rt.ExportTo(v, &config.Labels); err != nil {
			return fmt.Errorf("could not parse labels: %w", err)
		}
	}

	if v := c.Get("protobufRatio"); !isNully(v) {
		config.ProtobufRatio = v.ToFloat()
	}

	if v := c.Get("randSeed"); !isNully(v) {
		config.RandSeed = v.ToInteger()
	}

	return nil
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

	faker := gofakeit.New(config.RandSeed)
	if len(config.Labels) == 0 {
		config.Labels = newLabelPool(faker, config.Cardinalities)
	}

	return rt.ToValue(&Client{
		client:  &http.Client{},
		cfg:     config,
		vu:      r.vu,
		metrics: r.metrics,
		faker:   faker,
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
