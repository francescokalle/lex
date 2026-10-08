package search

import (
	"sort"
	"strings"
	"unicode"
)

// Stemmer reduces an inflected token to a stem so that morphological variants
// of a word match each other. Stemming is language-specific, so the index
// records its language and selects the matching stemmer; serving reuses it.
// See docs/TODO.md — this must be filled in per country/language.
type Stemmer interface {
	Stem(token string) string
}

// identityStemmer is the default for languages without a stemmer yet: it leaves
// tokens unchanged (behaves like plain unicode61 matching).
type identityStemmer struct{}

func (identityStemmer) Stem(t string) string { return t }

// StemmerFor returns the stemmer for a BCP-47 language code, or the identity
// stemmer if none is registered.
func StemmerFor(lang string) Stemmer {
	switch strings.ToLower(lang) {
	case "en":
		return englishStemmer{}
	case "fr":
		return frenchStemmer{}
	case "de":
		return germanStemmer{}
	case "es":
		return spanishStemmer{}
	case "pl":
		return polishStemmer{}
	case "fi":
		return finnishStemmer{}
	case "ja":
		return japaneseStemmer{}
	case "uk":
		return ukrainianStemmer{}
	default:
		return identityStemmer{}
	}
}

// --- Ukrainian ---

// ukrainianStemmer is a lightweight, conservative suffix-stripping stemmer for
// Ukrainian: it removes the most common inflectional endings so case/number
// variants of nouns and adjectives collapse (оренда/оренду/оренди → оренд,
// реєстрація/реєстрації → реєстрац). It is intentionally cautious — it only
// strips when a vowel-bearing stem of at least 3 letters remains — so it favours
// recall without aggressive over-merging. Not a full morphological analyser.
type ukrainianStemmer struct{}

var ukVowels = map[rune]bool{'а': true, 'е': true, 'и': true, 'і': true, 'о': true, 'у': true, 'ю': true, 'я': true, 'є': true, 'ї': true}

// ukEndings is sorted by descending rune length in init so the longest ending
// matches first.
var ukEndings = []string{
	"ами", "ями", "ого", "ому", "ими", "іми", "ією", "іях", "іям",
	"ах", "ях", "ам", "ям", "ів", "ом", "ем", "ою", "ею", "их", "ій",
	"ім", "ої", "ія", "ії", "ію", "ей", "єю", "ий",
	"а", "я", "и", "і", "о", "у", "ю", "е", "й", "ь", "ї", "є",
}

func init() {
	sort.SliceStable(ukEndings, func(i, j int) bool {
		return len([]rune(ukEndings[i])) > len([]rune(ukEndings[j]))
	})
}

func (ukrainianStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w) // too short to strip safely
	}
	for _, suf := range ukEndings {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsUkVowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	if len(w) > 3 && w[len(w)-1] == 'ь' {
		w = w[:len(w)-1]
	}
	return string(w)
}

func hasRuneSuffix(w, suf []rune) bool {
	if len(suf) > len(w) {
		return false
	}
	off := len(w) - len(suf)
	for i := range suf {
		if w[off+i] != suf[i] {
			return false
		}
	}
	return true
}

func containsUkVowel(rs []rune) bool {
	for _, r := range rs {
		if ukVowels[r] {
			return true
		}
	}
	return false
}

// --- English ---

// englishStemmer is a conservative, rule-based English stemmer inspired by
// Porter. It removes the most common plural and verb suffixes so inflection
// families collapse (renting/rented/rent → rent; acts/acting → act).
// It only strips when a vowel-bearing stem of at least 3 letters remains.
type englishStemmer struct{}

var enVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true}

// enSuffixes ordered by descending length (longest match first).
var enSuffixes = []string{
	"ization", "ational", "fulness", "ousness",
	"ing", "edly", "edly",
	"ies", "ied", "es", "ed", "ly", "al", "er", "or", "s",
}

func (englishStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range enSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsEnvowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsEnvowel(rs []rune) bool {
	for _, r := range rs {
		if enVowels[r] {
			return true
		}
	}
	return false
}

// --- French ---

// frenchStemmer is a conservative French suffix stripper. It removes common
// plural and adjectival endings so inflection families collapse
// (contrats/contrat → contrat; loi/lois → loi). Conservative: minimum 3-letter
// stem, vowel required.
type frenchStemmer struct{}

var frVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'y': true, 'â': true, 'à': true, 'é': true, 'è': true, 'ê': true, 'î': true, 'ï': true, 'ô': true, 'û': true, 'ù': true, 'ë': true, 'ü': true}

var frSuffixes = []string{
	"issement", "issantes", "issante", "issants",
	"ements", "ement", "ement", "ements",
	"ation", "ations",
	"eur", "euse", "eurs", "euses",
	"ive", "if", "ives", "ifs",
	"ité", "ités",
	"eux", "euse",
	"ais", "ait", "aient", "iez", "ions", "ant", "ent",
	"aux", "er", "e", "es", "s",
}

func (frenchStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range frSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsFRvowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsFRvowel(rs []rune) bool {
	for _, r := range rs {
		if frVowels[r] {
			return true
		}
	}
	return false
}

// --- German ---

// germanStemmer is a conservative German suffix stripper. It removes common
// nominal and adjectival endings so inflection families collapse
// (gesetze/gesetzen/gesetzes → gesetz). Conservative: minimum 3-letter stem,
// vowel required.
type germanStemmer struct{}

var deVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'ä': true, 'ö': true, 'ü': true}

