package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/grafana/loki/pkg/logql/stats"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/netext/httpext"
	"go.k6.io/k6/metrics"
)

const (
	ContentTypeProtobuf   = "application/x-protobuf"
	ContentTypeJSON       = "application/json"
	ContentEncodingSnappy = "snappy"
	ContentEncodingGzip   = "gzip"

	TenantPrefix = "xk6-tenant"
)

type Client struct {
	vu      modules.VU
	client  *http.Client
	logger  logrus.FieldLogger
	cfg     *Config
	metrics lokiMetrics
}

type Config struct {
	URL           url.URL
	UserAgent     string
	Timeout       time.Duration
	TenantID      string
	Labels        LabelPool
	ProtobufRatio float64
}

func (c *Client) InstantQuery(logQuery string, limit int) (httpext.Response, error) {
	q := &Query{
		Type:        InstantQuery,
		QueryString: logQuery,
		Limit:       limit,
	}
	q.SetInstant(time.Now())
	response, err := c.sendQuery(q)
	if err == nil && IsSuccessfulResponse(response.Status) {
		err = c.reportMetricsFromStats(response, InstantQuery)
	}
	return response, err
}

func (c *Client) InstantQueryAt(logQuery string, limit int, instant int64) (httpext.Response, error) {
	q := &Query{
		Type:        InstantQuery,
		QueryString: logQuery,
		Limit:       limit,
	}

	u := time.Unix(instant, 0)
	q.SetInstant(u)
	response, err := c.sendQuery(q)
	if err == nil && IsSuccessfulResponse(response.Status) {
		err = c.reportMetricsFromStats(response, InstantQuery)
	}
	return response, err
}

func (c *Client) RangeQuery(logQuery string, duration string, limit int) (httpext.Response, error) {
	now := time.Now()
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return httpext.Response{}, err
	}
	q := &Query{
		Type:        RangeQuery,
		QueryString: logQuery,
		Start:       now.Add(-dur),
		End:         now,
		Limit:       limit,
	}
	response, err := c.sendQuery(q)
	if err == nil && IsSuccessfulResponse(response.Status) {
		err = c.reportMetricsFromStats(response, RangeQuery)
	}
	return response, err
}

func (c *Client) LabelsQuery(duration string) (httpext.Response, error) {
	now := time.Now()
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return httpext.Response{}, err
	}
	q := &Query{
		Type:  LabelsQuery,
		Start: now.Add(-dur),
		End:   now,
	}
	return c.sendQuery(q)
}

func (c *Client) LabelValuesQuery(label string, duration string) (httpext.Response, error) {
	now := time.Now()
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return httpext.Response{}, err
	}
	q := &Query{
		Type:       LabelValuesQuery,
		Start:      now.Add(-dur),
		End:        now,
		PathParams: []interface{}{label},
	}
	return c.sendQuery(q)
}

func (c *Client) SeriesQuery(matchers string, duration string) (httpext.Response, error) {
	now := time.Now()
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return httpext.Response{}, err
	}
	q := &Query{
		Type:        SeriesQuery,
		QueryString: matchers,
		Start:       now.Add(-dur),
		End:         now,
	}
	return c.sendQuery(q)
}

// buildURL concatinates a URL `http://foo/bar` with a path `/buzz` and a query string `?query=...`.
func buildURL(u, p, qs string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, p)
	url.RawQuery = qs
	return url.String(), nil
}

func (c *Client) sendQuery(q *Query) (httpext.Response, error) {
	state := c.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}

	httpResp := httpext.NewResponse()
	path := q.Endpoint()

	urlString, err := buildURL(c.cfg.URL.String(), path, q.Values().Encode())
	if err != nil {
		return *httpext.NewResponse(), err
	}

	r, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return *httpResp, err
	}

	r.Header.Set("User-Agent", c.cfg.UserAgent)
	r.Header.Set("Accept", ContentTypeJSON)
	if c.cfg.TenantID != "" {
		r.Header.Set("X-Scope-OrgID", c.cfg.TenantID)
	} else {
		r.Header.Set("X-Scope-OrgID", fmt.Sprintf("%s-%d", TenantPrefix, state.VUID))
	}

	url, _ := httpext.NewURL(urlString, path)
	response, err := httpext.MakeRequest(c.vu.Context(), state, &httpext.ParsedHTTPRequest{
		URL:              &url,
		Req:              r,
		Throw:            state.Options.Throw.Bool,
		Redirects:        state.Options.MaxRedirects,
		Timeout:          c.cfg.Timeout,
		ResponseCallback: IsSuccessfulResponse,
	})
	if err != nil {
		return *httpResp, err
	}

	return *response, err
}

func (c *Client) Push() (httpext.Response, error) {
	// 5 streams per batch
	// batch size between 800KB and 1MB
	return c.PushParameterized(5, 800*1024, 1024*1024)
}

