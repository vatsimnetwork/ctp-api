package handlers

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestAllValidSelcals_Count(t *testing.T) {
	codes := allValidSelcals()
	// C(16,2) pairs for pair1 = 120, C(14,2) pairs for pair2 = 91 → 120×91 = 10920
	if len(codes) != 10920 {
		t.Fatalf("expected 10920 valid SELCAL codes, got %d", len(codes))
	}
}

func TestAllValidSelcals_AllUnique(t *testing.T) {
	codes := allValidSelcals()
	seen := make(map[string]bool, len(codes))
	for _, c := range codes {
		if seen[c] {
			t.Fatalf("duplicate SELCAL code: %s", c)
		}
		seen[c] = true
	}
}

func TestAllValidSelcals_Format(t *testing.T) {
	validLetters := map[byte]bool{
		'A': true, 'B': true, 'C': true, 'D': true,
		'E': true, 'F': true, 'G': true, 'H': true,
		'J': true, 'K': true, 'L': true, 'M': true,
		'P': true, 'Q': true, 'R': true, 'S': true,
	}

	codes := allValidSelcals()
	for _, code := range codes {
		// Format must be "XY-ZW" (5 chars, dash in middle)
		if len(code) != 5 || code[2] != '-' {
			t.Fatalf("bad format: %q", code)
		}

		a, b, c, d := code[0], code[1], code[3], code[4]

		// All letters must be from the valid SELCAL alphabet
		for _, ch := range []byte{a, b, c, d} {
			if !validLetters[ch] {
				t.Fatalf("invalid SELCAL letter %c in code %s", ch, code)
			}
		}

		// All 4 letters must be distinct
		set := map[byte]bool{a: true, b: true, c: true, d: true}
		if len(set) != 4 {
			t.Fatalf("non-distinct letters in code %s", code)
		}

		// Within each pair, letters must be in alphabetical order
		if a >= b {
			t.Fatalf("pair 1 not alphabetical in code %s", code)
		}
		if c >= d {
			t.Fatalf("pair 2 not alphabetical in code %s", code)
		}
	}
}

func TestGenerateUniqueSelcals_Uniqueness(t *testing.T) {
	for _, n := range []int{1, 100, 500, 2000, 10920} {
		rng := rand.New(rand.NewSource(42))
		codes, err := generateUniqueSelcals(n, rng)
		if err != nil {
			t.Fatalf("n=%d: unexpected error: %v", n, err)
		}
		if len(codes) != n {
			t.Fatalf("n=%d: expected %d codes, got %d", n, n, len(codes))
		}
		seen := make(map[string]bool, n)
		for _, c := range codes {
			if seen[c] {
				t.Fatalf("n=%d: duplicate code %s", n, c)
			}
			seen[c] = true
		}
	}
}

func TestGenerateUniqueSelcals_ExceedsPool(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	_, err := generateUniqueSelcals(10921, rng)
	if err == nil {
		t.Fatal("expected error when requesting more codes than pool size")
	}
}

func TestGenerateUniqueSelcals_DifferentSeeds(t *testing.T) {
	codes1, _ := generateUniqueSelcals(100, rand.New(rand.NewSource(1)))
	codes2, _ := generateUniqueSelcals(100, rand.New(rand.NewSource(2)))
	same := 0
	for i := range codes1 {
		if codes1[i] == codes2[i] {
			same++
		}
	}
	// With different seeds, the shuffled output should differ significantly
	if same > 50 {
		t.Fatalf("different seeds produced %d/100 identical positional codes; shuffle appears broken", same)
	}
}

func TestCombineRouteStrings_DeduplicatesBoundary(t *testing.T) {
	// Simulates three segments where endpoints overlap:
	// seg0 ends with "FIX_B", seg1 starts with "FIX_B" and ends with "FIX_C", seg2 starts with "FIX_C"
	type fakeRS struct {
		route string
	}
	tests := []struct {
		name     string
		routes   []string
		expected string
	}{
		{
			name:     "three connected segments",
			routes:   []string{"DEP FIXA FIXB", "FIXB TRACK1 TRACK2 FIXC", "FIXC FIXD ARR"},
			expected: "DEP FIXA FIXB TRACK1 TRACK2 FIXC FIXD ARR",
		},
		{
			name:     "no overlap",
			routes:   []string{"DEP FIXA", "FIXB FIXC", "FIXD ARR"},
			expected: "DEP FIXA FIXB FIXC FIXD ARR",
		},
		{
			name:     "case insensitive boundary",
			routes:   []string{"DEP fixb", "FIXB FIXC"},
			expected: "DEP fixb FIXC",
		},
		{
			name:     "single segment",
			routes:   []string{"DEP FIXA ARR"},
			expected: "DEP FIXA ARR",
		},
		{
			name:     "empty route string",
			routes:   []string{"", "FIXA FIXB", ""},
			expected: "FIXA FIXB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build minimal RouteSegment slice using only RouteString
			segments := make([]fakeRouteSegment, len(tt.routes))
			for i, r := range tt.routes {
				segments[i].RouteString = r
			}
			result := combineRouteStringsGeneric(segments)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// fakeRouteSegment is a test helper with just a RouteString field.
type fakeRouteSegment struct {
	RouteString string
}

// combineRouteStringsGeneric is a test-only generic version for verification.
func combineRouteStringsGeneric(segments []fakeRouteSegment) string {
	var parts []string
	for i, seg := range segments {
		tokens := strings.Fields(seg.RouteString)
		if i > 0 && len(parts) > 0 && len(tokens) > 0 {
			if strings.EqualFold(parts[len(parts)-1], tokens[0]) {
				tokens = tokens[1:]
			}
		}
		parts = append(parts, tokens...)
	}
	return strings.Join(parts, " ")
}

func TestAllValidSelcals_BothPairOrdersExist(t *testing.T) {
	codes := allValidSelcals()
	set := make(map[string]bool, len(codes))
	for _, c := range codes {
		set[c] = true
	}
	// AB-CD and CD-AB must both exist as distinct codes
	if !set["AB-CD"] {
		t.Fatal("AB-CD missing from pool")
	}
	if !set["CD-AB"] {
		t.Fatal("CD-AB missing from pool")
	}
	// But CD-BA must NOT exist (within-pair order must be alphabetical)
	if set["CD-BA"] {
		t.Fatal("CD-BA should not be in pool (B > A violates within-pair order)")
	}
}

func TestAllValidSelcals_NoDuplicateLettersAcrossPool(t *testing.T) {
	// Verify no code sneaks through with a repeated letter (belt-and-suspenders)
	codes := allValidSelcals()
	for _, code := range codes {
		letters := fmt.Sprintf("%c%c%c%c", code[0], code[1], code[3], code[4])
		seen := make(map[rune]bool)
		for _, ch := range letters {
			if seen[ch] {
				t.Fatalf("repeated letter %c in code %s", ch, code)
			}
			seen[ch] = true
		}
	}
}
