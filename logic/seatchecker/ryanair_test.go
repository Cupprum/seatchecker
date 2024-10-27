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
	a := Auth{"customerid", "token"}
	eId := "booking_id"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if !strings.Contains(r.URL.Path, a.CustomerID) {
			t.Fatalf("missing Customer ID in URL, received URL Path: %v", r.URL.Path)
		}
		if rT := r.Header["X-Auth-Token"][0]; a.Token != rT {
			t.Fatalf("invalid auth token, expected: %v, received: %v", a.Token, rT)
		}
		if !strings.Contains(r.URL.RawQuery, "active=true") {
			t.Fatal("missing url encoded query string parameter, name: active")
		}
		if !strings.Contains(r.URL.RawQuery, "order=ASC") {
			t.Fatal("missing url encoded query string parameter, name: order")
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
	rId, err := c.getBookingId(context.Background(), a)
	if err != nil {
		t.Fatalf("failed to get booking id: %v", err)
	}
	if eId != rId {
		t.Fatalf("wrong booking id, expected: %v, received %v", eId, rId)
	}
}

func TestGetBookingById(t *testing.T) {
	e := TripInfo{TripId: "trip_id", SessionToken: "session_token"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[TIVars]{}
		json.Unmarshal(rawB, &b)

		v := TIVars{AuthToken: "token", BookingInfo: BInfo{BookingId: "booking_id", SurrogateId: "customerid"}}
		if !reflect.DeepEqual(v, b.Variables) {
			t.Fatalf("wrong payload, expected: %v, received: %v", v, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[TIData]{
			Data: TIData{TI: e},
		}
		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}
	a := Auth{"customerid", "token"}
	r, err := c.getTripInfo(context.Background(), a, "booking_id")
	if err != nil {
		t.Fatalf("failed to get booking: %v", err)
	}
	if e.SessionToken != r.SessionToken {
		t.Fatalf("wrong session token, expected: %v, received %v", e.SessionToken, r.SessionToken)
	}
	if e.TripId != r.TripId {
		t.Fatalf("wrong trip id, expected: %v, received %v", e.TripId, r.TripId)
	}
}

func TestCreateBasket(t *testing.T) {
	a := TripInfo{"trip_id", "session_token", []Journey{{"depart_utc"}}}
	e := "basket_id"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[TripInfo]{}
		json.Unmarshal(rawB, &b)

		if !reflect.DeepEqual(a, b.Variables) {
			t.Fatalf("wrong payload, expected: %v, received: %v", a, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[BData]{
			Data: BData{Basket: Basket{Id: e}},
		}
		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}

	r, err := c.createBasket(context.Background(), a)
	if err != nil {
		t.Fatalf("failed to create basket: %v", err)
	}
	if e != r {
		t.Fatalf("wrong basket id, expected: %v, received %v", e, r)
	}
}

func TestGetSeatsQuery(t *testing.T) {
	id := "basket_id"
	e := FlightInfo{[]string{"01A", "01B", "01C"}, "30A"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := GqlQuery[FIVars]{}
		json.Unmarshal(rawB, &b)

		if id != b.Variables.BId {
			t.Fatalf("wrong payload, expected: %v, received: %v", id, b.Variables)
		}

		// Create fake response
		rres := GqlResponse[FIData]{
			Data: FIData{FlightInfos: []FlightInfo{e}},
		}

		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}

	r, err := c.getFlightInfo(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to query seats: %v", err)
	}

	if !reflect.DeepEqual(e, r) {
		t.Fatalf("wrong seats, expected: %v, received: %v", e, r)
	}
}

func TestGetNumberOfRows(t *testing.T) {
	m := "32A"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if !strings.Contains(r.URL.RawQuery, "aircraftModel="+m) {
			t.Fatalf("missing url encoded query string parameter, name: active")
		}

		// Create fake response
		rres := []NORResp{{SeatRows: [][]NORSeat{{{Row: 1}}, {{Row: 2}}}}}

		res, _ := json.Marshal(rres)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}

	r, err := c.getNumberOfRows(context.Background(), m)
	if err != nil {
		t.Fatalf("failed to get number of rows: %v\n", err)
	}

	if r != 2 {
		t.Fatalf("wrong number of seats, expected: 2, received: %v\n", r)
	}
}

func TestCalculateEmptySeats(t *testing.T) {
	rws := 4

	test := func(s []string, e EmptySeats) {
		r := calculateEmptySeats(rws, s)
		if r != e {
			et := e.generateText()
			rt := r.generateText()
			t.Fatalf("wrong number of calculated empty seats, expected: %v, received: %v\n", et, rt)
		}
	}

	fs := []string{
		"01A", "01B", "01C", "01D", "01E", "01F",
		"02A", "02B", "02C", "02D", "02E", "02F",
		"03A", "03B", "03C", "03D", "03E", "03F",
		"04A", "04B", "04C", "04D", "04E", "04F",
	}
	test(fs, EmptySeats{0, 0, 0})

	ss := []string{
		"01A", "01B", "01E",
		"02B", "02C", "02D", "02E", "02F",
		"03B", "03C", "03D", "03E", "03F",
		"04A", "04B", "04C", "04D", "04E",
	}
	test(ss, EmptySeats{4, 0, 2})

	ns := []string{}
	ms := rws * 2
	test(ns, EmptySeats{ms, ms, ms})
}

func TestQueryRyanair(t *testing.T) {
	// TODO: implement
}
