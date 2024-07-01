#!/usr/bin/env bash

# Create Infra
dagger call apply \
    --seatchecker="../../logic/seatchecker" \
    --notifier="../../logic/notifier" \
    --infra="../../infra" \
    --access_key="env:SEATCHECKER_AWS_ACCESS_KEY_ID" \
    --secret_key="env:SEATCHECKER_AWS_SECRET_ACCESS_KEY" \
    --ntfy_topic="env:SEATCHECKER_NTFY_TOPIC" \
    --honeycomb_api_key="env:SEATCHECKER_HONEYCOMB_API_KEY"

# # Destroy Infra
# dagger call destroy \
#     --infra="../../infra" \
#     --access_key="env:SEATCHECKER_AWS_ACCESS_KEY_ID" \
#     --secret_key="env:SEATCHECKER_AWS_SECRET_ACCESS_KEY"

