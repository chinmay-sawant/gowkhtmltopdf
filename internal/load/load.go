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

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const (
	defaultKeepAliveSec  = 30
	metaScanLimit        = 1024
	httpStatusOK         = 200
	httpStatusBadRequest = 400
	hexNibbleBits        = 4
	hexLetterOffset      = 10
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

// ErrNetworkPolicy is returned when a URL, redirect, or resolved network
// address is outside the loader's explicit network policy.
var ErrNetworkPolicy = errors.New("network policy denied request")

// ErrInvalidProxy is returned when a configured proxy is not an absolute URL
// with a scheme and host. NewLoader preserves its historical return shape and
// records this error for the first Load call; NewLoaderWithError exposes the
// fail-fast form for new callers.
var ErrInvalidProxy = errors.New("invalid proxy configuration")

// Package-level sentinels for the loader's internal failure modes, so
// dynamic messages wrap a static error and stay matchable with errors.Is.
var (
	errNilLoader           = errs.ErrNilLoader
	errNilContext          = errs.ErrNilContext
	errCannotLoad          = errors.New("cannot load")
	errUnsupportedCharset  = errors.New("unsupported charset")
	errBlockedFileAccess   = errors.New("blocked file access")
	errNoDocumentBase      = errors.New("cannot resolve relative subresource")
	errUnsupportedScheme   = errors.New("unsupported subresource scheme")
	errMalformedDataURL    = errors.New("malformed data URL")
	errInvalidDataURL      = errors.New("invalid URL escape in data URL")
	errTooManyRedirects    = errors.New("too many redirects")
	errBodyTooLarge        = errors.New("exceeds max body size")
	errInvalidBodyLimit    = errors.New("invalid max body size")
	errInvalidRedirects    = errors.New("invalid max redirects")
	errUninitializedLoader = errors.New("loader client is not initialized")
)

// Kind classifies a resolved input.
type Kind int

const (
	KindUnknown Kind = iota
	KindHTTP
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
	loader   *Loader
	base     string
	pageLoad settings.LoadPage
}

// NetworkPolicy controls network URL loading independently from the legacy
// local-file ACL. An empty AllowedHosts list permits any host that satisfies
// the scheme policy; a non-empty list is an exact or wildcard host allowlist.
// An explicitly allowlisted host may be private, which makes local test and
// trusted-service integrations possible without making that exception global.
type NetworkPolicy struct {
	AllowedSchemes          []string
	AllowedHosts            []string
	BlockPrivateNetworks    bool
	BlockCrossHostRedirects bool
}

// CompatibleNetworkPolicy preserves the historical loader behavior: HTTP(S)
// URLs are allowed, including localhost and private addresses, and redirects
// may cross hosts. It is the default for existing constructors.
func CompatibleNetworkPolicy() NetworkPolicy {
	return NetworkPolicy{ //nolint:exhaustruct // compatibility defaults
		AllowedSchemes: []string{"http", "https"},
	}
}

// RestrictedNetworkPolicy is suitable for untrusted HTML in an isolated
// service. Private/link-local destinations are blocked unless explicitly
// allowlisted, and redirects stay on the original host.
func RestrictedNetworkPolicy() NetworkPolicy {
	return NetworkPolicy{ //nolint:exhaustruct // explicit safety defaults
		AllowedSchemes:          []string{"http", "https"},
		BlockPrivateNetworks:    true,
		BlockCrossHostRedirects: true,
	}
}

func cloneNetworkPolicy(src NetworkPolicy) NetworkPolicy {
	dst := src
	dst.AllowedSchemes = cloneStrings(src.AllowedSchemes)
	dst.AllowedHosts = cloneStrings(src.AllowedHosts)

	return dst
}

// ApplyNetworkPolicy stores policy into dst with all slice fields cloned.
func ApplyNetworkPolicy(dst *settings.LoadGlobal, policy NetworkPolicy) {
	if dst == nil {
		return
	}

	dst.NetworkPolicySet = true
	dst.NetworkAllowedSchemes = cloneStrings(policy.AllowedSchemes)
	dst.NetworkAllowedHosts = cloneStrings(policy.AllowedHosts)
	dst.NetworkBlockPrivate = policy.BlockPrivateNetworks
	dst.NetworkBlockCrossHost = policy.BlockCrossHostRedirects
}

// ResolveEffectiveLoadGlobal returns the load settings for one conversion
// mode. global is the shared PDF/global policy; mode contains settings owned
// by a mode-specific request, such as image-mode proxy and ACL values.
//
// Mode-specific proxy settings override the shared proxy when present. ACL
// prefixes are additive and local-file access is enabled when either source
// enables it. Network policy remains shared-policy first: an explicit global
// NetworkPolicySet cannot be weakened by mode defaults, while a mode policy is
// used when no shared policy was configured. All slices are copied so the
// result is an owned effective snapshot for loader construction.
func ResolveEffectiveLoadGlobal(global, mode settings.LoadGlobal) settings.LoadGlobal {
	effective := global
	effective.Allow = append(cloneStrings(mode.Allow), global.Allow...)
	effective.NetworkAllowedSchemes = cloneStrings(global.NetworkAllowedSchemes)
	effective.NetworkAllowedHosts = cloneStrings(global.NetworkAllowedHosts)

	if mode.Proxy != "" {
		effective.Proxy = mode.Proxy
	}

	effective.EnableLocalFileAccess = global.EnableLocalFileAccess || mode.EnableLocalFileAccess

	if !global.NetworkPolicySet && mode.NetworkPolicySet {
		effective.NetworkPolicySet = true
		effective.NetworkAllowedSchemes = cloneStrings(mode.NetworkAllowedSchemes)
		effective.NetworkAllowedHosts = cloneStrings(mode.NetworkAllowedHosts)
		effective.NetworkBlockPrivate = mode.NetworkBlockPrivate
		effective.NetworkBlockCrossHost = mode.NetworkBlockCrossHost
	}

	return effective
}

// ForResource returns a context for resources relative to res. A nil
// resource is allowed for callers that only need absolute references; those
// references still go through the same loader policy.
func (l *Loader) ForResource(res *Resource, pageLoad settings.LoadPage) ResourceContext {
	var base string
	if res != nil {
		base = res.Base
	}

	return ResourceContext{loader: l, base: base, pageLoad: cloneLoadPage(pageLoad)}
}

// Loader is the owner of fetches for this document.
func (c ResourceContext) Loader() *Loader {
	return c.loader
}

// Base is the resolved document URL used for relative refs.
func (c ResourceContext) Base() string {
	return c.base
}

// PageLoad is the cloned per-page policy used for subresources.
func (c ResourceContext) PageLoad() settings.LoadPage {
	return c.pageLoad
}

// Fetch resolves ref against the document base and fetches it using the
// document's load policy.
func (c ResourceContext) Fetch(ctx context.Context, ref string) (*Resource, error) {
	if c.loader == nil {
		return nil, errNilLoader
	}

	if ctx == nil {
		return nil, errNilContext
	}

	return c.loader.FetchSub(ctx, c.base, ref, c.pageLoad)
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
	if a == nil {
		return false
	}

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

	if guessHostPort(input) {
		return KindHTTP, "http://" + input, nil
	}

	if target, ok := guessLocalFile(input); ok {
		return KindFile, target, nil
	}
	// default: treat as http URL
	return KindHTTP, "http://" + input, nil
}

// guessHostPort reports whether input is a host:port or a host with a port
// that should default to the http:// scheme.
func guessHostPort(input string) bool {
	return !isLocalPath(input) && (isHostPort(input) ||
		!strings.Contains(input, "/") && !strings.Contains(input, "\\") && strings.Contains(input, ":"))
}

// guessLocalFile resolves an existing local path to its file:// URL, or
// reports that no such path exists.
func guessLocalFile(input string) (string, bool) {
	if _, err := os.Stat(input); err != nil {
		return "", false
	}

	abs, err := filepath.Abs(input)
	if err != nil {
		return "file://" + filepath.ToSlash(input), true
	}

	return "file://" + filepath.ToSlash(abs), true
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

// IPResolver looks up host addresses. *net.Resolver implements it;
// tests inject a fake so Restricted pinning can be asserted without
// touching the system resolver.
type IPResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

// Loader fetches resources with the configured network and local-file policy.
type Loader struct {
	Client       *http.Client
	Global       settings.LoadGlobal
	Network      NetworkPolicy
	Log          io.Writer
	MaxBodySize  int64
	MaxRedirects int
	// Resolver looks up host addresses for Restricted private-IP checks
	// and pinned dials. Nil uses net.DefaultResolver.
	Resolver IPResolver

	// Global is an owned snapshot of the caller's load policy. Allow and
	// EnableLocalFileAccess remain exported compatibility fields for existing
	// internal callers; NewLoader initializes them from cloned policy values,
	// and file-access checks intentionally read these effective fields.
	Allow                 []string
	EnableLocalFileAccess bool
	initErr               error
	testDial              func(ctx context.Context, network, address string) (net.Conn, error)
}

// NewLoader builds a Loader from global load settings, applying the full
// load policy (proxy, allow prefixes, local-access flag) in one place.
func NewLoader(global settings.LoadGlobal) *Loader {
	loader, err := NewLoaderWithError(global)
	if err == nil {
		return loader
	}

	// Preserve the historical constructor shape for existing callers. New
	// callers should use NewLoaderWithError so initialization failures are
	// handled at their request boundary instead of on the first load.
	policy := ResolveEffectiveLoadGlobal(global, settings.LoadGlobal{}) //nolint:exhaustruct // empty mode override

	return &Loader{ //nolint:exhaustruct // intentional zero/partial fields
		Global:                policy,
		Network:               networkPolicyFromGlobal(global),
		Log:                   io.Discard,
		MaxBodySize:           DefaultMaxBodySize,
		MaxRedirects:          DefaultMaxRedirects,
		Allow:                 cloneStrings(policy.Allow),
		EnableLocalFileAccess: policy.EnableLocalFileAccess,
		initErr:               err,
	}
}

// NewLoaderWithError builds a Loader from global load settings and validates
// proxy configuration before installing the HTTP transport. It is the
// fail-fast constructor; NewLoader remains available for existing callers.
func NewLoaderWithError(global settings.LoadGlobal) (*Loader, error) {
	effective := ResolveEffectiveLoadGlobal(global, settings.LoadGlobal{}) //nolint:exhaustruct // empty mode override

	return NewLoaderWithNetworkPolicy(effective, networkPolicyFromGlobal(effective))
}

func networkPolicyFromGlobal(global settings.LoadGlobal) NetworkPolicy {
	if !global.NetworkPolicySet {
		return CompatibleNetworkPolicy()
	}

	return NetworkPolicy{
		AllowedSchemes:          cloneStrings(global.NetworkAllowedSchemes),
		AllowedHosts:            cloneStrings(global.NetworkAllowedHosts),
		BlockPrivateNetworks:    global.NetworkBlockPrivate,
		BlockCrossHostRedirects: global.NetworkBlockCrossHost,
	}
}

// NewLoaderWithNetworkPolicy builds a Loader with an explicit network policy.
// NewLoader and NewLoaderWithError remain compatibility constructors for
// callers that rely on the historical permissive HTTP behavior.
func NewLoaderWithNetworkPolicy(global settings.LoadGlobal, network NetworkPolicy) (*Loader, error) {
	policy := ResolveEffectiveLoadGlobal(global, settings.LoadGlobal{}) //nolint:exhaustruct // empty mode override

	loader := &Loader{ //nolint:exhaustruct // intentional zero/partial fields
		Global:                policy,
		Network:               cloneNetworkPolicy(network),
		Log:                   io.Discard,
		MaxBodySize:           DefaultMaxBodySize,
		MaxRedirects:          DefaultMaxRedirects,
		Allow:                 cloneStrings(policy.Allow),
		EnableLocalFileAccess: policy.EnableLocalFileAccess,
	}
	if err := loader.initClient(); err != nil {
		return nil, err
	}

	return loader, nil
}

func (l *Loader) initClient() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("create cookie jar: %w", err)
	}

	// Default TLS verification stays on (http.Transport system roots).
	dialer := &net.Dialer{ //nolint:exhaustruct // intentional zero/partial fields
		Timeout:   DefaultConnectTimeout,
		KeepAlive: defaultKeepAliveSec * time.Second,
	}
	transport := &http.Transport{ //nolint:exhaustruct // intentional zero/partial fields
		ForceAttemptHTTP2: true,
		DialContext:       l.policyDialContext(dialer),
	}

	if l.Global.Proxy != "" {
		pu, err := parseProxy(l.Global.Proxy)
		if err != nil {
			return err
		}

		transport.Proxy = http.ProxyURL(pu)
	}

	l.Client = &http.Client{ //nolint:exhaustruct // intentional zero/partial fields
		Transport: transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := l.checkNetworkURL(req.Context(), req.URL); err != nil {
				return err
			}

			if l.Network.BlockCrossHostRedirects && len(via) > 0 {
				origin := via[0].URL
				if !sameNetworkHost(origin, req.URL) {
					return fmt.Errorf("%w: cross-host redirect %s -> %s", ErrNetworkPolicy,
						origin.Host, req.URL.Host)
				}
			}

			// Go's client counts the requests made so far in via, so the
			// MaxRedirects-th redirect (len(via) == MaxRedirects) is the
			// last one allowed to complete.
			if len(via) > l.MaxRedirects {
				return fmt.Errorf("%w: stopped after %d redirects", errTooManyRedirects, l.MaxRedirects)
			}

			return nil
		},
	}

	return nil
}

