package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func httpRequest[T any](method string, url string, headers http.Header, payload any) (T, error) {
	var nilT T // Empty response for errors.

	buf := []byte{} // If payload not specified, send empty buffer.
	var err error
	if payload != nil {
		buf, err = json.Marshal(payload)
		if err != nil {
			return nilT, fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	c := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
	if err != nil {
		return nilT, fmt.Errorf("failed to form request: %v", err)
	}

	if headers != nil {
		req.Header = headers
	}
	req.Header.Add("Content-Type", "application/json") // Add default headers

	res, err := c.Do(req)
	if err != nil {
		return nilT, fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nilT, fmt.Errorf("failed to read response: %v", err)
	}

	var t T
	if err := json.Unmarshal(body, &t); err != nil {
		return nilT, fmt.Errorf("failed to unmarshal Json response: %v", err)
	}

	return t, nil
}

type Auth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func accountLogin(email string, password string) (Auth, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/usrprof/v2/accountLogin"
	p := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpRequest[Auth](method, url, nil, p)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return a, nil
}

func (a Auth) getBookingId() (string, error) {
	method := "GET"
	url, err := url.Parse("https://www.ryanair.com/api/orders/v2/orders")
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %v", err)
	}
	// Add Customer ID to the path.
	url = url.JoinPath(a.CustomerID)

	// Specify query string parameters.
	q := url.Query()
	q.Add("active", "true")
	q.Add("order", "ASC")
	url.RawQuery = q.Encode()

	h := http.Header{
		"x-auth-token": {a.Token},
	}

	type R struct {
		Items []struct {
			Flights []struct {
				BookingId string `json:"bookingId"`
			} `json:"flights"`
		} `json:"items"`
	}

	res, err := httpRequest[R](method, url.String(), h, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get orders: %v", err)
	}

	// Items only contain single item.
	// Flights contain a single booking with multiple segments of flight.
	return res.Items[0].Flights[0].BookingId, nil
}

type GqlQuery struct {
	Query     string `json:"query"`
	Variables any    `json:"variables"`
}

type GqlResponse[T any] struct {
	Data T `json:"data"`
}

type Booking struct {
	TripId       string `json:"tripId"`
	SessionToken string `json:"sessionToken"`
}

func (a Auth) getBookingById(bookingId string) (Booking, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/bookingfa/en-gb/graphql"

	q := `
		query GetBookingByBookingId($bookingInfo: GetBookingByBookingIdInputType, $authToken: String!) {
			getBookingByBookingId(bookingInfo: $bookingInfo, authToken: $authToken) {
				sessionToken
				tripId
			}
		}
	`
	type BookingInfo struct {
		BookingId   string `json:"bookingId"`
		SurrogateId string `json:"surrogateId"`
	}
	v := struct {
		BookingInfo BookingInfo `json:"bookingInfo"`
		AuthToken   string      `json:"authToken"`
	}{
		BookingInfo{bookingId, a.CustomerID},
		a.Token,
	}
	p := GqlQuery{Query: q, Variables: v}

	type Data struct {
		GetBookingByBookingId Booking `json:"getBookingByBookingId"`
	}

	res, err := httpRequest[GqlResponse[Data]](method, url, nil, p)
	if err != nil {
		return Booking{}, fmt.Errorf("failed to get booking: %v", err)
	}

	return res.Data.GetBookingByBookingId, nil
}

func createBasket(booking Booking) (string, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/basketapi/en-gb/graphql"

	query := `
		mutation CreateBasketForActiveTrip($tripId: String!, $sessionToken: String) {
			createBasketForActiveTrip(tripId: $tripId, sessionToken: $sessionToken) {
				...BasketCommon
			}
		}
		fragment BasketCommon on BasketType {
			id
		}
	`
	payload := GqlQuery{Query: query, Variables: booking}

	type Data struct {
		Basket struct {
			Id string `json:"id"`
		} `json:"createBasketForActiveTrip"`
	}

	res, err := httpRequest[GqlResponse[Data]](method, url, nil, payload)
	if err != nil {
		return "", fmt.Errorf("failed to create basket: %v", err)
	}

	return res.Data.Basket.Id, nil
}

func getSeatsQuery(basketId string) error {
	method := "POST"
	url := "https://www.ryanair.com/api/catalogapi/en-gb/graphql"

	query := `
		query GetSeatsQuery($basketId: String!) {
			seats(basketId: $basketId) {
				...SeatsResponse
			}
		}
		fragment SeatsResponse on SeatAvailability {
			unavailableSeats
		}
	`
	variables := struct {
		BId string `json:"basketId"`
	}{
		basketId,
	}
	payload := GqlQuery{Query: query, Variables: variables}

	type Data struct {
		Seats []struct {
			UnavailableSeats []string `json:"unavailableSeats"`
		} `json:"seats"`
	}

	res, err := httpRequest[GqlResponse[Data]](method, url, nil, payload)
	if err != nil {
		return fmt.Errorf("failed to get seats: %v", err)
	}

	fmt.Println(res)
	return nil
}

func main() {
	email := os.Getenv("SEATCHECKER_EMAIL")
	if email == "" {
		fmt.Fprintf(os.Stderr, "env var 'SEATCHECKER_EMAIL' is not configured")
		os.Exit(1)
	}
	password := os.Getenv("SEATCHECKER_PASSWORD")
	if password == "" {
		fmt.Fprintf(os.Stderr, "env var 'SEATCHECKER_PASSWORD' is not configured")
		os.Exit(1)
	}

	catchErr := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	auth, err := accountLogin(email, password)
	catchErr(err)

	bookingId, err := auth.getBookingId()
	catchErr(err)

	booking, err := auth.getBookingById(bookingId)
	catchErr(err)

	basketId, err := createBasket(booking)
	catchErr(err)

	err = getSeatsQuery(basketId)
	catchErr(err)
}
