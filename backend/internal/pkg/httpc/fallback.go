package httpc

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// HeaderNoRelay can be set on an http.Request to skip external RSS relay fallbacks
// (for example, during blind multi-path probing in site_scout).
const HeaderNoRelay = "X-Scout-No-Relay"

// ResponsePayload holds the status, headers, and body of a fetched URL.
type ResponsePayload struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type cachedRelayEntry struct {
	body      []byte
	expiresAt time.Time
}

var (
	relayCacheMu sync.RWMutex
	relayCache   = make(map[string]cachedRelayEntry)
)

const relayCacheTTL = 5 * time.Minute

// SanitizeXMLBytes strips UTF-8 BOM, leading whitespace before the first XML tag,
// and illegal XML 1.0 control characters (e.g. 0x1D) that cause XML parsers to abort.
func SanitizeXMLBytes(raw []byte) []byte {
	// Strip UTF-8 BOM if present
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	raw = bytes.TrimSpace(raw)

	return bytes.Map(func(r rune) rune {
		if r == 0x09 || r == 0x0A || r == 0x0D ||
			(r >= 0x20 && r <= 0xD7FF) ||
			(r >= 0xE000 && r <= 0xFFFD) ||
			(r >= 0x10000 && r <= 0x10FFFF) {
			return r
		}
		return -1
	}, raw)
}

func isAntiBotResponse(resp *http.Response, body []byte) bool {
	if resp == nil {
		return false
	}

	switch resp.StatusCode {
	case http.StatusForbidden,
		http.StatusUnauthorized,
		http.StatusServiceUnavailable,
		http.StatusTooManyRequests,
		http.StatusAccepted:
		return true
	}

	if resp.Header.Get("X-Amzn-Waf-Action") != "" ||
		resp.Header.Get("cf-mitigated") != "" ||
		resp.Header.Get("Cf-Mitigated") != "" {
		return true
	}

	if isWAFChallengeBody(body) {
		return true
	}

	return false
}

func isWAFChallengeBody(body []byte) bool {
	if len(body) == 0 || len(body) > 32768 {
		return false
	}
	snippet := string(body)
	return strings.Contains(snippet, "window.gokuProps") ||
		strings.Contains(snippet, "token.awswaf.com") ||
		strings.Contains(snippet, "AwsWafIntegration")
}

// DoWithFallback executes an HTTP request using the given client.
// If the request is rejected with an anti-bot status (403, 401, 503, 429, or 202 AWS WAF challenge)
// or encounters a TLS/network failure:
// 1. On macOS (darwin), it attempts Apple's native CFNetwork stack via osascript JXA.
// 2. Across all platforms (including Linux cloud servers), it attempts to resolve RSS/Atom feeds via
//    an RSS relay fallback so datacenter IP blocks (e.g. CloudFront AWS WAF 202 challenge) are bypassed.
func DoWithFallback(ctx context.Context, client *http.Client, req *http.Request) (*ResponsePayload, error) {
	if client == nil {
		client = http.DefaultClient
	}

	skipRelay := req.Header.Get(HeaderNoRelay) != ""
	req.Header.Del(HeaderNoRelay)

	var blockedBody []byte
	resp, doErr := client.Do(req)
	if doErr == nil {
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read response body: %w", readErr)
		}

		if !isAntiBotResponse(resp, body) {
			return &ResponsePayload{
				StatusCode: resp.StatusCode,
				Header:     resp.Header.Clone(),
				Body:       body,
			}, nil
		}

		blockedBody = body
		slog.Info("standard HTTP fetch returned anti-bot status",
			"status", resp.StatusCode,
			"waf_action", resp.Header.Get("X-Amzn-Waf-Action"),
			"url", req.URL.String(),
		)
	} else {
		slog.Info("standard HTTP fetch failed", "error", doErr, "url", req.URL.String())
	}

	if req.URL != nil {
		targetURL := req.URL.String()

		// Tier 1: Try native macOS fallback if running on darwin
		if runtime.GOOS == "darwin" && !isLoopbackHost(req.URL.Hostname()) {
			slog.Info("attempting native macOS fetch fallback", "url", targetURL)
			nativeBody, nativeErr := FetchNativeDarwin(ctx, targetURL)
			if nativeErr == nil && len(nativeBody) > 0 && !isWAFChallengeBody(nativeBody) {
				slog.Info("native macOS fetch succeeded", "url", targetURL, "bytes", len(nativeBody))
				h := make(http.Header)
				ct := http.DetectContentType(nativeBody)
				h.Set("Content-Type", ct)
				return &ResponsePayload{
					StatusCode: http.StatusOK,
					Header:     h,
					Body:       nativeBody,
				}, nil
			}
			if nativeErr != nil {
				slog.Warn("native macOS fetch fallback failed", "url", targetURL, "error", nativeErr)
			}
		}

		// Tier 2: Try cross-platform RSS relay fallback (essential on Linux cloud VMs blocked by AWS WAF / CloudFront)
		if !skipRelay && !isLoopbackHost(req.URL.Hostname()) {
			slog.Info("attempting RSS relay fallback", "url", targetURL)
			relayBody, relayErr := FetchViaRSSRelay(ctx, targetURL)
			if relayErr == nil && len(relayBody) > 0 {
				slog.Info("RSS relay fallback succeeded", "url", targetURL, "bytes", len(relayBody))
				h := make(http.Header)
				h.Set("Content-Type", "application/rss+xml; charset=utf-8")
				return &ResponsePayload{
					StatusCode: http.StatusOK,
					Header:     h,
					Body:       relayBody,
				}, nil
			}
			if relayErr != nil {
				slog.Warn("RSS relay fallback failed", "url", targetURL, "error", relayErr)
			}
		}
	}

	if doErr != nil {
		return nil, doErr
	}

	// Return original blocked response if fallbacks did not succeed
	return &ResponsePayload{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       blockedBody,
	}, nil
}

