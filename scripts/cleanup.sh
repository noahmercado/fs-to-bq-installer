#!/usr/bin/env bash

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CURRENT_USER=$(gcloud config get-value account)
PROJECT_ID=$(gcloud config get-value project)
CWD=$(pwd)

function delete_keys() {
    SERVICE_ACCOUNT="fs-to-bq-installer@${PROJECT_ID}.iam.gserviceaccount.com"
    # KEYS=$(gcloud iam service-accounts keys list --iam-account ${SERVICE_ACCOUNT} --format="json" | jq '.[] .name')
    KEYS=$(gcloud iam service-accounts keys list --iam-account ${SERVICE_ACCOUNT} --managed-by=user --format="value(KEY_ID)")

    NUM_OF_KEYS=$(echo "$KEYS" | wc -l | tr -d '[:space:]')

    echo "Deleting $NUM_OF_KEYS service account keys for ${SERVICE_ACCOUNT}..."

    for K in $(echo $KEYS | tr "\n" " ")
    do
        echo "Deleting Key ${K}"
        gcloud --quiet iam service-accounts keys delete "$K" --iam-account=${SERVICE_ACCOUNT}
    done
}

delete_keys