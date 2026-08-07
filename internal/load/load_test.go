package load

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gowkhtmltopdf/internal/settings"
)

func defaultLP() settings.LoadPage {
	return settings.DefaultLoadPage()
}

func TestGuessURL(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "page.html")
	if err := os.WriteFile(f, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		in   string
		kind Kind
		want string
	}{
		{"http://example.com/a.html", KindHTTP, "http://example.com/a.html"},
		{"https://example.com", KindHTTP, "https://example.com"},
		{"example.com:8080/a", KindHTTP, "http://example.com:8080/a"},
		{"<html>x</html>", KindInline, "inline:"},
		{"data:text/plain,hi", KindInline, "data:"},
		{f, KindFile, "file://"},
		{"not-an-existing-host", KindHTTP, "http://not-an-existing-host"},
	}
	for _, c := range cases {
		kind, target, err := GuessURL(c.in)
		if err != nil {
			t.Errorf("GuessURL(%q): %v", c.in, err)
			continue
		}
		if kind != c.kind || !strings.HasPrefix(target, c.want) {
			t.Errorf("GuessURL(%q) = %v, %q; want %v, prefix %q", c.in, kind, target, c.kind, c.want)
		}
	}
}

func TestIsHTML(t *testing.T) {
	if !IsHTML("<html><body></body></html>") {
		t.Error("inline html not detected")
	}
	if IsHTML("page.html") {
		t.Error("path misdetected as html")
	}
}

func TestLoadHTTPBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>ok</body></html>"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	res, err := l.Load(context.Background(), srv.URL+"/page", defaultLP())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d", res.StatusCode)
	}
}

func TestLoadHTTPCustomHeadersAndAuth(t *testing.T) {
	var mu sync.Mutex
	got := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got["x-token"] = r.Header.Get("X-Token")
		u, p, ok := r.BasicAuth()
		if ok {
			got["user"] = u
			got["pass"] = p
		}
		got["ua"] = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	lp := defaultLP()
	lp.CustomHeaders = map[string]string{"X-Token": "secret"}
	lp.Username = "bob"
	lp.Password = "hunter2"
	l := NewLoader(settings.LoadGlobal{})
	if _, err := l.Load(context.Background(), srv.URL, lp); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got["x-token"] != "secret" || got["user"] != "bob" || got["pass"] != "hunter2" {
		t.Errorf("headers/auth = %v", got)
	}
	if !strings.HasPrefix(got["ua"], "gowkhtmltopdf") {
		t.Errorf("ua = %q", got["ua"])
	}
}

func TestLoadHTTPPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q", ct)
		}
		r.ParseForm()
		if r.Form.Get("q") != "hello world" || r.Form.Get("x") != "1" {
			t.Errorf("form = %v", r.Form)
		}
		w.Write([]byte("posted"))
	}))
	defer srv.Close()

	lp := defaultLP()
	lp.Post = []settings.PostItem{{Name: "q", Value: "hello world"}, {Name: "x", Value: "1"}}
	l := NewLoader(settings.LoadGlobal{})
	res, err := l.Load(context.Background(), srv.URL, lp)
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Body) != "posted" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestLoadHTTPErrorCodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/404":
			http.NotFound(w, r)
		case "/401":
			w.WriteHeader(http.StatusUnauthorized)
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})

	_, err := l.Load(context.Background(), srv.URL+"/404", defaultLP())
	if err == nil {
		t.Fatal("404 must error with abort policy")
	}
	if he, ok := err.(*settings.HttpStatusError); !ok || he.HttpErrorCode() != 2 {
		t.Errorf("404 error = %v", err)
	}

	_, err = l.Load(context.Background(), srv.URL+"/401", defaultLP())
	if he, ok := err.(*settings.HttpStatusError); !ok || he.HttpErrorCode() != 3 {
		t.Errorf("401 error = %v", err)
	}

	// skip policy: no error, Skip=true
	lp := defaultLP()
	lp.LoadErrorHandling = settings.LoadErrorSkip
	res, err := l.Load(context.Background(), srv.URL+"/500", lp)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skip {
		t.Error("skip policy must set Skip")
	}

	// ignore policy: no error, empty body
	lp.LoadErrorHandling = settings.LoadErrorIgnore
	res, err = l.Load(context.Background(), srv.URL+"/500", lp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skip {
		t.Error("ignore policy must not set Skip")
	}
}

