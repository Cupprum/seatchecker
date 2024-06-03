package main

import (
	"context"
)

type Dagger struct{}

func (m *Dagger) Deploy(ctx context.Context) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithExec([]string{"echo", "test123"}).
		Stdout(ctx)
}
