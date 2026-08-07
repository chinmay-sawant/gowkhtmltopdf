// Package load reimplements the MultiPageLoader orchestration layer:
// URL guessing, HTTP(S)/file fetching, cookies, proxy, auth, local ACL,
// and POST bodies. Not a browser: it hands raw bytes to the HTML/CSS/layout
// pipeline. JS-related settings flags are accepted by the settings/CLI layer
// but not consumed here (no JS engine).
package load

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gowkhtmltopdf/internal/settings"
)

// Defaults for network behaviour (timeout, size caps).
const (
	DefaultConnectTimeout  = 30 * time.Second
	DefaultResponseTimeout = 60 * time.Second
	DefaultMaxBodySize     = 100 << 20 // 100 MiB
	DefaultMaxRedirects    = 10
)

// ErrAccessDenied is returned when the local-file ACL blocks a path.
var ErrAccessDenied = errors.New("local file access denied")

// Kind classifies a resolved input.
type Kind int

const (
	KindHTTP Kind = iota
	KindFile
	KindInline
)

// Resource is one fetched document.
type Resource struct {
	Kind        Kind
	URL         string // final URL (after redirects)
	Base        string // base URL for relative resolution
	Body        []byte
	ContentType string
	StatusCode  int
	Skip        bool // set when load-error policy = skip
}

// ResourceContext binds a loaded document's base URL and load policy. It is
// the narrow seam consumers should use when fetching CSS, images, fonts, or
// other document-relative resources. The loader remains the owner of URL
// resolution, ACL checks, body limits, and error handling.
type ResourceContext struct {
	loader *Loader
	base   string
	lp     settings.LoadPage
}

// ForResource returns a context for resources relative to res. A nil
// resource is allowed for callers that only need absolute references; those
// references still go through the same loader policy.
func (l *Loader) ForResource(res *Resource, lp settings.LoadPage) ResourceContext {
	var base string
	if res != nil {
		base = res.Base
	}
	return ResourceContext{loader: l, base: base, lp: lp}
}

// Fetch resolves ref against the document base and fetches it using the
// document's load policy.
func (c ResourceContext) Fetch(ctx context.Context, ref string) (*Resource, error) {
	if c.loader == nil {
		return nil, errors.New("nil resource loader")
	}
	return c.loader.FetchSub(ctx, c.base, ref, c.lp)
}

// AccessController implements the local-file ACL: default deny; an explicit
// allow-prefix list (--allow) expands access.
type AccessController struct {
	AllowPrefixes []string
}

// Allowed reports whether path may be read. The candidate path and every
// allow prefix are both resolved to their real, symlink-free locations
// before the prefix comparison, so a symlink planted inside an allowed
// directory cannot escape to a file outside it; `..` components and
// percent-encoded forms resolve to the same real path the subsequent read
// would follow. Paths that do not exist fall back to their cleaned
// absolute form (reading them fails anyway).
func (a *AccessController) Allowed(path string) bool {
	abs := resolvePath(path)
	if abs == "" {
		return false
	}
	for _, p := range a.AllowPrefixes {
		if p == "" {
			continue
		}
		ap := resolvePath(p)
		if abs == ap || strings.HasPrefix(abs, ap+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// resolvePath returns the clean absolute form of path, following symlinks
// when the path exists, and "" when it cannot be made absolute.
func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(abs)
}

// IsHTML reports whether s looks like inline HTML rather than a URL/path.
func IsHTML(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "<") || strings.HasPrefix(s, "\ufeff<")
}

// GuessURL mirrors wkhtmltopdf's guessUrlFromString: existing local path →
// file URL; http(s):// passthrough; host:port → http://host:port; else
// default to http://<input>.
func GuessURL(input string) (Kind, string, error) {
	if IsHTML(input) {
		return KindInline, "inline:" + input, nil
	}
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "inline:") {
		return KindInline, input, nil
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return KindHTTP, input, nil
	}
	if strings.HasPrefix(lower, "file://") {
		return KindFile, input, nil
	}
	if strings.HasPrefix(lower, "data:") {
		return KindInline, input, nil
	}
	// host:port or host without scheme
	if isHostPort(input) || !strings.Contains(input, "/") && !strings.Contains(input, "\\") && strings.Contains(input, ":") {
		if strings.Contains(input, ":") && !isLocalPath(input) {
			return KindHTTP, "http://" + input, nil
		}
	}
	// local file?
	if _, err := os.Stat(input); err == nil {
		abs, err := filepath.Abs(input)
		if err == nil {
			return KindFile, "file://" + filepath.ToSlash(abs), nil
		}
		return KindFile, "file://" + filepath.ToSlash(input), nil
	}
	// default: treat as http URL
	return KindHTTP, "http://" + input, nil
}