func TestLoadCookies(t *testing.T) {
	var mu sync.Mutex
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotCookie = r.Header.Get("Cookie")
		mu.Unlock()
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	lp := defaultLP()
	lp.Cookies = map[string]string{"session": "abc123"}
	l := NewLoader(settings.LoadGlobal{})
	if _, err := l.Load(context.Background(), srv.URL, lp); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotCookie, "session=abc123") {
		t.Errorf("cookie = %q", gotCookie)
	}
}

func TestACLDefaultDeny(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.html")
	os.WriteFile(secret, []byte("secret"), 0o644)

	lp := defaultLP() // BlockLocalFileAccess = true, no allow prefixes
	l := NewLoader(settings.LoadGlobal{})
	_, err := l.Load(context.Background(), secret, lp)
	if err == nil {
		t.Fatal("default policy must deny local file access")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("err = %v", err)
	}
}

func TestACLAllowPrefix(t *testing.T) {
	dir := t.TempDir()
	allowed := filepath.Join(dir, "public", "a.html")
	os.MkdirAll(filepath.Dir(allowed), 0o755)
	os.WriteFile(allowed, []byte("<html>ok</html>"), 0o644)

	lp := defaultLP()
	l := NewLoader(settings.LoadGlobal{})
	l.Allow = []string{filepath.Join(dir, "public")}

	res, err := l.Load(context.Background(), allowed, lp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}

	// sibling outside allow prefix stays denied
	outside := filepath.Join(dir, "other.html")
	os.WriteFile(outside, []byte("x"), 0o644)
	if _, err := l.Load(context.Background(), outside, lp); err == nil {
		t.Error("outside allow prefix must stay denied")
	}
}

func TestACLEnableLocalFileAccess(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "page.html")
	os.WriteFile(f, []byte("<html>ok</html>"), 0o644)

	lp := defaultLP()
	lp.BlockLocalFileAccess = false
	l := NewLoader(settings.LoadGlobal{})
	l.EnableLocalFileAccess = true
	if _, err := l.Load(context.Background(), f, lp); err != nil {
		t.Errorf("enabled local access must load: %v", err)
	}

	// global on but object still blocks → denied
	lp2 := defaultLP()
	l2 := NewLoader(settings.LoadGlobal{})
	l2.EnableLocalFileAccess = true
	if _, err := l2.Load(context.Background(), f, lp2); err == nil {
		t.Error("object block must still apply")
	}
}

func TestSubresourceFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/style.css":
			w.Header().Set("Content-Type", "text/css")
			w.Write([]byte("body{}"))
		case "/img/logo.png":
			w.Write([]byte("PNG"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	res, err := l.FetchSub(context.Background(), srv.URL+"/page.html", "/style.css", defaultLP())
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Body) != "body{}" {
		t.Errorf("css = %q", res.Body)
	}
	res, err = l.FetchSub(context.Background(), srv.URL+"/page.html", "img/logo.png", defaultLP())
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Body) != "PNG" {
		t.Errorf("img = %q", res.Body)
	}
}

func TestConcurrentLoads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := l.Load(context.Background(), srv.URL, defaultLP())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent load: %v", err)
		}
	}
}

func TestRedirectLimit(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 15 {
			http.Redirect(w, r, "/next", http.StatusFound)
			return
		}
		w.Write([]byte("done"))
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	_, err := l.Load(context.Background(), srv.URL, defaultLP())
	if err == nil {
		t.Fatal("redirect loop must error")
	}
}

// --- security: file:// scheme, path traversal, symlink escape ---

func TestACLFileURL(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "page.html")
	if err := os.WriteFile(f, []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// default policy denies file:// loads
	if _, err := NewLoader(settings.LoadGlobal{}).Load(context.Background(), "file://"+f, defaultLP()); err == nil {
		t.Error("default policy must deny file:// loads")
	}

	lp := defaultLP()
	lp.BlockLocalFileAccess = false
	l := NewLoader(settings.LoadGlobal{})
	l.EnableLocalFileAccess = true

	res, err := l.Load(context.Background(), "file://"+f, lp)
	if err != nil {
		t.Fatalf("file:// load: %v", err)
	}
	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}

	// file://localhost/... is the same machine
	if _, err := l.Load(context.Background(), "file://localhost"+f, lp); err != nil {
		t.Errorf("file://localhost load: %v", err)
	}

	// a remote file host is refused outright
	if _, err := l.Load(context.Background(), "file://evil.example.com"+f, lp); err == nil {
		t.Error("remote file host must be refused")
	}
}

