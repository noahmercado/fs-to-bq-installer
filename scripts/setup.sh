#!/usr/bin/env bash

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
CURRENT_USER=$(gcloud config get-value account)
PROJECT_ID=$(gcloud config get-value project)
KEY_PATH="${HOME}/key.json"
CWD=$(pwd)

function create_service_account() {
    SERVICE_ACCOUNT="fs-to-bq-installer@${PROJECT_ID}.iam.gserviceaccount.com"

    (gcloud iam service-accounts describe ${SERVICE_ACCOUNT} &> /dev/null && \
        echo "${SERVICE_ACCOUNT} already exists...") \
        || \
        (echo "Creating fs-to-bq-installer service account in ${PROJECT_ID} ..." && \
        gcloud iam service-accounts create \
        fs-to-bq-installer \
        --display-name="Firestore to BigQuery Installer Service Account")

    gcloud iam service-accounts add-iam-policy-binding \
        ${SERVICE_ACCOUNT} \
        --member="user:${CURRENT_USER}" \
        --role='roles/iam.serviceAccountTokenCreator'

    PROJECT_ROLES=(roles/editor)
    for ROLE in ${PROJECT_ROLES[@]}
    do
        echo "Assigning ${ROLE} to ${SERVICE_ACCOUNT}..."
        gcloud projects add-iam-policy-binding ${PROJECT_ID} \
            --member="serviceAccount:${SERVICE_ACCOUNT}" \
            --role=${ROLE}
    done

    gcloud iam service-accounts keys create \
        ${KEY_PATH} \
        --iam-account ${SERVICE_ACCOUNT}

    echo "Allowing time to propagate..."
    sleep 30
}

set -e
# keep track of the last executed command
trap 'last_command=$current_command; current_command=$BASH_COMMAND' DEBUG
# echo an error message before exiting
trap 'echo "\"${last_command}\" command exited with exit code $?."' EXIT

create_service_account