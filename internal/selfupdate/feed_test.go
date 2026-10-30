package selfupdate

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestFeed(base string, client *http.Client, public ed25519.PublicKey) feed {
	return feed{
		base:   base,
		client: client,
		keys:   map[string]ed25519.PublicKey{fixtureKeyID: public},
	}
}

func TestFeedLoadPinsSignatureToManifestVersionWhenLatestMoves(t *testing.T) {
	public, private := newFixtureKey(t)
	manifestV1 := fixtureManifest()
	dataV1, signatureV1 := signFixture(t, manifestV1, private)
	manifestV2 := fixtureManifest()
	manifestV2.Version = "1.2.4"
	dataV2, signatureV2 := signFixture(t, manifestV2, private)

	latest := dataV1
	byVersion := map[string][]byte{
		"1.2.3": dataV1,
		"1.2.4": dataV2,
	}
	var mu sync.Mutex
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNoAuthHeaders(t, r)
		mu.Lock()
		requested = append(requested, r.URL.Path)
		mu.Unlock()

		switch r.URL.Path {
		case "/latest/download/" + ManifestName:
			mu.Lock()
			current := latest
			latest = byVersion["1.2.4"]
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(current)
		case "/latest/download/" + SignatureName:
			// A floating signature lookup would now return the new release's signature.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(signatureV2)
		case "/download/v1.2.3/" + SignatureName:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(signatureV1)
		case "/download/v1.2.4/" + SignatureName:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(signatureV2)
		case "/download/v1.2.3/" + ManifestName:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(dataV1)
		case "/download/v1.2.4/" + ManifestName:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(dataV2)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	f := newTestFeed(server.URL, server.Client(), public)
	first, err := f.load(context.Background(), "")
	if err != nil {
		t.Fatalf("first load failed after latest moved: %v", err)
	}
	if first.Version != "1.2.3" {
		t.Fatalf("first load version = %q, want 1.2.3", first.Version)
	}
	second, err := f.load(context.Background(), "")
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if second.Version != "1.2.4" {
		t.Fatalf("second load version = %q, want 1.2.4", second.Version)
	}

	mu.Lock()
	gotPaths := append([]string(nil), requested...)
	mu.Unlock()
	wantPaths := []string{
		"/latest/download/" + ManifestName,
		"/download/v1.2.3/" + SignatureName,
		"/latest/download/" + ManifestName,
		"/download/v1.2.4/" + SignatureName,
	}
	if fmt.Sprint(gotPaths) != fmt.Sprint(wantPaths) {
		t.Fatalf("request paths = %v, want %v", gotPaths, wantPaths)
	}
}

func TestFeedLoadRejectsVersionDifferentFromPinnedRelease(t *testing.T) {
	public, private := newFixtureKey(t)
	manifest := fixtureManifest()
	manifest.Version = "1.2.4"
	data, _ := signFixture(t, manifest, private)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download/v1.2.3/"+ManifestName {
			t.Errorf("request path = %q, want pinned manifest path", r.URL.Path)
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()

	_, err := newTestFeed(server.URL, server.Client(), public).load(context.Background(), "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "release version changed") {
		t.Fatalf("load() error = %v, want pinned-version mismatch", err)
	}
}

func TestBeforeLaunchFetchesFreshFeedOnEveryInvocation(t *testing.T) {
	public, private := newFixtureKey(t)
	data, signature := signFixture(t, fixtureManifest(), private)
	var mu sync.Mutex
	manifestRequests, signatureRequests := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertNoAuthHeaders(t, r)
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/latest/download/" + ManifestName:
			manifestRequests++
			_, _ = w.Write(data)
		case "/download/v1.2.3/" + SignatureName:
			signatureRequests++
			_, _ = w.Write(signature)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("CLAUDODEX_NO_AUTO_UPDATE", "")

	manager := &Manager{
		cfg:  Config{Version: "1.2.3", Stderr: io.Discard},
		feed: newTestFeed(server.URL, server.Client(), public),
	}
	for i := 0; i < 2; i++ {
		if _, err := manager.BeforeLaunch(context.Background(), Request{}); err != nil {
			t.Fatalf("BeforeLaunch() call %d failed: %v", i+1, err)
		}
	}
	mu.Lock()
	gotManifestRequests, gotSignatureRequests := manifestRequests, signatureRequests
	mu.Unlock()
	if gotManifestRequests != 2 || gotSignatureRequests != 2 {
		t.Fatalf("feed requests after two launch checks = manifest %d, signature %d; want 2 each", gotManifestRequests, gotSignatureRequests)
	}
}

func TestFeedLoadBoundsManifestAndSignatureResponses(t *testing.T) {
	public, private := newFixtureKey(t)
	data, signature := signFixture(t, fixtureManifest(), private)

	tests := []struct {
		name string
		hand http.HandlerFunc
	}{
		{
			name: "manifest",
			hand: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/download/"+ManifestName {
					_, _ = w.Write(make([]byte, MaxManifestBytes+1))
					return
				}
				_, _ = w.Write(signature)
			},
		},
		{
			name: "signature",
			hand: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/download/"+ManifestName {
					_, _ = w.Write(data)
					return
				}
				_, _ = w.Write(make([]byte, 513))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.hand)
			defer server.Close()
			_, err := newTestFeed(server.URL, server.Client(), public).load(context.Background(), "")
			if err == nil || !strings.Contains(err.Error(), "too large") {
				t.Fatalf("load error = %v, want oversized %s response error", err, tt.name)
			}
		})
	}
}

