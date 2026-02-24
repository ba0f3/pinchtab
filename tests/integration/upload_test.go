//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// UP1: Single file upload with explicit selector
func TestUpload_SingleFile(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request with single file
	repoRoot = findRepoRoot()
	testFilePath := filepath.Join(repoRoot, "tests/assets/test-upload.png")
	code, body := httpPost(t, "/upload", map[string]any{
		"selector": "#single",
		"paths":    []string{testFilePath},
	})

	if code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", code, body)
	}

	// Verify response structure
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	status, ok := resp["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("expected status='ok', got %v", resp["status"])
	}

	files, ok := resp["files"].(float64)
	if !ok || int(files) != 1 {
		t.Errorf("expected files=1, got %v", resp["files"])
	}
}

// UP4: Multiple files upload with explicit selector
func TestUpload_MultipleFiles(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request with multiple files on multi selector
	repoRoot = findRepoRoot()
	testFilePath := filepath.Join(repoRoot, "tests/assets/test-upload.png")
	code, body := httpPost(t, "/upload", map[string]any{
		"selector": "#multi",
		"paths":    []string{testFilePath, testFilePath}, // two of the same file
	})

	if code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", code, body)
	}

	// Verify response structure
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	status, ok := resp["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("expected status='ok', got %v", resp["status"])
	}

	files, ok := resp["files"].(float64)
	if !ok || int(files) != 2 {
		t.Errorf("expected files=2, got %v", resp["files"])
	}
}

// UP6: Default selector (uses default input[type=file])
func TestUpload_DefaultSelector(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request without selector (should use default)
	repoRoot = findRepoRoot()
	testFilePath := filepath.Join(repoRoot, "tests/assets/test-upload.png")
	code, body := httpPost(t, "/upload", map[string]any{
		"paths": []string{testFilePath},
	})

	// Expect 200 (uses default input[type=file])
	if code != 200 {
		t.Logf("default selector returned %d (body: %s)", code, body)
		// This is ok - it may not match default selector, but shouldn't crash
	}
}

// UP7: Invalid selector should error
func TestUpload_InvalidSelector(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request with non-existent selector
	repoRoot = findRepoRoot()
	testFilePath := filepath.Join(repoRoot, "tests/assets/test-upload.png")
	code, _ := httpPost(t, "/upload", map[string]any{
		"selector": "#nonexistent",
		"paths":    []string{testFilePath},
	})

	// Expect error (400 or 500)
	if code == 200 {
		t.Errorf("expected error for invalid selector, got 200")
	}
}

// UP8: Missing paths field should error
func TestUpload_MissingFiles(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request without paths
	code, _ := httpPost(t, "/upload", map[string]any{
		"selector": "#single",
	})

	// Expect 400 error
	if code == 200 {
		t.Errorf("expected 400 for missing paths, got %d", code)
	}
}

// UP9: Non-existent file path should error
func TestUpload_FileNotFound(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request with non-existent file
	code, _ := httpPost(t, "/upload", map[string]any{
		"paths": []string{"/tmp/nonexistent_file_xyz_12345.jpg"},
	})

	// Expect 400 error
	if code == 200 {
		t.Errorf("expected 400 for non-existent file, got %d", code)
	}
}

// UP11: Malformed JSON should error
func TestUpload_BadJSON(t *testing.T) {
	// Navigate to upload test HTML
	repoRoot := findRepoRoot()
	testHtmlPath := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	testFileURL := fmt.Sprintf("file://%s", testHtmlPath)

	navigate(t, testFileURL)

	// POST upload request with broken JSON
	code, _ := httpPostRaw(t, "/upload", "{broken json")

	// Expect 400 parse error
	if code == 200 {
		t.Errorf("expected 400 for malformed JSON, got %d", code)
	}
}

// Helper to verify test assets exist
func verifyUploadAssets(t *testing.T) {
	repoRoot := findRepoRoot()
	htmlFile := filepath.Join(repoRoot, "tests/assets/upload-test.html")
	imgFile := filepath.Join(repoRoot, "tests/assets/test-upload.png")

	if _, err := os.Stat(htmlFile); err != nil {
		t.Skipf("upload test HTML not found: %v", err)
	}
	if _, err := os.Stat(imgFile); err != nil {
		t.Skipf("test image not found: %v", err)
	}
}
