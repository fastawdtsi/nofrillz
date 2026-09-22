// Package research acquires bounded source material independently of generation.
package research

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Source struct {
	URL          string     `json:"url"`
	Name         string     `json:"name"`
	Title        string     `json:"title"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	DiscoveredAt time.Time  `json:"discovered_at"`
}
type Candidate struct {
	Key, Title, Context string
	Sources             []Source
}
type Request struct {
	URLs   []string
	MaxAge time.Duration
	Now    time.Time
}
type Researcher interface {
	Discover(context.Context, Request) ([]Candidate, error)
}
type RSS struct{ client *http.Client }

func NewRSS() *RSS {
	// Resolve and validate the actual dial address, including after redirects.
	// Prevent admin-configured sources from reaching local/private infrastructure.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid source host")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("source DNS lookup failed")
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("source host has no addresses")
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, fmt.Errorf("source host must resolve to public addresses")
			}
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &RSS{client: &http.Client{Timeout: 20 * time.Second, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("too many source redirects")
		}
		return ValidateURL(req.URL.String())
	}}}
}
func publicIP(ip net.IP) bool {
	v4 := ip.To4()
	sharedAddress := v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !sharedAddress && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}
func ValidateURL(raw string) error {
	if len(raw) > 2048 {
		return fmt.Errorf("source URL exceeds 2048 bytes")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") {
		return fmt.Errorf("sources must be public HTTPS feed URLs without credentials")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return fmt.Errorf("private source address is not allowed")
	}
	if strings.EqualFold(u.Hostname(), "localhost") {
		return fmt.Errorf("private source address is not allowed")
	}
	return nil
}
func Fingerprint(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
func CanonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") || key == "fbclid" || key == "gclid" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	return strings.TrimRight(u.String(), "/")
}

var tags = regexp.MustCompile(`<[^>]*>`)

func plain(s string, max int) string {
	s = html.UnescapeString(tags.ReplaceAllString(s, " "))
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > max {
		r = r[:max]
	}
	return string(r)
}

type feedItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	Content     string `xml:"encoded"`
	PubDate     string `xml:"pubDate"`
	Date        string `xml:"date"`
}
type atomEntry struct {
	Title     string `xml:"title"`
	ID        string `xml:"id"`
	Summary   string `xml:"summary"`
	Content   string `xml:"content"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Links     []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
}

func Parse(data []byte, feedURL string, now time.Time, maxAge time.Duration) ([]Candidate, error) {
	var doc struct {
		XMLName xml.Name
		Title   string `xml:"title"`
		Channel struct {
			Title string     `xml:"title"`
			Items []feedItem `xml:"item"`
		} `xml:"channel"`
		Entries []atomEntry `xml:"entry"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid RSS/Atom document")
	}
	if doc.XMLName.Local != "rss" && doc.XMLName.Local != "feed" {
		return nil, fmt.Errorf("source is not RSS or Atom")
	}
	result := []Candidate{}
	name := plain(doc.Channel.Title, 160)
	if name == "" {
		name = plain(doc.Title, 160)
	}
	add := func(title, link, summary, date string) {
		if strings.TrimSpace(link) == "" {
			return
		}
		title = plain(title, 300)
		summary = plain(summary, 5000)
		base, _ := url.Parse(feedURL)
		ref, err := url.Parse(link)
		if err != nil || base == nil {
			return
		}
		link = CanonicalURL(base.ResolveReference(ref).String())
		if ValidateURL(link) != nil {
			return
		}
		var published *time.Time
		for _, layout := range []string{time.RFC3339Nano, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, "2006-01-02"} {
			if d, e := time.Parse(layout, strings.TrimSpace(date)); e == nil {
				v := d.UTC()
				published = &v
				break
			}
		}
		// Unknown/future/stale publication times cannot substantiate "current" news.
		if published == nil || published.After(now.Add(15*time.Minute)) || published.Before(now.Add(-maxAge)) || len(title) < 12 || len(summary) < 50 {
			return
		}
		result = append(result, Candidate{Key: Fingerprint(link), Title: title, Context: title + "\n" + summary, Sources: []Source{{URL: link, Name: name, Title: title, PublishedAt: published, DiscoveredAt: now}}})
	}
	for _, item := range doc.Channel.Items {
		summary := item.Description
		if len(item.Content) > len(summary) {
			summary = item.Content
		}
		date := item.PubDate
		if date == "" {
			date = item.Date
		}
		add(item.Title, item.Link, summary, date)
	}
	for _, item := range doc.Entries {
		link := ""
		for _, l := range item.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		summary := item.Summary
		if summary == "" {
			summary = item.Content
		}
		date := item.Published
		if date == "" {
			date = item.Updated
		}
		add(item.Title, link, summary, date)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Sources[0].PublishedAt.After(*result[j].Sources[0].PublishedAt) })
	if len(result) > 30 {
		result = result[:30]
	}
	return result, nil
}
func (r *RSS) Discover(ctx context.Context, request Request) ([]Candidate, error) {
	if len(request.URLs) == 0 || len(request.URLs) > 5 {
		return nil, fmt.Errorf("research requires 1–5 configured RSS/Atom sources")
	}
	all := []Candidate{}
	seen := map[string]bool{}
	for _, source := range request.URLs {
		if err := ValidateURL(source); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid source URL")
		}
		req.Header.Set("User-Agent", "NofrillzContent/1.0 (+RSS reader)")
		req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")
		resp, err := r.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("source retrieval failed")
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		resp.Body.Close()
		if resp.StatusCode != 200 || readErr != nil || len(data) > 1<<20 {
			return nil, fmt.Errorf("source response unavailable or exceeds 1 MB (HTTP %d)", resp.StatusCode)
		}
		items, err := Parse(data, source, request.Now, request.MaxAge)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !seen[item.Key] {
				seen[item.Key] = true
				all = append(all, item)
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Sources[0].PublishedAt.After(*all[j].Sources[0].PublishedAt) })
	return all, nil
}
