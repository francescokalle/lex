package store

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tggo/lex/internal/schema"
)

var update = flag.Bool("update", false, "update golden files")

func sampleAct() *schema.Act {
	return &schema.Act{
		Country:  "ua",
		TypeSlug: "kodeks",
		Year:     2003,
		Number:   "435-15",
		IDLocal:  "435-15",
		Expression: &schema.Expression{
			Title:            "Цивільний кодекс України",
			LangTag:          "uk",
			LangAlpha3:       "UKR",
			VersionDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			FirstInForceDate: time.Date(2004, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:           schema.StatusInForce,
			SourceURL:        "https://zakon.rada.gov.ua/laws/show/435-15",
			RetrievedAt:      time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC),
			Articles: []schema.Article{
				{Number: "1", Label: "Стаття 1", Text: "Цивільним законодавством регулюються відносини."},
				{Number: "2", Label: "Стаття 2", Text: "Учасниками цивільних відносин є фізичні та юридичні особи."},
			},
			Cites: []string{schema.ResourceURI("ua", "konstytutsiya", 1996, "254к/96-вр")},
		},
	}
}

func TestRoundTrip_memory(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	in := sampleAct()
	if err := s.AddAct(in); err != nil {
		t.Fatalf("AddAct: %v", err)
	}

	got, err := s.GetAct(in.ResourceURI())
	if err != nil {
		t.Fatalf("GetAct: %v", err)
	}
	assertActEqual(t, in, got)
}

