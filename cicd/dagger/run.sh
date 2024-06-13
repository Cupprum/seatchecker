#!/usr/bin/env bash

dagger call run \
    --logic="../../logic" \
    --infra="../../infra" \
    --access_key="env:GREENMO_AWS_ACCESS_KEY_ID" \
    --secret_key="env:GREENMO_AWS_SECRET_ACCESS_KEY"

