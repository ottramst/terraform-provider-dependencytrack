# API Key Authentication
provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"
  api_key  = "your-api-key-here"
}

# Username/Password Authentication
provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"
  username = "admin"
  password = "admin123"
}