func isHostPort(s string) bool {
	host, port, err := net.SplitHostPort(s)
	if err != nil || port == "" {
		return false
	}
	_ = host
	return true
}

func isLocalPath(s string) bool {
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") ||
		len(s) == 2 && s[1] == ':' // windows drive
}

// Loader fetches resources with the configured network and local-file policy.
type Loader struct {
	Client       *http.Client
	Global       settings.LoadGlobal
	Log          io.Writer
	MaxBodySize  int64
	MaxRedirects int

	// Allow prefixes and the effective local-access flag, applied by
	// NewLoader from settings.LoadGlobal (caller-side pokes are being
	// removed in parallel). Kept exported for tests.
	Allow                 []string
	EnableLocalFileAccess bool
}

// NewLoader builds a Loader from global load settings, applying the full
// load policy (proxy, allow prefixes, local-access flag) in one place.
func NewLoader(g settings.LoadGlobal) *Loader {
	l := &Loader{
		Global:                g,
		Log:                   io.Discard,
		MaxBodySize:           DefaultMaxBodySize,
		MaxRedirects:          DefaultMaxRedirects,
		Allow:                 g.Allow,
		EnableLocalFileAccess: g.EnableLocalFileAccess,
	}
	l.initClient()
	return l
}

func (l *Loader) initClient() {
	jar, _ := cookiejar.New(nil)
	// Default TLS verification stays on (http.Transport system roots).
	transport := &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{
			Timeout:   DefaultConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	if l.Global.Proxy != "" {
		if pu, err := url.Parse(l.Global.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(pu)
		}
	}
	l.Client = &http.Client{
		Transport: transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Go's client counts the requests made so far in via, so the
			// MaxRedirects-th redirect (len(via) == MaxRedirects) is the
			// last one allowed to complete.
			if len(via) > l.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", l.MaxRedirects)
			}
			return nil
		},
	}
}

// Load fetches the primary resource for an object. In-memory HTML
// (lp.InlineHTML) is returned as-is and skips GuessURL entirely; subresources
// resolve against lp.InlineBase when set. Every loaded document is checked
// for a supported charset at this seam (see checkDocumentCharset).
func (l *Loader) Load(ctx context.Context, input string, lp settings.LoadPage) (*Resource, error) {
	if len(lp.InlineHTML) > 0 {
		if err := checkBodyLimit("inline HTML", len(lp.InlineHTML), l.MaxBodySize); err != nil {
			return nil, err
		}
		res := &Resource{
			Kind:        KindInline,
			URL:         "inline:",
			Base:        lp.InlineBase,
			Body:        lp.InlineHTML,
			ContentType: "text/html",
		}
		return res, checkDocumentCharset(res)
	}
	kind, target, err := GuessURL(input)
	if err != nil {
		return nil, err
	}
	var res *Resource
	switch kind {
	case KindInline:
		if strings.HasPrefix(target, "inline:") {
			res = &Resource{Kind: KindInline, URL: "inline:", Body: []byte(target[len("inline:"):]), ContentType: "text/html"}
		} else if strings.HasPrefix(target, "data:") {
			var body []byte
			var ctype string
			body, ctype, err = decodeDataURLLimited(target, l.MaxBodySize)
			if err != nil {
				return nil, err
			}
			res = &Resource{Kind: KindInline, URL: target, Body: body, ContentType: ctype}
		}
	case KindFile:
		res, err = l.loadFile(ctx, target, lp)
	case KindHTTP:
		res, err = l.loadHTTP(ctx, target, lp)
	}
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("cannot load %q", input)
	}
	if !res.Skip {
		return res, checkDocumentCharset(res)
	}
	return res, nil
}

// checkDocumentCharset enforces the bytes-to-runes seam: only UTF-8/ASCII
// documents are supported. The charset is taken from the resource's
// Content-Type charset parameter, falling back to a <meta charset> (or
// <meta http-equiv="content-type">) declaration in the first 1 KiB of the
// body. Anything else is refused with a clear error instead of silently
// garbling the page.
func checkDocumentCharset(res *Resource) error {
	cs := charsetFromContentType(res.ContentType)
	if cs == "" {
		cs = metaCharset(res.Body)
	}
	if cs == "" || charsetSupported(cs) {
		return nil
	}
	return fmt.Errorf("unsupported charset: %s (only UTF-8/ASCII)", cs)
}

