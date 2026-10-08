// Command build downloads USLM XML titles from archive.org snapshots of
// uscode.house.gov, parses them, and writes a lex dataset (Badger store + FTS index).
// This is a one-shot build tool for creating the prebuilt lex-us.tar.gz release asset.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/tggo/lex/internal/search"
	"github.com/tggo/lex/internal/store"
	"github.com/tggo/lex/us/scripts/uslm"
)

const (
	archiveBase = "https://web.archive.org/web/%s/https://uscode.house.gov/download/releasepoints/us/pl/119/4"
	releaseTag  = "119-4"
	ua          = "lex/0.1 (+https://github.com/tggo/lex)"
	maxRetries  = 3
)

// snapshotTimestamps tries multiple snapshot dates in order.
var snapshotTimestamps = []string{"2024", "2025", "20240601", "20241201", "20250301"}

func main() {
	outDir := flag.String("out", "/tmp/us-dataset", "dataset root directory")
	flag.Parse()

	ctx := context.Background()
	client := &http.Client{Timeout: 5 * time.Minute}

	graphDir := filepath.Join(*outDir, "graph")
	indexPath := filepath.Join(*outDir, "index.fts")

	st, err := store.Open(graphDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	idx, err := search.OpenLang(indexPath, "en")
	if err != nil {
		log.Fatalf("open index: %v", err)
	}
	defer idx.Close()

	now := time.Now().UTC()
	total := 0

	for n := 1; n <= 54; n++ {
		log.Printf("Fetching title %d...", n)

		xmlBytes, err := fetchWithRetries(ctx, client, n)
		if err != nil {
			log.Printf("  SKIP title %d: %v", n, err)
			continue
		}

		doc, err := uslm.ParseDocument(xmlBytes)
		if err != nil {
			log.Printf("  SKIP title %d: parse: %v", n, err)
			continue
		}

		url := fmt.Sprintf("https://uscode.house.gov/download/releasepoints/us/pl/119/4/xml_usc%02d@%s.zip", n, releaseTag)
		act, err := uslm.ToAct(doc, url, now)
		if err != nil {
			log.Printf("  SKIP title %d: toAct: %v", n, err)
			continue
		}

		if err := st.AddAct(act); err != nil {
			log.Printf("  SKIP title %d: addAct: %v", n, err)
			continue
		}

		if err := idx.ReplaceAct(act); err != nil {
			log.Printf("  SKIP title %d: index: %v", n, err)
			continue
		}

		total++
		log.Printf("  OK title %d: %s (%d articles)", n, act.Expression.Title, len(act.Expression.Articles))
	}

	log.Printf("Built dataset with %d titles at %s", total, *outDir)
}

func fetchWithRetries(ctx context.Context, client *http.Client, n int) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		ts := snapshotTimestamps[attempt%len(snapshotTimestamps)]
		url := fmt.Sprintf(archiveBase+"/xml_usc%02d@%s.zip", ts, n, releaseTag)

		xmlBytes, err := fetchAndExtract(ctx, client, url)
		if err == nil {
			return xmlBytes, nil
		}
		lastErr = err
		log.Printf("  attempt %d for title %d: %v", attempt+1, n, err)

		// Wait before retry (exponential backoff)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(1<<attempt) * 2 * time.Second):
		}
	}
	return nil, lastErr
}

func fetchAndExtract(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}

	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}

	return nil, fmt.Errorf("no .xml in zip")
}
