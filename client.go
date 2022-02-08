package loki

import (
	"bytes"
	"context"
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
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/netext/httpext"
	k6_stats "go.k6.io/k6/stats"
)

const (
	ContentTypeProtobuf   = "application/x-protobuf"
	ContentTypeJSON       = "application/json"
	ContentEncodingSnappy = "snappy"
	ContentEncodingGzip   = "gzip"

	TenantPrefix = "xk6-tenant"
)

var (
	BytesProcessedTotal      = k6_stats.New("loki_bytes_processed_total", k6_stats.Counter, k6_stats.Data)
	BytesProcessedPerSeconds = k6_stats.New("loki_bytes_processed_per_second", k6_stats.Trend, k6_stats.Data)
	LinesProcessedTotal      = k6_stats.New("loki_lines_processed_total", k6_stats.Counter, k6_stats.Default)
	LinesProcessedPerSeconds = k6_stats.New("loki_lines_processed_per_second", k6_stats.Trend, k6_stats.Default)
)

type Client struct {
	client *http.Client
	logger logrus.FieldLogger
	cfg    *Config
}

type Config struct {
	URL           url.URL
	UserAgent     string
	Timeout       time.Duration
	TenantID      string
	Labels        LabelPool
	ProtobufRatio float64
}

func (c *Client) InstantQuery(ctx context.Context, logQuery string, limit int) (httpext.Response, error) {
	q := &Query{
		Type:        InstantQuery,
		QueryString: logQuery,
		Limit:       limit,
	}
	q.SetInstant(time.Now())
	response, err := c.sendQuery(ctx, q)
	if err == nil && IsSuccessfulResponse(response.Status) {
		err = reportMetricsFromStats(ctx, response, InstantQuery)
	}
	return response, err
}

func (c *Client) RangeQuery(ctx context.Context, logQuery string, duration string, limit int) (httpext.Response, error) {
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
	response, err := c.sendQuery(ctx, q)
	if err == nil && IsSuccessfulResponse(response.Status) {
		err = reportMetricsFromStats(ctx, response, RangeQuery)
	}
	return response, err
}

func (c *Client) LabelsQuery(ctx context.Context, duration string) (httpext.Response, error) {
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
	return c.sendQuery(ctx, q)
}

func (c *Client) LabelValuesQuery(ctx context.Context, label string, duration string) (httpext.Response, error) {
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
	return c.sendQuery(ctx, q)
}

func (c *Client) SeriesQuery(ctx context.Context, matchers string, duration string) (httpext.Response, error) {
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
	return c.sendQuery(ctx, q)
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

func (c *Client) sendQuery(ctx context.Context, q *Query) (httpext.Response, error) {
	state := lib.GetState(ctx)
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
	response, err := httpext.MakeRequest(ctx, state, &httpext.ParsedHTTPRequest{
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

func (c *Client) Push(ctx context.Context) (httpext.Response, error) {
	// 5 streams per batch
	// batch size between 800KB and 1MB
	return c.PushParameterized(ctx, 5, 800*1024, 1024*1024)
}

// PushParametrized is deprecated in favor or PushParameterized
func (c *Client) PushParametrized(ctx context.Context, streams, minBatchSize, maxBatchSize int) (httpext.Response, error) {
	c.logger.Warn("method pushParametrized() is deprecated and will be removed in future releases; please use pushParameterized() instead")
	return c.PushParameterized(ctx, streams, minBatchSize, maxBatchSize)
}

func (c *Client) PushParameterized(ctx context.Context, streams, minBatchSize, maxBatchSize int) (httpext.Response, error) {
	batch := newBatch(ctx, c.cfg.Labels, streams, minBatchSize, maxBatchSize)
	return c.pushBatch(ctx, batch)
}

func (c *Client) pushBatch(ctx context.Context, batch *Batch) (httpext.Response, error) {
	state := lib.GetState(ctx)
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

	res, err := c.send(ctx, state, buf, encodeSnappy)
	if err != nil {
		return *httpext.NewResponse(), errors.Wrap(err, "push request failed")
	}
	res.Request.Body = ""

	return res, nil
}

func (c *Client) send(ctx context.Context, state *lib.State, buf []byte, useProtobuf bool) (httpext.Response, error) {
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
	response, err := httpext.MakeRequest(ctx, state, &httpext.ParsedHTTPRequest{
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
		Stats stats.Result
	}
}

func reportMetricsFromStats(ctx context.Context, response httpext.Response, queryType QueryType) error {
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
	tags := k6_stats.NewSampleTags(map[string]string{"endpoint": queryType.Endpoint()})
	k6_stats.PushIfNotDone(ctx, lib.GetState(ctx).Samples, k6_stats.ConnectedSamples{
		Samples: []k6_stats.Sample{
			{
				Metric: BytesProcessedTotal,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.Stats.Summary.TotalBytesProcessed),
				Time:   now,
			},
			{
				Metric: BytesProcessedPerSeconds,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.Stats.Summary.BytesProcessedPerSecond),
				Time:   now,
			},
			{
				Metric: LinesProcessedTotal,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.Stats.Summary.TotalLinesProcessed),
				Time:   now,
			},
			{
				Metric: LinesProcessedPerSeconds,
				Tags:   tags,
				Value:  float64(responseWithStats.Data.Stats.Summary.LinesProcessedPerSecond),
				Time:   now,
			},
		},
	})
	return nil
}
