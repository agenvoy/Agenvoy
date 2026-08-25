package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	mcp_auth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

const (
	oauthKeyPrefix       = "MCP_OAUTH_"
	oauthClientKeyPrefix = "MCP_OAUTH_CLIENT_"
	oauthCallbackPath    = "/callback"
	oauthCallbackAddr    = "localhost:17988"
	oauthClientName      = "Agenvoy"

	DefaultRedirectURI = "http://" + oauthCallbackAddr + oauthCallbackPath
)

type oauthRecord struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	TokenURL     string    `json:"token_url"`
	AuthStyle    int       `json:"auth_style"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

type oauthClient struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	Dynamic      bool   `json:"dynamic,omitempty"`
}

func normalizeKeyName(name string) string {
	var sb strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(name)) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

func oauthKey(name string) string {
	return oauthKeyPrefix + normalizeKeyName(name)
}

func oauthClientKey(name string) string {
	return oauthClientKeyPrefix + normalizeKeyName(name)
}

func SaveOAuthClient(name, clientID, clientSecret, redirectURI string) error {
	return saveOAuthClient(name, oauthClient{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		RedirectURI:  strings.TrimSpace(redirectURI),
	})
}

func saveOAuthClient(name string, client oauthClient) error {
	raw, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	if err := keychain.Set(oauthClientKey(name), string(raw)); err != nil {
		return fmt.Errorf("keychain.Set: %w", err)
	}
	return nil
}

func loadOAuthClient(name string) (oauthClient, bool) {
	raw := strings.TrimSpace(keychain.Get(oauthClientKey(name)))
	if raw == "" {
		return oauthClient{}, false
	}
	var client oauthClient
	if err := json.Unmarshal([]byte(raw), &client); err != nil {
		slog.Debug("mcp oauth client unmarshal",
			slog.String("server", name),
			slog.String("error", err.Error()))
		return oauthClient{}, false
	}
	if client.ClientID == "" {
		return oauthClient{}, false
	}
	return client, true
}

func loadOAuth(name string) (oauthRecord, bool) {
	raw := strings.TrimSpace(keychain.Get(oauthKey(name)))
	if raw == "" {
		return oauthRecord{}, false
	}
	var rec oauthRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		slog.Debug("mcp oauth record unmarshal",
			slog.String("server", name),
			slog.String("error", err.Error()))
		return oauthRecord{}, false
	}
	if rec.AccessToken == "" || rec.TokenURL == "" {
		return oauthRecord{}, false
	}
	return rec, true
}

func saveOAuth(name string, rec oauthRecord) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	if err := keychain.Set(oauthKey(name), string(raw)); err != nil {
		return fmt.Errorf("keychain.Set: %w", err)
	}
	return nil
}

func ClearOAuth(name string) error {
	for _, key := range []string{oauthKey(name), oauthClientKey(name)} {
		if keychain.Get(key) == "" {
			continue
		}
		if err := keychain.Delete(key); err != nil {
			return fmt.Errorf("keychain.Delete %s: %w", key, err)
		}
	}
	return nil
}

func HasOAuth(name string) bool {
	_, ok := loadOAuth(name)
	return ok
}

func (r oauthRecord) config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     r.ClientID,
		ClientSecret: r.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  r.TokenURL,
			AuthStyle: oauth2.AuthStyle(r.AuthStyle),
		},
	}
}

func (r oauthRecord) token() *oauth2.Token {
	tokenType := r.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return &oauth2.Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenType:    tokenType,
		Expiry:       r.Expiry,
	}
}

func (r *oauthRecord) apply(token *oauth2.Token) {
	r.AccessToken = token.AccessToken
	r.TokenType = token.TokenType
	r.Expiry = token.Expiry
	if token.RefreshToken != "" {
		r.RefreshToken = token.RefreshToken
	}
}

type oauthHandler struct {
	name string
	mu   sync.Mutex
	src  oauth2.TokenSource
}

func (h *oauthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.src != nil {
		return h.src, nil
	}
	rec, ok := loadOAuth(h.name)
	if !ok {
		return nil, nil
	}
	h.src = &persistTokenSource{
		name: h.name,
		rec:  rec,
		base: rec.config().TokenSource(ctx, rec.token()),
	}
	return h.src, nil
}

func (h *oauthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	detail := go_pkg_utils.TruncateString(strings.TrimSpace(string(raw)), 512)
	if detail != "" {
		detail = " · " + detail
	}

	challenges, _ := oauthex.ParseWWWAuthenticate(resp.Header.Values("WWW-Authenticate"))
	challengeErr := ""
	for _, challenge := range challenges {
		if challenge.Scheme == "bearer" && challenge.Params["error"] != "" {
			challengeErr = challenge.Params["error"]
			break
		}
	}

	switch {
	case challengeErr == "insufficient_scope":
		return fmt.Errorf("token is missing required scopes, re-authenticate with /mcp → %s → login%s", h.name, detail)
	case resp.StatusCode == http.StatusForbidden:
		slog.Warn("mcp oauth forbidden",
			slog.String("server", h.name),
			slog.String("detail", detail))
		return nil
	case HasOAuth(h.name):
		return fmt.Errorf("oauth token rejected (%s), re-authenticate with /mcp → %s → login%s", resp.Status, h.name, detail)
	}
	return fmt.Errorf("oauth login required, run /mcp → %s → login%s", h.name, detail)
}

type persistTokenSource struct {
	name string
	mu   sync.Mutex
	rec  oauthRecord
	base oauth2.TokenSource
}

func (s *persistTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.base.Token()
	if err != nil {
		return nil, fmt.Errorf("oauth token: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if token.AccessToken == s.rec.AccessToken {
		return token, nil
	}
	s.rec.apply(token)
	if err := saveOAuth(s.name, s.rec); err != nil {
		slog.Warn("mcp oauth token persist",
			slog.String("server", s.name),
			slog.String("error", err.Error()))
	}
	return token, nil
}

type authCodeResult struct {
	code  string
	state string
	err   error
}

var pendingLogin sync.Map

func SubmitCallback(name, rawURL string) error {
	value, ok := pendingLogin.Load(name)
	if !ok {
		return fmt.Errorf("no oauth login waiting for %q", name)
	}
	pending := value.(chan authCodeResult)

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("parse redirect URL: %w", err)
	}
	query := parsed.Query()

	if failure := query.Get("error"); failure != "" {
		desc := query.Get("error_description")
		deliver(pending, authCodeResult{err: fmt.Errorf("%s: %s", failure, desc)})
		return nil
	}
	code := query.Get("code")
	if code == "" {
		return fmt.Errorf("redirect URL carries no code")
	}
	deliver(pending, authCodeResult{code: code, state: query.Get("state")})
	return nil
}

func deliver(result chan<- authCodeResult, res authCodeResult) {
	select {
	case result <- res:
	default:
	}
}

func Login(ctx context.Context, name string, onURL func(string)) error {
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	server, ok := cfg.Servers[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}
	server = server.Expand()
	if !server.IsHTTP() {
		return fmt.Errorf("server %q is not an HTTP server", name)
	}
	endpoint := strings.TrimSpace(server.URL)

	stored, hasClient := loadOAuthClient(name)
	result := make(chan authCodeResult, 1)
	callback, reuseClient, err := listenCallback(name, stored, hasClient, result)
	if err != nil {
		return err
	}
	defer callback.close()

	pendingLogin.Store(name, result)
	defer pendingLogin.Delete(name)

	recorder := &loginRecorder{base: http.DefaultTransport}
	handlerCfg := &mcp_auth.AuthorizationCodeHandlerConfig{
		RedirectURL: callback.redirectURL,
		Client:      &http.Client{Transport: recorder, Timeout: 60 * time.Second},
		AuthorizationCodeFetcher: func(ctx context.Context, args *mcp_auth.AuthorizationArgs) (*mcp_auth.AuthorizationResult, error) {
			if onURL != nil {
				onURL(offlineAccessURL(args.URL))
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case received := <-result:
				if received.err != nil {
					return nil, received.err
				}
				return &mcp_auth.AuthorizationResult{Code: received.code, State: received.state}, nil
			}
		},
	}
	if reuseClient {
		credentials := &oauthex.ClientCredentials{ClientID: stored.ClientID}
		if stored.ClientSecret != "" {
			credentials.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: stored.ClientSecret}
		}
		handlerCfg.PreregisteredClient = credentials
	} else {
		handlerCfg.DynamicClientRegistrationConfig = &mcp_auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				RedirectURIs:            []string{callback.redirectURL},
				ClientName:              registrationClientName(server.ClientName, endpoint),
				GrantTypes:              []string{"authorization_code", "refresh_token"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
			},
		}
	}

	handler, err := mcp_auth.NewAuthorizationCodeHandler(handlerCfg)
	if err != nil {
		return fmt.Errorf("NewAuthorizationCodeHandler: %w", err)
	}

	session, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "agenvoy", Version: "1.0.0"}, nil).
		Connect(ctx, &mcpsdk.StreamableClientTransport{
			Endpoint:     endpoint,
			HTTPClient:   headerClient(server.Headers),
			OAuthHandler: handler,
		}, nil)
	if err != nil {
		return loginError(err, name, reuseClient)
	}
	defer session.Close()

	if _, err := session.ListTools(ctx, nil); err != nil {
		if source, srcErr := handler.TokenSource(ctx); srcErr != nil || source == nil {
			return loginError(err, name, reuseClient)
		}
		slog.Debug("mcp oauth login tools/list",
			slog.String("server", name),
			slog.String("error", err.Error()))
	}

	source, err := handler.TokenSource(ctx)
	if err != nil {
		return fmt.Errorf("token source: %w", err)
	}
	if source == nil {
		return fmt.Errorf("server %q never requested authorization", name)
	}
	token, err := source.Token()
	if err != nil {
		return fmt.Errorf("token: %w", err)
	}

	clientID, clientSecret := stored.ClientID, stored.ClientSecret
	if !reuseClient {
		registered := recorder.registration()
		if registered.ClientID == "" {
			return fmt.Errorf("registered client id not observed during authorization")
		}
		clientID, clientSecret = registered.ClientID, registered.ClientSecret
		if err := saveOAuthClient(name, oauthClient{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  callback.redirectURL,
			Dynamic:      true,
		}); err != nil {
			return err
		}
	}

	tokenURL := recorder.tokenURL()
	if tokenURL == "" {
		return fmt.Errorf("token endpoint not observed during authorization")
	}

	rec := oauthRecord{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		AuthStyle:    int(oauth2.AuthStyleAutoDetect),
	}
	rec.apply(token)
	return saveOAuth(name, rec)
}

func loginError(err error, name string, reuseClient bool) error {
	if reuseClient {
		return err
	}
	return fmt.Errorf("%w · options: set \"oauth_client_name\" in mcp.json to a name the provider allowlists, open /mcp → %s → client to use a pre-registered client id, or re-add the server with a bearer token", err, name)
}

type callbackServer struct {
	server      *http.Server
	redirectURL string
}

func listenCallback(name string, stored oauthClient, hasClient bool, result chan<- authCodeResult) (*callbackServer, bool, error) {
	if hasClient {
		redirectURL := stored.RedirectURI
		if redirectURL == "" {
			redirectURL = DefaultRedirectURI
		}
		parsed, err := url.Parse(redirectURL)
		if err != nil {
			return nil, false, fmt.Errorf("parse redirect URI %q: %w", redirectURL, err)
		}
		if parsed.Port() == "" {
			return nil, false, fmt.Errorf("redirect URI %q must include a port", redirectURL)
		}
		path := parsed.Path
		if path == "" {
			path = oauthCallbackPath
		}
		listener, err := net.Listen("tcp", parsed.Host)
		if err == nil {
			return serveCallback(name, listener, redirectURL, path, result), true, nil
		}
		if !stored.Dynamic {
			return nil, false, fmt.Errorf("net.Listen %s: %w", parsed.Host, err)
		}
		slog.Debug("mcp oauth cached callback port busy, registering a new client",
			slog.String("server", name),
			slog.String("redirect", redirectURL))
	}

	if listener, err := net.Listen("tcp", oauthCallbackAddr); err == nil {
		return serveCallback(name, listener, DefaultRedirectURI, oauthCallbackPath, result), false, nil
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, false, fmt.Errorf("net.Listen: %w", err)
	}
	redirectURL := fmt.Sprintf("http://localhost:%d%s", listener.Addr().(*net.TCPAddr).Port, oauthCallbackPath)
	return serveCallback(name, listener, redirectURL, oauthCallbackPath, result), false, nil
}

func serveCallback(name string, listener net.Listener, redirectURL, path string, result chan<- authCodeResult) *callbackServer {
	server := &http.Server{
		Handler:           http.HandlerFunc(oauthCallbackHandler(path, result)),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Warn("mcp oauth callback server",
				slog.String("server", name),
				slog.String("error", err.Error()))
		}
	}()
	return &callbackServer{server: server, redirectURL: redirectURL}
}

func (c *callbackServer) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.server.Shutdown(ctx)
}

func oauthCallbackHandler(callbackPath string, result chan<- authCodeResult) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()

		fail := func(text string, err error) {
			fmt.Fprint(w, text)
			deliver(result, authCodeResult{err: err})
		}

		if failure := query.Get("error"); failure != "" {
			desc := query.Get("error_description")
			fail(fmt.Sprintf("Authorization failed: %s: %s", failure, desc), fmt.Errorf("%s: %s", failure, desc))
			return
		}
		code := query.Get("code")
		if code == "" {
			fail("Authorization failed: missing code", fmt.Errorf("missing code"))
			return
		}

		fmt.Fprint(w, "Authorization successful — you can close this tab.")
		deliver(result, authCodeResult{code: code, state: query.Get("state")})
	}
}

type loginRecorder struct {
	base          http.RoundTripper
	mu            sync.Mutex
	client        oauthClient
	tokenEndpoint string
}

func (r *loginRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, err
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return resp, nil
	}

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", req.URL, readErr)
	}
	raw = normalizeIssuers(raw)
	r.observe(req, raw)

	resp.Body = io.NopCloser(bytes.NewReader(raw))
	resp.ContentLength = int64(len(raw))
	return resp, nil
}

func (r *loginRecorder) observe(req *http.Request, raw []byte) {
	var dic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &dic); err != nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if value, ok := dic["token_endpoint"]; ok {
		var endpoint string
		if json.Unmarshal(value, &endpoint) == nil && endpoint != "" {
			r.tokenEndpoint = endpoint
		}
		return
	}
	if req.Method != http.MethodPost {
		return
	}
	if _, ok := dic["access_token"]; ok {
		if r.tokenEndpoint == "" {
			r.tokenEndpoint = req.URL.String()
		}
		return
	}
	if _, ok := dic["client_id"]; !ok {
		return
	}
	var registered struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(raw, &registered); err != nil {
		return
	}
	r.client = oauthClient{ClientID: registered.ClientID, ClientSecret: registered.ClientSecret}
}

func (r *loginRecorder) registration() oauthClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.client
}

func (r *loginRecorder) tokenURL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tokenEndpoint
}

func normalizeIssuers(raw []byte) []byte {
	var dic map[string]any
	if err := json.Unmarshal(raw, &dic); err != nil {
		return raw
	}
	list, ok := dic["authorization_servers"].([]any)
	if !ok {
		return raw
	}

	changed := false
	for i, entry := range list {
		issuer, ok := entry.(string)
		if !ok {
			continue
		}
		if trimmed := trimIssuerSlash(issuer); trimmed != issuer {
			list[i] = trimmed
			changed = true
		}
	}
	if !changed {
		return raw
	}

	out, err := json.Marshal(dic)
	if err != nil {
		return raw
	}
	return out
}

func trimIssuerSlash(issuer string) string {
	parsed, err := url.Parse(issuer)
	if err != nil {
		return issuer
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func offlineAccessURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "accounts.google.com") {
		return raw
	}
	query := parsed.Query()
	query.Set("access_type", "offline")
	query.Set("prompt", "consent")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func registrationClientName(configured, serverURL string) string {
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured
	}
	if parsed, err := url.Parse(serverURL); err == nil && strings.HasSuffix(strings.ToLower(parsed.Hostname()), "figma.com") {
		return "Claude Code"
	}
	return oauthClientName
}
