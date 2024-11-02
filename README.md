# seatchecker

![GitHub Action](https://github.com/Cupprum/seatchecker/actions/workflows/dagger.yml/badge.svg?branch=main) [![Dagger badge](https://badgen.net/badge/Dagger/o11y/blue?icon=terminal)](https://dagger.cloud/Cupprum/traces) [![Dagger badge](https://badgen.net/badge/Honeycomb/o11y/blue?icon=terminal)](https://ui.eu1.honeycomb.io/login)

## Functionality of this project:
Query Ryanair to check number of empty Window, Middle and Aisle seats on upcomming flight.

The random algorithm Ryanair uses for seat allocation, probably is not completely random. From my experience, it prefers giving out Middle seats when I check in sooner. This is probably not a coinsidence. 
Random allocation means no sale for Ryanair, so if they are not making money on you, at least they want to loose as little money as possible.

If people receive a bad seats, there is a chance they will pay extra for a Windows/Aisle seat.

This API starts an AWS Step Function, which regularly sends notifications to [ntfy.sh](https://ntfy.sh) so the user can check in for their flight when all the Middle seats are already in use and therefore receive a Windows/Aisle seat.

## The real purpose of this project:
Sam wanted to play around with new shiny things and was missing Golang.

## Parts of the project:
* .devcontainer -> development environment
* .github -> deployment pipeline definition
* chart -> chart of architecture for README
* cicd -> dagger deployment pipeline written in golang
* infra -> AWS Infrastructure deployed by Terraform
* logic -> AWS Lambda function writen in golang
* README -> the only piece of documentation

## Architecture

![seatchecker chart](chart/seatchecker.svg)

## How to work on the project:

This section contains short descriptions and set of commands to get started.

### Logic
The whole logic is basically a golang program.

Get to the directory: `cd logic/seatchecker`

**Install dependencies:** `go install`

**Run unit tests:** `go test`

**Run locally:** `go run .`

### CICD
Deployment pipeline is written in Dagger. Dagger executes pipelines in Docker, therefore they can also be executed in local environments, not just directly in GitHub Actions.

Get to the directory: `cd cicd/dagger`

**Install dependencies:** `go install`

**Generate dagger code:** `dagger develop`

### Infra
Infrastructure is written in Terraform.

Ideally all changes to infrastructure should be done by using the adhering dagger pipelines.

**A GitHub Action is configured to automatically apply changes** to Infrastructure by using the Dagger pipeline.

Get to the directory: `cd infra`

**Plan changes:**
```
dagger call plan \
    --seatchecker="../../logic/seatchecker" \
    --infra="../../infra" \
    --access_key="env:SEATCHECKER_AWS_ACCESS_KEY_ID" \
    --secret_key="env:SEATCHECKER_AWS_SECRET_ACCESS_KEY"
```

**Apply changes:**
```
dagger call apply \
     --seatchecker="../../logic/seatchecker" \
     --infra="../../infra" \
     --access_key="env:SEATCHECKER_AWS_ACCESS_KEY_ID" \
     --secret_key="env:SEATCHECKER_AWS_SECRET_ACCESS_KEY" \
     --honeycomb_api_key="env:SEATCHECKER_HONEYCOMB_API_KEY"
```

**Destroy infrastructure:**
```
dagger call destroy \
    --seatchecker="../../logic/seatchecker" \
    --infra="../../infra" \
    --access_key="env:SEATCHECKER_AWS_ACCESS_KEY_ID" \
    --secret_key="env:SEATCHECKER_AWS_SECRET_ACCESS_KEY"
```