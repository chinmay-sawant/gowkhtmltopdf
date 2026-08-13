package load_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/settings"
)

func defaultLP() settings.LoadPage {
	return settings.DefaultLoadPage()
}

const (
	mutatedShared = "mutated-shared"
	mutatedImage  = "mutated-image"
)

func TestResolveEffectiveLoadGlobalSharedPolicyWins(t *testing.T) {
	t.Parallel()

	global := settings.LoadGlobal{
		Proxy:                 "http://shared-proxy.example",
		Allow:                 []string{"/shared"},
		EnableLocalFileAccess: true,
		NetworkPolicySet:      true,
		NetworkAllowedSchemes: []string{"https"},
		NetworkAllowedHosts:   []string{"shared.example"},
		NetworkBlockPrivate:   true,
		NetworkBlockCrossHost: true,
	}
	mode := settings.LoadGlobal{ //nolint:exhaustruct // focused mode policy override
		Proxy:                 "http://image-proxy.example",
		Allow:                 []string{"/image"},
		NetworkPolicySet:      true,
		NetworkAllowedSchemes: []string{"http"},
		NetworkAllowedHosts:   []string{"image.example"},
	}

	got := load.ResolveEffectiveLoadGlobal(global, mode)
	want := settings.LoadGlobal{
		Proxy:                 "http://image-proxy.example",
		Allow:                 []string{"/image", "/shared"},
		EnableLocalFileAccess: true,
		NetworkPolicySet:      true,
		NetworkAllowedSchemes: []string{"https"},
		NetworkAllowedHosts:   []string{"shared.example"},
		NetworkBlockPrivate:   true,
		NetworkBlockCrossHost: true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effective load global = %+v, want %+v", got, want)
	}
}

func TestResolveEffectiveLoadGlobalModePolicyFallback(t *testing.T) {
	t.Parallel()

	global := settings.LoadGlobal{Proxy: "http://shared-proxy.example"} //nolint:exhaustruct // focused shared policy
	mode := settings.LoadGlobal{                                        //nolint:exhaustruct // mode policy
		NetworkPolicySet:      true,
		NetworkAllowedSchemes: []string{"https"},
		NetworkAllowedHosts:   []string{"image.example"},
		NetworkBlockPrivate:   true,
		NetworkBlockCrossHost: true,
	}

	got := load.ResolveEffectiveLoadGlobal(global, mode)
	if !got.NetworkPolicySet || !got.NetworkBlockPrivate || !got.NetworkBlockCrossHost {
		t.Fatalf("mode network policy was not carried into effective settings: %+v", got)
	}

	if !reflect.DeepEqual(got.NetworkAllowedSchemes, mode.NetworkAllowedSchemes) ||
		!reflect.DeepEqual(got.NetworkAllowedHosts, mode.NetworkAllowedHosts) {
		t.Fatalf("mode network allowlists = schemes %v hosts %v", got.NetworkAllowedSchemes, got.NetworkAllowedHosts)
	}
}

func TestResolveEffectiveLoadGlobalOwnsPolicySlices(t *testing.T) {
	t.Parallel()

	global := settings.LoadGlobal{ //nolint:exhaustruct // focused shared policy
		Allow:                 []string{"/shared"},
		NetworkAllowedSchemes: []string{"https"},
		NetworkAllowedHosts:   []string{"shared.example"},
	}
	mode := settings.LoadGlobal{ //nolint:exhaustruct // focused mode policy override
		Allow:                 []string{"/image"},
		NetworkAllowedSchemes: []string{"http"},
		NetworkAllowedHosts:   []string{"image.example"},
	}

	got := load.ResolveEffectiveLoadGlobal(global, mode)
	global.Allow[0] = mutatedShared
	global.NetworkAllowedSchemes[0] = mutatedShared
	global.NetworkAllowedHosts[0] = mutatedShared
	mode.Allow[0] = mutatedImage
	mode.NetworkAllowedSchemes[0] = mutatedImage
	mode.NetworkAllowedHosts[0] = mutatedImage

	if got.Allow[0] != "/image" || got.Allow[1] != "/shared" {
		t.Fatalf("effective ACL aliases source slices: %v", got.Allow)
	}

	if got.NetworkAllowedSchemes[0] != "https" || got.NetworkAllowedHosts[0] != "shared.example" {
		t.Fatalf("effective network policy aliases source slices: schemes=%v hosts=%v",
			got.NetworkAllowedSchemes, got.NetworkAllowedHosts)
	}
}

