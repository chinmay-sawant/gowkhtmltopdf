package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	fontCacheTestEntryBytes = 16
	readFontTestContents    = "0123456789"
)

// staticFileInfo overrides only the size the sized read path consults; the
// embedded nil FileInfo satisfies the interface for methods that never run.
type staticFileInfo struct {
	os.FileInfo
	size int64
}

func (f staticFileInfo) Size() int64 { return f.size }

func writeTestFont(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, bytes.Clone(testFont(t).data), 0o600); err != nil {
		t.Fatalf("write font: %v", err)
	}

	return path
}

func TestFontFileCacheHitMissAndKeyChange(t *testing.T) {
	t.Parallel()

	cache := newFontFileCacheWithLimits(1<<20, 8)
	key := fontFileKey{path: "/fonts/face.ttf", size: 64, modNano: 1}
	fnt := &Font{data: make([]byte, 64)}

	if _, ok := cache.get(key); ok {
		t.Fatal("empty cache reported a hit")
	}

	cache.put(key, fnt)

	got, ok := cache.get(key)
	if !ok || got != fnt {
		t.Fatalf("get() = (%p, %v), want the cached pointer %p", got, ok, fnt)
	}

	touched := key
	touched.modNano = 2

	if _, ok := cache.get(touched); ok {
		t.Fatal("mtime change must miss")
	}

	resized := key
	resized.size = 65

	if _, ok := cache.get(resized); ok {
		t.Fatal("size change must miss")
	}

	other := fontFileKey{path: "/fonts/other.ttf", size: 64, modNano: 1}
	if _, ok := cache.get(other); ok {
		t.Fatal("different path must miss")
	}
}

func TestFontFileCacheEvictsLeastRecentlyUsedEntry(t *testing.T) {
	t.Parallel()

	cache := newFontFileCacheWithLimits(1<<20, 2)
	keyA := fontFileKey{path: "a", modNano: 1}
	keyB := fontFileKey{path: "b", modNano: 1}
	keyC := fontFileKey{path: "c", modNano: 1}
	fntA := &Font{}
	fntB := &Font{}
	fntC := &Font{}

	cache.put(keyA, fntA)
	cache.put(keyB, fntB)

	if _, ok := cache.get(keyA); !ok {
		t.Fatal("A should be cached")
	}

	cache.put(keyC, fntC) // A was just used, so B is the oldest.

	if _, ok := cache.get(keyB); ok {
		t.Fatal("B should have been evicted")
	}

	if _, ok := cache.get(keyA); !ok {
		t.Fatal("A should survive as the recently used entry")
	}

	if _, ok := cache.get(keyC); !ok {
		t.Fatal("C should be cached")
	}
}

func TestFontFileCacheEvictsOverByteBudget(t *testing.T) {
	t.Parallel()

	// Room for exactly two entries; the third squeezes out the oldest.
	cache := newFontFileCacheWithLimits(2*fontCacheTestEntryBytes, 8)
	fntA := &Font{data: make([]byte, fontCacheTestEntryBytes)}
	fntB := &Font{data: make([]byte, fontCacheTestEntryBytes)}
	fntC := &Font{data: make([]byte, fontCacheTestEntryBytes)}

	cache.put(fontFileKey{path: "a", modNano: 1}, fntA)
	cache.put(fontFileKey{path: "b", modNano: 1}, fntB)
	cache.put(fontFileKey{path: "c", modNano: 1}, fntC)

	if _, ok := cache.get(fontFileKey{path: "a", modNano: 1}); ok {
		t.Fatal("A should have been evicted to honor the byte budget")
	}

	for _, path := range []string{"b", "c"} {
		if _, ok := cache.get(fontFileKey{path: path, modNano: 1}); !ok {
			t.Fatalf("%s should stay cached", path)
		}
	}
}

func TestFontFileCacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := newFontFileCache()

	const (
		workers = 8
		rounds  = 200
	)

	var group sync.WaitGroup

	for worker := range workers {
		group.Add(1)

		go func(worker int) {
			defer group.Done()

			for round := range rounds {
				key := fontFileKey{path: "face", size: int64(worker), modNano: int64(round % 4)}
				cache.put(key, &Font{data: make([]byte, fontCacheTestEntryBytes)})
				cache.get(key)
			}
		}(worker)
	}

	group.Wait()
}

func TestScanFontDirsSharesParsedFaceAcrossScans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTestFont(t, dir, "shared-face.ttf")

	first := ScanFontDirs([]string{dir})
	if len(first.faces) != 1 {
		t.Fatalf("first scan faces = %d, want 1", len(first.faces))
	}

	second := ScanFontDirs([]string{dir})
	if len(second.faces) != 1 {
		t.Fatalf("second scan faces = %d, want 1", len(second.faces))
	}

	if first.faces[0] != second.faces[0] {
		t.Fatal("second scan reparsed the face instead of hitting the cache")
	}

	// A touched file must invalidate: new key, new parse, new pointer.
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	third := ScanFontDirs([]string{dir})
	if len(third.faces) != 1 {
		t.Fatalf("third scan faces = %d, want 1", len(third.faces))
	}

	if third.faces[0] == second.faces[0] {
		t.Fatal("mtime change must invalidate the cached face")
	}
}

func TestReadFontFileMatchesStatSize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sized.ttf")
	data := []byte(readFontTestContents)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	got, err := readFontFile(path, info, nil)
	if err != nil {
		t.Fatalf("readFontFile: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("readFontFile = %q, want %q", got, data)
	}

	// The buffer is stat size + 1 so ReadFull stops one byte short and
	// reports the clean short read instead of allocating a second copy.
	if want := len(data) + 1; cap(got) != want {
		t.Fatalf("cap(readFontFile) = %d, want %d (stat size + 1)", cap(got), want)
	}
}

func TestReadFontFileFallsBackWhenFileGrewAfterStat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "grown.ttf")
	data := []byte(readFontTestContents)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readFontFile(path, staticFileInfo{size: 4}, nil)
	if err != nil {
		t.Fatalf("readFontFile: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("grown read = %q, want %q", got, data)
	}
}

func TestReadFontFileReturnsShortReadWhenFileShrankAfterStat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "shrunk.ttf")
	data := []byte(readFontTestContents)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readFontFile(path, staticFileInfo{size: 64}, nil)
	if err != nil {
		t.Fatalf("readFontFile: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("shrunk read = %q, want %q", got, data)
	}
}

func TestReadFontFileWithoutStatUsesLimitedRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nostat.ttf")
	data := []byte(readFontTestContents)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readFontFile(path, nil, os.ErrNotExist)
	if err != nil {
		t.Fatalf("readFontFile: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("fallback read = %q, want %q", got, data)
	}
}
