// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// SBOMFolderPath retrieves the folder path from an environment variable with a fallback
const (
	defaultSBOMFolder = "../testdata/github" // Update this if needed
)

func SBOMFolderPath() string {
	if path := os.Getenv("SBOMMV_TEST_FOLDER"); path != "" {
		return path
	}
	return defaultSBOMFolder
}

// mockGitHubRelease mimics a GitHub API release response
type mockGitHubRelease struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// upload from github_api to dtrack (without providing project name and project version)
func TestUploadGithubAPIToDTrack(t *testing.T) {
	folderPath := SBOMFolderPath()
	if folderPath == "" {
		t.Fatal("SBOMMV_TEST_FOLDER not set")
	}
	sbomFile := folderPath + "/sbomqs_github_api_sbom.spdx.json"
	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		t.Fatalf("GitHub SBOM file %s does not exist", sbomFile)
	}

	sbomData, err := os.ReadFile(sbomFile)
	if err != nil {
		t.Fatalf("Failed to read SBOM file %s: %v", sbomFile, err)
	}

	// mock github server
	githubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/repos/interlynk-io/sbomqs/dependency-graph/sbom" {
			response := map[string]json.RawMessage{"sbom": sbomData}
			w.Header().Set("Content-Type", "application/vnd.github.v3+json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found","documentation_url":"https://docs.github.com/rest","status":"404"}`))
	}))
	defer githubServer.Close()

	// mock dependency server
	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mock "/health" api
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}

		// mock "/api/version" api
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {

			// return empty project list
			w.Write([]byte(`[]`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "interlynk-io/sbomqs-latest-dependency-graph-sbom.json", "version": "latest"}`))
			return
		}

		// mock "/api/v1/bom" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=github",
		"--in-github-url=" + githubServer.URL + "/interlynk-io/sbomqs",
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"-D",
	})

	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	t.Log("Before Execute")
	err = cmd.Execute()

	w.Close()
	os.Stdout = origStdout

	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	assert.NoError(t, err, "Expected successful transfer")
	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "github"}`, "Expected Input adapter")

	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")

	assert.Contains(t, outBuf.String(), "Fetching SBOM Details", "Expected SBOM fetching message")
	assert.Contains(t, outBuf.String(), `{"repository": "sbomqs", "owner": "interlynk-io", "repo_url": "https://github.com/interlynk-io/sbomqs"}`, "Expected SBOM fetching details")

	assert.Contains(t, outBuf.String(), "Fetching SBOM via GitHub API", "Expected github fetch method")
	assert.Contains(t, outBuf.String(), "Total SBOMs fetched from all repos", "Expected SBOM fetched message")
	assert.Contains(t, outBuf.String(), `{"count": 1}`, "Expected total SBOM fetched details")

	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")

	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), `{"name": "interlynk-io/sbomqs-latest-dependency-graph-sbom.json", "version": "latest"}`, "Expected new project details")

	assert.Contains(t, outBuf.String(), "Processing Uploading SBOMs", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"project": "interlynk-io/sbomqs-latest-dependency-graph-sbom.json", "version": "latest"}`, "Expected project upload processsing")

	assert.Contains(t, outBuf.String(), "Fetched SBOM successfully", "Expected fetch success")
	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")

	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}

// upload from github_api to dtrack with a project name
func TestUploadGithubAPIToDTrack_WithProjectName(t *testing.T) {
	folderPath := SBOMFolderPath()
	if folderPath == "" {
		t.Fatal("SBOMMV_TEST_FOLDER not set")
	}
	sbomFile := folderPath + "/sbomqs_github_api_sbom.spdx.json"
	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		t.Fatalf("GitHub SBOM file %s does not exist", sbomFile)
	}

	sbomData, err := os.ReadFile(sbomFile)
	if err != nil {
		t.Fatalf("Failed to read SBOM file %s: %v", sbomFile, err)
	}

	// mock github server
	githubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/repos/interlynk-io/sbomqs/dependency-graph/sbom" {
			response := map[string]json.RawMessage{"sbom": sbomData}
			w.Header().Set("Content-Type", "application/vnd.github.v3+json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found","documentation_url":"https://docs.github.com/rest","status":"404"}`))
	}))
	defer githubServer.Close()

	// mock dependency server
	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mock "/health" api
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}

		// mock "/api/version" api
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {

			// return empty project list
			w.Write([]byte(`[]`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "sbommv_latest_github_api_to_dtrack-latest", "version": "latest"}`))
			return
		}

		// mock "/api/v1/bom" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=github",
		"--in-github-url=" + githubServer.URL + "/interlynk-io/sbomqs",
		"--in-github-method=api",
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"--out-dtrack-project-name=sbommv_latest_github_api_to_dtrack",
		"-D",
	})

	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	t.Log("Before Execute")
	err = cmd.Execute()

	w.Close()
	os.Stdout = origStdout

	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	assert.NoError(t, err, "Expected successful transfer")

	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "github"}`, "Expected Input adapter")

	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")

	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")

	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), `{"project": "sbommv_latest_github_api_to_dtrack-latest", "version": "latest", "uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1"}`, "Expected new project details")

	assert.Contains(t, outBuf.String(), "Processing Uploading SBOMs", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"project": "sbommv_latest_github_api_to_dtrack-latest", "version": "latest"}`, "Expected project upload processsing")

	assert.Contains(t, outBuf.String(), "Fetched SBOM successfully", "Expected fetch success")
	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")

	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}