func TestFeedGetRejectsForbiddenAndRateLimitedResponses(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			f := feed{base: server.URL, client: server.Client()}
			if _, err := f.get(context.Background(), "/metadata", 16); err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("get() error = %v, want HTTP %d failure", err, status)
			}
		})
	}
}

func TestFeedGetHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client := &http.Client{Timeout: 40 * time.Millisecond}
	f := feed{base: server.URL, client: client}

	_, err := f.get(context.Background(), "/slow", 16)
	if err == nil {
		t.Fatal("get() succeeded after the client timeout")
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("get() error = %T %v, want wrapped request timeout", err, err)
	}
}

func TestFeedGetRejectsTruncatedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "64")
		_, _ = io.WriteString(w, "short response")
	}))
	defer server.Close()
	f := feed{base: server.URL, client: server.Client()}

	if _, err := f.get(context.Background(), "/truncated", 128); err == nil {
		t.Fatal("get() accepted a body shorter than Content-Length")
	}
}

func TestAllowedReleaseURL(t *testing.T) {
	tests := []struct {
		url string
		ok  bool
	}{
		{url: "https://github.com/bassner/claudodex/releases", ok: true},
		{url: "https://release-assets.githubusercontent.com/file", ok: true},
		{url: "https://objects.githubusercontent.com/file", ok: true},
		{url: "https://github-releases.githubusercontent.com/file", ok: true},
		{url: "https://github.com:443/file", ok: true},
		{url: "http://github.com/file", ok: false},
		{url: "https://github.com.attacker.invalid/file", ok: false},
		{url: "https://user@github.com/file", ok: false},
		{url: "https://github.com:444/file", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			if got := allowedReleaseURL(u); got != tt.ok {
				t.Fatalf("allowedReleaseURL(%q) = %v, want %v", tt.url, got, tt.ok)
			}
		})
	}
}

func assertNoAuthHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	for _, name := range []string{
		"Authorization",
		"Proxy-Authorization",
		"Cookie",
		"X-API-Key",
		"X-Auth-Token",
		"X-GitHub-Api-Key",
	} {
		if value := r.Header.Get(name); value != "" {
			t.Errorf("release request sent unexpected %s header", name)
		}
	}
}