func (l *Loader) policyDialContext( //nolint:cyclop // hostname vs IP vs proxy policy
	dialer *net.Dialer,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if !l.Network.BlockPrivateNetworks {
			return l.dialRaw(ctx, dialer, network, address)
		}

		// A configured proxy is operator-supplied; DialContext talks to the
		// proxy hop, not the target. The target's private-IP policy is
		// applied in checkNetworkURL before the request is issued.
		if l.Global.Proxy != "" {
			return l.dialRaw(ctx, dialer, network, address)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid dial address %q: %w", ErrNetworkPolicy, address, err)
		}

		if l.networkHostExactAllowlisted(host) {
			return l.dialRaw(ctx, dialer, network, address)
		}

		if ip := net.ParseIP(host); ip != nil {
			if isPrivateNetworkIP(ip) {
				return nil, fmt.Errorf("%w: private address %s", ErrNetworkPolicy, ip)
			}

			return l.dialRaw(ctx, dialer, network, address)
		}

		ips, err := l.lookupIPs(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, err)
		}

		if err := rejectPrivateResolvedIPs(host, ips); err != nil {
			return nil, err
		}

		var first error

		for _, ip := range ips {
			conn, dialErr := l.dialRaw(ctx, dialer, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}

			first = dialErr
		}

		if first == nil {
			return nil, fmt.Errorf("%w: %s resolved to no addresses", ErrNetworkPolicy, host)
		}

		return nil, first
	}
}