func TestGuessURL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	filePath := filepath.Join(dir, "page.html")
	if err := os.WriteFile(filePath, []byte("<html></html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		inPath string
		kind   load.Kind
		want   string
	}{
		{"http://example.com/a.html", load.KindHTTP, "http://example.com/a.html"},
		{"https://example.com", load.KindHTTP, "https://example.com"},
		{"example.com:8080/a", load.KindHTTP, "http://example.com:8080/a"},
		{"<html>x</html>", load.KindInline, "inline:"},
		{"data:text/plain,hi", load.KindInline, "data:"},
		{filePath, load.KindFile, "file://"},
		{"not-an-existing-host", load.KindHTTP, "http://not-an-existing-host"},
	}
	for _, testCase := range cases {
		kind, target, err := load.GuessURL(testCase.inPath)
		if err != nil {
			t.Errorf("load.GuessURL(%q): %v", testCase.inPath, err)

			continue
		}

		if kind != testCase.kind || !strings.HasPrefix(target, testCase.want) {
			t.Errorf("load.GuessURL(%q) = %v, %q; want %v, prefix %q",
				testCase.inPath, kind, target, testCase.kind, testCase.want)
		}
	}
}

func TestIsHTML(t *testing.T) {
	t.Parallel()

	if !load.IsHTML("<html><body></body></html>") {
		t.Error("inline html not detected")
	}

	if load.IsHTML("page.html") {
		t.Error("path misdetected as html")
	}
}

func TestLoadHTTPBasic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/page" {
			respWriter.Header().Set("Content-Type", "text/html")
			_, _ = respWriter.Write([]byte("<html><body>ok</body></html>"))

			return
		}

		http.NotFound(respWriter, req)
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	res, err := loader.Load(t.Context(), srv.URL+"/page", defaultLP())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d", res.StatusCode)
	}
}

func TestLoadHTTPCustomHeadersAndAuth(t *testing.T) {
	t.Parallel()

	var muLock sync.Mutex

	got := map[string]string{}

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		muLock.Lock()
		defer muLock.Unlock()

		got["x-token"] = req.Header.Get("X-Token")
		targetURL, pathStr, okPath := req.BasicAuth()

		if okPath {
			got["user"] = targetURL
			got["pass"] = pathStr
		}

		got["ua"] = req.Header.Get("User-Agent")

		_, _ = respWriter.Write([]byte("ok"))
	}))
	defer srv.Close()

	pageLoad := defaultLP()
	pageLoad.CustomHeaders = map[string]string{"X-Token": "secret"}
	pageLoad.Username = "bob"
	pageLoad.Password = "hunter2"
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	if _, err := loader.Load(t.Context(), srv.URL, pageLoad); err != nil {
		t.Fatal(err)
	}

	muLock.Lock()
	defer muLock.Unlock()

	if got["x-token"] != "secret" || got["user"] != "bob" || got["pass"] != "hunter2" {
		t.Errorf("headers/auth = %v", got)
	}

	if !strings.HasPrefix(got["ua"], "gowkhtmltopdf") {
		t.Errorf("ua = %q", got["ua"])
	}
}

func TestLoadHTTPPost(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("method = %s", req.Method)
		}

		if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q", ct)
		}

		_ = req.ParseForm()

		if req.Form.Get("q") != "hello world" || req.Form.Get("x") != "1" {
			t.Errorf("form = %v", req.Form)
		}

		_, _ = respWriter.Write([]byte("posted"))
	}))
	defer srv.Close()

	pageLoad := defaultLP()
	pageLoad.Post = []settings.PostItem{{Name: "q", Value: "hello world"}, {Name: "x", Value: "1"}}
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	res, err := loader.Load(t.Context(), srv.URL, pageLoad)
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Body) != "posted" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestLoadHTTPErrorCodes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/404":
			http.NotFound(respWriter, req)
		case "/401":
			respWriter.WriteHeader(http.StatusUnauthorized)
		case "/500":
			respWriter.WriteHeader(http.StatusInternalServerError)
		default:
			_, _ = respWriter.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	for _, testCase := range []struct {
		path string
		code int
	}{
		{"/404", 2},
		{"/401", 3},
	} {
		_, err := loader.Load(t.Context(), srv.URL+testCase.path, defaultLP())
		checkAbortError(t, err, testCase.path, testCase.code)
	}

	// skip policy: no error, Skip=true
	pageLoad := defaultLP()
	pageLoad.LoadErrorHandling = settings.LoadErrorSkip

	res, err := loader.Load(t.Context(), srv.URL+"/500", pageLoad)
	if err != nil {
		t.Fatal(err)
	}

	if !res.Skip {
		t.Error("skip policy must set Skip")
	}

	// ignore policy: no error, empty body
	pageLoad.LoadErrorHandling = settings.LoadErrorIgnore

	res, err = loader.Load(t.Context(), srv.URL+"/500", pageLoad)
	if err != nil {
		t.Fatal(err)
	}

	if res.Skip {
		t.Error("ignore policy must not set Skip")
	}
}

