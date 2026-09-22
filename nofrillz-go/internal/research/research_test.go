package research

import (
	"testing"
	"time"
)

func TestRSSCanonicalizationFreshnessAndProvenance(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	xml := []byte(`<rss><channel><title>Science desk</title><item><title>A new telescope survey</title><link>https://example.org/survey?utm_source=feed#part</link><description>A telescope survey catalogues nearby stars in a newly released public data set.</description><pubDate>Tue, 22 Sep 2026 10:00:00 GMT</pubDate></item><item><title>An old announcement</title><link>https://example.org/old</link><description>This is old material with enough characters to pass the content length requirement.</description><pubDate>Mon, 01 Jan 2024 10:00:00 GMT</pubDate></item><item><title>Missing publication date</title><link>https://example.org/unknown</link><description>Unknown-date material must not be represented as current news from a fresh source.</description></item></channel></rss>`)
	items, err := Parse(xml, "https://example.org/feed", now, 7*24*time.Hour)
	if err != nil || len(items) != 1 {
		t.Fatalf("parse: %v, %d", err, len(items))
	}
	source := items[0].Sources[0]
	if source.Name != "Science desk" || source.URL != "https://example.org/survey" || source.PublishedAt == nil || !source.DiscoveredAt.Equal(now) {
		t.Fatalf("provenance: %+v", source)
	}
	if items[0].Key != Fingerprint("https://example.org/survey") {
		t.Fatal("tracking parameters changed identity")
	}
}
func TestSourcesRejectUnsafeURLs(t *testing.T) {
	for _, url := range []string{"http://example.org/rss", "https://localhost/feed", "https://127.0.0.1/feed", "https://169.254.169.254/feed", "https://user:pass@example.org/feed", "https://example.org:8443/feed"} {
		if ValidateURL(url) == nil {
			t.Fatalf("unsafe URL allowed: %s", url)
		}
	}
}
