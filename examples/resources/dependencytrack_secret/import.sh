# Secrets can be imported using their name. The secret value cannot be read
# from the API; the first apply after import re-sets it to the configured value.
terraform import dependencytrack_secret.example my-secret-name