// charsetSupported reports whether a charset name is one the pipeline can
// decode. Only UTF-8 and ASCII are accepted.
func charsetSupported(cs string) bool {
	switch strings.ToLower(strings.TrimSpace(cs)) {
	case "utf-8", "utf8", "us-ascii", "ascii":
		return true
	}
	return false
}

// charsetFromContentType extracts the charset parameter of a Content-Type
// value, or "" when absent or unparseable.
func charsetFromContentType(ct string) string {
	if ct == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return params["charset"]
}

// metaCharset scans the first 1 KiB of a document for a <meta charset> or
// <meta http-equiv="content-type"> declaration and returns the declared
// charset, or "" when there is none.
func metaCharset(body []byte) string {
	s := string(body)
	if len(s) > 1024 {
		s = s[:1024]
	}
	low := strings.ToLower(s)
	for {
		i := strings.Index(low, "<meta")
		if i < 0 {
			return ""
		}
		j := strings.IndexByte(s[i:], '>')
		if j < 0 {
			return ""
		}
		if cs := metaTagCharset(s[i : i+j]); cs != "" {
			return cs
		}
		s = s[i+j+1:]
		low = strings.ToLower(s)
	}
}

// metaTagCharset extracts the charset from one <meta ...> tag: the charset
// attribute directly, or the content attribute when http-equiv is
// content-type.
func metaTagCharset(tag string) string {
	if strings.EqualFold(attrValue(tag, "http-equiv"), "content-type") {
		if cs := charsetInContent(attrValue(tag, "content")); cs != "" {
			return cs
		}
	}
	if cs := attrValue(tag, "charset"); cs != "" {
		return cs
	}
	return ""
}

// charsetInContent extracts the charset=... parameter from a meta content
// attribute value (e.g. "text/html; charset=windows-1252").
func charsetInContent(content string) string {
	low := strings.ToLower(content)
	i := strings.Index(low, "charset")
	if i < 0 {
		return ""
	}
	rest := strings.TrimLeft(content[i+len("charset"):], " \t\n\r\f")
	if !strings.HasPrefix(rest, "=") {
		return ""
	}
	return strings.Trim(strings.TrimLeft(rest[1:], " \t\n\r\f"), "\"' ;")
}

// attrValue extracts the value of the named attribute (case-insensitive)
// from a <meta ...> tag body, or "".
func attrValue(tag, name string) string {
	low := strings.ToLower(tag)
	needle := strings.ToLower(name)
	for {
		i := strings.Index(low, needle)
		if i < 0 {
			return ""
		}
		if i > 0 {
			c := tag[i-1]
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f' {
				tag = tag[i+len(needle):]
				low = low[i+len(needle):]
				continue
			}
		}
		rest := strings.TrimLeft(tag[i+len(needle):], " \t\n\r\f")
		if !strings.HasPrefix(rest, "=") {
			tag = tag[i+len(needle):]
			low = low[i+len(needle):]
			continue
		}
		rest = strings.TrimLeft(rest[1:], " \t\n\r\f")
		if rest == "" {
			return ""
		}
		if rest[0] == '"' || rest[0] == '\'' {
			q := rest[0]
			if k := strings.IndexByte(rest[1:], q); k >= 0 {
				return rest[1 : k+1]
			}
			return ""
		}
		k := 0
		for k < len(rest) && !isMetaWhitespace(rest[k]) {
			k++
		}
		return rest[:k]
	}
}

func isMetaWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '>'
}

func (l *Loader) loadFile(ctx context.Context, path string, lp settings.LoadPage) (*Resource, error) {
	p, err := filePathFromURL(path)
	if err != nil {
		return nil, err
	}
	if !l.fileAccessAllowed(p, lp) {
		return nil, fmt.Errorf("%w: %s", ErrAccessDenied, p)
	}
	// blockLocalFileAccess=true blocks local reads even for the primary page
	// unless the user explicitly enabled local access.
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, l.MaxBodySize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > l.MaxBodySize {
		return nil, fmt.Errorf("file %s exceeds max body size %d", p, l.MaxBodySize)
	}
	return &Resource{
		Kind:        KindFile,
		URL:         "file://" + filepath.ToSlash(filepath.Clean(p)),
		Base:        "file://" + filepath.ToSlash(filepath.Dir(p)) + "/",
		Body:        b,
		ContentType: mime.TypeByExtension(filepath.Ext(p)),
		StatusCode:  200,
	}, nil
}

