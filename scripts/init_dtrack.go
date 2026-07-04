package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
)

// waitForHealth checks if the Dependency Track API responds with 200 on
// /api/version. This is used instead of /health/ready because in v5 the
// health endpoints moved to a separate management port (9000) which isn't
// necessarily exposed, whereas /api/version stays on the main API port and
// works on both v4 and v5. Using plain net/http (rather than dtrack.NewClient,
// which eagerly fetches /api/version itself) lets this poll succeed while the
// server is still booting.
// It retries up to maxRetries times with exponential backoff.
func waitForHealth(endpoint string, maxRetries int) error {
	versionURL := endpoint + "/api/version"
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(versionURL)
		if err == nil {
			statusOK := resp.StatusCode == http.StatusOK
			_ = resp.Body.Close()
			if statusOK {
				return nil
			}
		}

		if i < maxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("health check failed after %d attempts: endpoint %s is not responding", maxRetries, endpoint)
}

func main() {
	ctx := context.Background()

	// Initialize dtrack client
	endpoint := os.Getenv("DTRACK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	// Wait for the API to respond with 200 before continuing. This must happen
	// before dtrack.NewClient, which eagerly calls /api/version and would fail
	// while the server is still starting up.
	if err := waitForHealth(endpoint, 20); err != nil {
		log.Fatalf("Dependency Track server is not healthy: %v", err)
	}

	initC, err := dtrack.NewClient(endpoint)
	if err != nil {
		log.Fatalf("Unable to initialize dtrack client: %v", err)
	}

	// Try to force change default admin password
	// If it fails, the password may have already been changed
	var bearerToken string
	err = initC.User.ForceChangePassword(ctx, "admin", "admin", "admin123")
	if err != nil {
		// Try to authenticate with new credentials
		bearerToken, err = initC.User.Login(ctx, "admin", "admin123")
		if err != nil {
			log.Fatalf("Unable to authenticate with either old or new admin credentials: %v", err)
		}
	} else {
		// Authenticate with new admin credentials
		bearerToken, err = initC.User.Login(ctx, "admin", "admin123")
		if err != nil {
			log.Fatalf("Unable to authenticate with new admin credentials: %v", err)
		}
	}

	// Create an authenticated client with new admin credentials
	c, err := dtrack.NewClient(endpoint, dtrack.WithBearerToken(bearerToken))
	if err != nil {
		log.Fatalf("Unable to create authenticated dtrack client: %v", err)
	}

	// Fetch the "Administrators" team UUID
	var adminTeamUUID string
	errTeamFound := errors.New("team found") // Sentinel error to break out of ForEach

	err = dtrack.ForEach(func(po dtrack.PageOptions) (dtrack.Page[dtrack.Team], error) {
		return c.Team.GetAll(ctx, po)
	}, func(t dtrack.Team) error {
		if t.Name == "Administrators" {
			adminTeamUUID = t.UUID.String()
			return errTeamFound // Return sentinel error to stop iteration
		}
		return nil
	})

	// Check if iteration was stopped because team was found
	if err != nil && !errors.Is(err, errTeamFound) {
		log.Fatalf("Unable to fetch Administrators team UUID: %v", err)
	}

	if adminTeamUUID == "" {
		log.Fatalf("Administrators team not found")
	}

	// Convert the adminTeamUUID to UUID type
	teamUUID, err := uuid.Parse(adminTeamUUID)
	if err != nil {
		log.Fatalf("Invalid Administrators team UUID: %v", err)
	}

	// Generate new API key for the Administrators team
	apiKey, err := c.Team.GenerateAPIKey(ctx, teamUUID)
	if err != nil {
		log.Fatalf("Unable to generate API key for Administrators team: %v", err)
	}

	fmt.Println(apiKey.Key)
}
