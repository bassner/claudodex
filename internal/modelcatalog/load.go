package modelcatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bassner/claudodex/internal/auth"
	"github.com/bassner/claudodex/internal/codex"
	"github.com/bassner/claudodex/internal/filelock"
	"github.com/bassner/claudodex/internal/modelconfig"
)

const (
	catalogCacheSchema = 1
	catalogCacheMaxAge = 24 * time.Hour
	catalogCacheLimit  = 8 << 20
	catalogTimeout     = 5 * time.Second
)

type Options struct {
	Home        string
	BaseURL     string
	Version     string
	UserID      string
	Issuer      string
	Credentials codex.Credentials
	HTTPClient  *http.Client
	Requested   modelconfig.Config
	Pins        modelconfig.Pins
	Warnings    io.Writer
}

// Snapshot is one resolved view shared by the launcher and all its proxies.
type Snapshot struct {
	Models    []codex.ModelInfo
	Config    modelconfig.Config
	FetchedAt time.Time
	FromCache bool
}

type cacheScope struct {
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
	Issuer    string `json:"issuer"`
	Backend   string `json:"backend"`
	Protocol  string `json:"protocol"`
	FedRAMP   bool   `json:"fedramp"`
}

// Only the hash of scope is stored. The cache contains no credentials or PII.
type cachedCatalog struct {
	Schema     int               `json:"schema"`
	Scope      string            `json:"scope"`
	ObservedAt time.Time         `json:"observed_at"`
	FetchedAt  time.Time         `json:"fetched_at"`
	Invalid    bool              `json:"invalid,omitempty"`
	Models     []codex.ModelInfo `json:"models,omitempty"`
}

func NormalizeBaseURL(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		value = codex.DefaultBaseURL
	}
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "http" && u.Scheme != "https") || u.Opaque != "" {
		return "", fmt.Errorf("invalid Codex backend URL")
	}
	u.Host = strings.ToLower(u.Host)
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = u.Hostname()
		if strings.Contains(u.Host, ":") {
			u.Host = "[" + u.Host + "]"
		}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u.String(), nil
}

func Load(ctx context.Context, options Options) (Snapshot, error) {
	return loadAt(ctx, options, time.Now)
}

func loadAt(ctx context.Context, options Options, now func() time.Time) (Snapshot, error) {
	baseURL, err := NormalizeBaseURL(options.BaseURL)
	if err != nil {
		return Snapshot{}, err
	}
	// Reject credential destination errors before consulting any cached data.
	if err := codex.CheckCredentialDestination(options.Credentials, baseURL); err != nil {
		return Snapshot{}, err
	}
	protocol := codex.ModelsClientVersion(options.Version)
	key := scopeKey(options, baseURL, protocol)
	observedAt := now().UTC()
	fetchCtx, cancel := context.WithTimeout(ctx, catalogTimeout)
	models, fetchErr := (codex.Client{BaseURL: baseURL, Version: options.Version, HTTPClient: options.HTTPClient}).FetchModels(fetchCtx, options.Credentials)
	cancel()
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if fetchErr == nil {
		resolved, resolveErr := Resolve(models, options.Requested, options.Pins, protocol)
		entry := cachedCatalog{Schema: catalogCacheSchema, Scope: key, ObservedAt: observedAt, FetchedAt: now().UTC(), Models: models}
		if err := ValidateCatalog(models); err != nil {
			// Persist rejection rather than let a later outage resurrect old models.
			entry.Invalid, entry.Models = true, nil
		}
		if key != "" {
			persistCache(options, key, entry)
		}
		if resolveErr != nil {
			return Snapshot{}, resolveErr
		}
		return Snapshot{Models: models, Config: resolved, FetchedAt: entry.FetchedAt}, nil
	}
	if !temporaryCatalogError(fetchErr) {
		if key != "" {
			persistCache(options, key, cachedCatalog{Schema: catalogCacheSchema, Scope: key,
				ObservedAt: observedAt, FetchedAt: now().UTC(), Invalid: true})
		}
		return Snapshot{}, fmt.Errorf("fetch Codex model metadata: %w", fetchErr)
	}
	if key == "" {
		return Snapshot{}, fmt.Errorf("fetch Codex model metadata: %w (a cache requires known user and account identities)", fetchErr)
	}
	entry, err := readCache(options.Home, key, now().UTC())
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch Codex model metadata: %w (no recent usable cached catalog)", fetchErr)
	}
	resolved, err := Resolve(entry.Models, options.Requested, options.Pins, protocol)
	if err != nil {
		return Snapshot{}, fmt.Errorf("cached Codex model metadata cannot satisfy this launch: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if options.Warnings != nil {
		age := now().Sub(entry.FetchedAt).Round(time.Second)
		fmt.Fprintf(options.Warnings, "Claudodex: using model metadata cached %s ago after a temporary network failure.\n", age)
	}
	return Snapshot{Models: entry.Models, Config: resolved, FetchedAt: entry.FetchedAt, FromCache: true}, nil
}