// checkAbortError asserts that err is an HttpStatusError with the given
// HTTP error code.
func checkAbortError(t *testing.T, err error, path string, code int) {
	t.Helper()

	var he *settings.HttpStatusError
	if !errors.As(err, &he) || he.HttpErrorCode() != code {
		t.Errorf("%s error = %v, want HttpStatusError code %d", path, err, code)
	}
}

func TestLoadCookies(t *testing.T) {
	t.Parallel()

	var muLock sync.Mutex

	var gotCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		muLock.Lock()
		gotCookie = req.Header.Get("Cookie")
		muLock.Unlock()

		_, _ = respWriter.Write([]byte("ok"))
	}))
	defer srv.Close()

	pageLoad := defaultLP()
	pageLoad.Cookies = map[string]string{"session": "abc123"}
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	if _, err := loader.Load(t.Context(), srv.URL, pageLoad); err != nil {
		t.Fatal(err)
	}

	muLock.Lock()
	defer muLock.Unlock()

	if !strings.Contains(gotCookie, "session=abc123") {
		t.Errorf("cookie = %q", gotCookie)
	}
}

func TestACLDefaultDeny(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	pageLoad := defaultLP()                         // BlockLocalFileAccess = true, no allow prefixes
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	_, err := loader.Load(t.Context(), secret, pageLoad)
	if err == nil {
		t.Fatal("default policy must deny local file access")
	}

	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("err = %v", err)
	}
}

func TestACLAllowPrefix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	allowed := filepath.Join(dir, "public", "a.html")
	if err := os.MkdirAll(filepath.Dir(allowed), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(allowed, []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	pageLoad := defaultLP()
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.Allow = []string{filepath.Join(dir, "public")}

	res, err := loader.Load(t.Context(), allowed, pageLoad)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}

	// sibling outside allow prefix stays denied
	outside := filepath.Join(dir, "other.html")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := loader.Load(t.Context(), outside, pageLoad); err == nil {
		t.Error("outside allow prefix must stay denied")
	}
}

func TestACLEnableLocalFileAccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	filePath := filepath.Join(dir, "page.html")
	if err := os.WriteFile(filePath, []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.EnableLocalFileAccess = true

	if _, err := loader.Load(t.Context(), filePath, pageLoad); err != nil {
		t.Errorf("enabled local access must load: %v", err)
	}

	// global on but object still blocks → denied
	lp2 := defaultLP()
	l2 := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	l2.EnableLocalFileAccess = true

	if _, err := l2.Load(t.Context(), filePath, lp2); err == nil {
		t.Error("object block must still apply")
	}
}

func TestSubresourceFetch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/style.css":
			respWriter.Header().Set("Content-Type", "text/css")
			_, _ = respWriter.Write([]byte("body{}"))
		case "/img/logo.png":
			_, _ = respWriter.Write([]byte("PNG"))
		default:
			http.NotFound(respWriter, req)
		}
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	res, err := loader.FetchSub(t.Context(), srv.URL+"/page.html", "/style.css", defaultLP())
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Body) != "body{}" {
		t.Errorf("css = %q", res.Body)
	}

	res, err = loader.FetchSub(t.Context(), srv.URL+"/page.html", "img/logo.png", defaultLP())
	if err != nil {
		t.Fatal(err)
	}

	if string(res.Body) != "PNG" {
		t.Errorf("img = %q", res.Body)
	}
}

func TestConcurrentLoads(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		time.Sleep(20 * time.Millisecond)

		_, _ = respWriter.Write([]byte("ok"))
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	var waitGroup sync.WaitGroup

	errs := make(chan error, 8)

	for range 8 {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			_, err := loader.Load(t.Context(), srv.URL, defaultLP())
			errs <- err
		}()
	}

	waitGroup.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent load: %v", err)
		}
	}
}

func TestRedirectLimit(t *testing.T) {
	t.Parallel()

	var num int

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		num++
		if num < 15 {
			http.Redirect(respWriter, req, "/next", http.StatusFound)

			return
		}

		_, _ = respWriter.Write([]byte("done"))
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	_, err := loader.Load(t.Context(), srv.URL, defaultLP())
	if err == nil {
		t.Fatal("redirect loop must error")
	}
}

// --- security: file:// scheme, path traversal, symlink escape ---

