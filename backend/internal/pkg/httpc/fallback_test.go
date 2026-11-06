package httpc

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestIsAntiBotResponse(t *testing.T) {
	// 1. AWS CloudFront WAF 202 challenge with empty body
	wafResp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     http.Header{"X-Amzn-Waf-Action": []string{"challenge"}},
	}
	if !isAntiBotResponse(wafResp, nil) {
		t.Errorf("expected HTTP 202 with X-Amzn-Waf-Action: challenge to be detected as anti-bot")
	}

	// 2. HTTP 200 containing AWS WAF JS challenge HTML
	wafHTML := []byte(`<!DOCTYPE html><html><head><script>window.gokuProps = {"key":"abc"};</script></head></html>`)
	okResp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
	}
	if !isAntiBotResponse(okResp, wafHTML) {
		t.Errorf("expected HTTP 200 with window.gokuProps challenge HTML to be detected as anti-bot")
	}

	// 3. Normal HTTP 200 RSS XML
	rssBody := []byte(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>News - European Parliament</title></channel></rss>`)
	if isAntiBotResponse(okResp, rssBody) {
		t.Errorf("expected normal HTTP 200 RSS XML not to be flagged as anti-bot")
	}
}

func TestConvertRSS2JSONToXML(t *testing.T) {
	sample := &rss2jsonResponse{
		Status: "ok",
		Feed: rss2jsonFeed{
			URL:         "https://www.europarl.europa.eu/rss/doc/press-news/en.xml",
			Title:       "News - European Parliament",
			Link:        "https://www.europarl.europa.eu/news/en",
			Description: "European Parliament Press News",
		},
		Items: []rss2jsonItem{
			{
				Title:       "Agenda - The Week Ahead 09 – 15 November 2026",
				PubDate:     "2026-11-06 12:25:12",
				Link:        "https://www.europarl.europa.eu/news/en/agenda/weekly-agenda/2026-46",
				GUID:        "AGD_AGD-2026-46-00-00000_EN",
				Description: "Plenary session, Brussels",
				Content:     "Plenary session, Brussels<br>Source: EP",
				Categories:  []string{"Agenda"},
			},
		},
	}

	xmlBytes, err := convertRSS2JSONToXML(sample, sample.Feed.URL)
	if err != nil {
		t.Fatalf("convertRSS2JSONToXML failed: %v", err)
	}

	parsed, err := gofeed.NewParser().Parse(bytes.NewReader(xmlBytes))
	if err != nil {
		t.Fatalf("gofeed failed to parse reconstructed XML: %v", err)
	}
	if parsed.Title != "News - European Parliament" {
		t.Errorf("expected title 'News - European Parliament', got %q", parsed.Title)
	}
	if parsed.Link != "https://www.europarl.europa.eu/news/en" {
		t.Errorf("expected link 'https://www.europarl.europa.eu/news/en', got %q", parsed.Link)
	}
	if len(parsed.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(parsed.Items))
	}
	item := parsed.Items[0]
	if item.GUID != "AGD_AGD-2026-46-00-00000_EN" {
		t.Errorf("expected GUID 'AGD_AGD-2026-46-00-00000_EN', got %q", item.GUID)
	}
	if item.PublishedParsed == nil {
		t.Fatalf("expected PublishedParsed to be non-nil")
	}
}

func TestSanitizeXMLBytes(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF, ' ', '\n'}, []byte("<?xml version=\"1.0\"?>\x1d<rss version=\"2.0\"><channel><title>Clean</title></channel></rss>")...)
	clean := SanitizeXMLBytes(raw)
	parsed, err := gofeed.NewParser().Parse(bytes.NewReader(clean))
	if err != nil {
		t.Fatalf("expected sanitized XML to parse cleanly, got error: %v", err)
	}
	if parsed.Title != "Clean" {
		t.Errorf("expected title 'Clean', got %q", parsed.Title)
	}
}
