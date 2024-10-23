package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGetBookingId(t *testing.T) {
	cAReq := RAuth{"customerid", "token"}
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
	c := Client{scheme: "http", fqdn: ts.URL}
	rId, err := c.getBookingId(context.TODO(), cAReq)
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
	c := Client{scheme: "http", fqdn: ts.URL}
	cAReq := RAuth{"customerid", "token"}
	bA, err := c.getBookingById(context.TODO(), cAReq, "booking_id")
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
	c := Client{scheme: "http", fqdn: ts.URL}

	bId, err := c.createBasket(context.TODO(), bA)
	if err != nil {
		t.Fatalf("failed to create basket: %v", err)
	}
	if eId != bId {
		t.Fatalf("wrong basket id, expected: %v, received %v", eId, bId)
	}
}

func TestGetSeatsQuery(t *testing.T) {
	bId := "basket_id"
	s := SQSeats{[]string{"01A", "01B", "01C"}, "30A"}

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
			Data: SQData{Seats: []SQSeats{s}},
		}

		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}

	qs, err := c.getSeatsQuery(context.TODO(), bId)
	if err != nil {
		t.Fatalf("failed to query seats: %v", err)
	}

	if !reflect.DeepEqual(s, qs) {
		t.Fatalf("wrong seats, expected: %v, received: %v", s, qs)
	}
}

func TestGetNumberOfRows(t *testing.T) {
	m := "32A"
	rows := 30

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if !strings.Contains(r.URL.RawQuery, "aircraftModel="+m) {
			t.Fatalf("missing url encoded query string parameter, name: active")
		}

		// Create fake response
		rres := []Equipment{{SeatRows: [][]Seat{{{Row: 1}}, {{Row: rows}}}}}

		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}

	rrows, err := c.getNumberOfRows(context.TODO(), m)
	if err != nil {
		t.Fatalf("failed to get number of rows: %v\n", err)
	}

	if rows != rrows {
		t.Fatalf("wrong number of seats, expected: %v, received: %v\n", rows, rrows)
	}
}

func TestCalculateEmptySeats(t *testing.T) {
	r := 4

	test := func(s []string, ew int, em int, ea int) {
		w, m, a := calculateEmptySeats(r, s)
		if w != ew || m != em || a != ea {
			eTxt := generateText(ew, em, ea)
			rTxt := generateText(w, m, a)
			t.Fatalf("wrong number of calculated empty seats, expected: %v, received: %v\n", eTxt, rTxt)
		}
	}

	fs := []string{
		"01A", "01B", "01C", "01D", "01E", "01F",
		"02A", "02B", "02C", "02D", "02E", "02F",
		"03A", "03B", "03C", "03D", "03E", "03F",
		"04A", "04B", "04C", "04D", "04E", "04F",
	}
	test(fs, 0, 0, 0)

	ss := []string{
		"01A", "01B", "01E",
		"02B", "02C", "02D", "02E", "02F",
		"03B", "03C", "03D", "03E", "03F",
		"04A", "04B", "04C", "04D", "04E",
	}
	test(ss, 4, 0, 2)

	ns := []string{}
	ms := r * 2
	test(ns, ms, ms, ms)
}

func TestQueryRyanair(t *testing.T) {
	// TODO: implement
}