func (l *Loader) dialRaw(
	ctx context.Context, dialer *net.Dialer, network, address string,
) (net.Conn, error) {
	if l.testDial != nil {
		return l.testDial(ctx, network, address)
	}

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	return conn, nil
}

func (l *Loader) lookupIPs(ctx context.Context, host string) ([]net.IP, error) {
	resolver := l.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", host, err)
	}

	return ips, nil
}

func (l *Loader) checkNetworkURL(ctx context.Context, target *url.URL) error {
	if target == nil {
		return fmt.Errorf("%w: nil URL", ErrNetworkPolicy)
	}

	scheme := strings.ToLower(target.Scheme)
	if !containsFold(l.Network.AllowedSchemes, scheme) {
		return fmt.Errorf("%w: scheme %q is not allowed", ErrNetworkPolicy, target.Scheme)
	}

	host := target.Hostname()
	if host == "" {
		return fmt.Errorf("%w: URL host is empty", ErrNetworkPolicy)
	}

	if len(l.Network.AllowedHosts) > 0 && !l.networkHostAllowlisted(host) {
		return fmt.Errorf("%w: host %q is not allowlisted", ErrNetworkPolicy, host)
	}

	if !l.Network.BlockPrivateNetworks {
		return nil
	}

	// Exact allowlisted hosts may skip the private-IP check (trusted
	// internal services). Wildcard suffixes still resolve and block
	// private records.
	if l.networkHostExactAllowlisted(host) {
		return nil
	}

	return l.rejectPrivateTarget(ctx, host)
}

