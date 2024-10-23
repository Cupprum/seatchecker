package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	scheme string
	fqdn   string
}

type Request struct {
	ctx         context.Context
	method      string
	scheme      string
	fqdn        string
	path        string
	queryParams url.Values
	headers     http.Header
	body        any
}

func (r Request) creator() (*http.Request, error) {
	u, err := url.Parse(r.fqdn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	u = u.JoinPath(r.path)              // Specify path.
	u.Scheme = r.scheme                 // Specify scheme.
	u.RawQuery = r.queryParams.Encode() // Specify query string parameters.

	buf := []byte{} // If payload not specified, send empty buffer.
	if r.body != nil {
		buf, err = json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	req, err := http.NewRequestWithContext(r.ctx, r.method, u.String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, fmt.Errorf("failed to form request: %v", err)
	}

	if r.headers != nil {
		req.Header = r.headers
	}
	req.Header.Add("Content-Type", "application/json") // Add default headers

	return req, nil
}

func httpsRequest[T any](req Request) (T, error) {
	var nilT T // Empty response for errors.

	r, err := req.creator()
	if err != nil {
		return nilT, fmt.Errorf("failed to create request: %v", err)
	}

	c := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}),
		),
	}

	res, err := c.Do(r)
	if err != nil {
		return nilT, fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nilT, fmt.Errorf("request return invalid code: %v", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nilT, fmt.Errorf("failed to read response: %v", err)
	}

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nilT, fmt.Errorf("failed to unmarshal Json response: %v", err)
	}

	return t, nil
}

func httpsRequestGet[T any](ctx context.Context, c Client, path string, queryParams url.Values, headers http.Header) (T, error) {
	r := Request{
		ctx,
		"GET",
		c.scheme,
		c.fqdn,
		path,
		queryParams,
		headers,
		nil,
	}
	return httpsRequest[T](r)
}

func httpsRequestPost[T any](ctx context.Context, c Client, path string, body any) (T, error) {
	r := Request{
		ctx,
		"POST",
		c.scheme,
		c.fqdn,
		path,
		nil,
		nil,
		body,
	}
	return httpsRequest[T](r)
}
