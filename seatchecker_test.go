// package main

// import (
// 	"regexp"
// 	"testing"
// )

// // TestHelloName calls greetings.Hello with a name, checking
// // for a valid return value.
// func TestHelloName(t *testing.T) {
// 	name := "Gladys"
// 	want := regexp.MustCompile(`\b` + name + `\b`)
// 	msg, err := Hello("Gladys")
// 	if !want.MatchString(msg) || err != nil {
// 		t.Fatalf(`Hello("Gladys") = %q, %v, want match for %#q, nil`, msg, err, want)
// 	}
// }

// // TestHelloEmpty calls greetings.Hello with an empty string,
// // checking for an error.
//
//	func TestHelloEmpty(t *testing.T) {
//		msg, err := Hello("")
//		if msg != "" || err == nil {
//			t.Fatalf(`Hello("") = %q, %v, want "", error`, msg, err)
//		}
//	}
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountLogin(t *testing.T) {

	type CAuth struct {
		CustomerID string `json:"customerId"`
		Token      string `json:"token"`
	}

	b := CAuth{
		"abc123",
		"super_secret_token",
	}
	res, _ := json.Marshal(b)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fmt.Fprintln(w, res)
		enc := json.NewEncoder(w)
		enc.Encode(res)
	}))
	defer ts.Close()

	client := RClient{schema: "http", fqdn: ts.URL}

	cAuth, err := client.accountLogin("john@doe.com", "secret")
	if err != nil { // TODO: check also cAuth
		t.Fatalf("failed to get account login: %v", err)
	}

	fmt.Println("rafds")
	fmt.Println(cAuth)
	fmt.Println(err)
}