func (l *Loader) rejectPrivateTarget(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateNetworkIP(ip) {
			return fmt.Errorf("%w: private address %s", ErrNetworkPolicy, ip)
		}

		return nil
	}

	ips, err := l.lookupIPs(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}

	return rejectPrivateResolvedIPs(host, ips)
}

func rejectPrivateResolvedIPs(host string, ips []net.IP) error {
	if len(ips) == 0 {
		return fmt.Errorf("%w: %s resolved to no addresses", ErrNetworkPolicy, host)
	}

	for _, ip := range ips {
		if isPrivateNetworkIP(ip) {
			return fmt.Errorf("%w: %s resolves to private address %s", ErrNetworkPolicy, host, ip)
		}
	}

	return nil
}

func normalizeNetworkHost(host string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimSuffix(host, ".")))
}

func (l *Loader) networkHostAllowlisted(host string) bool {
	host = normalizeNetworkHost(host)

	if len(l.Network.AllowedHosts) == 0 {
		return false
	}

	for _, allowed := range l.Network.AllowedHosts {
		if hostMatchesAllowlist(host, allowed) {
			return true
		}
	}

	return false
}

func (l *Loader) networkHostExactAllowlisted(host string) bool {
	host = normalizeNetworkHost(host)
	for _, allowed := range l.Network.AllowedHosts {
		if normalizeNetworkHost(allowed) == host {
			return true
		}
	}

	return false
}

// hostMatchesAllowlist reports an exact host match or a label-boundary
// wildcard match. `*.example.com` matches `a.example.com`, not
// `notexample.com`. The bare apex (`example.com`) does not match `*.example.com`.
func hostMatchesAllowlist(host, allowed string) bool {
	allowed = normalizeNetworkHost(allowed)
	if allowed == host {
		return true
	}

	if !strings.HasPrefix(allowed, "*.") {
		return false
	}

	suffix := allowed[1:] // ".example.com"

	return len(host) > len(suffix) && strings.HasSuffix(host, suffix)
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}

	return false
}

func sameNetworkHost(left, right *url.URL) bool {
	return strings.EqualFold(left.Host, right.Host)
}

func mustCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}

	return network
}

func isPrivateNetworkIP(addr net.IP) bool {
	if addr == nil {
		return true
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
		return true
	}

	if mustCIDR("100.64.0.0/10").Contains(addr) {
		return true
	}

	if mustCIDR("169.254.169.0/24").Contains(addr) || addr.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}

	return false
}

func parseProxy(raw string) (*url.URL, error) {
	proxy, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidProxy, raw, err)
	}

	if proxy.Scheme == "" || proxy.Host == "" || proxy.Hostname() == "" {
		return nil, fmt.Errorf("%w %q: absolute URL with scheme and host required", ErrInvalidProxy, raw)
	}

	switch strings.ToLower(proxy.Scheme) {
	case "http", "https":
	default:
		return nil, fmt.Errorf("%w %q: only http and https proxies are supported", ErrInvalidProxy, raw)
	}

	return proxy, nil
}

// Load fetches the primary resource for an object. In-memory HTML
// (lp.InlineHTML) is returned as-is and skips GuessURL entirely; subresources
// resolve against lp.InlineBase when set. Every loaded document is checked
// for a supported charset at this seam (see checkDocumentCharset).
//
//nolint:cyclop // multi-branch resource loader
func (l *Loader) Load(ctx context.Context, input string, pageLoad settings.LoadPage) (*Resource, error) {
	if l == nil {
		return nil, errNilLoader
	}

	if l.initErr != nil {
		return nil, l.initErr
	}

	if err := l.validateLimits(); err != nil {
		return nil, err
	}

	if ctx == nil {
		return nil, errNilContext
	}

	pageLoad = cloneLoadPage(pageLoad)

	if len(pageLoad.InlineHTML) > 0 {
		if err := checkBodyLimit("inline HTML", len(pageLoad.InlineHTML), l.MaxBodySize); err != nil {
			return nil, err
		}

		res := &Resource{ //nolint:exhaustruct // intentional zero/partial fields
			Kind:        KindInline,
			URL:         "inline:",
			Base:        pageLoad.InlineBase,
			Body:        pageLoad.InlineHTML,
			ContentType: "text/html",
		}

		return res, checkDocumentCharset(res)
	}

	kind, target, err := GuessURL(input)
	if err != nil {
		return nil, err
	}

	res, err := l.loadByKind(ctx, kind, target, pageLoad)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("%w %q", errCannotLoad, input)
	}

	if !res.Skip {
		return res, checkDocumentCharset(res)
	}

	return res, nil
}

