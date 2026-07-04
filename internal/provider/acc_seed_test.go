package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// This file contains helpers that seed the Dependency-Track instance under
// test with data that cannot be created through the provider itself
// (components, vulnerabilities, BOM uploads), so read-only data sources such
// as dependencytrack_project_findings and dependencytrack_project_violations
// have something real to return.

// testAccAPIDo performs an authenticated JSON request against the
// Dependency-Track instance under test and returns the response status code.
// A non-nil out is decoded from the response body. Transport-level failures
// abort the test.
func testAccAPIDo(t *testing.T, method, path string, body, out any) int {
	t.Helper()

	endpoint := strings.TrimSuffix(os.Getenv("DEPENDENCYTRACK_ENDPOINT"), "/")
	apiKey := os.Getenv("DEPENDENCYTRACK_API_KEY")

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal %s %s request body: %s", method, path, err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, endpoint+path, reqBody)
	if err != nil {
		t.Fatalf("build %s %s request: %s", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform %s %s request: %s", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s response body: %s", method, path, err)
	}

	if out != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			t.Fatalf("decode %s %s response body: %s", method, path, err)
		}
	}

	return resp.StatusCode
}

// testAccSeedPreCheck skips the test when the acceptance-test environment is
// not configured. It mirrors the checks resource.Test performs via TF_ACC and
// testAccPreCheckAPIKey, but runs before resource.Test so seeding helpers can
// call the Dependency-Track API safely.
func testAccSeedPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}
	testAccPreCheckAPIKey(t)
}

// testAccSeedProject creates a project via the API and registers a cleanup
// that deletes it again. It returns the project UUID.
func testAccSeedProject(t *testing.T, name, version string) string {
	t.Helper()

	var project struct {
		UUID string `json:"uuid"`
	}
	status := testAccAPIDo(t, http.MethodPut, "/api/v1/project", map[string]string{
		"name":    name,
		"version": version,
	}, &project)
	if status < 200 || status >= 300 {
		t.Fatalf("creating seed project %q: unexpected status %d", name, status)
	}
	t.Cleanup(func() {
		testAccAPIDo(t, http.MethodDelete, "/api/v1/project/"+project.UUID, nil, nil)
	})

	return project.UUID
}

// testAccSeedProjectWithFinding creates a project containing one component
// with an internal vulnerability (CWE-79 and CWE-89) assigned to it, so the
// project has exactly one unsuppressed finding. It returns the project UUID.
// Vulnerability assignment is synchronous on both DT v4 and v5, so the
// finding is visible as soon as this helper returns.
func testAccSeedProjectWithFinding(t *testing.T) string {
	t.Helper()
	testAccSeedPreCheck(t)

	suffix := randomSuffix()
	projectUUID := testAccSeedProject(t, "tf-acc-findings-"+suffix, "1.0.0")

	var component struct {
		UUID string `json:"uuid"`
	}
	status := testAccAPIDo(t, http.MethodPut, "/api/v1/component/project/"+projectUUID, map[string]string{
		"name":       "tf-acc-vulnerable-component",
		"version":    "1.2.3",
		"classifier": "LIBRARY",
	}, &component)
	if status < 200 || status >= 300 {
		t.Fatalf("creating seed component: unexpected status %d", status)
	}

	vulnID := "INT-TF-ACC-" + suffix
	status = testAccAPIDo(t, http.MethodPut, "/api/v1/vulnerability", map[string]any{
		"vulnId":      vulnID,
		"source":      "INTERNAL",
		"description": "Terraform acceptance test vulnerability",
		"severity":    "HIGH",
		"cwes":        []map[string]int{{"cweId": 79}, {"cweId": 89}},
	}, nil)
	if status < 200 || status >= 300 {
		t.Fatalf("creating seed vulnerability %q: unexpected status %d", vulnID, status)
	}
	t.Cleanup(func() {
		testAccAPIDo(t, http.MethodDelete, "/api/v1/vulnerability/source/INTERNAL/vuln/"+vulnID, nil, nil)
	})

	status = testAccAPIDo(t, http.MethodPost, "/api/v1/vulnerability/source/INTERNAL/vuln/"+vulnID+"/component/"+component.UUID, nil, nil)
	if status < 200 || status >= 300 {
		t.Fatalf("assigning seed vulnerability to component: unexpected status %d", status)
	}

	return projectUUID
}

// testAccSeedProjectWithViolation creates a project whose single component
// (uploaded via a CycloneDX BOM) violates an operational policy, and waits
// until Dependency-Track's policy engine has recorded the violation. BOM
// processing is the only reliable way to trigger policy evaluation on both
// DT v4 and v5, and it is asynchronous, hence the polling. It returns the
// project UUID.
func testAccSeedProjectWithViolation(t *testing.T) string {
	t.Helper()
	testAccSeedPreCheck(t)

	suffix := randomSuffix()
	projectUUID := testAccSeedProject(t, "tf-acc-violations-"+suffix, "1.0.0")

	var policy struct {
		UUID string `json:"uuid"`
	}
	status := testAccAPIDo(t, http.MethodPut, "/api/v1/policy", map[string]string{
		"name":           "tf-acc-violation-policy-" + suffix,
		"operator":       "ANY",
		"violationState": "FAIL",
	}, &policy)
	if status < 200 || status >= 300 {
		t.Fatalf("creating seed policy: unexpected status %d", status)
	}
	t.Cleanup(func() {
		testAccAPIDo(t, http.MethodDelete, "/api/v1/policy/"+policy.UUID, nil, nil)
	})

	status = testAccAPIDo(t, http.MethodPut, "/api/v1/policy/"+policy.UUID+"/condition", map[string]string{
		"subject":  "VERSION",
		"operator": "NUMERIC_EQUAL",
		"value":    "4.5.6",
	}, nil)
	if status < 200 || status >= 300 {
		t.Fatalf("creating seed policy condition: unexpected status %d", status)
	}

	bom := `{"bomFormat":"CycloneDX","specVersion":"1.4","version":1,"components":[{"type":"library","name":"tf-acc-violating-component","version":"4.5.6"}]}`
	status = testAccAPIDo(t, http.MethodPut, "/api/v1/bom", map[string]string{
		"project": projectUUID,
		"bom":     base64.StdEncoding.EncodeToString([]byte(bom)),
	}, nil)
	if status < 200 || status >= 300 {
		t.Fatalf("uploading seed BOM: unexpected status %d", status)
	}

	// Wait for BOM processing and policy evaluation to complete.
	deadline := time.Now().Add(2 * time.Minute)
	for {
		var violations []json.RawMessage
		status = testAccAPIDo(t, http.MethodGet, fmt.Sprintf("/api/v1/violation/project/%s?suppressed=false", projectUUID), nil, &violations)
		if status == http.StatusOK && len(violations) > 0 {
			return projectUUID
		}

		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for a policy violation to appear on seed project %s (last status %d)", projectUUID, status)
		}
		time.Sleep(3 * time.Second)
	}
}
