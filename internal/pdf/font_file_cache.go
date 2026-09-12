package pdf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// fontFileCacheMaxBytes bounds the font bytes retained across conversions.
	// Ten golden-corpus faces need about 5 MiB; the headroom covers normal
	// opt-in system font sets without pinning a whole tree.
	fontFileCacheMaxBytes = 32 << 20
	// fontFileCacheMaxEntries bounds the entry count independently of bytes so
	// many tiny faces cannot grow the map without limit.
	fontFileCacheMaxEntries = 128
)

// fontFileKey identifies one file version. Size and mtime are part of the key,
// so a rewritten file misses and reparses instead of serving stale metrics.
type fontFileKey struct {
	path    string
	size    int64
	modNano int64
}

// fontFileCache is a bounded LRU of parsed fonts shared across conversions.
// Parsed *Font values are immutable after parse (their lazy name/face/cmap
// caches are sync.Once guarded), so one parse serves every conversion.
// Registries stay per-conversion because callers mutate them.
type fontFileCache struct {
	mu         sync.Mutex
	entries    map[fontFileKey]*Font
	order      []fontFileKey
	bytes      int64
	maxBytes   int64
	maxEntries int
}

//nolint:gochecknoglobals // cross-conversion cache
var parsedFontCache = newFontFileCache()

func newFontFileCache() *fontFileCache {
	return newFontFileCacheWithLimits(fontFileCacheMaxBytes, fontFileCacheMaxEntries)
}

func newFontFileCacheWithLimits(maxBytes int64, maxEntries int) *fontFileCache {
	return &fontFileCache{ //nolint:exhaustruct // zero-value mutex and byte counter are intentional
		entries:    make(map[fontFileKey]*Font),
		order:      make([]fontFileKey, 0, maxEntries),
		maxBytes:   maxBytes,
		maxEntries: maxEntries,
	}
}

// get returns the cached font for key and marks it most recently used.
func (c *fontFileCache) get(key fontFileKey) (*Font, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fnt, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)

			break
		}
	}

	c.order = append(c.order, key)

	return fnt, true
}

// put stores fnt, evicting least-recently-used entries until both the entry
// count and the retained byte budget fit.
func (c *fontFileCache) put(key fontFileKey, fnt *Font) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; ok {
		return
	}

	size := int64(len(fnt.data))

	for len(c.order) >= c.maxEntries ||
		(c.bytes+size > c.maxBytes && len(c.order) > 0) {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.bytes -= int64(len(c.entries[oldest].data))
		delete(c.entries, oldest)
	}

	c.entries[key] = fnt
	c.order = append(c.order, key)
	c.bytes += size
}

// loadFontFile returns a parsed font for one directory entry, serving it from
// the shared cache when the entry's version is known. Files larger than
// maxFontBytes are skipped from the stat alone, before any read.
func loadFontFile(path string, entry os.DirEntry) *Font {
	info, infoErr := entry.Info()
	if fontEntryTooLarge(info, infoErr) {
		return nil
	}

	key, cacheable := fontFileKeyFor(path, entry, info, infoErr)

	if cacheable {
		if fnt, ok := parsedFontCache.get(key); ok {
			return fnt
		}
	}

	data, err := readFontFile(path, info, infoErr)
	if err != nil || len(data) > maxFontBytes {
		return nil
	}

	fnt, err := ParseTTF(data)
	if err != nil {
		return nil
	}

	if fnt.PostScriptName == "" {
		fnt.PostScriptName = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
	}

	// Cache only reads that matched the stat behind the key; a file that
	// changed size mid-scan stays uncached instead of being mislabeled.
	if cacheable && int64(len(data)) == info.Size() {
		parsedFontCache.put(key, fnt)
	}

	return fnt
}

// fontEntryTooLarge reports whether a usable stat shows a file past the parse
// cap. A failed stat cannot prove the size, so the file stays eligible and the
// read path enforces the cap instead.
func fontEntryTooLarge(info os.FileInfo, infoErr error) bool {
	return infoErr == nil && info.Size() > maxFontBytes
}

// fontFileKeyFor builds the cache key for one directory entry. A failed stat,
// a symlink (whose target bytes can change while the link's own stat stays
// fixed), or an unresolvable absolute path disables caching for that file.
func fontFileKeyFor(path string, entry os.DirEntry, info os.FileInfo, infoErr error) (fontFileKey, bool) {
	// Every failure path returns the zero key; callers must honor the bool
	// before using it.
	var zero fontFileKey

	if infoErr != nil || info == nil || entry.Type()&os.ModeSymlink != 0 {
		return zero, false
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return zero, false
	}

	return fontFileKey{path: abs, size: info.Size(), modNano: info.ModTime().UnixNano()}, true
}

// readFontFile reads one font file, bounding the first allocation by the stat
// size. A file that shrank or grew after the stat falls back to the limited
// read path so the maxFontBytes cap holds either way.
func readFontFile(path string, info os.FileInfo, infoErr error) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open font %s: %w", path, err)
	}
	defer file.Close()

	if infoErr != nil || info == nil || info.Size() > maxFontBytes {
		return readFontFileLimited(file, int64(maxFontBytes)+1)
	}

	data := make([]byte, info.Size()+1)

	read, readErr := io.ReadFull(file, data)
	if errors.Is(readErr, io.ErrUnexpectedEOF) || errors.Is(readErr, io.EOF) {
		// The file ends at or before the stat size; the bytes already read
		// are its complete contents.
		return data[:read], nil
	}

	if readErr != nil {
		return nil, fmt.Errorf("read font %s: %w", path, readErr)
	}

	// ReadFull succeeded only when the file grew past the stat size; finish
	// it under the same cap with the limited fallback.
	tail, err := readFontFileLimited(file, int64(maxFontBytes)+1-int64(read))
	if err != nil {
		return nil, err
	}

	combined := make([]byte, len(data)+len(tail))
	copy(combined, data)
	copy(combined[len(data):], tail)

	return combined, nil
}

// readFontFileLimited mirrors the historical io.ReadAll(io.LimitReader) read
// used when the stat size is unknown or no longer trustworthy. A limit of
// zero or less reads nothing.
func readFontFileLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return nil, fmt.Errorf("read font data: %w", err)
	}

	return data, nil
}
