package main

import (
	"runtime"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.10.0", "1.9.9", 1},
		{"v2.0.0", "1.99.99", 1},
		{"1.2.3-beta", "1.2.3", 0},
	}
	for _, test := range cases {
		got, err := compareVersions(test.a, test.b)
		if err != nil {
			t.Errorf("compareVersions(%q, %q): %v", test.a, test.b, err)
			continue
		}
		if got != test.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
		}
	}

	for _, invalid := range []string{"", "latest", "1.x.0", "1..0"} {
		if _, err := normalizeVersion(invalid); err == nil {
			t.Errorf("normalizeVersion(%q) should fail", invalid)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	newer, err := isNewerVersion("v1.2.0", "1.1.0")
	if err != nil || !newer {
		t.Fatalf("1.2.0 should be newer than 1.1.0: %v %v", newer, err)
	}
	newer, err = isNewerVersion("1.1.0", "1.2.0")
	if err != nil || newer {
		t.Fatalf("1.1.0 should not be newer than 1.2.0: %v %v", newer, err)
	}
}

func TestSelectUpdateAsset(t *testing.T) {
	assets := []githubAsset{
		{Name: "SHA256SUMS.txt"},
		{Name: "Stockbook-x86_64.AppImage"},
		{Name: "stockbook.exe"},
		{Name: "Stockbook-windows-amd64.zip"},
	}
	asset, err := selectUpdateAsset(assets)
	if err != nil {
		t.Fatal(err)
	}
	switch runtime.GOOS {
	case "linux":
		if asset.Name != "Stockbook-x86_64.AppImage" {
			t.Errorf("linux asset = %q", asset.Name)
		}
	case "windows":
		if asset.Name != "stockbook.exe" {
			t.Errorf("windows asset = %q", asset.Name)
		}
	}
}

func TestAllowedDownloadURL(t *testing.T) {
	valid := []string{
		"https://github.com/munsif-solkar/stockbook-desktop/releases/download/v1.0.0/stockbook.exe",
		"https://objects.githubusercontent.com/some/asset",
		"https://release-assets.githubusercontent.com/some/asset",
	}
	for _, value := range valid {
		if !allowedDownloadURL(value) {
			t.Errorf("%q should be allowed", value)
		}
	}
	invalid := []string{
		"http://github.com/file",
		"https://example.com/file",
		"https://github.com.evil.example/file",
		"file:///etc/passwd",
	}
	for _, value := range invalid {
		if allowedDownloadURL(value) {
			t.Errorf("%q should be rejected", value)
		}
	}
}
