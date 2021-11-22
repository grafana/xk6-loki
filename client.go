package loki

import (
	"bytes"
	"context"
	"math/rand"
	"net/http"
	"net/url"
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
	r, err := http.NewRequest(http.MethodPost, c.cfg.URL.String(), nil)
	if err != nil {
		return *httpResp, err
	}

	r.Header.Set("User-Agent", c.cfg.UserAgent)
	if c.cfg.TenantID != "" {
		r.Header.Set("X-Scope-OrgID", c.cfg.TenantID)
	}
	if useProtobuf {
		r.Header.Set("Content-Type", ContentTypeProtobuf)
		r.Header.Add("Content-Encoding", ContentEncodingSnappy)
	} else {
		r.Header.Set("Content-Type", ContentTypeJSON)
	}

	url, _ := httpext.NewURL(c.cfg.URL.String(), "push")
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
	return n == 200
}
