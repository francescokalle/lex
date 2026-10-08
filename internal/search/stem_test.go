package search

import "testing"

func TestUkrainianStemmer_families(t *testing.T) {
	st := StemmerFor("uk")
	// Each inflection family must collapse to a single stem.
	families := [][]string{
		{"оренда", "оренду", "оренди", "оренді", "орендою"},
		{"спадщина", "спадщини", "спадщину", "спадщині"},
		{"реєстрація", "реєстрації", "реєстрацію", "реєстрацій"},
		{"контроль", "контролю", "контролі"},
		{"земля", "землю", "землі"},
		{"правовий", "правова", "правові"},
	}
	for _, fam := range families {
		want := st.Stem(fam[0])
		for _, w := range fam[1:] {
			if got := st.Stem(w); got != want {
				t.Errorf("stem(%q)=%q, stem(%q)=%q — family should collapse", fam[0], want, w, got)
			}
		}
	}
}

func TestUkrainianStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("uk")
	// Unrelated roots must keep distinct stems.
	pairs := [][2]string{
		{"оренда", "продаж"},
		{"закон", "земля"},
		{"спадщина", "реєстрація"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestUkrainianStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("uk")
	if st.Stem("рік") != "рік" { // <4 runes: unchanged
		t.Errorf("short word changed: %q", st.Stem("рік"))
	}
	if st.Stem("435") != "435" {
		t.Errorf("digits changed: %q", st.Stem("435"))
	}
}

func TestIdentityStemmer(t *testing.T) {
	st := StemmerFor("xx") // no stemmer → identity
	if st.Stem("Renting") != "Renting" {
		t.Errorf("identity changed token: %q", st.Stem("Renting"))
	}
}

// --- English ---

func TestEnglishStemmer_families(t *testing.T) {
	st := StemmerFor("en")
	// Conservative stemmer: test pairs that should collapse.
	pairs := [][2]string{
		{"renting", "rent"},
		{"rented", "rent"},
		{"rents", "rent"},
		{"acting", "act"},
		{"acted", "act"},
		{"acts", "act"},
		{"laws", "law"},
		{"contracts", "contract"},
		{"runs", "run"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestEnglishStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("en")
	pairs := [][2]string{
		{"rent", "run"},
		{"law", "lawyer"},
		{"contract", "contrast"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestEnglishStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("en")
	if st.Stem("go") != "go" {
		t.Errorf("short word changed: %q", st.Stem("go"))
	}
	if st.Stem("123") != "123" {
		t.Errorf("digits changed: %q", st.Stem("123"))
	}
}

// --- French ---

func TestFrenchStemmer_families(t *testing.T) {
	st := StemmerFor("fr")
	pairs := [][2]string{
		{"contrats", "contrat"},
		{"lois", "loi"},
		{"actes", "act"},
		{"gouvernements", "gouvern"},
		{"états", "état"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestFrenchStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("fr")
	pairs := [][2]string{
		{"contrat", "contraire"},
		{"loi", "lointain"},
		{"été", "état"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestFrenchStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("fr")
	if st.Stem("au") != "au" {
		t.Errorf("short word changed: %q", st.Stem("au"))
	}
	if st.Stem("42") != "42" {
		t.Errorf("digits changed: %q", st.Stem("42"))
	}
}

// --- German ---

func TestGermanStemmer_families(t *testing.T) {
	st := StemmerFor("de")
	pairs := [][2]string{
		{"gesetze", "gesetz"},
		{"gesetzen", "gesetz"},
		{"gesetzes", "gesetz"},
		{"rechte", "recht"},
		{"rechten", "recht"},
		{"staaten", "staat"},
		{"artikeln", "artikel"},
		{"nationalen", "national"},
		{"arbeiten", "arbeit"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestGermanStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("de")
	pairs := [][2]string{
		{"gesetz", "gesetzt"},
		{"recht", "rechts"},
		{"staat", "staatlich"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestGermanStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("de")
	if st.Stem("ab") != "ab" {
		t.Errorf("short word changed: %q", st.Stem("ab"))
	}
	if st.Stem("99") != "99" {
		t.Errorf("digits changed: %q", st.Stem("99"))
	}
}

// --- Spanish ---

func TestSpanishStemmer_families(t *testing.T) {
	st := StemmerFor("es")
	pairs := [][2]string{
		{"contratos", "contrat"},
		{"leyes", "ley"},
		{"gobiernos", "gobiern"},
		{"estados", "estad"},
		{"nacionales", "nacional"},
		{"artículos", "artícul"},
		{"trabajos", "trabaj"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestSpanishStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("es")
	pairs := [][2]string{
		{"ley", "leyenda"},
		{"estado", "estadual"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestSpanishStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("es")
	if st.Stem("el") != "el" {
		t.Errorf("short word changed: %q", st.Stem("el"))
	}
	if st.Stem("77") != "77" {
		t.Errorf("digits changed: %q", st.Stem("77"))
	}
}

// --- Polish ---

func TestPolishStemmer_families(t *testing.T) {
	st := StemmerFor("pl")
	pairs := [][2]string{
		{"budżetowej", "budżet"},
		{"budżetowych", "budżet"},
		{"ustawy", "ustaw"},
		{"prawa", "praw"},
		{"artykuły", "artykuł"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestPolishStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("pl")
	pairs := [][2]string{
		{"ustawa", "ustawka"},
		{"prawo", "prawnik"},
		{"rząd", "rządzić"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestPolishStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("pl")
	if st.Stem("od") != "od" {
		t.Errorf("short word changed: %q", st.Stem("od"))
	}
	if st.Stem("55") != "55" {
		t.Errorf("digits changed: %q", st.Stem("55"))
	}
}

// --- Finnish ---

func TestFinnishStemmer_families(t *testing.T) {
	st := StemmerFor("fi")
	pairs := [][2]string{
		{"talossa", "talo"},
		{"talosta", "talo"},
		{"valtiossa", "valtio"},
		{"valtiosta", "valtio"},
	}
	for _, p := range pairs {
		if got := st.Stem(p[0]); got != p[1] {
			t.Errorf("stem(%q)=%q, want %q", p[0], got, p[1])
		}
	}
}

func TestFinnishStemmer_doesNotOvermerge(t *testing.T) {
	st := StemmerFor("fi")
	pairs := [][2]string{
		{"laki", "lakia"},
	}
	for _, p := range pairs {
		if st.Stem(p[0]) == st.Stem(p[1]) {
			t.Errorf("stem(%q)==stem(%q)=%q — should differ", p[0], p[1], st.Stem(p[0]))
		}
	}
}

func TestFinnishStemmer_shortAndNonWord(t *testing.T) {
	st := StemmerFor("fi")
	if st.Stem("on") != "on" {
		t.Errorf("short word changed: %q", st.Stem("on"))
	}
	if st.Stem("33") != "33" {
		t.Errorf("digits changed: %q", st.Stem("33"))
	}
}

// --- Japanese ---

func TestJapaneseStemmer_bigrams(t *testing.T) {
	st := StemmerFor("ja")
	// Japanese has no spaces; bigram tokenization enables substring matching.
	tests := []struct {
		input string
		want  string
	}{
		{"法律", "法律"},
		{"法律令", "法律 律令"},
		{"民法", "民法"},
		{"民法総則", "民法 法総 総則"},
		{"憲法", "憲法"},
		{"a", "a"}, // single char: unchanged
		{"", ""},   // empty: unchanged
	}
	for _, tt := range tests {
		if got := st.Stem(tt.input); got != tt.want {
			t.Errorf("stem(%q)=%q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJapaneseStemmer_sharedBigrams(t *testing.T) {
	st := StemmerFor("ja")
	// Words sharing a substring share bigrams → matchable.
	s1 := st.Stem("法律")
	s2 := st.Stem("法律令")
	// "法律" is a bigram of both
	if !contains(s1, "法律") {
		t.Errorf("stem(%q) should contain bigram 法律", "法律")
	}
	if !contains(s2, "法律") {
		t.Errorf("stem(%q) should contain bigram 法律", "法律令")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- StemmerFor dispatch ---

func TestStemmerFor_dispatch(t *testing.T) {
	tests := []struct {
		lang string
		want Stemmer
	}{
		{"en", englishStemmer{}},
		{"fr", frenchStemmer{}},
		{"de", germanStemmer{}},
		{"es", spanishStemmer{}},
		{"pl", polishStemmer{}},
		{"fi", finnishStemmer{}},
		{"ja", japaneseStemmer{}},
		{"uk", ukrainianStemmer{}},
		{"xx", identityStemmer{}},
		{"", identityStemmer{}},
	}
	for _, tt := range tests {
		got := StemmerFor(tt.lang)
		if got != tt.want {
			t.Errorf("StemmerFor(%q) = %T, want %T", tt.lang, got, tt.want)
		}
	}
}

func TestStemmerFor_caseInsensitive(t *testing.T) {
	if StemmerFor("EN") != StemmerFor("en") {
		t.Error("StemmerFor should be case-insensitive")
	}
	if StemmerFor("Fr") != StemmerFor("fr") {
		t.Error("StemmerFor should be case-insensitive")
	}
}
