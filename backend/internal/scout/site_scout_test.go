package scout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScoutSiteWithDirectFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Mock Test Feed</title>
<link>http://example.com</link>
<description>Test Description</description>
<item>
<title>Item 1</title>
<link>http://example.com/1</link>
</item>
</channel>
</rss>`))
	}))
	defer ts.Close()

	feeds, err := ScoutSite(context.Background(), ts.URL+"/feed.xml", true)
	if err != nil {
		t.Fatalf("ScoutSite failed: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatalf("expected at least 1 feed, got 0")
	}
	if feeds[0].Title != "Mock Test Feed" {
		t.Errorf("expected title 'Mock Test Feed', got %q", feeds[0].Title)
	}
}

func TestScoutSiteWithHTMLAlternate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rss.xml" {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Alternate RSS Feed</title><link>http://example.com</link></channel></rss>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<link rel="alternate" type="application/rss+xml" title="Alternate RSS Feed" href="/rss.xml">
</head>
<body><h1>Hello World</h1></body>
</html>`))
	}))
	defer ts.Close()

	feeds, err := ScoutSite(context.Background(), ts.URL, true)
	if err != nil {
		t.Fatalf("ScoutSite failed: %v", err)
	}
	if len(feeds) == 0 {
		t.Fatalf("expected at least 1 feed, got 0")
	}
	if feeds[0].Title != "Alternate RSS Feed" {
		t.Errorf("expected 'Alternate RSS Feed', got %q", feeds[0].Title)
	}
}

func TestParseFeedBodySanitizesControlCharsAndBOM(t *testing.T) {
	rawXML := append([]byte{0xEF, 0xBB, 0xBF, '\n'}, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>News - European Parliament`+"\x1d"+`</title>
    <link>https://www.europarl.europa.eu/news/en</link>
    <description />
    <item>
      <title>Agenda - The Week Ahead</title>
      <link>https://www.europarl.europa.eu/news/en/agenda/weekly-agenda/2026-46</link>
      <guid isPermaLink="false">AGD_AGD-2026-46-00-00000_EN</guid>
    </item>
  </channel>
</rss>`)...)

	feed, err := parseFeedBody(rawXML, "https://www.europarl.europa.eu/rss/doc/press-news/en.xml")
	if err != nil {
		t.Fatalf("parseFeedBody failed: %v", err)
	}
	if feed.Title != "News - European Parliament" {
		t.Errorf("expected title 'News - European Parliament', got %q", feed.Title)
	}
	if feed.SiteURL != "https://www.europarl.europa.eu/news/en" {
		t.Errorf("expected site_url 'https://www.europarl.europa.eu/news/en', got %q", feed.SiteURL)
	}
}

