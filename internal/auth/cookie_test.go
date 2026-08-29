package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeHeaderValue(t *testing.T) {
	dirty := "SAPISID=12345;\r\n __Secure-1PSID=abc\n\t"
	clean := SanitizeHeaderValue(dirty)
	expected := "SAPISID=12345; __Secure-1PSID=abc"

	if clean != expected {
		t.Fatalf("expected '%s', got '%s'", expected, clean)
	}
}

func TestLoadCookiesJSONArray(t *testing.T) {
	tempDir := t.TempDir()
	cookiePath := filepath.Join(tempDir, "cookies.json")

	jsonArrayData := `[
		{"name": "__Secure-1PSID", "value": "psid_val"},
		{"name": "SAPISID", "value": "sapisid_val"}
	]`

	if err := os.WriteFile(cookiePath, []byte(jsonArrayData), 0644); err != nil {
		t.Fatal(err)
	}

	session := LoadCookies(cookiePath)
	if session.SAPISID != "sapisid_val" {
		t.Errorf("expected SAPISID 'sapisid_val', got '%s'", session.SAPISID)
	}

	expectedHeader := "__Secure-1PSID=psid_val; SAPISID=sapisid_val"
	if session.HeaderValue != expectedHeader {
		t.Errorf("expected header '%s', got '%s'", expectedHeader, session.HeaderValue)
	}
}
