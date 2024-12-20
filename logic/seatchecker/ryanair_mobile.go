package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Auth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c Client) accountLogin(ctx context.Context, email string, password string) (Auth, error) {
	ctx, span := tr.Start(ctx, "ryanair_account_login")
	defer span.End()
	span.SetAttributes(attribute.String("email", email)) // NOTE: delete after testing.

	p := "usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpsRequestPost[Auth](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get account login: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return Auth{}, err
	}
	span.AddEvent("Account login successful.")

	return a, nil
}
