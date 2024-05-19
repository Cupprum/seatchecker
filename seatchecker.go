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

type RClient struct {
	schema string
	fqdn   string
}

type Request struct {
	method      string
	schema      string
	fqdn        string
	path        string
	queryParams url.Values
	headers     http.Header
	body        any
}

func (r Request) creator() (*http.Request, error) {
	u, err := url.Parse(r.fqdn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	u = u.JoinPath(r.path)              // Specify path.
	u.Scheme = r.schema                 // Specify schema.
	u.RawQuery = r.queryParams.Encode() // Specify query string parameters.

	buf := []byte{} // If payload not specified, send empty buffer.
	if r.body != nil {
		buf, err = json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	req, err := http.NewRequest(r.method, u.String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, fmt.Errorf("failed to form request: %v", err)
	}

	if r.headers != nil {
		req.Header = r.headers
	}
	req.Header.Add("Content-Type", "application/json") // Add default headers

	return req, nil
}

func httpsRequest[T any](req Request) (T, error) {
	var nilT T // Empty response for errors.

	r, err := req.creator()
	if err != nil {
		return nilT, fmt.Errorf("failed to create request: %v", err)
	}

	c := &http.Client{}
	res, err := c.Do(r)
	if err != nil {
		return nilT, fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nilT, fmt.Errorf("failed to read response: %v", err)
	}

	// fmt.Println(string(b))

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nilT, fmt.Errorf("failed to unmarshal Json response: %v", err)
	}

	return t, nil
}

func httpsRequestGet[T any](c RClient, path string, queryParams url.Values, headers http.Header, body any) (T, error) {
	r := Request{
		"GET",
		c.schema,
		c.fqdn,
		path,
		queryParams,
		headers,
		body,
	}
	return httpsRequest[T](r)
}

func httpsRequestPost[T any](c RClient, path string, queryParams url.Values, headers http.Header, body any) (T, error) {
	r := Request{
		"POST",
		c.schema,
		c.fqdn,
		path,
		queryParams,
		headers,
		body,
	}
	return httpsRequest[T](r)
}

type CAuth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c RClient) accountLogin(email string, password string) (CAuth, error) {
	p := "api/usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpsRequestPost[CAuth](c, p, nil, nil, b)
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

func (c RClient) getBookingId(a CAuth) (string, error) {
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

	r, err := httpsRequestGet[BIdResp](c, p, q, h, nil)
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

func (c RClient) getBookingById(a CAuth, bookingId string) (BAuth, error) {
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

	r, err := httpsRequestPost[GqlResponse[BBIdData]](c, p, nil, nil, b)
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

func (c RClient) createBasket(a BAuth) (string, error) {
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

	r, err := httpsRequestPost[GqlResponse[BData]](c, p, nil, nil, b)
	if err != nil {
		return "", fmt.Errorf("failed to create basket: %v", err)
	}

	return r.Data.Basket.Id, nil
}

func (c RClient) getSeatsQuery(basketId string) error {
	p := "api/catalogapi/en-gb/graphql"

	q := `
		query GetSeatsQuery($basketId: String!) {
			seats(basketId: $basketId) {
				...SeatsResponse
			}
		}
		fragment SeatsResponse on SeatAvailability {
			unavailableSeats
		}
	`
	type Basket struct {
		BId string `json:"basketId"`
	}

	v := Basket{
		basketId,
	}
	b := GqlQuery[Basket]{Query: q, Variables: v}

	type Data struct {
		Seats []struct {
			UnavailableSeats []string `json:"unavailableSeats"`
		} `json:"seats"`
	}

	r, err := httpsRequestPost[GqlResponse[Data]](c, p, nil, nil, b)
	if err != nil {
		return fmt.Errorf("failed to get seats: %v", err)
	}

	fmt.Println(r)
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

	client := RClient{
		schema: "https",
		fqdn:   "www.ryanair.com",
	}

	cAuth, err := client.accountLogin(email, password)
	catchErr(err)

	bookingId, err := client.getBookingId(cAuth)
	catchErr(err)

	bAuth, err := client.getBookingById(cAuth, bookingId)
	catchErr(err)

	basketId, err := client.createBasket(bAuth)
	catchErr(err)

	err = client.getSeatsQuery(basketId)
	catchErr(err)
}
