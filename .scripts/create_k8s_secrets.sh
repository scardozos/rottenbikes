#!/bin/bash
set -euo pipefail

# Helper function to extract env value from file, stripping any quotes
get_env_val() {
  local file=$1
  local key=$2
  if [ -f "$file" ]; then
    grep "^${key}=" "$file" | cut -d'=' -f2- | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//" || true
  fi
}

create_secret() {
  local env_file=$1
  local secret_name=$2
  local namespace=$3

  if [ ! -f "$env_file" ]; then
    echo "Warning: $env_file not found. Skipping secret creation for $namespace."
    return
  fi

  echo "Reading secrets from $env_file..."
  local db_user
  local db_password
  local email_token
  local hcaptcha_secret

  db_user=$(get_env_val "$env_file" "DB_USER")
  db_password=$(get_env_val "$env_file" "DB_PASSWORD")
  email_token=$(get_env_val "$env_file" "EMAIL_SENDER_TOKEN_MAILTRAP")
  hcaptcha_secret=$(get_env_val "$env_file" "HCAPTCHA_SECRET")

  # Ensure required keys exist
  if [ -z "$db_user" ] || [ -z "$db_password" ] || [ -z "$email_token" ]; then
    echo "Error: Missing required keys (DB_USER, DB_PASSWORD, EMAIL_SENDER_TOKEN_MAILTRAP) in $env_file"
    exit 1
  fi

  echo "Creating/Updating Secret '$secret_name' in namespace '$namespace'..."
  
  # Build the command array
  local cmd=(
    kubectl create secret generic "$secret_name"
    --namespace "$namespace"
    --from-literal=DB_USER="$db_user"
    --from-literal=DB_PASSWORD="$db_password"
    --from-literal=EMAIL_SENDER_TOKEN_MAILTRAP="$email_token"
  )

  # Include HCAPTCHA_SECRET if it's set
  if [ -n "$hcaptcha_secret" ]; then
    cmd+=(--from-literal=HCAPTCHA_SECRET="$hcaptcha_secret")
  fi

  # Add dry-run and apply to make it idempotent
  cmd+=(--dry-run=client -o yaml)

  "${cmd[@]}" | kubectl apply -f -
  echo "Successfully applied secret '$secret_name' in namespace '$namespace'."
  echo
}

# Ensure namespaces exist or let k8s handle it (dry-run doesn't care, but apply will fail if ns doesn't exist)
# Here we assume the namespaces rottenbikes-dev and rottenbikes-prd exist (or will exist when deploying).

# 1. Dev Secret
create_secret ".env.dev" "rottenbikes-secrets-dev" "rottenbikes-dev"

# 2. Prod Secret
create_secret ".env.prod" "rottenbikes-secrets-prd" "rottenbikes-prd"