func TestRoundTrip_file(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "graph") // Badger uses a directory
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	in := sampleAct()
	if err := s.AddAct(in); err != nil {
		t.Fatalf("AddAct: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen the same directory: data must persist.
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.GetAct(in.ResourceURI())
	if err != nil {
		t.Fatalf("GetAct after reopen: %v", err)
	}
	assertActEqual(t, in, got)
}

func TestListAndEachAct(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	if err := s.AddAct(sampleAct()); err != nil {
		t.Fatal(err)
	}
	// A second act to confirm listing/ordering.
	second := sampleAct()
	second.Number = "100-1"
	second.Expression.Articles = nil
	second.Expression.Cites = nil
	if err := s.AddAct(second); err != nil {
		t.Fatal(err)
	}

	uris, err := s.ListResourceURIs()
	if err != nil {
		t.Fatal(err)
	}
	if len(uris) != 2 {
		t.Fatalf("got %d resource URIs, want 2", len(uris))
	}

	var seen []string
	if err := s.EachAct(func(a *schema.Act) error {
		seen = append(seen, a.Number)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 {
		t.Errorf("EachAct visited %d acts, want 2", len(seen))
	}
}

func TestEachAct_propagatesError(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	_ = s.AddAct(sampleAct())
	wantErr := fmt.Errorf("boom")
	if err := s.EachAct(func(*schema.Act) error { return wantErr }); err != wantErr {
		t.Errorf("EachAct error = %v, want %v", err, wantErr)
	}
}

func TestRoundTrip_amendedAndRepealedBy(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	amending := schema.ResourceURI("jp", "act", 2024, "506AC0000000033")
	repealing := schema.ResourceURI("jp", "act", 1907, "140AC0000000045")
	in := sampleAct()
	in.Expression.AmendedBy = []string{amending}
	in.Expression.RepealedBy = []string{repealing}
	if err := s.AddAct(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAct(in.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Expression.AmendedBy) != 1 || got.Expression.AmendedBy[0] != amending {
		t.Errorf("amendedBy = %v, want [%s]", got.Expression.AmendedBy, amending)
	}
	if len(got.Expression.RepealedBy) != 1 || got.Expression.RepealedBy[0] != repealing {
		t.Errorf("repealedBy = %v, want [%s]", got.Expression.RepealedBy, repealing)
	}
}

func TestRoundTrip_revisions(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	amending := schema.ResourceURI("jp", "act", 2024, "506AC0000000033")
	in := sampleAct() // current expression is 2026-01-01
	in.Revisions = []schema.Revision{
		// Out of order on input; must come back sorted by version date.
		{VersionDate: time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC), Status: schema.StatusInForce, AmendedBy: []string{amending}},
		{VersionDate: time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC), Status: schema.StatusInForce},
	}
	if err := s.AddAct(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAct(in.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Revisions) != 2 {
		t.Fatalf("revisions = %d, want 2", len(got.Revisions))
	}
	if !got.Revisions[0].VersionDate.Equal(time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("revisions not sorted: first = %v", got.Revisions[0].VersionDate)
	}
	if len(got.Revisions[1].AmendedBy) != 1 || got.Revisions[1].AmendedBy[0] != amending {
		t.Errorf("revision amendedBy = %v, want [%s]", got.Revisions[1].AmendedBy, amending)
	}
	// The current expression must be unaffected: still one titled version.
	if got.Expression.Title != in.Expression.Title {
		t.Errorf("current expression title = %q, want %q", got.Expression.Title, in.Expression.Title)
	}
}

func TestGetAct_notFound(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	if _, err := s.GetAct(schema.ResourceURI("ua", "zakon", 1999, "nope")); err == nil {
		t.Error("expected error for missing act, got nil")
	}
}

func TestDumpSorted_golden(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	if err := s.AddAct(sampleAct()); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.DumpSorted(&buf); err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", "civil_code.nt.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update first): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("triples mismatch with golden.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), want)
	}
}

// --- Inverse relation tests ---

func TestInverseAmends_emptyStore(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	got, err := s.InverseAmends(schema.ResourceURI("ua", "kodeks", 2003, "435-15"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestInverseRepeals_emptyStore(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	got, err := s.InverseRepeals(schema.ResourceURI("ua", "kodeks", 2003, "435-15"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestInverseAmends_singleAmender(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	target := sampleAct()
	if err := s.AddAct(target); err != nil {
		t.Fatal(err)
	}

	amender := &schema.Act{
		Country: "ua", TypeSlug: "zakon", Year: 2024, Number: "123-1",
		Expression: &schema.Expression{
			Title:       "Закон про внесення змін",
			LangTag:     "uk",
			VersionDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Amends:      []string{target.ResourceURI()},
		},
	}
	if err := s.AddAct(amender); err != nil {
		t.Fatal(err)
	}

	got, err := s.InverseAmends(target.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != amender.ResourceURI() {
		t.Errorf("InverseAmends = %v, want [%s]", got, amender.ResourceURI())
	}
}

func TestInverseRepeals_singleRepealer(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	target := sampleAct()
	if err := s.AddAct(target); err != nil {
		t.Fatal(err)
	}

	repealer := &schema.Act{
		Country: "ua", TypeSlug: "zakon", Year: 2025, Number: "456-1",
		Expression: &schema.Expression{
			Title:       "Закон про скасування",
			LangTag:     "uk",
			VersionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Repeals:     []string{target.ResourceURI()},
		},
	}
	if err := s.AddAct(repealer); err != nil {
		t.Fatal(err)
	}

	got, err := s.InverseRepeals(target.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != repealer.ResourceURI() {
		t.Errorf("InverseRepeals = %v, want [%s]", got, repealer.ResourceURI())
	}
}

func TestInverseAmends_multipleAmenders(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	target := sampleAct()
	if err := s.AddAct(target); err != nil {
		t.Fatal(err)
	}

	for i, num := range []string{"100-1", "200-2", "300-3"} {
		amender := &schema.Act{
			Country: "ua", TypeSlug: "zakon", Year: 2020 + i, Number: num,
			Expression: &schema.Expression{
				Title:       "Закон про внесення змін " + num,
				LangTag:     "uk",
				VersionDate: time.Date(2020+i, 1, 1, 0, 0, 0, 0, time.UTC),
				Amends:      []string{target.ResourceURI()},
			},
		}
		if err := s.AddAct(amender); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.InverseAmends(target.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 amenders, got %d: %v", len(got), got)
	}
	// Results must be sorted.
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Errorf("results not sorted: %v", got)
			break
		}
	}
}

func TestInverseAmends_malformedURI(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	// A malformed URI should not panic or error — just return empty.
	got, err := s.InverseAmends("not-a-valid-uri")
	if err != nil {
		t.Fatalf("unexpected error for malformed URI: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result for malformed URI, got %v", got)
	}
}

func TestInverseAmends_expressionURITarget(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	target := sampleAct()
	if err := s.AddAct(target); err != nil {
		t.Fatal(err)
	}

	amender := &schema.Act{
		Country: "ua", TypeSlug: "zakon", Year: 2024, Number: "789-1",
		Expression: &schema.Expression{
			Title:       "Закон про внесення змін",
			LangTag:     "uk",
			VersionDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			Amends:      []string{target.ExpressionURI()},
		},
	}
	if err := s.AddAct(amender); err != nil {
		t.Fatal(err)
	}

	// Querying by expression URI should also find the amender.
	got, err := s.InverseAmends(target.ExpressionURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != amender.ResourceURI() {
		t.Errorf("InverseAmends by expression URI = %v, want [%s]", got, amender.ResourceURI())
	}
}

func TestInverseAmends_noRelations(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	// Add an act with no amending relationships.
	act := sampleAct()
	act.Expression.Amends = nil
	if err := s.AddAct(act); err != nil {
		t.Fatal(err)
	}

	got, err := s.InverseAmends(act.ResourceURI())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no amenders, got %v", got)
	}
}

// --- GetActs batch tests ---

func TestGetActs_emptySlice(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	got, err := s.GetActs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d acts", len(got))
	}
}

func TestGetActs_single(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	act := sampleAct()
	if err := s.AddAct(act); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetActs([]string{act.ResourceURI()})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 act, got %d", len(got))
	}
	if got[0].ResourceURI() != act.ResourceURI() {
		t.Errorf("got %s, want %s", got[0].ResourceURI(), act.ResourceURI())
	}
}

func TestGetActs_multiple(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	uris := make([]string, 3)
	for i, num := range []string{"111-1", "222-2", "333-3"} {
		act := sampleAct()
		act.Number = num
		act.IDLocal = num
		if err := s.AddAct(act); err != nil {
			t.Fatal(err)
		}
		uris[i] = act.ResourceURI()
	}

	got, err := s.GetActs(uris)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 acts, got %d", len(got))
	}
	for i, a := range got {
		if a.ResourceURI() != uris[i] {
			t.Errorf("act[%d] = %s, want %s", i, a.ResourceURI(), uris[i])
		}
	}
}

func TestGetActs_malformedURI(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	_, err := s.GetActs([]string{"not-a-valid-uri"})
	if err == nil {
		t.Error("expected error for malformed URI, got nil")
	}
}

func TestGetActs_missingAct(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()
	_, err := s.GetActs([]string{schema.ResourceURI("ua", "zakon", 1999, "nonexistent")})
	if err == nil {
		t.Error("expected error for missing act, got nil")
	}
}

func TestGetActs_duplicateURIs(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	act := sampleAct()
	if err := s.AddAct(act); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetActs([]string{act.ResourceURI(), act.ResourceURI()})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 acts (duplicates allowed), got %d", len(got))
	}
}

func TestGetActs_partialFailure(t *testing.T) {
	s, _ := OpenMemory()
	defer s.Close()

	act := sampleAct()
	if err := s.AddAct(act); err != nil {
		t.Fatal(err)
	}

	// First URI is valid, second is missing — should return error.
	_, err := s.GetActs([]string{act.ResourceURI(), schema.ResourceURI("ua", "zakon", 1999, "missing")})
	if err == nil {
		t.Error("expected error when any URI is missing, got nil")
	}
}

func assertActEqual(t *testing.T, want, got *schema.Act) {
	t.Helper()
	if got.Country != want.Country || got.TypeSlug != want.TypeSlug ||
		got.Year != want.Year || got.Number != want.Number {
		t.Errorf("identity mismatch: got (%s,%s,%d,%s)", got.Country, got.TypeSlug, got.Year, got.Number)
	}
	w, g := want.Expression, got.Expression
	if g == nil {
		t.Fatal("got nil expression")
	}
	if g.Title != w.Title {
		t.Errorf("title = %q, want %q", g.Title, w.Title)
	}
	if g.LangTag != w.LangTag {
		t.Errorf("langTag = %q, want %q", g.LangTag, w.LangTag)
	}
	if g.LangAlpha3 != w.LangAlpha3 {
		t.Errorf("langAlpha3 = %q, want %q", g.LangAlpha3, w.LangAlpha3)
	}
	if !g.VersionDate.Equal(w.VersionDate) {
		t.Errorf("versionDate = %v, want %v", g.VersionDate, w.VersionDate)
	}
	if !g.FirstInForceDate.Equal(w.FirstInForceDate) {
		t.Errorf("firstInForce = %v, want %v", g.FirstInForceDate, w.FirstInForceDate)
	}
	if g.Status != w.Status {
		t.Errorf("status = %v, want %v", g.Status, w.Status)
	}
	if g.SourceURL != w.SourceURL {
		t.Errorf("sourceURL = %q, want %q", g.SourceURL, w.SourceURL)
	}
	if !g.RetrievedAt.Equal(w.RetrievedAt) {
		t.Errorf("retrievedAt = %v, want %v", g.RetrievedAt, w.RetrievedAt)
	}
	if len(g.Articles) != len(w.Articles) {
		t.Fatalf("articles = %d, want %d", len(g.Articles), len(w.Articles))
	}
	for i := range w.Articles {
		if g.Articles[i] != w.Articles[i] {
			t.Errorf("article[%d] = %+v, want %+v", i, g.Articles[i], w.Articles[i])
		}
	}
	if len(g.Cites) != len(w.Cites) {
		t.Fatalf("cites = %d, want %d", len(g.Cites), len(w.Cites))
	}
	for i := range w.Cites {
		if g.Cites[i] != w.Cites[i] {
			t.Errorf("cites[%d] = %q, want %q", i, g.Cites[i], w.Cites[i])
		}
	}
}
