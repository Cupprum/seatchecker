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

func (c Client) getBookingId(ctx context.Context, a RAuth) (string, error) {
	ctx, span := tr.Start(ctx, "get_booking_id")
	defer span.End()

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
	span.SetAttributes(attribute.String("bookingId", id))
	return id, nil
}

type GqlQuery[T any] struct {
	Query     string `json:"query"`
	Variables T      `json:"variables"`
}

type GqlResponse[T any] struct {
	Data T `json:"data"`
}

// TODO: rename structs regarding authentication to something more useful.
type SessionInfo struct {
	TripId       string `json:"tripId"`
	SessionToken string `json:"sessionToken"`
}

type BInfo struct {
	BookingId   string `json:"bookingId"`
	SurrogateId string `json:"surrogateId"`
}

type SIVars struct {
	BookingInfo BInfo  `json:"bookingInfo"`
	AuthToken   string `json:"authToken"`
}

type SIData struct {
	GetBookingByBookingId SessionInfo `json:"getBookingByBookingId"`
}

func (c Client) getSessionInfo(ctx context.Context, a RAuth, id string) (SessionInfo, error) {
	ctx, span := tr.Start(ctx, "get_booking_by_id")
	defer span.End()
	span.SetAttributes(attribute.String("bookingId", id))

	p := "api/bookingfa/en-gb/graphql"

	q := `
		query GetBookingByBookingId($bookingInfo: GetBookingByBookingIdInputType, $authToken: String!) {
			getBookingByBookingId(bookingInfo: $bookingInfo, authToken: $authToken) {
				sessionToken
				tripId
			}
		}
	`
	v := SIVars{
		BInfo{id, a.CustomerID},
		a.Token,
	}
	b := GqlQuery[SIVars]{Query: q, Variables: v}

	r, err := httpsRequestPost[GqlResponse[SIData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get booking: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SessionInfo{}, err
	}

	bAuth := r.Data.GetBookingByBookingId
	span.SetAttributes(attribute.String("tripId", bAuth.TripId))
	return bAuth, nil
}

type BBasket struct {
	Id string `json:"id"`
}

type BData struct {
	Basket BBasket `json:"createBasketForActiveTrip"`
}

func (c Client) createBasket(ctx context.Context, a SessionInfo) (string, error) {
	ctx, span := tr.Start(ctx, "create_basket")
	defer span.End()

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
	b := GqlQuery[SessionInfo]{Query: q, Variables: a}

	r, err := httpsRequestPost[GqlResponse[BData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to create basket: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	id := r.Data.Basket.Id
	span.SetAttributes(attribute.String("basketId", id))
	return id, nil
}

type SQBasket struct {
	BId string `json:"basketId"`
}

type SQSeats struct {
	UnavailableSeats []string `json:"unavailableSeats"`
	EquipmentModel   string   `json:"equipmentModel"`
}

type SQData struct {
	Seats []SQSeats `json:"seats"`
}

func (c Client) getSeatsQuery(ctx context.Context, id string) (SQSeats, error) {
	ctx, span := tr.Start(ctx, "get_seats_query")
	defer span.End()
	span.SetAttributes(attribute.String("basketId", id))

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
	v := SQBasket{id}
	b := GqlQuery[SQBasket]{Query: q, Variables: v}

	r, err := httpsRequestPost[GqlResponse[SQData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get seats: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SQSeats{}, err
	}

	// TODO: what is this supposed to return? always first? what if i ma on the way back?
	s := r.Data.Seats[0]
	span.SetAttributes(attribute.String("equipmentModel", s.EquipmentModel))
	span.SetAttributes(attribute.Int("unavailableSeats", len(s.UnavailableSeats)))
	return s, nil
}

type Seat struct {
	Row int `json:"row"`
}

type Equipment struct {
	SeatRows [][]Seat `json:"seatRows"`
}

func (c Client) getNumberOfRows(ctx context.Context, m string) (int, error) {
	ctx, span := tr.Start(ctx, "get_number_of_rows")
	defer span.End()
	span.SetAttributes(attribute.String("model", m))

	p := "api/booking/v5/en-ie/res/seatmap"

	q := url.Values{}
	q.Add("aircraftModel", m)

	e, err := httpsRequestGet[[]Equipment](ctx, c, p, q, nil)
	if err != nil {
		err = fmt.Errorf("failed to get seatmap: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	// TODO: add comments
	sr := e[0].SeatRows
	r := sr[len(sr)-1][0].Row
	span.SetAttributes(attribute.Int("number_of_rows", r))
	return r, nil
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

func (c Client) queryRyanair(ctx context.Context, a RAuth) (EmptySeats, error) {
	ctx, span := tr.Start(ctx, "query_ryanair")
	defer span.End()

	log.Println("Get closest Booking ID.")
	bookingId, err := c.getBookingId(ctx, a)
	if err != nil {
		err = fmt.Errorf("get booking ID failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, err
	}
	span.AddEvent("Booking ID retrieved successfully.", trace.WithAttributes(attribute.String("bookingId", bookingId)))

	log.Printf("Get Booking with ID: %s.\n", bookingId)
	bAuth, err := c.getSessionInfo(ctx, a, bookingId)
	if err != nil {
		err := fmt.Errorf("get booking failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, err
	}
	span.AddEvent("Booking retrieved successfully.", trace.WithAttributes(attribute.String("tripId", bAuth.TripId)))

	log.Println("Create basket.")
	basketId, err := c.createBasket(ctx, bAuth)
	if err != nil {
		err = fmt.Errorf("basket creation failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, err
	}
	span.AddEvent("Basket created successfully.", trace.WithAttributes(attribute.String("basketId", basketId)))

	log.Println("Get seats.")
	seats, err := c.getSeatsQuery(ctx, basketId)
	if err != nil {
		err = fmt.Errorf("get seats failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, err
	}
	span.AddEvent("Seats retrieved successfully.", trace.WithAttributes(
		attribute.String("equipmentModel", seats.EquipmentModel),
		attribute.Int("unavailableSeats", len(seats.UnavailableSeats))))

	log.Println("Get number of rows in the plane.")
	rows, err := c.getNumberOfRows(ctx, seats.EquipmentModel)
	if err != nil {
		err = fmt.Errorf("get number of rows in the plane failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return EmptySeats{}, err
	}
	span.AddEvent("Number of rows retrieved successfully.", trace.WithAttributes(attribute.Int("rows", rows)))

	log.Println("Calculate number of empty seats.")
	ss := calculateEmptySeats(rows, seats.UnavailableSeats)
	span.AddEvent("Empty seats calculated successfully.", trace.WithAttributes(
		attribute.Int("window", ss.Window),
		attribute.Int("middle", ss.Middle),
		attribute.Int("aisle", ss.Aisle)))
	return ss, nil
}
