package selfupdate

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type feed struct {
	client *http.Client
	base   string
	keys   map[string]ed25519.PublicKey
}

func productionFeed() feed {
	return feed{
		base: "https://github.com/" + Repository + "/releases",
		keys: TrustedKeys(),
		client: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || !allowedReleaseURL(req.URL) {
				return errors.New("unexpected release redirect")
			}
			return nil
		}},
	}
}

func allowedReleaseURL(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	switch u.Hostname() {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "github-releases.githubusercontent.com":
		return true
	default:
		return false
	}
}

func (f feed) get(ctx context.Context, path string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Claudodex-release-check")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check GitHub release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release server returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("release response too large")
	}
	return data, nil
}

func (f feed) load(ctx context.Context, version string) (Manifest, error) {
	path := "/latest/download/"
	if version != "" {
		var err error
		version, err = StableVersion(version)
		if err != nil {
			return Manifest{}, err
		}
		path = "/download/v" + version + "/"
	}
	data, err := f.get(ctx, path+ManifestName, MaxManifestBytes)
	if err != nil {
		return Manifest{}, err
	}
	untrusted, err := ParseManifest(data)
	if err != nil && !errors.Is(err, ErrUnsupportedSchema) {
		return Manifest{}, err
	}
	if version != "" && untrusted.Version != version {
		return Manifest{}, errors.New("release version changed")
	}
	sig, err := f.get(ctx, "/download/v"+untrusted.Version+"/"+SignatureName, 512)
	if err != nil {
		return Manifest{}, err
	}
	return VerifyManifest(data, sig, f.keys)
}
