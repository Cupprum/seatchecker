package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestRequestCreator(t *testing.T) {
	q := url.Values{}
	q.Add("query_parameter", "test_query_parameter")
	h := http.Header{
		"header": {"test_header"},
	}
	b := struct {
		Payload string `json:"payload"`
	}{
		Payload: "test_payload",
	}

	r := Request{
		ctx:         context.Background(),
		method:      "GET",
		scheme:      "http",
		fqdn:        "test",
		path:        "test_path",
		queryParams: q,
		headers:     h,
		body:        b,
	}

	cr, err := r.creator()
	if err != nil {
		t.Fatalf("failed to create request: %v\n", err)
	}

	if r.method != cr.Method {
		t.Fatalf("wrong method, expected: %v, received: %v\n", r.method, cr.Method)
	}
	if r.scheme != cr.URL.Scheme {
		t.Fatalf("wrong scheme, expected: %v, received: %v\n", r.scheme, cr.URL.Scheme)
	}
	if r.fqdn != cr.URL.Host {
		t.Fatalf("wrong url, expected: %v, received: %v\n", r.fqdn, cr.URL.Host)
	}
	if !strings.Contains(cr.URL.Path, r.path) {
		t.Fatalf("wrong path, expected: %v, received: %v\n", r.path, cr.URL.Path)
	}
	if !reflect.DeepEqual(r.queryParams, cr.URL.Query()) {
		t.Fatalf("wrong query parameters, expected: %v, received: %v\n", r.queryParams, cr.URL.Query())
	}
	if !reflect.DeepEqual(r.headers, cr.Header) {
		t.Fatalf("wrong headers, expected: %v, received: %v\n", r.headers, cr.Header)
	}

	eb := "{\"payload\":\"test_payload\"}"
	rrb, _ := io.ReadAll(cr.Body)
	rb := string(rrb)
	if eb != rb {
		t.Fatalf("wrong body, expected: %v, received: %v\n", eb, rb)
	}
}

func TestHttpsRequest(t *testing.T) {
	ra := RAuth{
		CustomerID: "test_customer_id",
		Token:      "test_token",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, _ := json.Marshal(ra)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	r := Request{
		context.Background(),
		"POST",
		"http",
		ts.URL,
		"test_path",
		nil,
		nil,
		nil,
	}
	rra, _ := httpsRequest[RAuth](r)

	if !reflect.DeepEqual(ra, rra) {
		t.Fatalf("returned struct is incorrect, expected: %v, received: %v\n", ra, rra)
	}
}
