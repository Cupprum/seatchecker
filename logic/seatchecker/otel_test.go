package main

import (
	"context"
	"testing"
)

// NOTE: This is executed as a setup before the rest of the test suite.
func TestMain(m *testing.M) {
	defer setupOtel(context.Background())()
	m.Run()
}
