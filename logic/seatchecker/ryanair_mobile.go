package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type RAuth struct {
	CustomerID string `json:"customerId"`
	Token      string `json:"token"`
}

func (c Client) accountLogin(ctx context.Context, email string, password string) (RAuth, error) {
	ctx, span := tr.Start(ctx, "ryanair_account_login")
	defer span.End()

	p := "usrprof/v2/accountLogin"

	b := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		email,
		password,
	}

	a, err := httpsRequestPost[RAuth](ctx, c, p, b)
	if err != nil {
		err = fmt.Errorf("failed to get account login: %v", err)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err, trace.WithStackTrace(true))
		return RAuth{}, err
	}

	return a, nil
}