var deSuffixes = []string{
	"ungen", "ungen", "ungen", "ungen",
	"ischen", "ische", "ischer", "isches",
	"ischen", "ischer", "isches",
	"liche", "licher", "liches",
	"heit", "heiten",
	"keit", "keiten",
	"ung", "ungen",
	"en", "er", "es", "e", "n",
}

func (germanStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range deSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsDEVowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsDEVowel(rs []rune) bool {
	for _, r := range rs {
		if deVowels[r] {
			return true
		}
	}
	return false
}

// --- Spanish ---

// spanishStemmer is a conservative Spanish suffix stripper. It removes common
// nominal and verbal endings so inflection families collapse
// (contratos/contrato → contrato; leyes/ley → ley). Conservative: minimum
// 3-letter stem, vowel required.
type spanishStemmer struct{}

var esVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'á': true, 'é': true, 'í': true, 'ó': true, 'ú': true, 'ü': true}

var esSuffixes = []string{
	"imientos", "imiento",
	"amiento", "amientos",
	"adora", "adoras", "ador", "adores",
	"antes", "ante",
	"ancia", "ancias",
	"mente",
	"ción", "ciones",
	"eros", "era", "eras", "ero",
	"icos", "ica", "icas",	"ico",
	"ar", "os", "as", "a", "o", "es", "s",
}

func (spanishStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range esSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsESvowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsESvowel(rs []rune) bool {
	for _, r := range rs {
		if esVowels[r] {
			return true
		}
	}
	return false
}

// --- Polish ---

// polishStemmer is a conservative Polish suffix stripper. It removes common
// nominal and adjectival endings so inflection families collapse
// (budżetowej/budżetowych/budżetowy → budżet). Conservative: minimum 3-letter
// stem, vowel required.
type polishStemmer struct{}

var plVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'y': true, 'ą': true, 'ę': true, 'ó': true}

var plSuffixes = []string{
	"owanie", "owania", "owaniu",
	"owany", "owane", "owanych", "owanym", "owanej", "owaną", "owanymi",
	"owego", "owemu", "owej",
	"owych", "owym", "owymi", "owa", "owe", "owi", "owy",
	"anie", "ania", "aniu",
	"enie", "enia", "eniu",
	"en", "enia",
	"ej", "ym", "ymi", "ych", "ych",
	"ego", "emu", "em",
	"ami", "ach",
	"a", "e", "i", "o", "u", "y", "ę", "ą", "ie",
}

func (polishStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range plSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsPLvowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsPLvowel(rs []rune) bool {
	for _, r := range rs {
		if plVowels[r] {
			return true
		}
	}
	return false
}

// --- Finnish ---

// finnishStemmer is a conservative Finnish suffix stripper. Finnish has rich
// inflection; this removes the most common case endings so inflection families
// collapse (laki/lain → laki; talo/talossa/talosta → talo). Conservative:
// minimum 3-letter stem, vowel required.
type finnishStemmer struct{}

var fiVowels = map[rune]bool{'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'y': true, 'ä': true, 'ö': true}

var fiSuffixes = []string{
	"issä", "istä", "illa", "ille", "ilta", "issa", "ista",
	"ein", "eja",
	"ssa", "sta",
	"na", "nä",
	"lle",
	"ksi",
	"in",
	"an", "en", "on", "un", "yn",
	"ja", "jä",
}

func (finnishStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 4 {
		return string(w)
	}
	for _, suf := range fiSuffixes {
		sr := []rune(suf)
		stem := len(w) - len(sr)
		if stem < 3 || !hasRuneSuffix(w, sr) {
			continue
		}
		if containsFIvowel(w[:stem]) {
			w = w[:stem]
			break
		}
	}
	return string(w)
}

func containsFIvowel(rs []rune) bool {
	for _, r := range rs {
		if fiVowels[r] {
			return true
		}
	}
	return false
}

// --- Japanese ---

// japaneseStemmer handles Japanese, which has no word spaces. Instead of
// suffix stemming, it produces character bigrams from the token. This allows
// substring matching: "法律" and "法律令" share the bigram "法律".
// The stem function returns space-separated bigrams.
type japaneseStemmer struct{}

func (japaneseStemmer) Stem(token string) string {
	w := []rune(strings.ToLower(token))
	if len(w) < 2 {
		return string(w)
	}
	// Produce bigrams: for "法律" → "法律"; for "法律令" → "法律 律令"
	var parts []string
	for i := 0; i < len(w)-1; i++ {
		parts = append(parts, string(w[i:i+2]))
	}
	return strings.Join(parts, " ")
}

// --- tokenization shared by indexing, querying, and snippets ---

// tokenize splits text into word tokens on any non-letter, non-digit rune.
func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// stemColumn produces the space-joined stems indexed for a piece of text.
func stemColumn(st Stemmer, text string) string {
	toks := tokenize(text)
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		out = append(out, st.Stem(strings.ToLower(t)))
	}
	return strings.Join(out, " ")
}
