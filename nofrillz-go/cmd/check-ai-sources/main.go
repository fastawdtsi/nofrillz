// check-ai-sources validates public feeds through the poster's actual source
// adapter. It does not call models, write accounts, or publish posts.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"nofrillz/internal/research"
)

type result struct {
	URL        string            `json:"url"`
	CheckedAt  time.Time         `json:"checked_at"`
	Candidates int               `json:"candidates"`
	Error      string            `json:"error,omitempty"`
	Examples   []research.Source `json:"examples"`
}

func main() {
	age := flag.Int("max-age-hours", 168, "Maximum accepted source age in hours")
	flag.Parse()
	if *age < 1 || *age > 2160 || len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "usage: check-ai-sources [-max-age-hours 168] https://publisher/feed ...")
		os.Exit(2)
	}
	now := time.Now().UTC()
	results := make([]result, len(flag.Args()))
	adapter := research.NewRSS()
	limit := make(chan struct{}, 4)
	var workers sync.WaitGroup
	for i, url := range flag.Args() {
		workers.Add(1)
		go func(i int, url string) {
			defer workers.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			items, err := adapter.Discover(ctx, research.Request{URLs: []string{url}, Now: now, MaxAge: time.Duration(*age) * time.Hour})
			value := result{URL: url, CheckedAt: now, Candidates: len(items), Examples: []research.Source{}}
			if err != nil {
				value.Error = err.Error()
			}
			for _, item := range items[:min(2, len(items))] {
				value.Examples = append(value.Examples, item.Sources...)
			}
			results[i] = value
		}(i, url)
	}
	workers.Wait()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