func TestACLFileURL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	filePath := filepath.Join(dir, "page.html")
	if err := os.WriteFile(filePath, []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	// default policy denies file:// loads
	denyLoader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	if _, err := denyLoader.Load(t.Context(), "file://"+filePath, defaultLP()); err == nil {
		t.Error("default policy must deny file:// loads")
	}

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.EnableLocalFileAccess = true

	res, err := loader.Load(t.Context(), "file://"+filePath, pageLoad)
	if err != nil {
		t.Fatalf("file:// load: %v", err)
	}

	if !strings.Contains(string(res.Body), "ok") {
		t.Errorf("body = %q", res.Body)
	}

	// file://localhost/... is the same machine
	if _, err := loader.Load(t.Context(), "file://localhost"+filePath, pageLoad); err != nil {
		t.Errorf("file://localhost load: %v", err)
	}

	// a remote file host is refused outright
	if _, err := loader.Load(t.Context(), "file://evil.example.com"+filePath, pageLoad); err == nil {
		t.Error("remote file host must be refused")
	}
}

func TestACLPathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}

	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	inPath := filepath.Join(public, "a.html")
	if err := os.WriteFile(inPath, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.Allow = []string{public}

	// a file inside the prefix stays readable
	if _, err := loader.Load(t.Context(), inPath, defaultLP()); err != nil {
		t.Fatalf("inside prefix: %v", err)
	}

	// ../ escape as a plain path
	esc := public + "/../secret.html"
	if _, err := loader.Load(t.Context(), esc, defaultLP()); err == nil {
		t.Error("path traversal escape must be denied")
	}

	// ../ escape via a file:// URL, raw and percent-encoded
	for _, targetURL := range []string{
		"file://" + public + "/../secret.html",
		"file://" + public + "/%2e%2e/secret.html",
	} {
		if _, err := loader.Load(t.Context(), targetURL, defaultLP()); err == nil {
			t.Errorf("traversal via %q must be denied", targetURL)
		}
	}
}

func TestACLSymlinkEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	public := filepath.Join(dir, "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}

	secret := filepath.Join(dir, "secret.html")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	realPath := filepath.Join(public, "real.html")
	if err := os.WriteFile(realPath, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}

	escapeLink := filepath.Join(public, "escape.html")
	if err := os.Symlink(secret, escapeLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	inLink := filepath.Join(public, "in.html")
	if err := os.Symlink(realPath, inLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.Allow = []string{public}

	// a symlink inside the prefix pointing outside it must be denied
	if _, err := loader.Load(t.Context(), escapeLink, defaultLP()); err == nil {
		t.Error("symlink escape must be denied")
	}
	// a symlink pointing inside the prefix stays allowed
	if _, err := loader.Load(t.Context(), inLink, defaultLP()); err != nil {
		t.Errorf("symlink inside prefix: %v", err)
	}
}

func TestSubresourceFileACL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte("<html>x</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	img := filepath.Join(dir, "x.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := "file://" + dir + "/page.html"

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	if _, err := loader.FetchSub(t.Context(), base, "x.png", defaultLP()); err == nil {
		t.Error("file subresource must be denied by default")
	}

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	l2 := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	l2.EnableLocalFileAccess = true

	res, err := l2.FetchSub(t.Context(), base, "x.png", pageLoad)
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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		defer conn.Close()
		br := bufio.NewReader(conn)

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}

			if line == "\r\n" || line == "\n" {
				break
			}
		}

		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 8192\r\nConnection: close\r\n\r\n"))
		_, _ = conn.Write(make([]byte, 128))
	}()

	return "http://" + listener.Addr().String()
}

func TestMaxBodySizeHTTP(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/big": // chunked, no Content-Length
			_, _ = respWriter.Write(make([]byte, 4096))
		case "/exact":
			_, _ = respWriter.Write(make([]byte, 1024))
		case "/small":
			_, _ = respWriter.Write(make([]byte, 64))
		}
	}))
	defer srv.Close()

	liar := lyingContentLength(t)

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.MaxBodySize = 1024

	for _, targetURL := range []string{srv.URL + "/big", liar} {
		_, err := loader.Load(t.Context(), targetURL, defaultLP())
		if err == nil {
			t.Errorf("%s: oversized body must be rejected", targetURL)

			continue
		}

		if !strings.Contains(err.Error(), "max body size") {
			t.Errorf("%s: err = %v", targetURL, err)
		}
	}

	for _, pathStr := range []string{"/exact", "/small"} {
		res, err := loader.Load(t.Context(), srv.URL+pathStr, defaultLP())
		if err != nil {
			t.Errorf("%s: %v", pathStr, err)

			continue
		}

		if len(res.Body) > 1024 {
			t.Errorf("%s: body length %d", pathStr, len(res.Body))
		}
	}
}

func TestMaxBodySizeFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	big := filepath.Join(dir, "big.html")
	if err := os.WriteFile(big, make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}

	okPath := filepath.Join(dir, "ok.html")
	if err := os.WriteFile(okPath, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}

	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.EnableLocalFileAccess = true
	loader.MaxBodySize = 1024

	_, err := loader.Load(t.Context(), big, pageLoad)
	if err == nil {
		t.Error("oversized local file must be rejected")
	} else if !strings.Contains(err.Error(), "max body size") {
		t.Errorf("err = %v", err)
	}

	if _, err := loader.Load(t.Context(), okPath, pageLoad); err != nil {
		t.Errorf("small file: %v", err)
	}
}

func TestSlowServerTimeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)

		_, _ = respWriter.Write([]byte("too late"))
	}))
	defer srv.Close()

	pageLoad := defaultLP()
	pageLoad.Timeout = 1
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	start := time.Now()

	_, err := loader.Load(t.Context(), srv.URL, pageLoad)
	if err == nil {
		t.Fatal("slow server must time out")
	}

	if d := time.Since(start); d > 2500*time.Millisecond {
		t.Errorf("timeout took %v", d)
	}
}

func TestContextCancelAbortsBodyRead(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		respWriter.Header().Set("Content-Type", "text/html")
		respWriter.WriteHeader(http.StatusOK)

		if filePath, okPath := respWriter.(http.Flusher); okPath {
			filePath.Flush()
		}

		close(started)
		<-release // hang until the test lets go
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	errCh := make(chan error, 1)

	go func() {
		_, err := loader.Load(ctx, srv.URL, defaultLP())
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		num, err := strconv.Atoi(strings.TrimPrefix(req.URL.Path, "/r/"))
		if err != nil || num <= 0 {
			_, _ = respWriter.Write([]byte("done"))

			return
		}

		http.Redirect(respWriter, req, fmt.Sprintf("/r/%d", num-1), http.StatusFound)
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.MaxRedirects = 2

	res, err := loader.Load(t.Context(), srv.URL+"/r/2", defaultLP())
	if err != nil {
		t.Fatalf("exactly MaxRedirects redirects must succeed: %v", err)
	}

	if string(res.Body) != "done" {
		t.Errorf("body = %q", res.Body)
	}

	if _, err := loader.Load(t.Context(), srv.URL+"/r/3", defaultLP()); err == nil {
		t.Error("one more than MaxRedirects must fail")
	}
}

// TestHTTPLocalhostAllowedByDesign documents the intended SSRF posture:
// the loader fetches any URL the document references - including
// http://localhost - exactly like upstream wkhtmltopdf. Only file:// reads
// are gated by the ACL.
func TestHTTPLocalhostAllowedByDesign(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		_, _ = respWriter.Write([]byte("localhost ok"))
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	res, err := loader.Load(t.Context(), srv.URL, defaultLP())
	if err != nil {
		t.Fatalf("http://127.0.0.1 must be fetchable: %v", err)
	}

	if string(res.Body) != "localhost ok" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestRestrictedNetworkPolicyBlocksPrivateAddress(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("restricted policy must reject the private address before serving")
	}))
	defer srv.Close()

	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct // compatibility global settings are intentionally empty
		load.RestrictedNetworkPolicy(),
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := loader.Load(t.Context(), srv.URL, defaultLP()); !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("error = %v, want load.ErrNetworkPolicy", err)
	}
}

func TestRestrictedNetworkPolicyAllowsExplicitHostException(t *testing.T) { //nolint:dupl // explicit-host counterpart
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		_, _ = respWriter.Write([]byte("explicitly trusted"))
	}))
	defer srv.Close()

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{parsed.Hostname()}
	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct // compatibility global settings are intentionally empty
		policy,
	)

	if err != nil {
		t.Fatal(err)
	}

	res, err := loader.Load(t.Context(), srv.URL, defaultLP())
	if err != nil {
		t.Fatalf("explicit host exception: %v", err)
	}

	if string(res.Body) != "explicitly trusted" {
		t.Errorf("body = %q", res.Body)
	}
}

func TestRestrictedNetworkPolicyBlocksCrossHostRedirect(t *testing.T) {
	t.Parallel()

	target := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		_, _ = respWriter.Write([]byte("redirect target"))
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		http.Redirect(respWriter, req, target.URL, http.StatusFound)
	}))
	defer origin.Close()

	originURL, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatal(err)
	}

	targetURL, err := url.Parse(target.URL)

	if err != nil {
		t.Fatal(err)
	}

	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{originURL.Hostname(), targetURL.Hostname()}
	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct // compatibility global settings are intentionally empty
		policy,
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := loader.Load(t.Context(), origin.URL, defaultLP()); !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("error = %v, want cross-host network policy rejection", err)
	}
}