func isLoopbackHost(host string) bool {
	h := strings.TrimSpace(strings.ToLower(host))
	if h == "localhost" || strings.HasSuffix(h, ".local") {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	return false
}

// FetchNativeDarwin fetches a URL using macOS's built-in Foundation framework.
// Because it routes through Apple's native CFNetwork/Security stack, it uses Apple's
// standard Safari/WebKit TLS handshake, passing Akamai and Cloudflare edge verification.
func FetchNativeDarwin(ctx context.Context, targetURL string) ([]byte, error) {
	script := fmt.Sprintf(`
ObjC.import("Foundation");
var url = $.NSURL.URLWithString(%q);
if (!url.isNil()) {
    var data = $.NSData.dataWithContentsOfURL(url);
    if (!data.isNil()) {
        $.NSFileHandle.fileHandleWithStandardOutput.writeData(data);
    }
}
`, targetURL)

	timeout := 25 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		remaining := time.Until(dl)
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "osascript", "-l", "JavaScript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("native fetch execution failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	out := stdout.Bytes()
	if len(out) == 0 {
		return nil, fmt.Errorf("native fetch returned empty response")
	}

	return out, nil
}

type rss2jsonResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Feed    rss2jsonFeed   `json:"feed"`
	Items   []rss2jsonItem `json:"items"`
}

type rss2jsonFeed struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

type rss2jsonItem struct {
	Title       string   `json:"title"`
	PubDate     string   `json:"pubDate"`
	Link        string   `json:"link"`
	GUID        string   `json:"guid"`
	Author      string   `json:"author"`
	Thumbnail   string   `json:"thumbnail"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Categories  []string `json:"categories"`
}

type syntheticRSSRoot struct {
	XMLName xml.Name            `xml:"rss"`
	Version string              `xml:"version,attr"`
	Channel syntheticRSSChannel `xml:"channel"`
}

type syntheticRSSChannel struct {
	Title       string             `xml:"title"`
	Link        string             `xml:"link"`
	Description string             `xml:"description"`
	Items       []syntheticRSSItem `xml:"item"`
}

type syntheticRSSItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        string   `xml:"guid,omitempty"`
	PubDate     string   `xml:"pubDate,omitempty"`
	Author      string   `xml:"author,omitempty"`
	Description string   `xml:"description,omitempty"`
	Categories  []string `xml:"category,omitempty"`
}

// FetchViaRSSRelay fetches an RSS/Atom feed via the rss2json edge API and reconstructs
// standard RSS 2.0 XML bytes so downstream XML parsers (gofeed) work transparently.
func FetchViaRSSRelay(ctx context.Context, targetURL string) ([]byte, error) {
	now := time.Now()
	relayCacheMu.RLock()
	if entry, ok := relayCache[targetURL]; ok && now.Before(entry.expiresAt) {
		cached := make([]byte, len(entry.body))
		copy(cached, entry.body)
		relayCacheMu.RUnlock()
		return cached, nil
	}
	relayCacheMu.RUnlock()

	apiURL := "https://api.rss2json.com/v1/api.json?rss_url=" + url.QueryEscape(targetURL)

	timeout := 12 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create relay request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("relay http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("relay returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read relay body: %w", err)
	}

	var parsed rss2jsonResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode relay json: %w", err)
	}

	if !strings.EqualFold(parsed.Status, "ok") {
		if parsed.Message != "" {
			return nil, fmt.Errorf("relay status %s: %s", parsed.Status, parsed.Message)
		}
		return nil, fmt.Errorf("relay status %s", parsed.Status)
	}

	xmlBytes, err := convertRSS2JSONToXML(&parsed, targetURL)
	if err != nil {
		return nil, err
	}

	relayCacheMu.Lock()
	relayCache[targetURL] = cachedRelayEntry{
		body:      xmlBytes,
		expiresAt: time.Now().Add(relayCacheTTL),
	}
	relayCacheMu.Unlock()

	return xmlBytes, nil
}

func convertRSS2JSONToXML(parsed *rss2jsonResponse, fallbackURL string) ([]byte, error) {
	if parsed == nil {
		return nil, fmt.Errorf("nil relay response")
	}

	title := strings.TrimSpace(parsed.Feed.Title)
	if title == "" {
		title = fallbackURL
	}
	siteLink := strings.TrimSpace(parsed.Feed.Link)
	if siteLink == "" {
		siteLink = fallbackURL
	}

	items := make([]syntheticRSSItem, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		desc := strings.TrimSpace(it.Content)
		if desc == "" {
			desc = strings.TrimSpace(it.Description)
		}
		guid := strings.TrimSpace(it.GUID)
		if guid == "" {
			guid = strings.TrimSpace(it.Link)
		}
		items = append(items, syntheticRSSItem{
			Title:       strings.TrimSpace(it.Title),
			Link:        strings.TrimSpace(it.Link),
			GUID:        guid,
			PubDate:     formatRSSPubDate(it.PubDate),
			Author:      strings.TrimSpace(it.Author),
			Description: desc,
			Categories:  it.Categories,
		})
	}

	root := syntheticRSSRoot{
		Version: "2.0",
		Channel: syntheticRSSChannel{
			Title:       title,
			Link:        siteLink,
			Description: strings.TrimSpace(parsed.Feed.Description),
			Items:       items,
		},
	}

	marshaled, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal synthetic rss xml: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(marshaled)
	return buf.Bytes(), nil
}

func formatRSSPubDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC1123Z)
		}
	}
	return raw
}
