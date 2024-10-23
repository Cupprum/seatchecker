package main

import (
	"fmt"
)

type RAuth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c Client) accountLogin(email string, password string) (RAuth, error) {
	p := "usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpsRequestPost[RAuth](c, p, b)
	if err != nil {
		return RAuth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return a, nil
}