// TestLoadInlineHTML: an explicit in-memory HTML source is returned as-is
// and skips GuessURL entirely; subresources resolve against InlineBase.
func TestLoadInlineHTML(t *testing.T) {
	t.Parallel()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	pageLoad := defaultLP()
	pageLoad.InlineHTML = []byte("<html><body>inline</body></html>")
	pageLoad.InlineBase = "https://example.com/docs/page.html"

	// The input would be treated as an http:// URL by GuessURL; InlineHTML
	// must short-circuit it without any guessing or fetching.
	res, err := loader.Load(t.Context(), "this is not a url", pageLoad)
	if err != nil {
		t.Fatal(err)
	}

	if res.Kind != load.KindInline {
		t.Errorf("kind = %v, want load.KindInline", res.Kind)
	}

	if string(res.Body) != "<html><body>inline</body></html>" {
		t.Errorf("body = %q", res.Body)
	}

	if res.Base != "https://example.com/docs/page.html" {
		t.Errorf("base = %q, want InlineBase", res.Base)
	}

	lp2 := defaultLP()
	lp2.InlineHTML = []byte("<html></html>")

	res2, err := loader.Load(t.Context(), "ignored", lp2)
	if err != nil {
		t.Fatal(err)
	}

	if res2.Base != "" {
		t.Errorf("empty InlineBase must leave Base empty, got %q", res2.Base)
	}
}

