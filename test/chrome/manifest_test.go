package chrome_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type caseManifest struct {
	CaseCount int            `json:"caseCount"`
	Cases     []manifestCase `json:"cases"`
}

type manifestCase struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Fixture     string `json:"fixture"`
	GoTarget    string `json:"goTarget"`
	Expected    string `json:"expected"`
	Status      string `json:"status"`
	Combination string `json:"combination"`
}

func readManifest(tb testing.TB) caseManifest {
	tb.Helper()

	raw, err := os.ReadFile("manifest.json")
	if err != nil {
		tb.Fatalf("read manifest: %v", err)
	}

	var manifest caseManifest

	if err := json.Unmarshal(raw, &manifest); err != nil {
		tb.Fatalf("decode manifest: %v", err)
	}

	return manifest
}

func TestManifestHasFortyUniqueCases(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)
	if manifest.CaseCount != 40 {
		t.Fatalf("manifest caseCount = %d, want 40", manifest.CaseCount)
	}

	if len(manifest.Cases) != manifest.CaseCount {
		t.Fatalf("manifest cases = %d, want %d", len(manifest.Cases), manifest.CaseCount)
	}

	ids := make(map[string]struct{}, len(manifest.Cases))
	validTargets := map[string]bool{
		"layout-unit":      true,
		"chrome-reference": true,
		"golden-fixture":   true,
	}
	validStatuses := map[string]bool{
		"scaffold":    true,
		"completed":   true,
		"blocked":     true,
		"unsupported": true,
	}

	for _, item := range manifest.Cases {
		assertManifestCase(t, item, ids, validTargets, validStatuses)
	}
}

func TestManifestSourcesExistWhenChromiumCheckoutIsPresent(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)
	root := filepath.Join("..", "..", "chromium")

	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		t.Skip("local Chromium checkout is not present")
	}

	for _, item := range manifest.Cases {
		source := strings.SplitN(item.Source, "::", 2)[0]
		if _, err := os.Stat(filepath.Join(root, source)); err != nil {
			t.Errorf("case %q source %q: %v", item.ID, source, err)
		}
	}
}

func assertManifestCase(
	t *testing.T,
	item manifestCase,
	ids map[string]struct{},
	validTargets, validStatuses map[string]bool,
) {
	t.Helper()

	assertManifestMetadata(t, item, ids, validTargets, validStatuses)
	assertFixture(t, item)
}

func assertManifestMetadata(
	t *testing.T,
	item manifestCase,
	ids map[string]struct{},
	validTargets, validStatuses map[string]bool,
) {
	t.Helper()

	if item.ID == "" {
		t.Error("case has empty id")
	}

	if _, exists := ids[item.ID]; exists {
		t.Errorf("duplicate case id %q", item.ID)
	}

	ids[item.ID] = struct{}{}

	if !validTargets[item.GoTarget] {
		t.Errorf("case %q has unknown Go target %q", item.ID, item.GoTarget)
	}

	if !validStatuses[item.Status] {
		t.Errorf("case %q has unknown status %q", item.ID, item.Status)
	}

	if item.Source == "" || item.Combination == "" || item.Expected == "" {
		t.Errorf("case %q is missing source, combination, or expected behavior", item.ID)
	}
}

func assertFixture(t *testing.T, item manifestCase) {
	t.Helper()

	if filepath.Clean(item.Fixture) != item.Fixture || !strings.HasPrefix(item.Fixture, "cases/") {
		t.Errorf("case %q has unsafe fixture path %q", item.ID, item.Fixture)

		return
	}

	fixture, err := os.ReadFile(item.Fixture)
	if err != nil {
		t.Errorf("case %q read fixture %q: %v", item.ID, item.Fixture, err)

		return
	}

	fixtureText := string(fixture)
	if !strings.HasPrefix(fixtureText, "<!doctype html>") {
		t.Errorf("case %q fixture does not start with a doctype", item.ID)
	}

	if !strings.Contains(fixtureText, "Port status: "+item.Status) {
		t.Errorf("case %q fixture lacks %q marker", item.ID, "Port status: "+item.Status)
	}
}