// loadByKind fetches a resolved (kind, target) pair; nil means the kind was
// not handled by any loader branch.
func (l *Loader) loadByKind(
	ctx context.Context, kind Kind, target string, pageLoad settings.LoadPage,
) (*Resource, error) {
	var res *Resource

	switch kind {
	case KindInline:
		switch {
		case strings.HasPrefix(target, "inline:"):
			body := []byte(target[len("inline:"):])
			res = &Resource{ //nolint:exhaustruct // intentional zero/partial fields
				Kind:        KindInline,
				URL:         "inline:",
				Body:        body,
				ContentType: "text/html",
			}
		case strings.HasPrefix(target, "data:"):
			body, ctype, err := decodeDataURLLimited(target, l.MaxBodySize)
			if err != nil {
				return nil, err
			}

			res = &Resource{ //nolint:exhaustruct // intentional zero/partial fields
				Kind:        KindInline,
				URL:         target,
				Body:        body,
				ContentType: ctype,
			}
		}
	case KindFile:
		return l.loadFile(ctx, target, pageLoad)
	case KindHTTP:
		return l.loadHTTP(ctx, target, pageLoad)
	case KindUnknown:
		return nil, fmt.Errorf("%w: %v", errCannotLoad, target)
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
	charset := charsetFromContentType(res.ContentType)
	if charset == "" {
		charset = metaCharset(res.Body)
	}

	if charset == "" || charsetSupported(charset) {
		return nil
	}

	return fmt.Errorf("%w: %s (only UTF-8/ASCII)", errUnsupportedCharset, charset)
}

// charsetSupported reports whether a charset name is one the pipeline can
// decode. Only UTF-8 and ASCII are accepted.
func charsetSupported(charset string) bool {
	switch strings.ToLower(strings.TrimSpace(charset)) {
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
	if len(body) > metaScanLimit {
		body = body[:metaScanLimit]
	}

	html := string(body)
	low := strings.ToLower(html)

	for {
		pos := strings.Index(low, "<meta")
		if pos < 0 {
			return ""
		}

		closeGT := strings.IndexByte(html[pos:], '>')
		if closeGT < 0 {
			return ""
		}

		if cs := metaTagCharset(html[pos : pos+closeGT]); cs != "" {
			return cs
		}

		html = html[pos+closeGT+1:]
		low = strings.ToLower(html)
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

	pos := strings.Index(low, "charset")
	if pos < 0 {
		return ""
	}

	rest := strings.TrimLeft(content[pos+len("charset"):], " \t\n\r\f")
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
		pos := strings.Index(low, needle)
		if pos < 0 {
			return ""
		}

		// a match is an attribute only when whitespace precedes it
		if pos == 0 || isMetaWhitespace(tag[pos-1]) {
			if val, ok := metaAttrValue(tag[pos+len(needle):]); ok {
				return val
			}
		}

		tag = tag[pos+len(needle):]
		low = low[pos+len(needle):]
	}
}

// metaAttrValue extracts the value following "attr=" at the front of rest,
// reporting whether rest began with the "=".
func metaAttrValue(rest string) (string, bool) {
	rest = strings.TrimLeft(rest, " \t\n\r\f")
	if !strings.HasPrefix(rest, "=") {
		return "", false
	}

	rest = strings.TrimLeft(rest[1:], " \t\n\r\f")
	if rest == "" {
		return "", true
	}

	if rest[0] == '"' || rest[0] == '\'' {
		q := rest[0]
		if k := strings.IndexByte(rest[1:], q); k >= 0 {
			return rest[1 : k+1], true
		}

		return "", true
	}

	k := 0
	for k < len(rest) && !isMetaWhitespace(rest[k]) {
		k++
	}

	return rest[:k], true
}

func isMetaWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '>'
}

func (l *Loader) validateLimits() error {
	if err := validateBodyLimit(l.MaxBodySize); err != nil {
		return err
	}

	if l.MaxRedirects < 0 {
		return fmt.Errorf("%w: %d must be non-negative", errInvalidRedirects, l.MaxRedirects)
	}

	return nil
}

func validateBodyLimit(maxBytes int64) error {
	if maxBytes < 0 {
		return fmt.Errorf("%w: %d must be non-negative", errInvalidBodyLimit, maxBytes)
	}

	return nil
}

// bodyReadLimit adds one probe byte so callers can distinguish a body at the
// limit from one that exceeds it. Avoid overflowing when the compatibility
// field is set to the largest int64 value.
func bodyReadLimit(maxBytes int64) int64 {
	if maxBytes == 1<<63-1 {
		return maxBytes
	}

	return maxBytes + 1
}

// readFileBody reads at most maxBytes plus one byte and closes the file when
// ctx is cancelled. os.File reads do not otherwise observe context
// cancellation, so the watcher makes a blocked local-file read abort at the
// same request boundary as an HTTP read.
func readFileBody(ctx context.Context, file *os.File, maxBytes int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("file read ctx: %w", err)
	}

	stop := make(chan struct{})
	watcherDone := make(chan struct{})

	go func() {
		defer close(watcherDone)

		select {
		case <-ctx.Done():
			_ = file.Close()
		case <-stop:
		}
	}()

	body, err := io.ReadAll(io.LimitReader(file, bodyReadLimit(maxBytes)))

	close(stop)
	<-watcherDone

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("file read canceled: %w", ctxErr)
		}

		return nil, fmt.Errorf("file read io: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("file read finish: %w", err)
	}

	return body, nil
}

