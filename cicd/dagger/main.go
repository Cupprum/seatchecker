package main

import (
	"context"
	"fmt"
	"time"
)

type Cicd struct{}

func Logic(src *Directory) *Directory {
	base := dag.Container().From("golang:latest")
	// Install packages required to package Lambda.
	base = base.
		WithExec([]string{"apt", "update"}).
		WithExec([]string{"apt", "install", "zip", "-y"})
	// Configure source directory.
	base = base.
		WithDirectory("/src", src).
		WithWorkdir("/src")
	// Configure Lambda runtime.
	base = base.
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "arm64")
	// Add cache for Go.
	base = base.
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod-logic")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", dag.CacheVolume("go-build-logic")).
		WithEnvVariable("GOCACHE", "/go/build-cache")
	base = base.WithExec([]string{"go", "install"})

	base.WithExec([]string{"go", "test"})

	build := base.WithExec([]string{"go", "build",
		"-tags", "lambda.norpc", // Do not include RPC part of library.
		"-o", "seatchecker",
		"-ldflags", "-w", // Reduce size of output binary.
		"seatchecker.go"})

	out := build.
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", "seatchecker", "/out/bootstrap"}). // Lambda requires it to be called bootstrap.
		WithWorkdir("/out").
		WithExec([]string{"zip", "seatchecker.zip", "bootstrap"})

	return out.Directory("/out")
}

func Infra(out *Directory, infra *Directory,
	ak *Secret, sk *Secret,
) *Container {
	base := dag.Container().From("hashicorp/terraform:latest")
	// Configure source directories.
	base = base.
		WithDirectory("/out", out).
		WithDirectory("/infra", infra).
		WithWorkdir("/infra")
	// Configure AWS credentials.
	base = base.
		WithSecretVariable("AWS_ACCESS_KEY_ID", ak).
		WithSecretVariable("AWS_SECRET_ACCESS_KEY", sk).
		WithEnvVariable("AWS_REGION", "eu-central-1")
	// Configure cache for Terraform plugins.
	base = base.
		WithMountedCache("/infra/terraform.d/plugins", dag.CacheVolume("terraform-plugins"))

	tf := base.WithExec([]string{"init"})

	// Ensure that Terraform apply operation is never cached.
	epoch := fmt.Sprintf("%d", time.Now().Unix())
	tf = tf.
		WithEnvVariable("CACHEBUSTER", epoch).
		WithExec([]string{"apply", "-auto-approve"})

	return tf
}

func (m *Cicd) Run(ctx context.Context,
	logic *Directory, infra *Directory,
	access_key *Secret, secret_key *Secret,
) (string, error) {
	out := Logic(logic)
	i := Infra(out, infra, access_key, secret_key)

	return i.Stdout(ctx)
}
