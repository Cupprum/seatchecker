package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type CAuth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c Client) accountLogin(email string, password string) (CAuth, error) {
	p := "api/usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	// TODO: empty login seems to be successful, how can this be...
	a, err := httpsRequestPost[CAuth](c, p, b)
	if err != nil {
		return CAuth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return a, nil
}

type BIdFlight struct {
	BookingId string `json:"bookingId"`
}
type BIdItem struct {
	Flights []BIdFlight `json:"flights"`
}
type BIdResp struct {
	Items []BIdItem `json:"items"`
}

func (c Client) getBookingId(a CAuth) (string, error) {
	p, err := url.JoinPath("api/orders/v2/orders", a.CustomerID)
	if err != nil {
		return "", fmt.Errorf("failed to create path: %v", err)
	}

	q := url.Values{}
	q.Add("active", "true")
	q.Add("order", "ASC")

	h := http.Header{
		"X-Auth-Token": {a.Token},
	}

	r, err := httpsRequestGet[BIdResp](c, p, q, h)
	if err != nil {
		return "", fmt.Errorf("failed to get orders: %v", err)
	}

	// Items only contain single item.
	// Flights contain a single booking with multiple segments of flight.
	return r.Items[0].Flights[0].BookingId, nil
}

type GqlQuery[T any] struct {
	Query     string `json:"query"`
	Variables T      `json:"variables"`
}

type GqlResponse[T any] struct {
	Data T `json:"data"`
}

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

func (c Client) getBookingById(a CAuth, bookingId string) (BAuth, error) {
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

	r, err := httpsRequestPost[GqlResponse[BBIdData]](c, p, b)
	if err != nil {
		return BAuth{}, fmt.Errorf("failed to get booking: %v", err)
	}

	return r.Data.GetBookingByBookingId, nil
}

type BBasket struct {
	Id string `json:"id"`
}

type BData struct {
	Basket BBasket `json:"createBasketForActiveTrip"`
}

func (c Client) createBasket(a BAuth) (string, error) {
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

	r, err := httpsRequestPost[GqlResponse[BData]](c, p, b)
	if err != nil {
		return "", fmt.Errorf("failed to create basket: %v", err)
	}

	return r.Data.Basket.Id, nil
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

func (c Client) getSeatsQuery(basketId string) (SQSeats, error) {
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

	r, err := httpsRequestPost[GqlResponse[SQData]](c, p, b)
	if err != nil {
		return SQSeats{}, fmt.Errorf("failed to get seats: %v", err)
	}

	// TODO: what is this supposed to return? always first? what if i ma on the way back?
	s := r.Data.Seats[0]

	return s, nil
}

type Seat struct {
	Row int `json:"row"`
}

type Equipment struct {
	SeatRows [][]Seat `json:"seatRows"`
}

func (c Client) getNumberOfRows(model string) (int, error) {
	p := "api/booking/v5/en-ie/res/seatmap"

	q := url.Values{}
	q.Add("aircraftModel", model)

	e, err := httpsRequestGet[[]Equipment](c, p, q, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get seatmap: %v", err)
	}

	// TODO: add comments
	sr := e[0].SeatRows
	maxRow := sr[len(sr)-1][0].Row

	return maxRow, nil
}

// TODO: can this return struct and generateText work on that struct and some other struct?
func calculateEmptySeats(rows int, seats []string) (int, int, int) {
	window := rows * 2
	middle := rows * 2
	aisle := rows * 2

	for _, s := range seats {
		switch string(s[2]) {
		case "A", "F":
			window -= 1
		case "B", "E":
			middle -= 1
		case "C", "D":
			aisle -= 1
		}
	}

	return window, middle, aisle
}

// TODO: update return values to something more normal
func queryRyanair(email string, password string) (int, int, int, error) {
	client := Client{
		scheme: "https",
		fqdn:   "www.ryanair.com",
	}

	log.Printf("Start account login for user: %s.\n", email)
	cAuth, err := client.accountLogin(email, password)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("login failed: %v", err)
	}
	log.Println("Account login finished successfully.")

	log.Println("Get closest Booking ID.")
	bookingId, err := client.getBookingId(cAuth)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get booking ID failed: %v", err)
	}
	log.Printf("Booking ID retrieved successfully: %s.\n", bookingId)

	log.Printf("Get Booking with ID: %s.\n", bookingId)
	bAuth, err := client.getBookingById(cAuth, bookingId)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get booking failed: %v", err)
	}
	log.Printf("Booking retrieved successfully, Trip ID: %s.\n", bAuth.TripId)

	log.Println("Create basket.")
	basketId, err := client.createBasket(bAuth)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("basket creation failed: %v", err)
	}
	log.Printf("Basket created successfully, Basket ID: %s.\n", basketId)

	log.Println("Get seats.")
	seats, err := client.getSeatsQuery(basketId)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get seats failed: %v", err)
	}
	log.Println("Seats retrieved successfully.")
	log.Printf("Model: %v\n", seats.EquipmentModel)
	log.Printf("Number of occupied seats: %v\n", len(seats.UnavailableSeats))

	log.Println("Get number of rows in the plane.")
	rows, err := client.getNumberOfRows(seats.EquipmentModel)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get number of rows in the plane failed: %v", err)
	}
	log.Println("Number of rows retrieved successfully.")
	log.Println(rows)

	log.Println("Calculate number of empty seats.")
	window, middle, aisle := calculateEmptySeats(rows, seats.UnavailableSeats)

	return window, middle, aisle, nil
}