// upload from github_api to dtrack with a project name and project version
func TestUploadGithubAPIToDTrack_WithProjectNameAndVersion(t *testing.T) {
	folderPath := SBOMFolderPath()
	if folderPath == "" {
		t.Fatal("SBOMMV_TEST_FOLDER not set")
	}
	sbomFile := folderPath + "/sbomqs_github_api_sbom.spdx.json"
	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		t.Fatalf("GitHub SBOM file %s does not exist", sbomFile)
	}

	sbomData, err := os.ReadFile(sbomFile)
	if err != nil {
		t.Fatalf("Failed to read SBOM file %s: %v", sbomFile, err)
	}

	// mock github server
	githubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/repos/interlynk-io/sbomqs/dependency-graph/sbom" {
			response := map[string]json.RawMessage{"sbom": sbomData}
			w.Header().Set("Content-Type", "application/vnd.github.v3+json")
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found","documentation_url":"https://docs.github.com/rest","status":"404"}`))
	}))
	defer githubServer.Close()

	// mock dependency server
	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mock "/health" api
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}

		// mock "/api/version" api
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {

			// return empty project list
			w.Write([]byte(`[]`))
			return
		}

		// mock "/api/v1/project" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "test-project-v1.0.1", "version": "v1.0.1"}`))
			return
		}

		// mock "/api/v1/bom" api
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=github",
		"--in-github-url=" + githubServer.URL + "/interlynk-io/sbomqs",
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"--out-dtrack-project-name=test-project",
		"--out-dtrack-project-version=v1.0.1",
		"-D",
	})

	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	t.Log("Before Execute")
	err = cmd.Execute()

	w.Close()
	os.Stdout = origStdout

	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	assert.NoError(t, err, "Expected successful transfer")
	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "github"}`, "Expected Input adapter")

	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")

	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")

	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), `{"project": "test-project-v1.0.1", "version": "v1.0.1", "uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1"}`, "Expected new project details")

	assert.Contains(t, outBuf.String(), "Processing Uploading SBOMs", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"project": "test-project-v1.0.1", "version": "v1.0.1"}`, "Expected project upload processsing")

	assert.Contains(t, outBuf.String(), "Fetched SBOM successfully", "Expected fetch success")
	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")

	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}