func TestACLPathTraversal(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(public, "a.html")
	if err := os.WriteFile(in, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(settings.LoadGlobal{})
	l.Allow = []string{public}

	// a file inside the prefix stays readable
	if _, err := l.Load(context.Background(), in, defaultLP()); err != nil {
		t.Fatalf("inside prefix: %v", err)
	}

	// ../ escape as a plain path
	esc := public + "/../secret.html"
	if _, err := l.Load(context.Background(), esc, defaultLP()); err == nil {
		t.Error("path traversal escape must be denied")
	}

	// ../ escape via a file:// URL, raw and percent-encoded
	for _, u := range []string{
		"file://" + public + "/../secret.html",
		"file://" + public + "/%2e%2e/secret.html",
	} {
		if _, err := l.Load(context.Background(), u, defaultLP()); err == nil {
			t.Errorf("traversal via %q must be denied", u)
		}
	}
}

func TestACLSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(public, "real.html")
	if err := os.WriteFile(real, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	escapeLink := filepath.Join(public, "escape.html")
	if err := os.Symlink(secret, escapeLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	inLink := filepath.Join(public, "in.html")
	if err := os.Symlink(real, inLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	l := NewLoader(settings.LoadGlobal{})
	l.Allow = []string{public}

	// a symlink inside the prefix pointing outside it must be denied
	if _, err := l.Load(context.Background(), escapeLink, defaultLP()); err == nil {
		t.Error("symlink escape must be denied")
	}
	// a symlink pointing inside the prefix stays allowed
	if _, err := l.Load(context.Background(), inLink, defaultLP()); err != nil {
		t.Errorf("symlink inside prefix: %v", err)
	}
}

func TestSubresourceFileACL(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte("<html>x</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(dir, "x.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := "file://" + dir + "/page.html"

	l := NewLoader(settings.LoadGlobal{})
	if _, err := l.FetchSub(context.Background(), base, "x.png", defaultLP()); err == nil {
		t.Error("file subresource must be denied by default")
	}

	lp := defaultLP()
	lp.BlockLocalFileAccess = false
	l2 := NewLoader(settings.LoadGlobal{})
	l2.EnableLocalFileAccess = true
	res, err := l2.FetchSub(context.Background(), base, "x.png", lp)
	if err != nil {
		t.Fatalf("enabled: %v", err)
	}
	if string(res.Body) != "PNG" {
		t.Errorf("body = %q", res.Body)
	}
}

// --- security: body caps, timeouts, redirects ---

// lyingContentLength opens a raw TCP server that advertises a Content-Length
// far above the cap but sends almost nothing: the loader must reject it on
// the header alone, without reading the body.
func lyingContentLength(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		br := bufio.NewReader(c)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" || line == "\n" {
				break
			}
		}
		c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 8192\r\nConnection: close\r\n\r\n"))
		c.Write(make([]byte, 128))
	}()
	return "http://" + ln.Addr().String()
}

func TestMaxBodySizeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/big": // chunked, no Content-Length
			w.Write(make([]byte, 4096))
		case "/exact":
			w.Write(make([]byte, 1024))
		case "/small":
			w.Write(make([]byte, 64))
		}
	}))
	defer srv.Close()

	liar := lyingContentLength(t)

	l := NewLoader(settings.LoadGlobal{})
	l.MaxBodySize = 1024

	for _, u := range []string{srv.URL + "/big", liar} {
		_, err := l.Load(context.Background(), u, defaultLP())
		if err == nil {
			t.Errorf("%s: oversized body must be rejected", u)
			continue
		}
		if !strings.Contains(err.Error(), "max body size") {
			t.Errorf("%s: err = %v", u, err)
		}
	}
	for _, p := range []string{"/exact", "/small"} {
		res, err := l.Load(context.Background(), srv.URL+p, defaultLP())
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		if len(res.Body) > 1024 {
			t.Errorf("%s: body length %d", p, len(res.Body))
		}
	}
}

func TestMaxBodySizeFile(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "big.html")
	if err := os.WriteFile(big, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	ok := filepath.Join(dir, "ok.html")
	if err := os.WriteFile(ok, make([]byte, 64), 0o644); err != nil {
		t.Fatal(err)
	}

	lp := defaultLP()
	lp.BlockLocalFileAccess = false
	l := NewLoader(settings.LoadGlobal{})
	l.EnableLocalFileAccess = true
	l.MaxBodySize = 1024

	_, err := l.Load(context.Background(), big, lp)
	if err == nil {
		t.Error("oversized local file must be rejected")
	} else if !strings.Contains(err.Error(), "max body size") {
		t.Errorf("err = %v", err)
	}
	if _, err := l.Load(context.Background(), ok, lp); err != nil {
		t.Errorf("small file: %v", err)
	}
}

func TestSlowServerTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.Write([]byte("too late"))
	}))
	defer srv.Close()

	lp := defaultLP()
	lp.Timeout = 1
	l := NewLoader(settings.LoadGlobal{})
	start := time.Now()
	_, err := l.Load(context.Background(), srv.URL, lp)
	if err == nil {
		t.Fatal("slow server must time out")
	}
	if d := time.Since(start); d > 2500*time.Millisecond {
		t.Errorf("timeout took %v", d)
	}
}

func TestContextCancelAbortsBodyRead(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(started)
		<-release // hang until the test lets go
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	l := NewLoader(settings.LoadGlobal{})
	errCh := make(chan error, 1)
	go func() {
		_, err := l.Load(ctx, srv.URL, defaultLP())
		errCh <- err
	}()
	<-started
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("cancelled context must abort the load")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("load did not abort after context cancellation")
	}
	close(release)
}

func TestRedirectLimitExact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/r/"))
		if err != nil || n <= 0 {
			w.Write([]byte("done"))
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/r/%d", n-1), http.StatusFound)
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	l.MaxRedirects = 2

	res, err := l.Load(context.Background(), srv.URL+"/r/2", defaultLP())
	if err != nil {
		t.Fatalf("exactly MaxRedirects redirects must succeed: %v", err)
	}
	if string(res.Body) != "done" {
		t.Errorf("body = %q", res.Body)
	}

	if _, err := l.Load(context.Background(), srv.URL+"/r/3", defaultLP()); err == nil {
		t.Error("one more than MaxRedirects must fail")
	}
}

// TestHTTPLocalhostAllowedByDesign documents the intended SSRF posture:
// the loader fetches any URL the document references - including
// http://localhost - exactly like upstream wkhtmltopdf. Only file:// reads
// are gated by the ACL.
func TestHTTPLocalhostAllowedByDesign(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("localhost ok"))
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	res, err := l.Load(context.Background(), srv.URL, defaultLP())
	if err != nil {
		t.Fatalf("http://127.0.0.1 must be fetchable: %v", err)
	}
	if string(res.Body) != "localhost ok" {
		t.Errorf("body = %q", res.Body)
	}
}

