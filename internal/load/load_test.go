package load

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestWaitJSDelay(t *testing.T) {
	start := time.Now()
	WaitJSDelay(context.Background(), 30)
	if d := time.Since(start); d < 25*time.Millisecond {
		t.Errorf("jsdelay slept %v", d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	WaitJSDelay(ctx, 500)
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
