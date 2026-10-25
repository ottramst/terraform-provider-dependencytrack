# Service account API keys can be imported using service_account/public_id.
# The plain text key cannot be read back, so `key` is null after import.
terraform import dependencytrack_service_account_api_key.example ci-pipeline/a1b2c3d4
