package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const releaseBase = "https://github.com/briceduke/ax/releases/download"

// Getter fetches a URL. Tests inject a fake.
type Getter func(url string) ([]byte, error)

// DefaultGet is the real HTTP client. GitHub requires a User-Agent.
func DefaultGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ax-upgrade")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, res.Status)
	}
	return body, nil
}

// LatestTag reads the latest GitHub release tag (v0.2.0).
func LatestTag(get Getter) (string, error) {
	if get == nil {
		return "", fmt.Errorf("no downloader")
	}
	body, err := get("https://api.github.com/repos/briceduke/ax/releases/latest")
	if err != nil {
		return "", err
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("latest release has no tag")
	}
	return rel.TagName, nil
}

func pinFromTag(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

func assetURLs(ver string) []string {
	base := fmt.Sprintf("%s/v%s/ax_%s_%s_%s", releaseBase, ver, ver, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		return []string{base + ".exe", base + ".zip"}
	}
	return []string{base, base + ".tar.gz"}
}

func downloadRelease(get Getter, ver string) ([]byte, error) {
	if get == nil {
		return nil, fmt.Errorf("no downloader")
	}
	var last error
	for _, url := range assetURLs(ver) {
		data, err := get(url)
		if err != nil {
			last = err
			continue
		}
		payload, err := unpackAx(data)
		if err != nil {
			last = err
			continue
		}
		return payload, nil
	}
	if last == nil {
		last = fmt.Errorf("no release asset for %s %s/%s", ver, runtime.GOOS, runtime.GOARCH)
	}
	return nil, last
}
