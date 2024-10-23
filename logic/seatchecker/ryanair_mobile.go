package main

import (
	"fmt"
)

type CAuth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c Client) accountLogin(email string, password string) (CAuth, error) {
	p := "usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpsRequestPost[CAuth](c, p, b)
	if err != nil {
		return CAuth{}, fmt.Errorf("failed to get account login: %v", err)
	}

	return a, nil
}