func TestDataURLHonorsBodyLimitForPrimaryAndSubresource(t *testing.T) {
	t.Parallel()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.MaxBodySize = 4
	pageLoad := defaultLP()

	if _, err := loader.Load(t.Context(), "data:text/plain,12345", pageLoad); err == nil {
		t.Fatal("oversized primary data URL must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("primary error = %v", err)
	}

	if _, err := loader.FetchSub(t.Context(), "", "data:text/plain,12345", pageLoad); err == nil {
		t.Fatal("oversized data subresource must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("subresource error = %v", err)
	}

	if _, err := loader.Load(t.Context(), "data:text/plain;base64,MTIzNDU=", pageLoad); err == nil {
		t.Fatal("oversized base64 data URL must be rejected")
	} else if !strings.Contains(err.Error(), "data URL exceeds max body size 4") {
		t.Fatalf("base64 error = %v", err)
	}

	res, err := loader.FetchSub(t.Context(), "", "data:text/plain,1234", pageLoad)
	if err != nil {
		t.Fatalf("data URL at the body limit: %v", err)
	}

	if string(res.Body) != "1234" {
		t.Errorf("body = %q, want 1234", res.Body)
	}
}

func TestInlineHTMLHonorsBodyLimit(t *testing.T) {
	t.Parallel()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.MaxBodySize = 4
	pageLoad := defaultLP()
	pageLoad.InlineHTML = []byte("12345")

	if _, err := loader.Load(t.Context(), "ignored", pageLoad); err == nil {
		t.Fatal("oversized inline HTML must be rejected")
	} else if !strings.Contains(err.Error(), "inline HTML exceeds max body size 4") {
		t.Fatalf("error = %v", err)
	}
}

func TestEmptyInlineBaseRejectsRelativeSubresources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	path := filepath.Join(dir, "local.css")
	if err := os.WriteFile(path, []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	// Even with local access enabled, an inline document without a base must
	// not reinterpret a relative reference as a process-working-directory
	// file. The reference is unresolved, not an implicit local path.
	loader.EnableLocalFileAccess = true
	if _, err := loader.FetchSub(t.Context(), "", path, defaultLP()); err == nil {
		t.Fatal("relative reference without a base must be rejected")
	} else if !strings.Contains(err.Error(), "without a document base URL") {
		t.Fatalf("error = %v", err)
	}

	res, err := loader.FetchSub(t.Context(), "", "data:text/plain,ok", defaultLP())
	if err != nil {
		t.Fatalf("absolute data reference without a base: %v", err)
	}

	if string(res.Body) != "ok" {
		t.Errorf("body = %q, want ok", res.Body)
	}
}

func TestResourceContextBindsBaseAndPolicy(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	styleDir := filepath.Join(dir, "styles")
	if err := os.Mkdir(styleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	stylePath := filepath.Join(styleDir, "site.css")
	if err := os.WriteFile(stylePath, []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	loader.EnableLocalFileAccess = true
	pageURL := "file://" + filepath.ToSlash(filepath.Join(dir, "page.html"))
	base := &load.Resource{Base: pageURL} //nolint:exhaustruct // intentional zero/partial fields
	pageLoad := defaultLP()
	pageLoad.BlockLocalFileAccess = false
	ctx := loader.ForResource(base, pageLoad)

	res, err := ctx.Fetch(t.Context(), "styles/site.css")
	if err != nil {
		t.Fatalf("relative fetch = %v", err)
	}

	if string(res.Body) != "body{}" {
		t.Errorf("body = %q, want body{}", res.Body)
	}
}

func TestLoadCharsetContentType(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.Header().Set("Content-Type", req.URL.Query().Get("ct"))
		_, _ = respWriter.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields
	for _, testCase := range []struct {
		ct     string
		okPath bool
		want   string
	}{
		{"text/html", true, ""},
		{"text/html; charset=utf-8", true, ""},
		{"text/html; charset=UTF-8", true, ""},
		{"text/html; charset=us-ascii", true, ""},
		{"text/html; charset=ISO-8859-1", false, "unsupported charset: ISO-8859-1 (only UTF-8/ASCII)"},
		{"text/html; charset=windows-1252", false, "unsupported charset: windows-1252 (only UTF-8/ASCII)"},
	} {
		_, err := loader.Load(t.Context(), srv.URL+"?ct="+url.QueryEscape(testCase.ct), defaultLP())
		if testCase.okPath && err != nil {
			t.Errorf("ct %q: %v", testCase.ct, err)
		}

		if !testCase.okPath {
			if err == nil {
				t.Errorf("ct %q: expected error", testCase.ct)
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("ct %q: err = %v, want contains %q", testCase.ct, err, testCase.want)
			}
		}
	}
}

func TestLoadCharsetMetaDecl(t *testing.T) {
	t.Parallel()
	// Content-Type without a charset parameter: the <meta> declaration is
	// the only charset signal, and it must be honored at the load seam.
	var body string

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		respWriter.Header().Set("Content-Type", "text/html")
		_, _ = respWriter.Write([]byte(body))
	}))
	defer srv.Close()

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // intentional zero/partial fields

	for _, testCase := range []struct {
		name, head string
		okPath     bool
		want       string
	}{
		{"utf8-charset", `<meta charset="utf-8">`, true, ""},
		{"utf8-content-type", `<meta http-equiv="content-type" content="text/html; charset=UTF-8">`, true, ""},
		{"no-meta", `<html><body>x</body></html>`, true, ""},
		{"latin1-charset", `<meta charset="windows-1252">`, false, "unsupported charset: windows-1252 (only UTF-8/ASCII)"},
		{
			"latin1-content-type",
			`<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1">`,
			false, "unsupported charset: ISO-8859-1 (only UTF-8/ASCII)",
		},
	} {
		body = testCase.head + "<title>t</title></head><body>x</body></html>"

		_, err := loader.Load(t.Context(), srv.URL+"/"+testCase.name, defaultLP())
		if testCase.okPath && err != nil {
			t.Errorf("%s: %v", testCase.name, err)
		}

		if !testCase.okPath {
			if err == nil {
				t.Errorf("%s: expected error", testCase.name)
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("%s: err = %v, want contains %q", testCase.name, err, testCase.want)
			}
		}
	}
}

type fakeResolver struct {
	ips   map[string][]net.IP
	calls []string
}

func (f *fakeResolver) LookupIP(_ context.Context, _, host string) ([]net.IP, error) {
	f.calls = append(f.calls, host)
	if ips, ok := f.ips[strings.ToLower(host)]; ok {
		return ips, nil
	}

	return nil, &net.DNSError{ //nolint:exhaustruct // stdlib error type with many optional fields
		Err: "no such host", Name: host, IsNotFound: true,
	}
}

func newRestrictedLoader(t *testing.T, resolver load.IPResolver) *load.Loader {
	t.Helper()

	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct
		load.RestrictedNetworkPolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	loader.Resolver = resolver

	return loader
}

func TestRestrictedPinnedDialNeverRedialsHostname(t *testing.T) {
	t.Parallel()

	const publicHost = "public.example.test"

	resolver := &fakeResolver{ips: map[string][]net.IP{ //nolint:exhaustruct // calls field filled by LookupIP under test
		publicHost: {net.ParseIP("93.184.216.34")},
	}}
	loader := newRestrictedLoader(t, resolver)

	var dialed []string

	loader.SetTestDial(func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = append(dialed, address)

		return nil, errors.New("dial blocked in test") //nolint:err113 // test-local sentinel with dynamic context
	})

	_, err := loader.Load(t.Context(), "http://"+publicHost+"/", defaultLP())
	if err == nil {
		t.Fatal("expected dial failure")
	}

	if len(dialed) == 0 {
		t.Fatal("Restricted path never dialed")
	}

	for _, address := range dialed {
		host, _, splitErr := net.SplitHostPort(address)
		if splitErr != nil {
			t.Fatalf("dialed %q: %v", address, splitErr)
		}

		if host == publicHost {
			t.Fatalf("pinned dial re-used hostname %q", address)
		}

		if host == "127.0.0.1" || host == "::1" {
			t.Fatalf("pinned dial saw loopback %q", address)
		}

		if ip := net.ParseIP(host); ip == nil || !ip.Equal(net.ParseIP("93.184.216.34")) {
			t.Fatalf("pinned dial address = %q, want 93.184.216.34", address)
		}
	}
}

