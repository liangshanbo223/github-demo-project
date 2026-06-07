package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/admin8800/s-ui/config"
	"github.com/admin8800/s-ui/database"
	"github.com/admin8800/s-ui/logger"
	"github.com/op/go-logging"
)

func TestWebServer_Integration(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	logger.InitLogger(logging.DEBUG)

	// Setup test environment database
	tempDir, err := os.MkdirTemp("", "s-ui-web-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("SUI_DB_FOLDER", tempDir)

	err = database.InitDB(config.GetDBPath())
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Instantiate server
	s := NewServer()
	engine, err := s.initRouter()
	if err != nil {
		t.Fatalf("initRouter failed: %v", err)
	}

	// 1. Test anonymous access redirect (middleware/NoRoute check)
	reqAnonymous := httptest.NewRequest("GET", "/app/api/status", nil)
	wAnonymous := httptest.NewRecorder()
	engine.ServeHTTP(wAnonymous, reqAnonymous)

	if wAnonymous.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected redirect status 307 for anonymous access, got %d", wAnonymous.Code)
	}
	location := wAnonymous.Header().Get("Location")
	if !strings.HasSuffix(location, "/login") {
		t.Errorf("Expected redirect destination to end with /login, got '%s'", location)
	}

	// 2. Test Login Authentication with Form data
	form := url.Values{}
	form.Add("user", "admin")
	form.Add("pass", "admin")

	reqLogin := httptest.NewRequest("POST", "/app/api/login", strings.NewReader(form.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wLogin := httptest.NewRecorder()
	engine.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("Expected login status 200, got %d. Body: %s", wLogin.Code, wLogin.Body.String())
	}

	// Ensure cookie session is set
	cookies := wLogin.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "s-ui" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("Expected session cookie 's-ui' not found in login response")
	}

	// 3. Test Authorized access to status endpoint
	reqStatus := httptest.NewRequest("GET", "/app/api/status", nil)
	reqStatus.AddCookie(sessionCookie)
	wStatus := httptest.NewRecorder()
	engine.ServeHTTP(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Errorf("Expected status 200 for authorized /app/api/status, got %d", wStatus.Code)
	}

	var statusResp map[string]interface{}
	err = json.Unmarshal(wStatus.Body.Bytes(), &statusResp)
	if err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}
	if !statusResp["success"].(bool) {
		t.Errorf("Expected API success true, got false. Body: %s", wStatus.Body.String())
	}

	// 4. Test Authorized access to Reality scanning endpoints
	reqScanStatus := httptest.NewRequest("GET", "/app/api/scanStatus", nil)
	reqScanStatus.AddCookie(sessionCookie)
	wScanStatus := httptest.NewRecorder()
	engine.ServeHTTP(wScanStatus, reqScanStatus)

	if wScanStatus.Code != http.StatusOK {
		t.Errorf("Expected status 200 for scanStatus, got %d", wScanStatus.Code)
	}

	var scanStatusResp map[string]interface{}
	err = json.Unmarshal(wScanStatus.Body.Bytes(), &scanStatusResp)
	if err != nil {
		t.Fatalf("Failed to parse scanStatus response: %v", err)
	}
	if scanStatusResp["status"] != "success" {
		t.Errorf("Expected status 'success' for scanStatus, got '%s'", scanStatusResp["status"])
	}

	// 5. Test Server Public IP endpoint
	reqServerIp := httptest.NewRequest("GET", "/app/api/serverIp", nil)
	reqServerIp.AddCookie(sessionCookie)
	wServerIp := httptest.NewRecorder()
	engine.ServeHTTP(wServerIp, reqServerIp)

	if wServerIp.Code != http.StatusOK {
		t.Errorf("Expected status 200 for serverIp, got %d", wServerIp.Code)
	}
}