// filePathFromURL extracts the local filesystem path from a file:// URL
// (or passes a plain path through). Remote file hosts - anything other
// than the empty host and localhost - are refused.
func filePathFromURL(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil || u.Scheme != "file" {
		return path, nil
	}
	if u.Host != "" && u.Host != "localhost" {
		return "", fmt.Errorf("blocked file access to %q", path)
	}
	if u.Path != "" {
		return u.Path, nil
	}
	return strings.TrimPrefix(path, "file://"), nil
}

// fileAccessAllowed implements the frozen security policy: blocked by
// default; allowed when the global enable flag is on AND the object's block
// flag is off, or when an --allow prefix matches.
func (l *Loader) fileAccessAllowed(path string, lp settings.LoadPage) bool {
	ctrl := &AccessController{AllowPrefixes: l.Allow}
	if ctrl.Allowed(path) {
		return true
	}
	return l.EnableLocalFileAccess && !lp.BlockLocalFileAccess
}

func (l *Loader) loadHTTP(ctx context.Context, target string, lp settings.LoadPage) (*Resource, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	method := http.MethodGet
	var body io.Reader
	if len(lp.Post) > 0 {
		method = http.MethodPost
		body = strings.NewReader(urlEncodePost(lp.Post))
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gowkhtmltopdf/0.1 (pure-Go wkhtmltopdf reimplementation)")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if lp.Username != "" {
		req.SetBasicAuth(lp.Username, lp.Password)
	}
	for k, v := range lp.CustomHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range lp.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	timeout := time.Duration(lp.Timeout) * time.Second
	if timeout <= 0 {
		timeout = DefaultResponseTimeout
	}
	c := l.Client
	client := *c
	client.Timeout = timeout

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// skip / ignore policy
		switch lp.LoadErrorHandling {
		case settings.LoadErrorSkip:
			return &Resource{Kind: KindHTTP, URL: u.String(), StatusCode: resp.StatusCode, Skip: true}, nil
		case settings.LoadErrorIgnore:
			return &Resource{Kind: KindHTTP, URL: u.String(), StatusCode: resp.StatusCode, Body: nil, Skip: false}, nil
		default:
			return nil, &settings.HttpStatusError{Status: resp.StatusCode, URL: u.String()}
		}
	}

	// Enforce the body cap: reject oversized bodies outright rather than
	// silently truncating them. Content-Length short-circuits the download
	// when present; the read-side limit is the authoritative check for
	// chunked or unknown-length responses.
	if resp.ContentLength > l.MaxBodySize {
		return nil, fmt.Errorf("response from %s exceeds max body size %d (Content-Length %d)",
			u.String(), l.MaxBodySize, resp.ContentLength)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, l.MaxBodySize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > l.MaxBodySize {
		return nil, fmt.Errorf("response from %s exceeds max body size %d", u.String(), l.MaxBodySize)
	}
	ctype := resp.Header.Get("Content-Type")
	final := resp.Request.URL.String()
	return &Resource{
		Kind:        KindHTTP,
		URL:         final,
		Base:        final,
		Body:        b,
		ContentType: ctype,
		StatusCode:  resp.StatusCode,
	}, nil
}

// FetchSub fetches a subresource (css/img) resolved against base.
func (l *Loader) FetchSub(ctx context.Context, base, ref string, lp settings.LoadPage) (*Resource, error) {
	refURL, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	if !refURL.IsAbs() && base == "" {
		return nil, fmt.Errorf("cannot resolve relative subresource %q without a document base URL", ref)
	}
	var abs *url.URL
	if refURL.IsAbs() {
		abs = refURL
	} else {
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, err
		}
		abs = baseURL.ResolveReference(refURL)
	}
	switch abs.Scheme {
	case "file", "":
		p := abs.Path
		if abs.Host != "" && abs.Host != "localhost" {
			return nil, fmt.Errorf("blocked file access to %q", abs.String())
		}
		return l.loadFile(ctx, p, lp)
	case "http", "https":
		return l.loadHTTP(ctx, abs.String(), lp)
	case "data":
		body, ctype, err := decodeDataURLLimited(abs.String(), l.MaxBodySize)
		if err != nil {
			return nil, err
		}
		return &Resource{Kind: KindInline, URL: abs.String(), Body: body, ContentType: ctype}, nil
	}
	return nil, fmt.Errorf("unsupported subresource scheme %q", abs.Scheme)
}

