package main

import (
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
		t.Fatalf("failed to create request: %v", err)
	}

	if r.method != cr.Method {
		t.Fatalf("wrong method, expected: %v, received: %v", r.method, cr.Method)
	}
	if r.scheme != cr.URL.Scheme {
		t.Fatalf("wrong scheme, expected: %v, received: %v", r.scheme, cr.URL.Scheme)
	}
	if r.fqdn != cr.URL.Host {
		t.Fatalf("wrong url, expected: %v, received: %v", r.fqdn, cr.URL.Host)
	}
	if !strings.Contains(cr.URL.Path, r.path) {
		t.Fatalf("wrong path, expected: %v, received: %v", r.path, cr.URL.Path)
	}
	if !reflect.DeepEqual(r.queryParams, cr.URL.Query()) {
		t.Fatalf("wrong query parameters, expected: %v, received: %v", r.queryParams, cr.URL.Query())
	}
	if !reflect.DeepEqual(r.headers, cr.Header) {
		t.Fatalf("wrong headers, expected: %v, received: %v", r.headers, cr.Header)
	}

	eb := "{\"payload\":\"test_payload\"}"
	rrb, _ := io.ReadAll(cr.Body)
	rb := string(rrb)
	if eb != rb {
		t.Fatalf("wrong body, expected: %v, received: %v", eb, rb)
	}
}

func TestHttpsRequest(t *testing.T) {
	ca := CAuth{
		CustomerID: "test_customer_id",
		Token:      "test_token",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, _ := json.Marshal(ca)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	r := Request{
		"POST",
		"http",
		ts.URL,
		"test_path",
		nil,
		nil,
		nil,
	}
	rca, _ := httpsRequest[CAuth](r)
	fmt.Println(rca)

	if !reflect.DeepEqual(ca, rca) {
		t.Fatalf("returned struct is incorrect, expected: %v, received: %v", ca, rca)
	}
}

func TestAccountLogin(t *testing.T) {
	e, p := "john@doe.com", "password"
	cAReq := CAuth{"customerid", "token"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawB, _ := io.ReadAll(r.Body)
		b := map[string]any{}
		json.Unmarshal(rawB, &b)
		if e != b["email"] {
			t.Fatalf("wrong email, expected: %v, received: %v", e, b["email"])
		}
		if p != b["password"] {
			t.Fatalf("wrong password, expected: %v, received: %v", e, b["password"])
		}
		res, _ := json.Marshal(cAReq)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	c := RClient{scheme: "http", fqdn: ts.URL}

	cARes, err := c.accountLogin(e, p)
	if err != nil {
		t.Fatalf("failed to get account login: %v", err)
	}

	if cAReq != cARes {
		t.Fatalf("wrong response, expected: %v, received %v", cAReq, cARes)
	}
}

func TestGetBookingId(t *testing.T) {
	cAReq := CAuth{"customerid", "token"}
	eId := "booking_id"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if !strings.Contains(r.URL.Path, cAReq.CustomerID) {
			t.Fatalf("missing Customer ID in URL, received URL Path: %v", r.URL.Path)
		}
		rT := r.Header["X-Auth-Token"][0]
		if cAReq.Token != rT {
			t.Fatalf("invalid auth token, expected: %v, received: %v", cAReq.Token, rT)
		}
		if !strings.Contains(r.URL.RawQuery, "active=true") {
			t.Fatalf("missing url encoded query string parameter, name: active")
		}
		if !strings.Contains(r.URL.RawQuery, "order=ASC") {
			t.Fatalf("missing url encoded query string parameter, name: order")
		}

		// Create fake response
		rres := BIdResp{
			Items: []BIdItem{
				{
					Flights: []BIdFlight{
						{BookingId: eId},
					},
				},
			},
		}
		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := RClient{scheme: "http", fqdn: ts.URL}
	rId, err := c.getBookingId(cAReq)
	if err != nil {
		t.Fatalf("failed to get booking id: %v", err)
	}
	if eId != rId {
		t.Fatalf("wrong booking id, expected: %v, received %v", eId, rId)
	}
}

func TestGetBookingById(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[BBIdVars]{}
		json.Unmarshal(rawB, &b)

		v := BBIdVars{AuthToken: "token", BookingInfo: BBIdInfo{BookingId: "booking_id", SurrogateId: "customerid"}}
		if !reflect.DeepEqual(v, b.Variables) {
			t.Fatalf("wrong payload, expected: %v, received: %v", v, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[BBIdData]{
			Data: BBIdData{GetBookingByBookingId: BAuth{TripId: "trip_id", SessionToken: "session_token"}},
		}
		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := RClient{scheme: "http", fqdn: ts.URL}
	cAReq := CAuth{"customerid", "token"}
	bA, err := c.getBookingById(cAReq, "booking_id")
	if err != nil {
		t.Fatalf("failed to get booking: %v", err)
	}
	if bA.SessionToken != "session_token" {
		t.Fatalf("wrong session token, expected: session_token, received %v", bA.SessionToken)
	}
	if bA.TripId != "trip_id" {
		t.Fatalf("wrong trip id, expected: trip_id, received %v", bA.TripId)
	}
}

func TestCreateBasket(t *testing.T) {
	bA := BAuth{"trip_id", "session_token"}
	eId := "basket_id"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[BAuth]{}
		json.Unmarshal(rawB, &b)

		if !reflect.DeepEqual(bA, b.Variables) {
			t.Fatalf("wrong payload, expected: %v, received: %v", bA, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[BData]{
			Data: BData{Basket: BBasket{Id: eId}},
		}
		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := RClient{scheme: "http", fqdn: ts.URL}

	bId, err := c.createBasket(bA)
	if err != nil {
		t.Fatalf("failed to create basket: %v", err)
	}
	if eId != bId {
		t.Fatalf("wrong basket id, expected: %v, received %v", eId, bId)
	}
}

func TestGetSeatsQuery(t *testing.T) {
	bId := "basket_id"
	s := []string{"01A", "01B", "01C"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[SQBasket]{}
		json.Unmarshal(rawB, &b)

		if bId != b.Variables.BId {
			t.Fatalf("wrong payload, expected: %v, received: %v", bId, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[SQData]{
			Data: SQData{Seats: []SQSeats{{UnavailableSeats: s}}},
		}

		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := RClient{scheme: "http", fqdn: ts.URL}

	qs, err := c.getSeatsQuery(bId)
	if err != nil {
		t.Fatalf("failed to query seats: %v", err)
	}

	if !reflect.DeepEqual(s, qs) {
		t.Fatalf("wrong seats, expected: %v, received: %v", s, qs)
	}
}
