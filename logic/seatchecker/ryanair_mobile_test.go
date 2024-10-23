package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountLogin(t *testing.T) {
	ctx := context.Background()
	defer setupOtel(ctx)()

	e, p := "john@doe.com", "password"
	rAReq := RAuth{"customerid", "token"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := map[string]any{}
		json.Unmarshal(rawB, &b)
		if e != b["email"] {
			t.Fatalf("wrong email, expected: %v, received: %v", e, b["email"])
		}
		if p != b["password"] {
			t.Fatalf("wrong password, expected: %v, received: %v", e, b["password"])
		}

		// Create fake response
		res, _ := json.Marshal(rAReq)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}
	cARes, err := c.accountLogin(ctx, e, p)
	if err != nil {
		t.Fatalf("failed to get account login: %v", err)
	}
	if rAReq != cARes {
		t.Fatalf("wrong response, expected: %v, received %v", rAReq, cARes)
	}
}