func (l *Loader) loadFile(ctx context.Context, path string, pageLoad settings.LoadPage) (*Resource, error) {
	if ctx == nil {
		return nil, errNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load file: %w", err)
	}

	filePath, err := filePathFromURL(path)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load file: %w", err)
	}

	if !l.fileAccessAllowed(filePath, pageLoad) {
		return nil, fmt.Errorf("%w: %s", ErrAccessDenied, filePath)
	}
	// blockLocalFileAccess=true blocks local reads even for the primary page
	// unless the user explicitly enabled local access.
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}

	defer file.Close()

	body, err := readFileBody(ctx, file, l.MaxBodySize)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	if int64(len(body)) > l.MaxBodySize {
		return nil, fmt.Errorf("file %s %w %d", filePath, errBodyTooLarge, l.MaxBodySize)
	}

	return &Resource{ //nolint:exhaustruct // intentional zero/partial fields
		Kind:        KindFile,
		URL:         "file://" + filepath.ToSlash(filepath.Clean(filePath)),
		Base:        "file://" + filepath.ToSlash(filepath.Dir(filePath)) + "/",
		Body:        body,
		ContentType: mime.TypeByExtension(filepath.Ext(filePath)),
		StatusCode:  httpStatusOK,
	}, nil
}

// filePathFromURL extracts the local filesystem path from a file:// URL
// (or passes a plain path through). Remote file hosts - anything other
// than the empty host and localhost - are refused.
func filePathFromURL(path string) (string, error) {
	parsed, err := url.Parse(path)
	if err != nil {
		// unparseable strings fall back to the plain path
		return path, nil //nolint:nilerr // deliberate: parse failure means plain path
	}

	if parsed.Scheme != "file" {
		return path, nil
	}

	if parsed.Host != "" && parsed.Host != "localhost" {
		return "", fmt.Errorf("%w to %q", errBlockedFileAccess, path)
	}

	if parsed.Path != "" {
		return parsed.Path, nil
	}

	return strings.TrimPrefix(path, "file://"), nil
}

// fileAccessAllowed implements the frozen security policy: blocked by
// default; allowed when the global enable flag is on AND the object's block
// flag is off, or when an --allow prefix matches.
func (l *Loader) fileAccessAllowed(path string, pageLoad settings.LoadPage) bool {
	ctrl := &AccessController{AllowPrefixes: l.Allow}
	if ctrl.Allowed(path) {
		return true
	}

	return l.EnableLocalFileAccess && !pageLoad.BlockLocalFileAccess
}

func (l *Loader) loadHTTP(ctx context.Context, target string, pageLoad settings.LoadPage) (*Resource, error) {
	if l.Client == nil {
		return nil, errUninitializedLoader
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", target, err)
	}

	if err := l.checkNetworkURL(ctx, parsed); err != nil {
		return nil, err
	}

	req, err := buildHTTPRequest(ctx, parsed, pageLoad)
	if err != nil {
		return nil, err
	}

	c := l.Client
	client := *c
	client.Timeout = requestTimeout(pageLoad)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", parsed.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= httpStatusBadRequest {
		// skip / ignore / abort policy
		return l.loadErrorResponse(parsed, resp, pageLoad)
	}

	// Enforce the body cap: reject oversized bodies outright rather than
	// silently truncating them. Content-Length short-circuits the download
	// when present; the read-side limit is the authoritative check for
	// chunked or unknown-length responses.
	if resp.ContentLength > l.MaxBodySize {
		return nil, fmt.Errorf("response from %s %w %d (Content-Length %d)",
			parsed.String(), errBodyTooLarge, l.MaxBodySize, resp.ContentLength)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, bodyReadLimit(l.MaxBodySize)))
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", parsed.String(), err)
	}

	if int64(len(bodyBytes)) > l.MaxBodySize {
		return nil, fmt.Errorf("response from %s %w %d", parsed.String(), errBodyTooLarge, l.MaxBodySize)
	}

	ctype := resp.Header.Get("Content-Type")
	final := resp.Request.URL.String()

	return &Resource{ //nolint:exhaustruct // intentional zero/partial fields
		Kind:        KindHTTP,
		URL:         final,
		Base:        final,
		Body:        bodyBytes,
		ContentType: ctype,
		StatusCode:  resp.StatusCode,
	}, nil
}

func cloneLoadPage(src settings.LoadPage) settings.LoadPage {
	dst := src
	dst.CustomHeaders = cloneStringMap(src.CustomHeaders)
	dst.Cookies = cloneStringMap(src.Cookies)
	dst.Post = clonePostItems(src.Post)
	dst.InlineHTML = cloneBytes(src.InlineHTML)

	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)

	return dst
}

func clonePostItems(src []settings.PostItem) []settings.PostItem {
	if src == nil {
		return nil
	}

	dst := make([]settings.PostItem, len(src))
	copy(dst, src)

	return dst
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}

