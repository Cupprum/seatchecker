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
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "arm64").
		WithExec([]string{"go", "install"}).
		WithExec([]string{"go", "build",
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
		WithExec([]string{"mv", "seatchecker", "bootstrap"}). // Lambda requires it to be called bootstrap.
		WithExec([]string{"zip", "seatchecker.zip", "bootstrap"}).
		WithExec([]string{"rm", "bootstrap"})
}

func (m *Cicd) Deploy(out *Directory, infra *Directory,
	access_key *Secret, secret_key *Secret,
) *Container {
	return dag.Container().
		From("hashicorp/terraform:latest").
		WithEnvVariable("tes", "10").
		WithDirectory("/out", out).
		WithDirectory("/infra", infra).
		WithWorkdir("/infra").
		WithSecretVariable("AWS_ACCESS_KEY_ID", access_key).
		WithSecretVariable("AWS_SECRET_ACCESS_KEY", secret_key).
		WithEnvVariable("AWS_REGION", "eu-central-1").
		WithExec([]string{"init"}).
		WithExec([]string{"apply", "-auto-approve"})
}

func (m *Cicd) Run(ctx context.Context,
	logic *Directory, infra *Directory,
	access_key *Secret, secret_key *Secret,
) (string, error) {
	build := m.Build(logic)
	pack := m.Package(build.Directory("/out"))
	deploy := m.Deploy(pack.Directory("/out"), infra, access_key, secret_key)

	return deploy.Stdout(ctx)
}