// PushParametrized is deprecated in favor or PushParameterized
func (c *Client) PushParametrized(streams, minBatchSize, maxBatchSize int) (httpext.Response, error) {
	if state := c.vu.State(); state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	} else {
		state.Logger.Warn("method pushParametrized() is deprecated and will be removed in future releases; please use pushParameterized() instead")
	}
	return c.PushParameterized(streams, minBatchSize, maxBatchSize)
}

func (c *Client) PushParameterized(streams, minBatchSize, maxBatchSize int) (httpext.Response, error) {
	state := c.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}

	batch := c.newBatch(c.cfg.Labels, streams, minBatchSize, maxBatchSize)
	return c.pushBatch(batch)
}

func (c *Client) pushBatch(batch *Batch) (httpext.Response, error) {
	state := c.vu.State()
	if state == nil {
		return *httpext.NewResponse(), errors.New("state is nil")
	}

	var buf []byte
	var err error

	// Use snappy encoded Protobuf for 90% of the requests
	// Use JSON encoding for 10% of the requests
	encodeSnappy := rand.Float64() < c.cfg.ProtobufRatio
	if encodeSnappy {
		buf, _, err = batch.encodeSnappy()
	} else {
		buf, _, err = batch.encodeJSON()
	}
	if err != nil {
		return *httpext.NewResponse(), errors.Wrap(err, "failed to encode payload")
	}

	res, err := c.send(state, buf, encodeSnappy)
	if err != nil {
		return *httpext.NewResponse(), errors.Wrap(err, "push request failed")
	}
	res.Request.Body = ""

	return res, nil
}

func (c *Client) send(state *lib.State, buf []byte, useProtobuf bool) (httpext.Response, error) {
	httpResp := httpext.NewResponse()
	path := "/loki/api/v1/push"
	r, err := http.NewRequest(http.MethodPost, c.cfg.URL.String()+path, nil)
	if err != nil {
		return *httpResp, err
	}

	r.Header.Set("User-Agent", c.cfg.UserAgent)
	r.Header.Set("Accept", ContentTypeJSON)
	if c.cfg.TenantID != "" {
		r.Header.Set("X-Scope-OrgID", c.cfg.TenantID)
	} else {
		r.Header.Set("X-Scope-OrgID", fmt.Sprintf("%s-%d", TenantPrefix, state.VUID))
	}
	if useProtobuf {
		r.Header.Set("Content-Type", ContentTypeProtobuf)
		r.Header.Add("Content-Encoding", ContentEncodingSnappy)
	} else {
		r.Header.Set("Content-Type", ContentTypeJSON)
	}

	url, _ := httpext.NewURL(c.cfg.URL.String()+path, path)
	response, err := httpext.MakeRequest(c.vu.Context(), state, &httpext.ParsedHTTPRequest{
		URL:              &url,
		Req:              r,
		Body:             bytes.NewBuffer(buf),
		Throw:            state.Options.Throw.Bool,
		Redirects:        state.Options.MaxRedirects,
		Timeout:          c.cfg.Timeout,
		ResponseCallback: IsSuccessfulResponse,
	})
	if err != nil {
		return *httpResp, err
	}

	return *response, err
}

func IsSuccessfulResponse(n int) bool {
	// report all 2xx respones as successful requests
	return n/100 == 2
}

type responseWithStats struct {
	Data struct {
		stats stats.Result
	}
}

func (c *Client) reportMetricsFromStats(response httpext.Response, queryType QueryType) error {
	responseBody, ok := response.Body.(string)
	if !ok {
		return errors.New("response body is not a string")
	}
	responseWithStats := responseWithStats{}
	err := json.Unmarshal([]byte(responseBody), &responseWithStats)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling response body to response with stats")
	}
	now := time.Now()
	tags := metrics.NewSampleTags(map[string]string{"endpoint": queryType.Endpoint()})
	ctx := c.vu.Context()
	metrics.PushIfNotDone(ctx, c.vu.State().Samples, metrics.ConnectedSamples{
		Samples: []metrics.Sample{
			{
				Metric: c.metrics.BytesProcessedTotal,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.stats.Summary.TotalBytesProcessed),
				Time:   now,
			},
			{
				Metric: c.metrics.BytesProcessedPerSeconds,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.stats.Summary.BytesProcessedPerSecond),
				Time:   now,
			},
			{
				Metric: c.metrics.LinesProcessedTotal,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.stats.Summary.TotalLinesProcessed),
				Time:   now,
			},
			{
				Metric: c.metrics.LinesProcessedPerSeconds,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.stats.Summary.LinesProcessedPerSecond),
				Time:   now,
			},
		},
	})
	return nil
}