// buildHTTPRequest assembles the request for target according to the page
// load policy: method/POST body, user agent, basic auth, custom headers and
// cookies.
func buildHTTPRequest(ctx context.Context, parsed *url.URL, pageLoad settings.LoadPage) (*http.Request, error) {
	method := http.MethodGet

	var reqBody io.Reader

	if len(pageLoad.Post) > 0 {
		method = http.MethodPost
		reqBody = strings.NewReader(urlEncodePost(pageLoad.Post))
	}

	req, err := http.NewRequestWithContext(ctx, method, parsed.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", parsed.String(), err)
	}

	req.Header.Set("User-Agent", "github.com/chinmay-sawant/gowkhtmltopdf/0.1 (pure-Go wkhtmltopdf reimplementation)")

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if pageLoad.Username != "" {
		req.SetBasicAuth(pageLoad.Username, pageLoad.Password)
	}

	for k, v := range pageLoad.CustomHeaders {
		req.Header.Set(k, v)
	}

	for k, v := range pageLoad.Cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v}) //nolint:exhaustruct // intentional zero/partial fields
	}

	return req, nil
}

// requestTimeout resolves the effective per-request timeout from the page
// load policy.
func requestTimeout(pageLoad settings.LoadPage) time.Duration {
	timeout := time.Duration(pageLoad.Timeout) * time.Second
	if timeout <= 0 {
		return DefaultResponseTimeout
	}

	return timeout
}

// loadErrorResponse applies the --load-error-handling policy to an error
// status response.
func (l *Loader) loadErrorResponse(
	parsed *url.URL, resp *http.Response, pageLoad settings.LoadPage,
) (*Resource, error) {
	switch pageLoad.LoadErrorHandling {
	case settings.LoadErrorSkip:
		return &Resource{ //nolint:exhaustruct // intentional zero/partial fields
			Kind:       KindHTTP,
			URL:        parsed.String(),
			StatusCode: resp.StatusCode,
			Skip:       true,
		}, nil
	case settings.LoadErrorIgnore:
		return &Resource{ //nolint:exhaustruct // intentional zero/partial fields
			Kind:       KindHTTP,
			URL:        parsed.String(),
			StatusCode: resp.StatusCode,
		}, nil
	case settings.LoadErrorAbort:
		return nil, &settings.HttpStatusError{Status: resp.StatusCode, URL: parsed.String()}
	}

	return nil, &settings.HttpStatusError{Status: resp.StatusCode, URL: parsed.String()}
}

// FetchSub fetches a subresource (css/img) resolved against base.
//
//nolint:cyclop // multi-branch subresource loader
func (l *Loader) FetchSub(ctx context.Context, base, ref string, pageLoad settings.LoadPage) (*Resource, error) {
	if l == nil {
		return nil, errNilLoader
	}

	if l.initErr != nil {
		return nil, l.initErr
	}

	if err := l.validateLimits(); err != nil {
		return nil, err
	}

	if ctx == nil {
		return nil, errNilContext
	}

	pageLoad = cloneLoadPage(pageLoad)

	abs, err := resolveReference(base, ref)
	if err != nil {
		return nil, err
	}

	switch abs.Scheme {
	case "file", "":
		if abs.Host != "" && abs.Host != "localhost" {
			return nil, fmt.Errorf("%w to %q", errBlockedFileAccess, abs.String())
		}

		return l.loadFile(ctx, abs.Path, pageLoad)
	case "http", "https":
		return l.loadHTTP(ctx, abs.String(), pageLoad)
	case "data":
		body, ctype, err := decodeDataURLLimited(abs.String(), l.MaxBodySize)
		if err != nil {
			return nil, err
		}

		return &Resource{ //nolint:exhaustruct // intentional zero/partial fields
			Kind:        KindInline,
			URL:         abs.String(),
			Body:        body,
			ContentType: ctype,
		}, nil
	}

	return nil, fmt.Errorf("%w %q", errUnsupportedScheme, abs.Scheme)
}

// resolveReference resolves ref against base, mirroring the loader's URL
// policy: absolute references pass through; relative ones need a base.
func resolveReference(base, ref string) (*url.URL, error) {
	refURL, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return nil, fmt.Errorf("parse ref %q: %w", ref, err)
	}

	if refURL.IsAbs() {
		return refURL, nil
	}

	if base == "" {
		return nil, fmt.Errorf("%w %q without a document base URL", errNoDocumentBase, ref)
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("parse base %q: %w", base, err)
	}

	return baseURL.ResolveReference(refURL), nil
}

func urlEncodePost(items []settings.PostItem) string {
	vals := url.Values{}
	for _, it := range items {
		vals.Add(it.Name, it.Value)
	}

	return vals.Encode()
}

// decodeDataURLLimited decodes a data URL while enforcing the same non-negative
// body cap used by file and HTTP resources.
func decodeDataURLLimited(s string, maxBytes int64) ([]byte, string, error) {
	if err := validateBodyLimit(maxBytes); err != nil {
		return nil, "", err
	}

	rest := strings.TrimPrefix(s, "data:")

	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return nil, "", errMalformedDataURL
	}

	meta, data := rest[:comma], rest[comma+1:]
	ctype := "text/plain"

	if meta != "" {
		parts := strings.Split(meta, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "base64") {
				dec, err := decodeBase64Limited(data, maxBytes)
				if err != nil {
					return nil, "", err
				}

				return dec, ctype, nil
			}

			if !strings.Contains(part, "=") {
				ctype = part
			}
		}
	}

	dec, err := unescapeDataLimited(data, maxBytes)
	if err != nil {
		return nil, "", err
	}

	return dec, ctype, nil
}