func TestRestrictedBlocksPrivateResolvedRecords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ip   string
	}{
		{name: "loopback", ip: "127.0.0.1"},
		{name: "rfc1918", ip: "10.1.2.3"},
		{name: "link-local", ip: "169.254.1.1"},
		{name: "metadata", ip: "169.254.169.254"},
		{name: "cgnat", ip: "100.64.0.1"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			host := testCase.name + ".private.test"
			resolver := &fakeResolver{ips: map[string][]net.IP{ //nolint:exhaustruct // calls field filled by LookupIP under test
				host: {net.ParseIP(testCase.ip)},
			}}
			loader := newRestrictedLoader(t, resolver)

			_, err := loader.Load(t.Context(), "http://"+host+"/", defaultLP())
			if !errors.Is(err, load.ErrNetworkPolicy) {
				t.Fatalf("error = %v, want ErrNetworkPolicy", err)
			}
		})
	}
}

func TestRestrictedBlocksMixedPublicAndLoopbackRecords(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{ips: map[string][]net.IP{ //nolint:exhaustruct // calls field filled by LookupIP under test
		"mixed.example.test": {net.ParseIP("93.184.216.34"), net.ParseIP("127.0.0.1")},
	}}
	loader := newRestrictedLoader(t, resolver)

	_, err := loader.Load(t.Context(), "http://mixed.example.test/", defaultLP())
	if !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("error = %v, want ErrNetworkPolicy", err)
	}
}

func TestRestrictedProxyToPrivateTargetDenied(t *testing.T) {
	t.Parallel()

	global := settings.LoadGlobal{ //nolint:exhaustruct
		Proxy: "http://127.0.0.1:9",
	}

	loader, err := load.NewLoaderWithNetworkPolicy(global, load.RestrictedNetworkPolicy())
	if err != nil {
		t.Fatal(err)
	}

	_, err = loader.Load(t.Context(), "http://169.254.169.254/latest/meta-data/", defaultLP())
	if !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("error = %v, want ErrNetworkPolicy for private target via proxy", err)
	}
}

func TestRestrictedWildcardAllowlistStillBlocksPrivateIP(t *testing.T) {
	t.Parallel()

	resolver := &fakeResolver{ips: map[string][]net.IP{ //nolint:exhaustruct // calls field filled by LookupIP under test
		"evil.com": {net.ParseIP("127.0.0.1")},
	}}
	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{"*.com"}

	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct
		policy,
	)
	if err != nil {
		t.Fatal(err)
	}

	loader.Resolver = resolver

	if _, err := loader.Load(t.Context(), "http://evil.com/", defaultLP()); !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("error = %v, want ErrNetworkPolicy (wildcard must not skip IP check)", err)
	}
}

func TestRestrictedExactAllowlistPermitsPrivateLiteral(t *testing.T) { //nolint:dupl // mirrors explicit-host test
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, _ *http.Request) {
		_, _ = respWriter.Write([]byte("trusted private"))
	}))
	defer srv.Close()

	parsed, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{parsed.Hostname()}
	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct
		policy,
	)

	if err != nil {
		t.Fatal(err)
	}

	res, err := loader.Load(t.Context(), srv.URL, defaultLP())

	if err != nil {
		t.Fatalf("exact allowlist should permit private literal: %v", err)
	}

	if string(res.Body) != "trusted private" {
		t.Fatalf("body = %q", res.Body)
	}
}

func TestRestrictedSecondHopPrivateURLDenied(t *testing.T) {
	t.Parallel()

	loader := newRestrictedLoader(t, nil)

	_, err := loader.FetchSub(t.Context(), "https://example.com/page", "http://127.0.0.1/secret.png", defaultLP())
	if !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("FetchSub error = %v, want ErrNetworkPolicy", err)
	}

	meta := "http://169.254.169.254/latest/meta-data/"
	_, err = loader.FetchSub(t.Context(), "https://example.com/page", meta, defaultLP())

	if !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("metadata FetchSub error = %v, want ErrNetworkPolicy", err)
	}
}

func TestWildcardAllowlistIsLabelBoundary(t *testing.T) {
	t.Parallel()

	policy := load.RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{"*.example.com"}
	loader, err := load.NewLoaderWithNetworkPolicy(
		settings.LoadGlobal{}, //nolint:exhaustruct
		policy,
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := loader.Load(t.Context(), "http://notexample.com/", defaultLP()); !errors.Is(err, load.ErrNetworkPolicy) {
		t.Fatalf("notexample.com should not match *.example.com: %v", err)
	}
}
