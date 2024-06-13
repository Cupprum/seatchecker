package main

import (
	"context"
)

type Cicd struct{}

func (m *Cicd) Build(source *Directory) *Container {
	return dag.Container().
		From("golang:latest").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{ // "GOOS=linux", "GOARCH=arm64",
			"go", "build",
			"-tags", "lambda.norpc", // TODO: maybe this is useless
			"-o", "seatchecker",
			"seatchecker.go"}).
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", "seatchecker", "/out/seatchecker"})
}

func (m *Cicd) Package(source *Directory) *Container {
	return dag.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "zip"}).
		WithDirectory("/out", source).
		WithWorkdir("/out").
		WithExec([]string{"zip", "seatchecker.zip", "seatchecker"}).
		WithExec([]string{"ls"})
}

func (m *Cicd) Deploy(ctx context.Context, source *Directory) (string, error) {
	build := m.Build(source)
	pack := m.Package(build.Directory("/out"))

	return pack.Stdout(ctx)
}
