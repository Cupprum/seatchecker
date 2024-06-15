package main

import (
	"context"
)

type Cicd struct{}

func Logic(src *Directory) *Directory {
	base := dag.Container().
		From("golang:latest").
		WithExec([]string{"apt", "update"}).
		WithExec([]string{"apt", "install", "zip", "-y"}). // Required to package lambda.
		WithEnvVariable("tes", "4").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "arm64").
		WithExec([]string{"go", "install"})

	base.WithExec([]string{"go", "test"})

	build := base.WithExec([]string{"go", "build",
		"-tags", "lambda.norpc", // Do not include RPC part of library.
		"-o", "seatchecker",
		"-ldflags", "-w", // Reduce size of output binary.
		"seatchecker.go"})

	pack := build.
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", "seatchecker", "/out/bootstrap"}). // Lambda requires it to be called bootstrap.
		WithWorkdir("/out").
		WithExec([]string{"zip", "seatchecker.zip", "bootstrap"})

	return pack.Directory("/out")
}

func Infra(out *Directory, infra *Directory,
	ak *Secret, sk *Secret,
) *Container {
	return dag.Container().
		From("hashicorp/terraform:latest").
		WithEnvVariable("tes", "4"). // TODO: increment to force full run
		WithDirectory("/out", out).
		WithDirectory("/infra", infra).
		WithWorkdir("/infra").
		WithSecretVariable("AWS_ACCESS_KEY_ID", ak).
		WithSecretVariable("AWS_SECRET_ACCESS_KEY", sk).
		WithEnvVariable("AWS_REGION", "eu-central-1").
		WithExec([]string{"init"}).
		WithExec([]string{"apply", "-auto-approve"})
}

func (m *Cicd) Run(ctx context.Context,
	logic *Directory, infra *Directory,
	access_key *Secret, secret_key *Secret,
) (string, error) {
	out := Logic(logic)
	i := Infra(out, infra, access_key, secret_key)

	return i.Stdout(ctx)
}