func decodeBase64Limited(encoded string, maxBytes int64) ([]byte, error) {
	if err := validateBodyLimit(maxBytes); err != nil {
		return nil, err
	}

	// Count first so a large amount of whitespace cannot force an equally
	// large compacted allocation before the body limit is checked.
	compactLen := compactBase64Len(encoded)

	if maxBytes >= 0 {
		decodedLen := base64.StdEncoding.DecodedLen(compactLen)

		if compactLen > 0 && encoded != "" {
			// DecodedLen includes the bytes represented by padding. Account for
			// it so a one-byte data URL is not rejected with a one-byte limit.
			decodedLen -= base64PaddingBytes(encoded)
		}

		if int64(decodedLen) > maxBytes {
			return nil, fmt.Errorf("data URL %w %d", errBodyTooLarge, maxBytes)
		}
	}

	dec, err := base64.StdEncoding.DecodeString(stripBase64Whitespace(encoded, compactLen))
	if err != nil {
		return nil, fmt.Errorf("decode base64 data URL: %w", err)
	}

	if err := checkBodyLimit("data URL", len(dec), maxBytes); err != nil {
		return nil, err
	}

	return dec, nil
}

// compactBase64Len counts the payload bytes of a base64 string, skipping
// the whitespace base64 permits.
func compactBase64Len(encoded string) int {
	count := 0

	for i := range len(encoded) {
		if !isBase64Space(encoded[i]) {
			count++
		}
	}

	return count
}

// base64PaddingBytes counts the trailing '=' padding characters (up to two,
// past any whitespace) of a base64 string.
func base64PaddingBytes(encoded string) int {
	seen := 0

	for i := len(encoded) - 1; i >= 0 && seen < 2; i-- {
		switch encoded[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case '=':
			seen++
		default:
			seen = 2
		}
	}

	return seen
}

// stripBase64Whitespace returns s without its whitespace, grown to capacity.
func stripBase64Whitespace(encoded string, capacity int) string {
	var buf strings.Builder

	buf.Grow(capacity)

	for i := range len(encoded) {
		if !isBase64Space(encoded[i]) {
			buf.WriteByte(encoded[i])
		}
	}

	return buf.String()
}

func isBase64Space(b byte) bool {
	return b == '\n' || b == '\r' || b == ' ' || b == '\t'
}

func unescapeDataLimited(raw string, maxBytes int64) ([]byte, error) {
	if err := validateBodyLimit(maxBytes); err != nil {
		return nil, err
	}

	limit, capacity := unescapeCapacity(raw, maxBytes)

	out := make([]byte, 0, capacity)

	for pos := 0; pos < len(raw); pos++ {
		var cur byte

		switch raw[pos] {
		case '%':
			dec, ok := decodePercentEscape(raw, pos)
			if !ok {
				return nil, errInvalidDataURL
			}

			cur = dec
			pos += 2
		case '+':
			cur = ' '
		default:
			cur = raw[pos]
		}

		if limit >= 0 && int64(len(out)) >= limit {
			return nil, fmt.Errorf("data URL %w %d", errBodyTooLarge, limit)
		}

		out = append(out, cur)
	}

	return out, nil
}

// unescapeCapacity derives the decoded-body limit and the preallocation
// hint for a data URL payload. The limit is only a capacity hint when the
// source is short: percent escapes and '+' can make the decoded body
// shorter, never longer, than the source string.
func unescapeCapacity(raw string, maxBytes int64) (int64, int) {
	if maxBytes >= 0 && int64(len(raw)) < maxBytes {
		maxBytes = int64(len(raw))
	}

	capacity := len(raw)
	if maxBytes >= 0 && int64(capacity) > maxBytes {
		capacity = int(maxBytes)
	}

	return maxBytes, capacity
}

// decodePercentEscape decodes the %xx escape starting at raw[pos:].
func decodePercentEscape(raw string, pos int) (byte, bool) {
	if pos+2 >= len(raw) {
		return 0, false
	}

	hiNib, okHi := fromHex(raw[pos+1])
	loNib, okLo := fromHex(raw[pos+2])

	if !okHi || !okLo {
		return 0, false
	}

	return hiNib<<hexNibbleBits | loNib, true
}

func fromHex(hexChar byte) (byte, bool) {
	switch {
	case hexChar >= '0' && hexChar <= '9':
		return hexChar - '0', true
	case hexChar >= 'a' && hexChar <= 'f':
		return hexChar - 'a' + hexLetterOffset, true
	case hexChar >= 'A' && hexChar <= 'F':
		return hexChar - 'A' + hexLetterOffset, true
	default:
		return 0, false
	}
}

func checkBodyLimit(source string, length int, maxBytes int64) error {
	if err := validateBodyLimit(maxBytes); err != nil {
		return err
	}

	if int64(length) > maxBytes {
		return fmt.Errorf("%s %w %d", source, errBodyTooLarge, maxBytes)
	}

	return nil
}
