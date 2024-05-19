package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

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

	c := RClient{schema: "http", fqdn: ts.URL}

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
	c := RClient{schema: "http", fqdn: ts.URL}
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
	c := RClient{schema: "http", fqdn: ts.URL}
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
