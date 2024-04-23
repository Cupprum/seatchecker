package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type auth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

type flights []struct {
	BookingId string `json:"bookingId"`
}

func httpRequest(method string, url string, headers http.Header, payload map[string]any, response any) error {
	b := new(bytes.Buffer)
	if payload != nil {
		json.NewEncoder(b).Encode(payload)
	}

	c := &http.Client{}
	req, err := http.NewRequest(method, url, b)
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
		return fmt.Errorf("failed to unmarshall Json respone: %v", err)
	}

	return nil
}

func accountLogin(email string, password string) (auth, error) {
	method := "POST"
	url := "https://www.ryanair.com/api/usrprof/v2/accountLogin"
	payload := map[string]any{
		"email":    email,
		"password": password,
	}

	var a auth

	err := httpRequest(method, url, nil, payload, &a)
	if err != nil {
		return auth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return a, nil
}

func getOrders(auth auth) (flights, error) {
	method := "GET"
	url := fmt.Sprintf("https://www.ryanair.com/api/orders/v2/orders/%s?active=true&order=ASC", auth.CustomerID)
	headers := http.Header{
		"x-auth-token": {auth.Token},
	}

	// TODO: how much stuff is in items?
	var res struct {
		Items []struct {
			Flights flights `json:"flights"`
		} `json:"items"`
	}

	err := httpRequest(method, url, headers, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %v", err)
	}

	// TODO: base on what should i do filtering here?
	return res.Items[0].Flights, nil
}

func getBookingId(auth auth) (string, error) {
	flights, err := getOrders(auth)
	if err != nil {
		return "", fmt.Errorf("failed to get booking id: %v", err)
	}
	// TODO: base on what should i do filtering here?
	return flights[0].BookingId, nil
}

func getBookingById(bookingId string, a auth) (string, string, error) {
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
	variables := map[string]any{
		"bookingInfo": map[string]string{
			"bookingId":   bookingId,
			"surrogateId": a.CustomerID,
		},
		"authToken": a.Token,
	}
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	var response struct {
		Data struct {
			GetBookingByBookingId struct {
				TripId       string `json:"tripId"`
				SessionToken string `json:"sessionToken"`
			} `json:"getBookingByBookingId"`
		} `json:"data"`
	}

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		return "", "", fmt.Errorf("failed to get booking: %v", err)
	}

	return response.Data.GetBookingByBookingId.TripId, response.Data.GetBookingByBookingId.SessionToken, nil
}

func createBasket(tripId string, sessionToken string) (string, error) {
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
	variables := map[string]any{
		"tripId":       tripId,
		"sessionToken": sessionToken,
	}
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	var response struct {
		Data struct {
			CreateBasketForActiveTrip struct {
				Id string `json:"id"`
			} `json:"createBasketForActiveTrip"`
		} `json:"data"`
	}

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		return "", fmt.Errorf("failed to create basket: %v", err)
	}

	return response.Data.CreateBasketForActiveTrip.Id, nil
}

func getSeatsQuery(basketId string) {
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
	variables := map[string]any{
		"basketId": basketId,
	}
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	var response struct {
		Data struct {
			Seats []struct {
				UnavailableSeats []string `json:"unavailableSeats"`
			} `json:"seats"`
		} `json:"data"`
	}

	err := httpRequest(method, url, nil, payload, &response)
	if err != nil {
		fmt.Println("failed to get seats:", err)
		// return fmt.Errorf("failed to get seats: %v", err)
	}

	fmt.Println(response)
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

	auth, err := accountLogin(email, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	bookingId, err := getBookingId(auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	tripId, sessionToken, err := getBookingById(bookingId, auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	basketId, err := createBasket(tripId, sessionToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	getSeatsQuery(basketId)
}
