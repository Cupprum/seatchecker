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
type BAuth struct {
	TripId       string `json:"tripId"`
	SessionToken string `json:"sessionToken"`
}

type BBIdInfo struct {
	BookingId   string `json:"bookingId"`
	SurrogateId string `json:"surrogateId"`
}

type BBIdVars struct {
	BookingInfo BBIdInfo `json:"bookingInfo"`
	AuthToken   string   `json:"authToken"`
}

type BBIdData struct {
	GetBookingByBookingId BAuth `json:"getBookingByBookingId"`
}

// TODO: rename to getSessionToken or something similar.
func (c Client) getBookingById(ctx context.Context, a RAuth, bookingId string) (BAuth, error) {
	ctx, span := tr.Start(ctx, "get_booking_by_id")
	defer span.End()
	span.SetAttributes(attribute.String("bookingId", bookingId))

	p := "api/bookingfa/en-gb/graphql"

	q := `
		query GetBookingByBookingId($bookingInfo: GetBookingByBookingIdInputType, $authToken: String!) {
			getBookingByBookingId(bookingInfo: $bookingInfo, authToken: $authToken) {
				sessionToken
				tripId
			}
		}
	`
	v := BBIdVars{
		BBIdInfo{bookingId, a.CustomerID},
		a.Token,
	}
	b := GqlQuery[BBIdVars]{Query: q, Variables: v}

	r, err := httpsRequestPost[GqlResponse[BBIdData]](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get booking: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return BAuth{}, err
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

func (c Client) createBasket(ctx context.Context, a BAuth) (string, error) {
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
	b := GqlQuery[BAuth]{Query: q, Variables: a}

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

func (c Client) getSeatsQuery(ctx context.Context, basketId string) (SQSeats, error) {
	ctx, span := tr.Start(ctx, "get_seats_query")
	defer span.End()
	span.SetAttributes(attribute.String("basketId", basketId))

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
	v := SQBasket{
		basketId,
	}
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

func (c Client) getNumberOfRows(ctx context.Context, model string) (int, error) {
	ctx, span := tr.Start(ctx, "get_number_of_rows")
	defer span.End()
	span.SetAttributes(attribute.String("model", model))

	p := "api/booking/v5/en-ie/res/seatmap"

	q := url.Values{}
	q.Add("aircraftModel", model)

	e, err := httpsRequestGet[[]Equipment](ctx, c, p, q, nil)
	if err != nil {
		err = fmt.Errorf("failed to get seatmap: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	// TODO: add comments
	sr := e[0].SeatRows
	maxRow := sr[len(sr)-1][0].Row
	span.SetAttributes(attribute.Int("rows", maxRow))
	return maxRow, nil
}

func calculateEmptySeats(rows int, seats []string) SeatState {
	ss := SeatState{Window: rows * 2, Middle: rows * 2, Aisle: rows * 2}

	for _, s := range seats {
		// Second character represents the seat columns.
		switch string(s[2]) {
		case "A", "F":
			ss.Window -= 1
		case "B", "E":
			ss.Middle -= 1
		case "C", "D":
			ss.Aisle -= 1
		}
	}
	return ss
}

func (c Client) queryRyanair(ctx context.Context, cAuth RAuth) (SeatState, error) {
	ctx, span := tr.Start(ctx, "query_ryanair")
	defer span.End()

	// TODO: turn logs into something tracelike.
	log.Println("Get closest Booking ID.")
	bookingId, err := c.getBookingId(ctx, cAuth)
	if err != nil {
		err = fmt.Errorf("get booking ID failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SeatState{}, err
	}
	span.AddEvent("Booking ID retrieved successfully.", trace.WithAttributes(attribute.String("bookingId", bookingId)))

	log.Printf("Get Booking with ID: %s.\n", bookingId)
	bAuth, err := c.getBookingById(ctx, cAuth, bookingId)
	if err != nil {
		err := fmt.Errorf("get booking failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SeatState{}, err
	}
	span.AddEvent("Booking retrieved successfully.", trace.WithAttributes(attribute.String("tripId", bAuth.TripId)))

	log.Println("Create basket.")
	basketId, err := c.createBasket(ctx, bAuth)
	if err != nil {
		err = fmt.Errorf("basket creation failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SeatState{}, err
	}
	span.AddEvent("Basket created successfully.", trace.WithAttributes(attribute.String("basketId", basketId)))

	log.Println("Get seats.")
	seats, err := c.getSeatsQuery(ctx, basketId)
	if err != nil {
		err = fmt.Errorf("get seats failed: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return SeatState{}, err
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
		return SeatState{}, err
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