func scopeKey(options Options, backend, protocol string) string {
	if strings.TrimSpace(options.Credentials.AccountID) == "" || strings.TrimSpace(options.UserID) == "" {
		return ""
	}
	scope := cacheScope{AccountID: options.Credentials.AccountID, UserID: options.UserID,
		Issuer: options.Issuer, Backend: backend, Protocol: protocol, FedRAMP: options.Credentials.FedRAMP}
	data, _ := json.Marshal(scope)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func temporaryCatalogError(err error) bool {
	var upstream *codex.UpstreamError
	if errors.As(err, &upstream) {
		return upstream.Status == http.StatusRequestTimeout || upstream.Status == http.StatusTooManyRequests ||
			(upstream.Status >= 500 && upstream.Status <= 599)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	var network net.Error
	return errors.As(err, &network)
}

func cacheDirectory(home string) (string, error) {
	return privateCacheDirectory(home, "model-catalog")
}

func privateCacheDirectory(home, name string) (string, error) {
	dir, err := auth.DataDir(home)
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Mode().Perm()&0o200 == 0 {
		return "", fmt.Errorf("model catalog cache directory must be private")
	}
	return dir, nil
}

func readCache(home, key string, now time.Time) (cachedCatalog, error) {
	dir, err := cacheDirectory(home)
	if err != nil {
		return cachedCatalog{}, err
	}
	path := filepath.Join(dir, key+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return cachedCatalog{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Mode().Perm()&0o400 == 0 || info.Size() > catalogCacheLimit {
		return cachedCatalog{}, fmt.Errorf("unsafe model catalog cache")
	}
	file, err := os.Open(path)
	if err != nil {
		return cachedCatalog{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return cachedCatalog{}, fmt.Errorf("model catalog cache changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, catalogCacheLimit+1))
	if err != nil || len(data) > catalogCacheLimit {
		return cachedCatalog{}, fmt.Errorf("invalid model catalog cache size")
	}
	var entry cachedCatalog
	if err := json.Unmarshal(data, &entry); err != nil {
		return cachedCatalog{}, fmt.Errorf("invalid model catalog cache")
	}
	if entry.Schema != catalogCacheSchema || entry.Scope != key || entry.Invalid ||
		entry.FetchedAt.IsZero() || entry.ObservedAt.IsZero() || entry.FetchedAt.After(now) ||
		entry.ObservedAt.After(now) || entry.ObservedAt.After(entry.FetchedAt) ||
		now.Sub(entry.FetchedAt) > catalogCacheMaxAge {
		return cachedCatalog{}, fmt.Errorf("expired or rejected model catalog cache")
	}
	if err := ValidateCatalog(entry.Models); err != nil {
		return cachedCatalog{}, err
	}
	if err := checkCacheBarriers(home, key, entry.ObservedAt); err != nil {
		return cachedCatalog{}, err
	}
	return entry, nil
}

func writeCache(home, key string, entry cachedCatalog) error {
	dir, err := cacheDirectory(home)
	if err != nil {
		return err
	}
	lockCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	unlock, err := filelock.Acquire(lockCtx, filepath.Join(dir, key+".lock"))
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(dir, key+".json")
	// Older in-flight responses cannot overwrite a later observed rejection.
	info, statErr := os.Lstat(path)
	if statErr == nil && (!info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > catalogCacheLimit) {
		return fmt.Errorf("unsafe existing model catalog cache")
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if file, err := os.Open(path); err == nil {
		opened, openErr := file.Stat()
		if statErr != nil || openErr != nil || !os.SameFile(info, opened) {
			_ = file.Close()
			return fmt.Errorf("model catalog cache changed while opening")
		}
		data, readErr := io.ReadAll(io.LimitReader(file, catalogCacheLimit+1))
		_ = file.Close()
		var current cachedCatalog
		if readErr == nil && len(data) <= catalogCacheLimit && json.Unmarshal(data, &current) == nil && current.Schema == catalogCacheSchema &&
			current.Scope == key && current.ObservedAt.After(entry.ObservedAt) {
			return nil
		}
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if len(data) > catalogCacheLimit {
		return fmt.Errorf("model catalog is too large to cache")
	}
	file, err := os.CreateTemp(dir, ".catalog-*.json")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	pruneCacheBarriers(home, key, entry.ObservedAt)
	return nil
}

// A cache lock timeout or failed replacement must not preserve earlier data
// after an authoritative observation. Immutable rejection records use a
// separate private directory and no shared cache lock: an older writer cannot
// overwrite a later rejection, even when it finishes after this launch.
func persistCache(options Options, key string, entry cachedCatalog) {
	if err := writeCache(options.Home, key, entry); err == nil {
		return
	}
	if err := writeCacheBarrier(options.Home, key, entry.ObservedAt); err != nil {
		// If the state filesystem itself cannot record the barrier, make an
		// existing cache unreadable. A later successful refresh replaces it.
		if dir, dirErr := cacheDirectory(options.Home); dirErr == nil {
			path := filepath.Join(dir, key+".json")
			if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
				if file, err := os.Open(path); err == nil {
					if opened, err := file.Stat(); err == nil && os.SameFile(info, opened) {
						_ = file.Chmod(0)
					}
					_ = file.Close()
				}
			}
		}
	}
	if options.Warnings != nil {
		fmt.Fprintln(options.Warnings, "Claudodex: model metadata could not be cached; recovery during a network outage is unavailable.")
	}
}

func writeCacheBarrier(home, key string, observedAt time.Time) error {
	dir, err := privateCacheDirectory(home, "model-catalog-rejections")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, key+"-*.json")
	if err != nil {
		return err
	}
	// Leave even a truncated marker in place: malformed barriers fail closed.
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(observedAt); err != nil {
		return err
	}
	return file.Sync()
}

func cacheBarriers(home, key string, limit int, visit func(string, time.Time)) error {
	dir, err := privateCacheDirectory(home, "model-catalog-rejections")
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(dir, key+"-*.json"))
	if err != nil || (limit > 0 && len(paths) > limit) {
		return fmt.Errorf("model catalog rejection state is unavailable")
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > 128 {
			return fmt.Errorf("unsafe model catalog rejection state")
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		var observed time.Time
		decoder := json.NewDecoder(io.LimitReader(file, 129))
		decodeErr := decoder.Decode(&observed)
		var trailing any
		endErr := decoder.Decode(&trailing)
		_ = file.Close()
		if statErr != nil || !os.SameFile(info, opened) || decodeErr != nil || !errors.Is(endErr, io.EOF) || observed.IsZero() {
			return fmt.Errorf("invalid model catalog rejection state")
		}
		visit(path, observed)
	}
	return nil
}

func checkCacheBarriers(home, key string, observed time.Time) error {
	blocked := false
	err := cacheBarriers(home, key, 256, func(_ string, barrier time.Time) {
		if !barrier.Before(observed) {
			blocked = true
		}
	})
	if err != nil || blocked {
		return fmt.Errorf("model catalog has a later authoritative observation")
	}
	return nil
}

func pruneCacheBarriers(home, key string, observed time.Time) {
	_ = cacheBarriers(home, key, 0, func(path string, barrier time.Time) {
		if !barrier.After(observed) {
			_ = os.Remove(path)
		}
	})
}
