package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Auth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func httpRequest(method string, url string, headers http.Header, payload any, response any) error {
	buf := []byte{}
	var err error
	if payload != nil {
		buf, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	c := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
	if err != nil {
		return fmt.Errorf("failed to form request: %v", err)
	}

	if headers != nil {
		req.Header = headers
	}
	// Add default headers
	req.Header.Add("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("failed to unmarshal Json response: %v", err)
	}

	return nil
}

func accountLogin(email string, password string) (Auth, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/usrprof/v2/accountLogin"
	payload := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	var auth Auth

	err := httpRequest(method, url, nil, payload, &auth)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return auth, nil
}

type Flights []struct {
	BookingId string `json:"bookingId"`
}

func (auth Auth) getOrders() (Flights, error) {
	method := "GET"
	// TODO: look at flags
	url := fmt.Sprintf("https://www.ryanair.com/api/orders/v2/orders/%s?active=true&order=ASC", auth.CustomerID)
	headers := http.Header{
		"x-auth-token": {auth.Token},
	}

	// TODO: how much stuff is in items?
	var res struct {
		Items []struct {
			Flights Flights `json:"flights"`
		} `json:"items"`
	}

	err := httpRequest(method, url, headers, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %v", err)
	}

	// TODO: base on what should i do filtering here?
	return res.Items[0].Flights, nil
}

func (auth Auth) getBookingId() (string, error) {
	flights, err := auth.getOrders()
	if err != nil {
		return "", fmt.Errorf("failed to get booking id: %v", err)
	}
	// TODO: base on what should i do filtering here?
	return flights[0].BookingId, nil
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

func (auth Auth) getBookingById(bookingId string) (Booking, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/bookingfa/en-gb/graphql"

	query := `
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
	variables := struct {
		BookingInfo BookingInfo `json:"bookingInfo"`
		AuthToken   string      `json:"authToken"`
	}{
		BookingInfo{bookingId, auth.CustomerID},
		auth.Token,
	}
	payload := GqlQuery{Query: query, Variables: variables}

	type Data struct {
		GetBookingByBookingId Booking `json:"getBookingByBookingId"`
	}
	var response GqlResponse[Data]

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		return Booking{}, fmt.Errorf("failed to get booking: %v", err)
	}

	return response.Data.GetBookingByBookingId, nil
}

func createBasket(booking Booking) (string, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/basketapi/en-ie/graphql"

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
	var response GqlResponse[Data]

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		return "", fmt.Errorf("failed to create basket: %v", err)
	}

	return response.Data.Basket.Id, nil
}

func getSeatsQuery(basketId string) error {
	method := "POST"
	url := "https://www.ryanair.com/api/catalogapi/en-ie/graphql"

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
	var response GqlResponse[Data]

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		return fmt.Errorf("failed to get seats: %v", err)
	}

	fmt.Println(response)
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
	// fmt.Println(bookingId)

	booking, err := auth.getBookingById(bookingId)
	catchErr(err)

	basketId, err := createBasket(booking)
	catchErr(err)

	err = getSeatsQuery(basketId)
	catchErr(err)
}
