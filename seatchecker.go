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

func httpRequest(method string, url string, headers http.Header, payload map[string]any, response any) error {
	b := new(bytes.Buffer)
	if payload != nil {
		json.NewEncoder(b).Encode(payload)
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, b)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if headers != nil {
		req.Header = headers
	}
	// Add default headers
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if err := json.Unmarshal(body, response); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	return nil
}

// TODO: add exception handling
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
		fmt.Println(err)
		return auth{}, err
	}

	return a, nil
}

// TODO: maybe this is more of a GetBookingId
func getOrders(auth auth) (string, error) {
	method := "GET"
	url := fmt.Sprintf("https://www.ryanair.com/api/orders/v2/orders/%s?active=true&order=ASC", auth.CustomerID)
	headers := http.Header{
		"x-auth-token": {auth.Token},
	}

	var response struct {
		Items []struct {
			Flights []struct {
				BookingId string `json:"bookingId"`
			} `json:"flights"`
		} `json:"items"`
	}

	err := httpRequest(method, url, headers, nil, &response)
	if err != nil {
		fmt.Println("Failed to execute request:", err)
		return "", err
	}

	// TODO: base on what should i do filtering here?
	return response.Items[0].Flights[0].BookingId, nil
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
		fmt.Println("Failed to execute request:", err)
		return "", "", err
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
		fmt.Println("Failed to execute request:", err)
		return "", err
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
		fmt.Println("Failed to execute request:", err)
		// return "", err
	}

	fmt.Println(response)
}

func main() {
	email := os.Getenv("SEATCHECKER_EMAIL")
	if email == "" {
		fmt.Println("Env var 'SEATCHECKER_EMAIL' is not configured.")
		return
	}
	password := os.Getenv("SEATCHECKER_PASSWORD")
	if password == "" {
		fmt.Println("Env var 'SEATCHECKER_PASSWORD' is not configured.")
		return
	}

	auth, _ := accountLogin(email, password)
	fmt.Println(auth)

	bookingId, _ := getOrders(auth)
	// fmt.Println(bookingId)

	tripId, sessionToken, _ := getBookingById(bookingId, auth)

	basketId, _ := createBasket(tripId, sessionToken)

	getSeatsQuery(basketId)
}
