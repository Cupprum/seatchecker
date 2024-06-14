package main

import (
	"context"
	"fmt"
)

type Cicd struct{}

func (m *Cicd) Build(src *Directory) *Container {
	return dag.Container().
		From("golang:latest").
		WithEnvVariable("tes", "4").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "arm64").
		WithExec([]string{"go", "install"}).
		WithExec([]string{"go", "build",
			"-tags", "lambda.norpc", // Do not include RPC part of library.
			"-o", "seatchecker",
			"-ldflags", "-w", // Reduce size of output binary.
			"seatchecker.go"}).
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", "seatchecker", "/out/seatchecker"})
}

func (m *Cicd) Package(out *Directory) *Container {
	return dag.Container().
		From("alpine:latest").
		WithEnvVariable("tes", "4").
		WithExec([]string{"apk", "add", "zip"}).
		WithDirectory("/out", out).
		WithWorkdir("/out").
		WithExec([]string{"mv", "seatchecker", "bootstrap"}). // Lambda requires it to be called bootstrap.
		WithExec([]string{"zip", "seatchecker.zip", "bootstrap"}).
		WithExec([]string{"rm", "bootstrap"}) // Cleanup so the transfered directory is smaller.
}

func (m *Cicd) Deploy(out *Directory, infra *Directory,
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
	fmt.Println("Build.")
	build := m.Build(logic)

	fmt.Println("Package.")
	pack := m.Package(build.Directory("/out"))

	fmt.Println("Deploy.")
	deploy := m.Deploy(pack.Directory("/out"), infra, access_key, secret_key)

	return deploy.Stdout(ctx)
}