func urlEncodePost(items []settings.PostItem) string {
	vals := url.Values{}
	for _, it := range items {
		vals.Add(it.Name, it.Value)
	}
	return vals.Encode()
}

func decodeDataURL(s string) ([]byte, string, error) {
	return decodeDataURLLimited(s, -1)
}

// decodeDataURLLimited decodes a data URL while enforcing the same body cap
// used by file and HTTP resources. A negative max means unlimited and exists
// only for the package-local compatibility helper above.
func decodeDataURLLimited(s string, max int64) ([]byte, string, error) {
	rest := strings.TrimPrefix(s, "data:")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return nil, "", fmt.Errorf("malformed data URL")
	}
	meta, data := rest[:comma], rest[comma+1:]
	ctype := "text/plain"
	if meta != "" {
		parts := strings.Split(meta, ";")
		for _, p := range parts {
			if strings.HasPrefix(p, "base64") {
				dec, err := decodeBase64Limited(data, max)
				if err != nil {
					return nil, "", err
				}
				return dec, ctype, nil
			}
			if !strings.Contains(p, "=") {
				ctype = p
			}
		}
	}
	dec, err := unescapeDataLimited(data, max)
	if err != nil {
		return nil, "", err
	}
	return dec, ctype, nil
}

func decodeBase64(s string) ([]byte, error) {
	return decodeBase64Limited(s, -1)
}

func decodeBase64Limited(s string, max int64) ([]byte, error) {
	// Count first so a large amount of whitespace cannot force an equally
	// large compacted allocation before the body limit is checked.
	compactLen := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' && s[i] != '\r' && s[i] != ' ' && s[i] != '\t' {
			compactLen++
		}
	}
	if max >= 0 {
		decodedLen := base64.StdEncoding.DecodedLen(compactLen)
		if compactLen > 0 && s != "" {
			// DecodedLen includes the bytes represented by padding. Account for
			// it so a one-byte data URL is not rejected with a one-byte limit.
			seen := 0
			for i := len(s) - 1; i >= 0 && seen < 2; i-- {
				switch s[i] {
				case ' ', '\t', '\n', '\r':
					continue
				case '=':
					decodedLen--
					seen++
				default:
					seen = 2
				}
			}
		}
		if int64(decodedLen) > max {
			return nil, fmt.Errorf("data URL exceeds max body size %d", max)
		}
	}
	var b strings.Builder
	b.Grow(compactLen)
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' && s[i] != '\r' && s[i] != ' ' && s[i] != '\t' {
			b.WriteByte(s[i])
		}
	}
	dec, err := base64.StdEncoding.DecodeString(b.String())
	if err != nil {
		return nil, err
	}
	if err := checkBodyLimit("data URL", len(dec), max); err != nil {
		return nil, err
	}
	return dec, nil
}

func unescapeDataLimited(s string, max int64) ([]byte, error) {
	if max >= 0 && int64(len(s)) < max {
		// This is only a capacity hint. Percent escapes and '+' can make the
		// decoded body shorter, never longer, than the source string.
		max = int64(len(s))
	}
	capacity := len(s)
	if max >= 0 && int64(capacity) > max {
		capacity = int(max)
	}
	out := make([]byte, 0, capacity)
	for i := 0; i < len(s); i++ {
		var b byte
		switch s[i] {
		case '%':
			if i+2 >= len(s) {
				return nil, fmt.Errorf("invalid URL escape in data URL")
			}
			hi, okHi := fromHex(s[i+1])
			lo, okLo := fromHex(s[i+2])
			if !okHi || !okLo {
				return nil, fmt.Errorf("invalid URL escape in data URL")
			}
			b = hi<<4 | lo
			i += 2
		case '+':
			b = ' '
		default:
			b = s[i]
		}
		if max >= 0 && int64(len(out)) >= max {
			return nil, fmt.Errorf("data URL exceeds max body size %d", max)
		}
		out = append(out, b)
	}
	return out, nil
}

func fromHex(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

func checkBodyLimit(source string, length int, max int64) error {
	if max >= 0 && int64(length) > max {
		return fmt.Errorf("%s exceeds max body size %d", source, max)
	}
	return nil
}
