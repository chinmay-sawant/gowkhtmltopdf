// Package load reimplements the MultiPageLoader orchestration layer:
// URL guessing, HTTP(S)/file fetching, cookies, proxy, auth, local ACL,
// POST bodies, and the jsdelay/windowStatus/runScript stubs. Not a browser:
// it hands raw bytes to the HTML/CSS/layout pipeline.
package load

import (
	"context"
	"crypto/tls"
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

// AccessController implements the local-file ACL: default deny; an explicit
// allow-prefix list (--allow) expands access.
type AccessController struct {
	AllowPrefixes []string
}

// NewAccessController builds the controller from global + object load
// settings: access is allowed iff the global flag is on AND the object's
// blockLocalFileAccess is off, or an allow prefix matches.
func NewAccessController(g settings.PdfGlobal, lp settings.LoadPage) *AccessController {
	return &AccessController{AllowPrefixes: append([]string{}, g.Allow...)}
}

// Allowed reports whether path may be read.
func (a *AccessController) Allowed(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	for _, p := range a.AllowPrefixes {
		if p == "" {
			continue
		}
		ap, err := filepath.Abs(p)
		if err != nil {
			ap = p
		}
		ap = filepath.Clean(ap)
		if abs == ap || strings.HasPrefix(abs, ap+string(filepath.Separator)) {
			return true
		}
	}
	return false
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

// Loader fetches resources concurrently with aggregate progress.
type Loader struct {
	Client       *http.Client
	Global       settings.LoadGlobal
	Log          io.Writer
	OnProgress   func(percent int)
	MaxBodySize  int64
	MaxRedirects int
	InsecureTLS  bool

	// Allow prefixes and the effective local-access flag, injected by the
	// caller from settings (convert wires PdfGlobal.Allow + EnableLocalFileAccess).
	Allow                 []string
	EnableLocalFileAccess bool

	active int
}

// NewLoader builds a Loader from global load settings.
func NewLoader(g settings.LoadGlobal) *Loader {
	l := &Loader{
		Global:       g,
		Log:          io.Discard,
		MaxBodySize:  DefaultMaxBodySize,
		MaxRedirects: DefaultMaxRedirects,
	}
	l.initClient()
	return l
}

func (l *Loader) initClient() {
	tlsCfg := &tls.Config{}
	if l.InsecureTLS {
		tlsCfg.InsecureSkipVerify = true // #nosec G402 -- explicit --insecure opt-in
	}
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		TLSClientConfig:   tlsCfg,
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
			if len(via) >= l.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", l.MaxRedirects)
			}
			return nil
		},
	}
}

// ApplyCert loads a client certificate (PEM key+crt) onto the transport.
func (l *Loader) ApplyCert(certPEM, keyPEM []byte) error {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("client cert: %w", err)
	}
	if tr, ok := l.Client.Transport.(*http.Transport); ok {
		tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
		return nil
	}
	return errors.New("client cert: unexpected transport")
}

// Load fetches the primary resource for an object.
func (l *Loader) Load(ctx context.Context, input string, lp settings.LoadPage) (*Resource, error) {
	kind, target, err := GuessURL(input)
	if err != nil {
		return nil, err
	}
	switch kind {
	case KindInline:
		if strings.HasPrefix(target, "inline:") {
			return &Resource{Kind: KindInline, URL: "inline:", Body: []byte(target[len("inline:"):]), ContentType: "text/html"}, nil
		}
		if strings.HasPrefix(target, "data:") {
			body, ctype, err := decodeDataURL(target)
			if err != nil {
				return nil, err
			}
			return &Resource{Kind: KindInline, URL: target, Body: body, ContentType: ctype}, nil
		}
	case KindFile:
		return l.loadFile(ctx, target, lp)
	case KindHTTP:
		return l.loadHTTP(ctx, target, lp)
	}
	return nil, fmt.Errorf("cannot load %q", input)
}

func (l *Loader) loadFile(ctx context.Context, path string, lp settings.LoadPage) (*Resource, error) {
	p := strings.TrimPrefix(path, "file://")
	p, err := url.PathUnescape(p)
	if err != nil {
		p = strings.TrimPrefix(path, "file://")
	}
	if !l.fileAccessAllowed(p, lp) {
		return nil, fmt.Errorf("%w: %s", ErrAccessDenied, p)
	}
	// blockLocalFileAccess=true blocks local reads even for the primary page
	// unless the user explicitly enabled local access.
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return &Resource{
		Kind:        KindFile,
		URL:         "file://" + filepath.ToSlash(filepath.Clean(p)),
		Base:        "file://" + filepath.ToSlash(filepath.Dir(p)),
		Body:        b,
		ContentType: mime.TypeByExtension(filepath.Ext(p)),
		StatusCode:  200,
	}, nil
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

	limited := io.LimitReader(resp.Body, l.MaxBodySize)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
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
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	refURL, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	abs := baseURL.ResolveReference(refURL)
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
		body, ctype, err := decodeDataURL(abs.String())
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
				dec, err := decodeBase64(data)
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
	dec, err := url.QueryUnescape(data)
	if err != nil {
		return nil, "", err
	}
	return []byte(dec), ctype, nil
}

func decodeBase64(s string) ([]byte, error) {
	r := strings.NewReplacer("\n", "", "\r", "", " ", "")
	return base64.StdEncoding.DecodeString(r.Replace(s))
}

// WaitJSDelay sleeps ms (or ctx cancel), the MVP stand-in for JS execution.
func WaitJSDelay(ctx context.Context, ms int) {
	if ms <= 0 {
		return
	}
	t := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// WarnJSStubs logs once that windowStatus/runScript are accepted but ignored
// (no JS engine in MVP).
func WarnJSStubs(log io.Writer, lp settings.LoadPage) {
	if lp.WindowStatus != "" {
		fmt.Fprintf(log, "warning: --window-status ignored (no JavaScript engine in MVP)\n")
	}
	if lp.RunScript != "" {
		fmt.Fprintf(log, "warning: --run-script ignored (no JavaScript engine in MVP)\n")
	}
	if lp.DebugJavaScript {
		fmt.Fprintf(log, "warning: --debug-javascript ignored (no JavaScript engine in MVP)\n")
	}
}
