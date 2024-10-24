package main

import (
	"context"
	"testing"
)

func TestMain(m *testing.M) {
	defer setupOtel(context.Background())()
	m.Run()
}