// TestLoadInlineHTML: an explicit in-memory HTML source is returned as-is
// and skips GuessURL entirely; subresources resolve against InlineBase.
func TestLoadInlineHTML(t *testing.T) {
	l := NewLoader(settings.LoadGlobal{})
	lp := defaultLP()
	lp.InlineHTML = []byte("<html><body>inline</body></html>")
	lp.InlineBase = "https://example.com/docs/page.html"

	// The input would be treated as an http:// URL by GuessURL; InlineHTML
	// must short-circuit it without any guessing or fetching.
	res, err := l.Load(context.Background(), "this is not a url", lp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != KindInline {
		t.Errorf("kind = %v, want KindInline", res.Kind)
	}
	if string(res.Body) != "<html><body>inline</body></html>" {
		t.Errorf("body = %q", res.Body)
	}
	if res.Base != "https://example.com/docs/page.html" {
		t.Errorf("base = %q, want InlineBase", res.Base)
	}

	lp2 := defaultLP()
	lp2.InlineHTML = []byte("<html></html>")
	res2, err := l.Load(context.Background(), "ignored", lp2)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Base != "" {
		t.Errorf("empty InlineBase must leave Base empty, got %q", res2.Base)
	}
}

func TestDataURLHonorsBodyLimitForPrimaryAndSubresource(t *testing.T) {
	l := NewLoader(settings.LoadGlobal{})
	l.MaxBodySize = 4
	lp := defaultLP()

	if _, err := l.Load(context.Background(), "data:text/plain,12345", lp); err == nil {
		t.Fatal("oversized primary data URL must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("primary error = %v", err)
	}
	if _, err := l.FetchSub(context.Background(), "", "data:text/plain,12345", lp); err == nil {
		t.Fatal("oversized data subresource must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("subresource error = %v", err)
	}
	if _, err := l.Load(context.Background(), "data:text/plain;base64,MTIzNDU=", lp); err == nil {
		t.Fatal("oversized base64 data URL must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("base64 error = %v", err)
	}

	res, err := l.FetchSub(context.Background(), "", "data:text/plain,1234", lp)
	if err != nil {
		t.Fatalf("data URL at the body limit: %v", err)
	}
	if string(res.Body) != "1234" {
		t.Errorf("body = %q, want 1234", res.Body)
	}
}

func TestInlineHTMLHonorsBodyLimit(t *testing.T) {
	l := NewLoader(settings.LoadGlobal{})
	l.MaxBodySize = 4
	lp := defaultLP()
	lp.InlineHTML = []byte("12345")
	if _, err := l.Load(context.Background(), "ignored", lp); err == nil {
		t.Fatal("oversized inline HTML must be rejected")
	} else if !strings.Contains(err.Error(), "inline HTML exceeds max body size 4") {
		t.Fatalf("error = %v", err)
	}
}

func TestEmptyInlineBaseRejectsRelativeSubresources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "local.css")
	if err := os.WriteFile(path, []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(settings.LoadGlobal{})
	// Even with local access enabled, an inline document without a base must
	// not reinterpret a relative reference as a process-working-directory
	// file. The reference is unresolved, not an implicit local path.
	l.EnableLocalFileAccess = true
	if _, err := l.FetchSub(context.Background(), "", path, defaultLP()); err == nil {
		t.Fatal("relative reference without a base must be rejected")
	} else if !strings.Contains(err.Error(), "without a document base URL") {
		t.Fatalf("error = %v", err)
	}

	res, err := l.FetchSub(context.Background(), "", "data:text/plain,ok", defaultLP())
	if err != nil {
		t.Fatalf("absolute data reference without a base: %v", err)
	}
	if string(res.Body) != "ok" {
		t.Errorf("body = %q, want ok", res.Body)
	}
}

func TestResourceContextBindsBaseAndPolicy(t *testing.T) {
	dir := t.TempDir()
	styleDir := filepath.Join(dir, "styles")
	if err := os.Mkdir(styleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stylePath := filepath.Join(styleDir, "site.css")
	if err := os.WriteFile(stylePath, []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLoader(settings.LoadGlobal{})
	l.EnableLocalFileAccess = true
	base := &Resource{Base: "file://" + filepath.ToSlash(filepath.Join(dir, "page.html"))}
	lp := defaultLP()
	lp.BlockLocalFileAccess = false
	ctx := l.ForResource(base, lp)
	res, err := ctx.Fetch(context.Background(), "styles/site.css")
	if err != nil {
		t.Fatalf("relative fetch = %v", err)
	}
	if string(res.Body) != "body{}" {
		t.Errorf("body = %q, want body{}", res.Body)
	}
}

func TestLoadCharsetContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.URL.Query().Get("ct"))
		w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	for _, tc := range []struct {
		ct   string
		ok   bool
		want string
	}{
		{"text/html", true, ""},
		{"text/html; charset=utf-8", true, ""},
		{"text/html; charset=UTF-8", true, ""},
		{"text/html; charset=us-ascii", true, ""},
		{"text/html; charset=ISO-8859-1", false, "unsupported charset: ISO-8859-1 (only UTF-8/ASCII)"},
		{"text/html; charset=windows-1252", false, "unsupported charset: windows-1252 (only UTF-8/ASCII)"},
	} {
		_, err := l.Load(context.Background(), srv.URL+"?ct="+url.QueryEscape(tc.ct), defaultLP())
		if tc.ok && err != nil {
			t.Errorf("ct %q: %v", tc.ct, err)
		}
		if !tc.ok {
			if err == nil {
				t.Errorf("ct %q: expected error", tc.ct)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("ct %q: err = %v, want contains %q", tc.ct, err, tc.want)
			}
		}
	}
}

func TestLoadCharsetMetaDecl(t *testing.T) {
	// Content-Type without a charset parameter: the <meta> declaration is
	// the only charset signal, and it must be honored at the load seam.
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	l := NewLoader(settings.LoadGlobal{})
	for _, tc := range []struct {
		name, head string
		ok         bool
		want       string
	}{
		{"utf8-charset", `<meta charset="utf-8">`, true, ""},
		{"utf8-content-type", `<meta http-equiv="content-type" content="text/html; charset=UTF-8">`, true, ""},
		{"no-meta", `<html><body>x</body></html>`, true, ""},
		{"latin1-charset", `<meta charset="windows-1252">`, false, "unsupported charset: windows-1252 (only UTF-8/ASCII)"},
		{"latin1-content-type", `<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1">`, false, "unsupported charset: ISO-8859-1 (only UTF-8/ASCII)"},
	} {
		body = tc.head + "<title>t</title></head><body>x</body></html>"
		_, err := l.Load(context.Background(), srv.URL+"/"+tc.name, defaultLP())
		if tc.ok && err != nil {
			t.Errorf("%s: %v", tc.name, err)
		}
		if !tc.ok {
			if err == nil {
				t.Errorf("%s: expected error", tc.name)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("%s: err = %v, want contains %q", tc.name, err, tc.want)
			}
		}
	}
}
