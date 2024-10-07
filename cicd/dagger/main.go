package main

import (
	"context"
	"fmt"
	"time"
)

type Cicd struct{}

func PackageGoLambda(src *Directory, module string) *Directory {
	base := dag.Container().From("golang:latest")
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
		WithMountedCache("/go/pkg/mod", dag.CacheVolume(fmt.Sprintf("go-mod-%s", module))).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", dag.CacheVolume(fmt.Sprintf("go-build-%s", module))).
		WithEnvVariable("GOCACHE", "/go/build-cache")
	base = base.WithExec([]string{"go", "install"})
	// Ensure that build operation is never cached.
	epoch := fmt.Sprintf("%d", time.Now().Unix())
	base = base.WithEnvVariable("CACHEBUSTER", epoch)

	base.WithExec([]string{"go", "test"})

	build := base.WithExec([]string{"go", "build",
		"-tags", "lambda.norpc", // Do not include RPC part of library.
		"-o", module,
		"-ldflags", "-w", // Reduce size of output binary.
		"."})

	out := build.
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", module, "/out/bootstrap"}).
		Directory("/out")

	return out
}

func TerraformContainer(infra *Directory, ak *Secret, sk *Secret) *Container {
	base := dag.Container().From("hashicorp/terraform:latest")
	// Configure source directories.
	base = base.
		WithDirectory("/infra", infra).
		WithWorkdir("/infra")
	// Configure AWS credentials.
	base = base.
		WithSecretVariable("AWS_ACCESS_KEY_ID", ak).
		WithSecretVariable("AWS_SECRET_ACCESS_KEY", sk).
		WithEnvVariable("AWS_REGION", "eu-central-1")
	// Configure cache for Terraform plugins.
	base = base.
		WithEnvVariable("TF_PLUGIN_CACHE_DIR", "/opt/terraform-plugin-dir").
		WithMountedCache("/opt/terraform-plugin-dir", dag.CacheVolume("tf-plugin-cache"))
	base = base.
		WithExec([]string{"init"})

	return base
}

// Apply infrastructure changes.
func (m *Cicd) Apply(
	ctx context.Context,
	// Directory containing the seatchecker lambda source code.
	seatchecker *Directory,
	// Directory containing the infrastructure.
	infra *Directory,
	// AWS Access Key ID.
	access_key *Secret,
	// AWS Secret Access Key.
	secret_key *Secret,
	// Honeycomb.io api key representing the team.
	honeycomb_api_key *Secret,
) (string, error) {
	// Logic.
	sc := PackageGoLambda(seatchecker, "seatchecker")

	// Infra with basic configuration.
	tf := TerraformContainer(infra, access_key, secret_key).
		WithDirectory("/out/seatchecker", sc).
		WithSecretVariable("TF_VAR_honeycomb_api_key", honeycomb_api_key)

	// Ensure that Terraform apply operation is never cached.
	epoch := fmt.Sprintf("%d", time.Now().Unix())
	tf = tf.WithEnvVariable("CACHEBUSTER", epoch)
	tf = tf.WithExec([]string{"apply", "-auto-approve"})

	return tf.Stdout(ctx)
}

// Destroy the infrastructure.
func (m *Cicd) Destroy(
	ctx context.Context,
	// Directory containing the seatchecker lambda source code.
	seatchecker *Directory,
	// Directory containing the infrastructure.
	infra *Directory,
	// AWS Access Key ID.
	access_key *Secret,
	// AWS Secret Access Key.
	secret_key *Secret,
) (string, error) {
	sc := PackageGoLambda(seatchecker, "seatchecker")

	tf := TerraformContainer(infra, access_key, secret_key).
		WithDirectory("/out/seatchecker", sc)
	tf = tf.WithExec([]string{"destroy", "-auto-approve"})

	return tf.Stdout(ctx)
}
