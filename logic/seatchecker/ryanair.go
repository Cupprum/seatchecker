package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type BIdFlight struct {
	BookingId string `json:"bookingId"`
}
type BIdItem struct {
	Flights []BIdFlight `json:"flights"`
}
type BIdResp struct {
	Items []BIdItem `json:"items"`
}

func (c Client) getBookingId(ctx context.Context, a Auth) (string, error) {
	ctx, span := tr.Start(ctx, "get_booking_id")
	defer span.End()
	span.SetAttributes(attribute.String("customer_id", a.CustomerID)) // NOTE: delete after testing.

	p, err := url.JoinPath("api/orders/v2/orders", a.CustomerID)
	if err != nil {
		err = fmt.Errorf("failed to create path: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	q := url.Values{}
	q.Add("active", "true")
	q.Add("order", "ASC")

	h := http.Header{
		"X-Auth-Token": {a.Token},
	}

	r, err := httpsRequestGet[BIdResp](ctx, c, p, q, h)
	if err != nil {
		err = fmt.Errorf("failed to get orders: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Items only contain single item.
	// Flights contain a single booking with multiple segments of flight.
	id := r.Items[0].Flights[0].BookingId
	return id, nil
}

type GqlQuery[T any] struct {
	Query     string `json:"query"`
	Variables T      `json:"variables"`
}

type GqlResponse[T any] struct {
	Data T `json:"data"`
}

type TripInfo struct {
	TripId       string    `json:"tripId"`
	SessionToken string    `json:"sessionToken"`
	Journeys     []Journey `json:"journeys"`
}

type Journey struct {
	DepartUTC string `json:"departUTC"`
}

type BInfo struct {
	BookingId   string `json:"bookingId"`
	SurrogateId string `json:"surrogateId"`
}

type TIVars struct {
	BookingInfo BInfo  `json:"bookingInfo"`
	AuthToken   string `json:"authToken"`
}

type TIData struct {
	TI TripInfo `json:"getBookingByBookingId"`
}

func (c Client) getTripInfo(ctx context.Context, a Auth, id string) (TripInfo, error) {
	ctx, span := tr.Start(ctx, "get_trip_info")
	defer span.End()
	span.SetAttributes(attribute.String("booking_id", id)) // NOTE: delete after testing.

	p := "api/bookingfa/en-gb/graphql"

	q := `
		query GetBookingByBookingId($bookingInfo: GetBookingByBookingIdInputType, $authToken: String!) {
			getBookingByBookingId(bookingInfo: $bookingInfo, authToken: $authToken) {
				sessionToken
				tripId
				journeys {
		        	...JourneysFrag
      			}
			}
		}
		fragment JourneysFrag on BookingJourneyResponseModelType {
			departUTC
		}
	`
	v := TIVars{
		BInfo{id, a.CustomerID},
		a.Token,
	}
	b := GqlQuery[TIVars]{Query: q, Variables: v}

	r, err := httpsRequestPost[GqlResponse[TIData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get booking: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return TripInfo{}, err
	}

	ti := r.Data.TI
	return ti, nil
}

type Basket struct {
	Id string `json:"id"`
}

type BData struct {
	Basket Basket `json:"createBasketForActiveTrip"`
}

func (c Client) createBasket(ctx context.Context, ti TripInfo) (string, error) {
	ctx, span := tr.Start(ctx, "create_basket")
	defer span.End()
	span.SetAttributes(attribute.String("trip_id", ti.TripId)) // NOTE: delete after testing.

	p := "api/basketapi/en-gb/graphql"

	q := `
		mutation CreateBasketForActiveTrip($tripId: String!, $sessionToken: String) {
			createBasketForActiveTrip(tripId: $tripId, sessionToken: $sessionToken) {
				...BasketCommon
			}
		}
		fragment BasketCommon on BasketType {
			id
		}
	`
	b := GqlQuery[TripInfo]{Query: q, Variables: ti}

	r, err := httpsRequestPost[GqlResponse[BData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to create basket: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	id := r.Data.Basket.Id
	return id, nil
}

type FlightInfo struct {
	UnavailableSeats []string `json:"unavailableSeats"`
	EquipmentModel   string   `json:"equipmentModel"`
}

type FIVars struct {
	BId string `json:"basketId"`
}

type FIData struct {
	FlightInfos []FlightInfo `json:"seats"`
}

func (c Client) getFlightInfo(ctx context.Context, id string) (FlightInfo, error) {
	ctx, span := tr.Start(ctx, "get_flight_info")
	defer span.End()
	span.SetAttributes(attribute.String("basket_id", id)) // NOTE: delete after testing.

	p := "api/catalogapi/en-gb/graphql"

	q := `
		query GetSeatsQuery($basketId: String!) {
			seats(basketId: $basketId) {
				...SeatsResponse
			}
		}
		fragment SeatsResponse on SeatAvailability {
			unavailableSeats
			equipmentModel
		}
	`
	v := FIVars{id}
	b := GqlQuery[FIVars]{Query: q, Variables: v}

	r, err := httpsRequestPost[GqlResponse[FIData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get seats: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return FlightInfo{}, err
	}

	// TODO: what is this supposed to return? always first? what if i ma on the way back?
	fi := r.Data.FlightInfos[0]
	return fi, nil
}

type NORSeat struct {
	Row int `json:"row"`
}

type NORResp struct {
	SeatRows [][]NORSeat `json:"seatRows"`
}

func (c Client) getNumberOfRows(ctx context.Context, m string) (int, error) {
	ctx, span := tr.Start(ctx, "get_number_of_rows")
	defer span.End()
	span.SetAttributes(attribute.String("model", m)) // NOTE: delete after testing.

	p := "api/booking/v5/en-ie/res/seatmap"

	q := url.Values{}
	q.Add("aircraftModel", m)

	rs, err := httpsRequestGet[[]NORResp](ctx, c, p, q, nil)
	if err != nil {
		err = fmt.Errorf("failed to get seatmap: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	// Get first response
	r := rs[0]
	nor := len(r.SeatRows)

	span.SetAttributes(attribute.Int("number_of_rows", nor))
	return nor, nil
}

func calculateEmptySeats(rows int, seats []string) EmptySeats {
	es := EmptySeats{rows * 2, rows * 2, rows * 2}

	for _, s := range seats {
		// Second character represents the seat columns.
		switch string(s[2]) {
		case "A", "F":
			es.Window -= 1
		case "B", "E":
			es.Middle -= 1
		case "C", "D":
			es.Aisle -= 1
		}
	}
	return es
}

func (c Client) getEmptySeats(ctx context.Context, a Auth) (EmptySeats, []string, error) {
	ctx, span := tr.Start(ctx, "ryanair_get_empty_seats")
	defer span.End()
	span.SetAttributes(attribute.String("customer_id", a.CustomerID)) // NOTE: delete after testing.

	throwErr := func(err error) (EmptySeats, []string, error) {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, nil, err
	}

	log.Println("Get closest Booking ID.")
	bookingId, err := c.getBookingId(ctx, a)
	if err != nil {
		err = fmt.Errorf("get booking ID failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Booking ID retrieved successfully.")

	log.Println("Get Trip info.")
	ti, err := c.getTripInfo(ctx, a, bookingId)
	if err != nil {
		err := fmt.Errorf("get trip info failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Trip info retrieved successfully.")

	log.Println("Create basket.")
	basketId, err := c.createBasket(ctx, ti)
	if err != nil {
		err = fmt.Errorf("basket creation failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Basket created successfully.")

	log.Println("Get Flight info.")
	fi, err := c.getFlightInfo(ctx, basketId)
	if err != nil {
		err = fmt.Errorf("get flight info failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Flight info retrieved successfully.")

	log.Println("Get number of rows in the plane.")
	nor, err := c.getNumberOfRows(ctx, fi.EquipmentModel)
	if err != nil {
		err = fmt.Errorf("get number of rows in the plane failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Number of rows retrieved successfully.")

	log.Println("Calculate number of empty seats.")
	es := calculateEmptySeats(nor, fi.UnavailableSeats)
	span.AddEvent("Empty seats calculated successfully.", trace.WithAttributes(
		attribute.Int("window", es.Window),
		attribute.Int("middle", es.Middle),
		attribute.Int("aisle", es.Aisle)))

	var js []string
	for _, j := range ti.Journeys {
		js = append(js, j.DepartUTC)
	}
	return es, js, nil
}
