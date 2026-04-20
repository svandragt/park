package cmd

import (
	"net/http"
	"strings"
	"time"
)

// resolveRemote follows HTTP redirects to find the canonical remote URL.
// Returns rawURL unchanged if it can't be resolved or no redirect occurs.
func resolveRemote(rawURL string) string {
	httpsURL := sshToHTTPS(rawURL)
	if httpsURL == "" {
		return rawURL
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := client.Head(httpsURL)
	if err != nil {
		return rawURL
	}
	defer resp.Body.Close()

	final := normalizeURL(resp.Request.URL.String())
	if final != httpsURL {
		return final
	}
	return httpsURL
}

func normalizeRemote(u string) string {
	if u == "" {
		return ""
	}
	if s := sshToHTTPS(u); s != "" {
		return s
	}
	return normalizeURL(u)
}

// sshToHTTPS converts git@host:org/repo to https://host/org/repo.
// Returns empty string if the URL isn't a recognized format.
func sshToHTTPS(remote string) string {
	if strings.HasPrefix(remote, "https://") {
		return normalizeURL(remote)
	}
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) == 2 {
			return "https://" + parts[0] + "/" + normalizeURL(parts[1])
		}
	}
	return ""
}

func normalizeURL(u string) string {
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")
	return u
}
