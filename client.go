package loki

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/pkg/errors"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/netext/httpext"
)

const (
	ContentTypeProtobuf   = "application/x-protobuf"
	ContentTypeJSON       = "application/json"
	ContentEncodingSnappy = "snappy"
	ContentEncodingGzip   = "gzip"

	TenantPrefix = "xk6-tenant"
)

type Client struct {
	client *http.Client
	logger log.Logger
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
	return c.sendQuery(ctx, q)
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
	return c.sendQuery(ctx, q)
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
		return *httpext.NewResponse(ctx), errors.New("state is nil")
	}

	httpResp := httpext.NewResponse(ctx)
	path := q.Endpoint()

	urlString, err := buildURL(c.cfg.URL.String(), path, q.Values().Encode())
	if err != nil {
		return *httpext.NewResponse(ctx), err
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
	response, err := httpext.MakeRequest(ctx, &httpext.ParsedHTTPRequest{
		URL:              &url,
		Req:              r,
		Throw:            state.Options.Throw.Bool,
		Redirects:        state.Options.MaxRedirects,
		Timeout:          c.cfg.Timeout,
		ResponseCallback: ResponseCallback,
	})
	if err != nil {
		return *httpResp, err
	}

	return *response, err
}

func (c *Client) Push(ctx context.Context) (httpext.Response, error) {
	return c.PushParametrized(ctx, 5, 500, 1000)
}

func (c *Client) PushParametrized(ctx context.Context, streams, minBatchSize, maxBatchSize int) (httpext.Response, error) {
	entries := generateEntries(ctx, c.cfg.TenantID, c.cfg.Labels, streams, minBatchSize, maxBatchSize)
	batch := newBatch(entries...)
	return c.pushBatch(ctx, batch)
}

func (c *Client) pushBatch(ctx context.Context, batch *Batch) (httpext.Response, error) {
	state := lib.GetState(ctx)
	if state == nil {
		return *httpext.NewResponse(ctx), errors.New("state is nil")
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
		return *httpext.NewResponse(ctx), errors.Wrap(err, "failed to encode payload")
	}

	res, err := c.send(ctx, state, buf, encodeSnappy)
	if err != nil {
		return *httpext.NewResponse(ctx), errors.Wrap(err, "push request failed")
	}
	res.Request.Body = ""

	return res, nil
}

func (c *Client) send(ctx context.Context, state *lib.State, buf []byte, useProtobuf bool) (httpext.Response, error) {
	httpResp := httpext.NewResponse(ctx)
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
	response, err := httpext.MakeRequest(ctx, &httpext.ParsedHTTPRequest{
		URL:              &url,
		Req:              r,
		Body:             bytes.NewBuffer(buf),
		Throw:            state.Options.Throw.Bool,
		Redirects:        state.Options.MaxRedirects,
		Timeout:          c.cfg.Timeout,
		ResponseCallback: ResponseCallback,
	})
	if err != nil {
		return *httpResp, err
	}

	return *response, err
}

func ResponseCallback(n int) bool {
	// report all 2xx respones as successful requests
	return n/100 == 2
}
