package main

import (
	"context"
	"fmt"
	"time"
)

type Cicd struct{}

func PackageGoLambda(src *Directory, module string, test bool) *Directory {
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
		WithMountedCache("/go/pkg/mod", dag.CacheVolume(fmt.Sprintf("go-mod-%s", module))).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", dag.CacheVolume(fmt.Sprintf("go-build-%s", module))).
		WithEnvVariable("GOCACHE", "/go/build-cache")
	base = base.WithExec([]string{"go", "install"})

	// TODO: do i need a feature flag here?
	if test {
		base.WithExec([]string{"go", "test"})
	}

	build := base.WithExec([]string{"go", "build",
		"-tags", "lambda.norpc", // Do not include RPC part of library.
		"-o", module,
		"-ldflags", "-w", // Reduce size of output binary.
		fmt.Sprintf("%s.go", module)})

	out := build.
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"mv", module, "/out/bootstrap"}). // Lambda requires it to be called bootstrap.
		WithWorkdir("/out").
		WithExec([]string{"zip", fmt.Sprintf("%s.zip", module), "bootstrap"}).
		WithExec([]string{"rm", "bootstrap"})

	return out.Directory("/out")
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
		WithEnvVariable("TF_PLUGIN_CACHE_DIR", "/infra/terraform.d/plugins").
		WithMountedCache("/infra/terraform.d/plugins", dag.CacheVolume("terraform-plugins"))
	base = base.
		WithExec([]string{"init"})

	return base
}

func (m *Cicd) Apply(
	ctx context.Context,
	seatchecker *Directory,
	notifier *Directory,
	infra *Directory,
	access_key *Secret,
	secret_key *Secret,
) (string, error) {
	// Logic.
	sc := PackageGoLambda(seatchecker, "seatchecker", true)
	nt := PackageGoLambda(notifier, "notifier", false)

	// Combine outputs.
	out := dag.Container().From("golang:latest").
		WithDirectory("/in/seatchecker", sc).
		WithDirectory("/in/notifier", nt).
		WithExec([]string{"mkdir", "/out"}).
		WithExec([]string{"cp", "-r", "/in/seatchecker/.", "/out"}).
		WithExec([]string{"cp", "-r", "/in/notifier/.", "/out"}).
		Directory("/out")

	// Infra.
	tf := TerraformContainer(infra, access_key, secret_key).
		WithDirectory("/out", out)

	// Ensure that Terraform apply operation is never cached.
	epoch := fmt.Sprintf("%d", time.Now().Unix())
	tf = tf.WithEnvVariable("CACHEBUSTER", epoch)
	tf = tf.WithExec([]string{"apply", "-auto-approve"})

	return tf.Stdout(ctx)
}

func (m *Cicd) Destroy(
	ctx context.Context,
	infra *Directory,
	access_key *Secret,
	secret_key *Secret,
) (string, error) {
	tf := TerraformContainer(infra, access_key, secret_key)
	tf = tf.WithExec([]string{"destroy", "-auto-approve"})

	return tf.Stdout(ctx)
}
