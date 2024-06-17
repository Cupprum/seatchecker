#!/usr/bin/env bash

# Create Infra
dagger call apply \
    --seatchecker="../../logic/seatchecker" \
    --notifier="../../logic/notifier" \
    --infra="../../infra" \
    --access_key="env:GREENMO_AWS_ACCESS_KEY_ID" \
    --secret_key="env:GREENMO_AWS_SECRET_ACCESS_KEY" -vvv

# # Destroy Infra
# dagger call destroy \
#     --infra="../../infra" \
#     --access_key="env:GREENMO_AWS_ACCESS_KEY_ID" \
#     --secret_key="env:GREENMO_AWS_SECRET_ACCESS_KEY" -vvv