// TEST:  uploaded folder to dtrack (without project name and project version)
func TestUploadFolderToDTrack(t *testing.T) {
	// Check if SBOM folder exists
	folderPath := SBOMFolderPath()
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		t.Skipf("SBOM folder %s does not exist, skipping test", folderPath)
	}

	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`[]`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "com.github.interlynk-io/sbomqs-main", "version": "latest"}`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	// Set environment variable for Dependency-Track API key
	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	// Setup command
	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=folder",
		"--in-folder-path=" + folderPath,
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"-D",
	})

	// Set up buffers for capturing output
	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	// Create a pipe to capture os.Stdout (logger output)
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Redirect command output/error (optional, for completeness)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	// Run the command
	t.Log("Before Execute")
	err = cmd.Execute()

	// Close the writer and restore os.Stdout
	w.Close()
	os.Stdout = origStdout

	// Copy pipe contents to outBuf
	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	// Assertions
	assert.NoError(t, err, "Expected no error for valid SBOM transfer")
	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "folder"}`, "Expected Input adapter")

	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")

	assert.Contains(t, outBuf.String(), "Locally SBOM located folder", "Expected sbom fetching")

	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")

	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), `{"project": "com.github.interlynk-io/sbomqs-main", "version": "latest", "uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1"}`, "Expected new project details")

	assert.Contains(t, outBuf.String(), "Processing Uploading SBOMs", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"project": "com.github.interlynk-io/sbomqs-main", "version": "latest"}`, "Expected project upload processsing")

	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}

// TEST: uploaded folder to dtrack with a provided project name
func TestUploadFolderToDTrack_WithProjectName(t *testing.T) {
	// Check if SBOM folder exists
	folderPath := SBOMFolderPath()
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		t.Skipf("SBOM folder %s does not exist, skipping test", folderPath)
	}

	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`[]`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "test-project-main", "version": "latest"}`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	// Set environment variable for Dependency-Track API key
	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	// Setup command
	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=folder",
		"--in-folder-path=" + folderPath,
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"--out-dtrack-project-name=test-project",
		"--processing-mode=sequential",
		"-D",
	})

	// Set up buffers for capturing output
	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	// Create a pipe to capture os.Stdout (logger output)
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Redirect command output/error (optional, for completeness)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	// Run the command
	t.Log("Before Execute")
	err = cmd.Execute()

	// Close the writer and restore os.Stdout
	w.Close()
	os.Stdout = origStdout

	// Copy pipe contents to outBuf
	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	// Assertions
	assert.NoError(t, err, "Expected no error for valid SBOM transfer")
	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "folder"}`, "Expected Input adapter")
	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")
	assert.Contains(t, outBuf.String(), "Locally SBOM located folder", "Expected sbom fetching")
	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")
	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}

// TEST:  uploaded folder to dtrack with a project name and version
func TestUploadFolderToDTrack_WithProjectNameAndVersion(t *testing.T) {
	// Check if SBOM folder exists
	folderPath := SBOMFolderPath()
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		t.Skipf("SBOM folder %s does not exist, skipping test", folderPath)
	}

	dtrackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/version" {
			w.Write([]byte(`{"version":"4.12.5","timestamp":"2025-02-17T15:58:13Z","uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`[]`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/project" {
			w.Write([]byte(`{"uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1", "name": "test-project-v1.0.1", "version": "v1.0.1"}`))
			return
		}
		if r.Method == "PUT" && r.URL.Path == "/api/v1/bom" {
			w.WriteHeader(http.StatusOK)
			token := uuid.New().String()
			response := fmt.Sprintf(`{"token":"%s"}`, token)
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid endpoint"}`))
	}))
	defer dtrackServer.Close()

	// Set environment variable for Dependency-Track API key
	os.Setenv("DTRACK_API_KEY", "dummy-key")
	defer os.Unsetenv("DTRACK_API_KEY")

	// Setup command
	cmd := rootCmd
	cmd.SetArgs([]string{
		"transfer",
		"--input-adapter=folder",
		"--in-folder-path=" + folderPath,
		"--output-adapter=dtrack",
		"--out-dtrack-url=" + dtrackServer.URL,
		"--out-dtrack-project-name=test-project",
		"--out-dtrack-project-version=v1.0.1",
		"--processing-mode=sequential",
		"-D",
	})

	// Set up buffers for capturing output
	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	// Create a pipe to capture os.Stdout (logger output)
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Redirect command output/error (optional, for completeness)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)

	// Run the command
	t.Log("Before Execute")
	err = cmd.Execute()

	// Close the writer and restore os.Stdout
	w.Close()
	os.Stdout = origStdout

	// Copy pipe contents to outBuf
	_, err = io.Copy(outBuf, r)
	if err != nil {
		t.Fatalf("Failed to copy pipe output: %v", err)
	}

	t.Logf("Execute error: %v", err)
	t.Log("Output:", outBuf.String())
	t.Log("Errors:", errBuf.String())

	// Assertions
	assert.NoError(t, err, "Expected no error for valid SBOM transfer")
	assert.Contains(t, outBuf.String(), "Initializing Input Adapter", "Expected Input adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"InputAdapter": "folder"}`, "Expected Input adapter")

	assert.Contains(t, outBuf.String(), "Initializing Output Adapter", "Expected Output adapter Initialization")
	assert.Contains(t, outBuf.String(), `{"OutputAdapter": "dtrack"}`, "Expected Output adapter")

	assert.Contains(t, outBuf.String(), "Locally SBOM located folder", "Expected sbom fetching")

	assert.Contains(t, outBuf.String(), "Initializing SBOMs uploading to Dependency-Track sequentially", "Expected upload start")

	assert.Contains(t, outBuf.String(), "New project will be created", "Expected project creation")
	assert.Contains(t, outBuf.String(), `{"project": "test-project-v1.0.1", "version": "v1.0.1", "uuid": "39a35c94-b369-46e2-b67f-aed235cbc9c1"}`, "Expected new project details")

	assert.Contains(t, outBuf.String(), "Processing Uploading SBOMs", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"project": "test-project-v1.0.1", "version": "v1.0.1"}`, "Expected project upload processsing")

	assert.Contains(t, outBuf.String(), "upload", "Expected successful upload completion")
	assert.Contains(t, outBuf.String(), `{"sboms": 1, "success": 1, "failed": 0}`, "Expected upload counts")
}
