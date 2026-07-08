package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
	"sigs.k8s.io/yaml"
)

const (
	defaultServer       = "http://localhost:8080"
	defaultOutput       = "table"
	defaultPollInterval = 2 * time.Second
	defaultTimeout      = 30 * time.Minute
)

type cliConfig struct {
	Server                string
	Username              string
	Password              string
	Token                 string
	ConfigPath            string
	Output                string
	Timeout               time.Duration
	PollInterval          time.Duration
	InsecureSkipTLSVerify bool
	InsecureExplicitlySet bool
	CACert                string
	DefaultProject        string
	PasswordStdin         bool
}

type rawConfig struct {
	Server        string
	Username      string
	Password      string
	Token         string
	SSLVerify     *bool
	CACert        string
	DefaultOutput string
	DefaultPoll   string
	DefaultTO     string
	DefaultProj   string
}

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type apiError struct {
	StatusCode int
	Message    string
}

func (e apiError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("api request failed with status %d", e.StatusCode)
	}
	return e.Message
}

type workflowFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type workflow struct {
	ID             int            `json:"id,omitempty"`
	Name           string         `json:"name"`
	YAML           string         `json:"yaml,omitempty"`
	DefinitionKind string         `json:"definition_kind,omitempty"`
	Files          []workflowFile `json:"files,omitempty"`
	ProjectID      *int           `json:"project_id,omitempty"`
	SourcePath     string         `json:"source_path,omitempty"`
	SourceKind     string         `json:"source_kind,omitempty"`
	SourceRoot     string         `json:"source_root,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	UpdatedAt      string         `json:"updated_at,omitempty"`
}

type project struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Branch       string `json:"branch"`
	SyncStatus   string `json:"sync_status"`
	SyncError    string `json:"sync_error,omitempty"`
	LastSyncedAt string `json:"last_synced_at,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type run struct {
	ID                int    `json:"id"`
	RunID             string `json:"run_id"`
	WorkflowName      string `json:"workflow_name"`
	Status            string `json:"status"`
	VarsJSON          string `json:"vars_json,omitempty"`
	VolumesJSON       string `json:"volumes_json,omitempty"`
	SecretVolumesJSON string `json:"secret_volumes_json,omitempty"`
	StartedAt         string `json:"started_at,omitempty"`
	CompletedAt       string `json:"completed_at,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	Steps             []step `json:"steps,omitempty"`
}

type step struct {
	RunID        string `json:"run_id"`
	ForkID       string `json:"fork_id,omitempty"`
	WorkflowName string `json:"workflow_name"`
	StepName     string `json:"step_name"`
	StepType     string `json:"step_type,omitempty"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type pvcMount struct {
	Name      string `json:"name"`
	PVC       string `json:"pvc"`
	MountPath string `json:"mount_path"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

type secretMount struct {
	Name      string `json:"name"`
	Secret    string `json:"secret"`
	MountPath string `json:"mount_path"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

type runCreateRequest struct {
	WorkflowName       string            `json:"workflow_name"`
	WorkflowEntrypoint string            `json:"workflow_entrypoint,omitempty"`
	Vars               map[string]string `json:"vars,omitempty"`
	Debug              bool              `json:"debug,omitempty"`
	Volumes            []pvcMount        `json:"volumes,omitempty"`
	SecretVolumes      []secretMount     `json:"secret_volumes,omitempty"`
}

type waitResult struct {
	Status         string  `json:"status"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	Project        any     `json:"project,omitempty"`
	Run            any     `json:"run,omitempty"`
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	if err := runCLI(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}

func exitCode(err error) int {
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return 3
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return 124
	}
	return 1
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	cfg, rest, err := resolveConfig(args, stdin)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(stdout)
		return nil
	}
	if err != nil {
		return err
	}
	if len(rest) == 0 || rest[0] == "help" {
		printUsage(stdout)
		return nil
	}
	client, err := newAPIClient(cfg)
	if err != nil {
		return err
	}

	switch rest[0] {
	case "auth":
		return runAuth(ctx, client, cfg, rest[1:], stdout, stdin)
	case "health":
		return runHealth(ctx, client, cfg, stdout)
	case "projects":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runProjects(ctx, client, cfg, rest[1:], stdout)
	case "workflows":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runWorkflows(ctx, client, cfg, rest[1:], stdout)
	case "runs":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runRuns(ctx, client, cfg, rest[1:], stdout)
	case "pvcs":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runSimpleList(ctx, client, cfg, rest[1:], "/pvcs", stdout)
	case "secrets":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runSimpleList(ctx, client, cfg, rest[1:], "/secrets", stdout)
	case "preferences":
		if err := ensureAuth(ctx, client, cfg, stdin); err != nil {
			return err
		}
		return runPreferences(ctx, client, cfg, rest[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  markovd-cli [global flags] auth login [--username USER] [--password PASS | --password-stdin] [--save]
  markovd-cli [global flags] health
  markovd-cli [global flags] projects list|get|create|delete|sync|files|import ...
  markovd-cli [global flags] workflows list|get|create|update|delete|validate|diagram ...
  markovd-cli [global flags] runs list|get|create|wait|cancel|delete|logs ...
  markovd-cli [global flags] pvcs list
  markovd-cli [global flags] secrets list
  markovd-cli [global flags] preferences get|set

Global flags:
  --server URL
  --username USER
  --password PASS
  --password-stdin
  --token TOKEN
  --config PATH
  --output table|json|yaml
  --timeout DURATION
  --poll-interval DURATION
  --insecure-skip-tls-verify
  --ca-cert PATH`)
}

func resolveConfig(args []string, stdin io.Reader) (cliConfig, []string, error) {
	cfg := cliConfig{
		Server:       defaultServer,
		Output:       defaultOutput,
		Timeout:      defaultTimeout,
		PollInterval: defaultPollInterval,
	}

	global, rest, err := parseGlobalFlags(args)
	if err != nil {
		return cfg, nil, err
	}

	for _, path := range defaultConfigPaths() {
		mergeRawConfig(&cfg, readConfigIfExists(path))
	}
	if global.ConfigPath != "" {
		raw, err := readConfig(global.ConfigPath)
		if err != nil {
			return cfg, nil, err
		}
		mergeRawConfig(&cfg, raw)
		cfg.ConfigPath = global.ConfigPath
	}
	applyEnv(&cfg)
	applyFlagOverrides(&cfg, global)
	if cfg.PasswordStdin {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return cfg, nil, err
		}
		cfg.Password = strings.TrimRight(string(b), "\r\n")
	}
	return cfg, rest, nil
}

func parseGlobalFlags(args []string) (cliConfig, []string, error) {
	fs := flag.NewFlagSet("markovd-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var cfg cliConfig
	fs.StringVar(&cfg.Server, "server", "", "")
	fs.StringVar(&cfg.Username, "username", "", "")
	fs.StringVar(&cfg.Password, "password", "", "")
	fs.BoolVar(&cfg.PasswordStdin, "password-stdin", false, "")
	fs.StringVar(&cfg.Token, "token", "", "")
	fs.StringVar(&cfg.ConfigPath, "config", "", "")
	fs.StringVar(&cfg.Output, "output", "", "")
	var timeout, poll string
	fs.StringVar(&timeout, "timeout", "", "")
	fs.StringVar(&poll, "poll-interval", "", "")
	fs.BoolVar(&cfg.InsecureSkipTLSVerify, "insecure-skip-tls-verify", false, "")
	fs.StringVar(&cfg.CACert, "ca-cert", "", "")
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "insecure-skip-tls-verify" {
			cfg.InsecureExplicitlySet = true
		}
	})
	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return cfg, nil, fmt.Errorf("invalid --timeout: %w", err)
		}
		cfg.Timeout = d
	}
	if poll != "" {
		d, err := time.ParseDuration(poll)
		if err != nil {
			return cfg, nil, fmt.Errorf("invalid --poll-interval: %w", err)
		}
		cfg.PollInterval = d
	}
	return cfg, fs.Args(), nil
}

func parseCommandFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var flagArgs []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flagArgs = append(flagArgs, arg)
		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		if isBoolFlag(f) {
			continue
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("flag needs an argument: --%s", name)
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}
	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}
	return append(positionals, fs.Args()...), nil
}

func isBoolFlag(f *flag.Flag) bool {
	getter, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && getter.IsBoolFlag()
}

func defaultConfigPaths() []string {
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".config", "markovd", "cli-config.toml"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "markovd", "cli-config.toml"))
	}
	paths = append(paths, ".markovd-cli-config.toml")
	return paths
}

func applyFlagOverrides(cfg *cliConfig, flags cliConfig) {
	if flags.Server != "" {
		cfg.Server = flags.Server
	}
	if flags.Username != "" {
		cfg.Username = flags.Username
	}
	if flags.Password != "" {
		cfg.Password = flags.Password
	}
	if flags.PasswordStdin {
		cfg.PasswordStdin = true
	}
	if flags.Token != "" {
		cfg.Token = flags.Token
	}
	if flags.Output != "" {
		cfg.Output = flags.Output
	}
	if flags.Timeout != 0 {
		cfg.Timeout = flags.Timeout
	}
	if flags.PollInterval != 0 {
		cfg.PollInterval = flags.PollInterval
	}
	if flags.InsecureExplicitlySet {
		cfg.InsecureSkipTLSVerify = flags.InsecureSkipTLSVerify
		cfg.InsecureExplicitlySet = true
	}
	if flags.CACert != "" {
		cfg.CACert = flags.CACert
	}
}

func applyEnv(cfg *cliConfig) {
	if v := os.Getenv("MARKOVD_URL"); v != "" {
		cfg.Server = v
	}
	if v := os.Getenv("MARKOVD_USERNAME"); v != "" {
		cfg.Username = v
	}
	if v := os.Getenv("MARKOVD_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("MARKOVD_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("MARKOVD_OUTPUT"); v != "" {
		cfg.Output = v
	}
	if v := os.Getenv("MARKOVD_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv("MARKOVD_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.PollInterval = d
		}
	}
	if v := os.Getenv("MARKOVD_INSECURE_SKIP_TLS_VERIFY"); parseBool(v) {
		cfg.InsecureSkipTLSVerify = true
		cfg.InsecureExplicitlySet = true
	}
	if v := os.Getenv("MARKOVD_CA_CERT"); v != "" {
		cfg.CACert = v
	}
}

func readConfigIfExists(path string) rawConfig {
	raw, err := readConfig(path)
	if err != nil {
		return rawConfig{}
	}
	return raw
}

func readConfig(path string) (rawConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return rawConfig{}, err
	}
	return parseConfigTOML(string(b))
}

func parseConfigTOML(s string) (rawConfig, error) {
	var cfg rawConfig
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return cfg, fmt.Errorf("invalid config line %q", line)
		}
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		switch section {
		case "":
			switch key {
			case "server":
				cfg.Server = parseString(val)
			case "username":
				cfg.Username = parseString(val)
			case "password":
				cfg.Password = parseString(val)
			case "token":
				cfg.Token = parseString(val)
			case "ssl_verify":
				b := parseBool(val)
				cfg.SSLVerify = &b
			case "ca_cert":
				cfg.CACert = parseString(val)
			}
		case "defaults":
			switch key {
			case "output":
				cfg.DefaultOutput = parseString(val)
			case "poll_interval":
				cfg.DefaultPoll = parseString(val)
			case "timeout":
				cfg.DefaultTO = parseString(val)
			case "project":
				cfg.DefaultProj = parseString(val)
			}
		}
	}
	return cfg, scanner.Err()
}

func stripComment(line string) string {
	inQuote := false
	var quote rune
	for i, r := range line {
		if (r == '"' || r == '\'') && (i == 0 || line[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quote = r
			} else if quote == r {
				inQuote = false
			}
		}
		if r == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func parseString(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(parseString(v))) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func mergeRawConfig(cfg *cliConfig, raw rawConfig) {
	if raw.Server != "" {
		cfg.Server = raw.Server
	}
	if raw.Username != "" {
		cfg.Username = raw.Username
	}
	if raw.Password != "" {
		cfg.Password = raw.Password
	}
	if raw.Token != "" {
		cfg.Token = raw.Token
	}
	if raw.SSLVerify != nil {
		cfg.InsecureSkipTLSVerify = !*raw.SSLVerify
	}
	if raw.CACert != "" {
		cfg.CACert = raw.CACert
	}
	if raw.DefaultOutput != "" {
		cfg.Output = raw.DefaultOutput
	}
	if raw.DefaultPoll != "" {
		if d, err := time.ParseDuration(raw.DefaultPoll); err == nil {
			cfg.PollInterval = d
		}
	}
	if raw.DefaultTO != "" {
		if d, err := time.ParseDuration(raw.DefaultTO); err == nil {
			cfg.Timeout = d
		}
	}
	if raw.DefaultProj != "" {
		cfg.DefaultProject = raw.DefaultProj
	}
}

func newAPIClient(cfg cliConfig) (*apiClient, error) {
	base := strings.TrimRight(cfg.Server, "/")
	if _, err := url.ParseRequestURI(base); err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipTLSVerify} //nolint:gosec
	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no PEM certificates found in %s", cfg.CACert)
		}
		tlsConfig.RootCAs = pool
	}
	return &apiClient{
		baseURL: base,
		token:   cfg.Token,
		client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		}},
	}, nil
}

func (c *apiClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		var errBody map[string]any
		if json.Unmarshal(respBody, &errBody) == nil {
			if v, ok := errBody["error"].(string); ok {
				msg = v
			}
		}
		return apiError{StatusCode: resp.StatusCode, Message: msg}
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

func ensureAuth(ctx context.Context, c *apiClient, cfg cliConfig, stdin io.Reader) error {
	if c.token != "" {
		return nil
	}
	if cfg.Username == "" {
		return errors.New("authentication required: provide --token or --username/--password")
	}
	pass := cfg.Password
	if pass == "" {
		if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			fmt.Fprintf(os.Stderr, "Password for %s: ", cfg.Username)
			b, err := term.ReadPassword(int(f.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return err
			}
			pass = string(b)
		}
	}
	if pass == "" {
		return errors.New("authentication required: provide password, MARKOVD_PASSWORD, or --password-stdin")
	}
	token, err := login(ctx, c, cfg.Username, pass)
	if err != nil {
		return err
	}
	c.token = token
	return nil
}

func login(ctx context.Context, c *apiClient, username, password string) (string, error) {
	var resp struct {
		Token string `json:"token"`
	}
	err := c.do(ctx, http.MethodPost, "/auth/login", map[string]string{"username": username, "password": password}, &resp)
	return resp.Token, err
}

func runAuth(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer, stdin io.Reader) error {
	if len(args) == 0 || args[0] != "login" {
		return errors.New("usage: auth login [--username USER] [--password PASS | --password-stdin] [--save]")
	}
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	username := cfg.Username
	password := cfg.Password
	passwordStdin := cfg.PasswordStdin
	save := false
	fs.StringVar(&username, "username", username, "")
	fs.StringVar(&password, "password", password, "")
	fs.BoolVar(&passwordStdin, "password-stdin", passwordStdin, "")
	fs.BoolVar(&save, "save", false, "")
	if _, err := parseCommandFlags(fs, args[1:]); err != nil {
		return err
	}
	if username == "" {
		return errors.New("username required")
	}
	if passwordStdin && password == "" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		password = strings.TrimRight(string(b), "\r\n")
	}
	if password == "" {
		return errors.New("password required")
	}
	token, err := login(ctx, c, username, password)
	if err != nil {
		return err
	}
	c.token = token
	if save {
		cfg.Username = username
		cfg.Token = token
		cfg.Password = ""
		if err := saveUserConfig(cfg); err != nil {
			return err
		}
	}
	return writeOutput(stdout, cfg.Output, map[string]string{"token": token})
}

func saveUserConfig(cfg cliConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".config", "markovd", "cli-config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "server = %q\n", cfg.Server)
	if cfg.Username != "" {
		fmt.Fprintf(&b, "username = %q\n", cfg.Username)
	}
	if cfg.Token != "" {
		fmt.Fprintf(&b, "token = %q\n", cfg.Token)
	}
	fmt.Fprintf(&b, "ssl_verify = %t\n", !cfg.InsecureSkipTLSVerify)
	if cfg.CACert != "" {
		fmt.Fprintf(&b, "ca_cert = %q\n", cfg.CACert)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func runHealth(ctx context.Context, c *apiClient, cfg cliConfig, stdout io.Writer) error {
	var out any
	if err := c.do(ctx, http.MethodGet, "/health", nil, &out); err != nil {
		return err
	}
	return writeOutput(stdout, cfg.Output, out)
}

func runProjects(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("projects command required")
	}
	switch args[0] {
	case "list":
		var out []project
		if err := c.do(ctx, http.MethodGet, "/projects", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "get":
		if len(args) < 2 {
			return errors.New("usage: projects get <id-or-name>")
		}
		p, err := resolveProject(ctx, c, args[1])
		if err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, p)
	case "create":
		fs := flag.NewFlagSet("projects create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		name, repoURL, branch := "", "", "main"
		fs.StringVar(&name, "name", "", "")
		fs.StringVar(&repoURL, "url", "", "")
		fs.StringVar(&branch, "branch", "main", "")
		if _, err := parseCommandFlags(fs, args[1:]); err != nil {
			return err
		}
		if name == "" || repoURL == "" {
			return errors.New("projects create requires --name and --url")
		}
		var out project
		err := c.do(ctx, http.MethodPost, "/projects", map[string]string{"name": name, "url": repoURL, "branch": branch}, &out)
		if err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "delete":
		if len(args) < 2 {
			return errors.New("usage: projects delete <id-or-name>")
		}
		p, err := resolveProject(ctx, c, args[1])
		if err != nil {
			return err
		}
		if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/projects/%d", p.ID), nil, nil); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, map[string]any{"deleted": true, "project": p})
	case "sync":
		return runProjectSync(ctx, c, cfg, args[1:], stdout)
	case "files":
		if len(args) < 2 {
			return errors.New("usage: projects files <id-or-name>")
		}
		p, err := resolveProject(ctx, c, args[1])
		if err != nil {
			return err
		}
		var out any
		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/projects/%d/files", p.ID), nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "import":
		if len(args) < 3 {
			return errors.New("usage: projects import <id-or-name> <path> [--kind file|directory]")
		}
		fs := flag.NewFlagSet("projects import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		kind := "file"
		fs.StringVar(&kind, "kind", "file", "")
		if _, err := parseCommandFlags(fs, args[3:]); err != nil {
			return err
		}
		p, err := resolveProject(ctx, c, args[1])
		if err != nil {
			return err
		}
		body := map[string]any{"definitions": []map[string]string{{"path": args[2], "kind": kind}}}
		var out any
		if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/projects/%d/import", p.ID), body, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	default:
		return fmt.Errorf("unknown projects command %q", args[0])
	}
}

func runProjectSync(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("projects sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	wait := false
	fs.BoolVar(&wait, "wait", false, "")
	positionals, err := parseCommandFlags(fs, args)
	if err != nil {
		return err
	}
	if len(positionals) < 1 {
		return errors.New("usage: projects sync <id-or-name> [--wait]")
	}
	p, err := resolveProject(ctx, c, positionals[0])
	if err != nil {
		return err
	}
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/projects/%d/sync", p.ID), nil, &p); err != nil {
		return err
	}
	if !wait {
		return writeOutput(stdout, cfg.Output, p)
	}
	start := time.Now()
	final, err := waitProject(ctx, c, cfg, p.ID, stdout)
	if err != nil {
		return err
	}
	if cfg.Output == "json" || cfg.Output == "yaml" {
		return writeOutput(stdout, cfg.Output, waitResult{Status: final.SyncStatus, ElapsedSeconds: time.Since(start).Seconds(), Project: final})
	}
	fmt.Fprintf(stdout, "project %d %s %s in %.1fs\n", final.ID, final.Name, final.SyncStatus, time.Since(start).Seconds())
	return nil
}

func waitProject(ctx context.Context, c *apiClient, cfg cliConfig, id int, stdout io.Writer) (project, error) {
	waitCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	last := ""
	for {
		var p project
		if err := c.do(waitCtx, http.MethodGet, fmt.Sprintf("/projects/%d", id), nil, &p); err != nil {
			return p, err
		}
		if cfg.Output == "table" && p.SyncStatus != last {
			fmt.Fprintf(stdout, "project %d %s %s\n", p.ID, p.Name, p.SyncStatus)
			last = p.SyncStatus
		}
		switch p.SyncStatus {
		case "synced":
			return p, nil
		case "error":
			if p.SyncError != "" {
				return p, fmt.Errorf("project sync failed: %s", p.SyncError)
			}
			return p, errors.New("project sync failed")
		}
		select {
		case <-waitCtx.Done():
			return p, waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func resolveProject(ctx context.Context, c *apiClient, ident string) (project, error) {
	if id, err := strconv.Atoi(ident); err == nil {
		var p project
		return p, c.do(ctx, http.MethodGet, fmt.Sprintf("/projects/%d", id), nil, &p)
	}
	var projects []project
	if err := c.do(ctx, http.MethodGet, "/projects", nil, &projects); err != nil {
		return project{}, err
	}
	var matches []project
	for _, p := range projects {
		if p.Name == ident {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return project{}, fmt.Errorf("project %q not found", ident)
	}
	if len(matches) > 1 {
		return project{}, fmt.Errorf("project %q is ambiguous", ident)
	}
	return matches[0], nil
}

func runWorkflows(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("workflows command required")
	}
	switch args[0] {
	case "list":
		var out []workflow
		if err := c.do(ctx, http.MethodGet, "/workflows", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "get":
		if len(args) < 2 {
			return errors.New("usage: workflows get <name>")
		}
		var out workflow
		if err := c.do(ctx, http.MethodGet, "/workflows/"+url.PathEscape(args[1]), nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "create", "update", "validate":
		return runWorkflowUpsert(ctx, c, cfg, args, stdout)
	case "delete":
		if len(args) < 2 {
			return errors.New("usage: workflows delete <name>")
		}
		if err := c.do(ctx, http.MethodDelete, "/workflows/"+url.PathEscape(args[1]), nil, nil); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, map[string]any{"deleted": true, "name": args[1]})
	case "diagram":
		if len(args) < 2 {
			return errors.New("usage: workflows diagram <name>")
		}
		var out any
		if err := c.do(ctx, http.MethodGet, "/workflows/"+url.PathEscape(args[1])+"/diagram", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	default:
		return fmt.Errorf("unknown workflows command %q", args[0])
	}
}

func runWorkflowUpsert(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	cmd := args[0]
	fs := flag.NewFlagSet("workflows "+cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name, filePath, dirPath := "", "", ""
	fs.StringVar(&name, "name", "", "")
	fs.StringVar(&filePath, "file", "", "")
	fs.StringVar(&dirPath, "directory", "", "")
	parseArgs := args[1:]
	if cmd == "update" && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		name = args[1]
		parseArgs = args[2:]
	}
	positionals, err := parseCommandFlags(fs, parseArgs)
	if err != nil {
		return err
	}
	if name == "" && len(positionals) > 0 {
		name = positionals[0]
	}
	if cmd != "validate" && name == "" {
		return errors.New("workflow name required")
	}
	if (filePath == "") == (dirPath == "") {
		return errors.New("provide exactly one of --file or --directory")
	}
	kind := "file"
	var files []workflowFile
	if filePath != "" {
		files, err = readWorkflowFile(filePath)
	} else {
		kind = "directory"
		files, err = readWorkflowDirectory(dirPath)
	}
	if err != nil {
		return err
	}
	body := map[string]any{"name": name, "definition_kind": kind, "files": files}
	var out any
	switch cmd {
	case "create":
		err = c.do(ctx, http.MethodPost, "/workflows", body, &out)
	case "update":
		err = c.do(ctx, http.MethodPut, "/workflows/"+url.PathEscape(name), body, &out)
	case "validate":
		err = c.do(ctx, http.MethodPost, "/workflows/validate", body, &out)
	}
	if err != nil {
		return err
	}
	return writeOutput(stdout, cfg.Output, out)
}

func readWorkflowFile(path string) ([]workflowFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return []workflowFile{{Path: filepath.Base(path), Content: string(b)}}, nil
}

func readWorkflowDirectory(root string) ([]workflowFile, error) {
	var files []workflowFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, workflowFile{Path: rel, Content: string(b)})
		return nil
	})
	return files, err
}

func runRuns(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("runs command required")
	}
	switch args[0] {
	case "list":
		var out []run
		if err := c.do(ctx, http.MethodGet, "/runs", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "get":
		if len(args) < 2 {
			return errors.New("usage: runs get <run-id>")
		}
		var out run
		if err := c.do(ctx, http.MethodGet, "/runs/"+url.PathEscape(args[1]), nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "create":
		return runCreateRun(ctx, c, cfg, args[1:], stdout)
	case "wait":
		if len(args) < 2 {
			return errors.New("usage: runs wait <run-id>")
		}
		start := time.Now()
		final, err := waitRun(ctx, c, cfg, args[1], stdout)
		if err != nil {
			return err
		}
		if cfg.Output == "json" || cfg.Output == "yaml" {
			return writeOutput(stdout, cfg.Output, waitResult{Status: final.Status, ElapsedSeconds: time.Since(start).Seconds(), Run: final})
		}
		fmt.Fprintf(stdout, "run %s %s in %.1fs\n", final.RunID, final.Status, time.Since(start).Seconds())
		return nil
	case "cancel":
		if len(args) < 2 {
			return errors.New("usage: runs cancel <run-id>")
		}
		var out run
		if err := c.do(ctx, http.MethodPost, "/runs/"+url.PathEscape(args[1])+"/cancel", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "delete":
		if len(args) < 2 {
			return errors.New("usage: runs delete <run-id>")
		}
		if err := c.do(ctx, http.MethodDelete, "/runs/"+url.PathEscape(args[1]), nil, nil); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, map[string]any{"deleted": true, "run_id": args[1]})
	case "logs":
		return runLogs(ctx, c, cfg, args[1:], stdout)
	default:
		return fmt.Errorf("unknown runs command %q", args[0])
	}
}

func runCreateRun(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("runs create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	entrypoint, varsFile := "", ""
	debug, wait := false, false
	var vars, volumes, secretVolumes stringList
	fs.StringVar(&entrypoint, "workflow", "", "")
	fs.Var(&vars, "var", "")
	fs.StringVar(&varsFile, "vars-file", "", "")
	fs.BoolVar(&debug, "debug", false, "")
	fs.Var(&volumes, "volume", "")
	fs.Var(&secretVolumes, "secret-volume", "")
	fs.BoolVar(&wait, "wait", false, "")
	positionals, err := parseCommandFlags(fs, args)
	if err != nil {
		return err
	}
	if len(positionals) < 1 {
		return errors.New("usage: runs create <workflow-name> [flags]")
	}
	varMap, err := parseVars(vars, varsFile)
	if err != nil {
		return err
	}
	pvcs, err := parsePVCMounts(volumes)
	if err != nil {
		return err
	}
	secrets, err := parseSecretMounts(secretVolumes)
	if err != nil {
		return err
	}
	req := runCreateRequest{
		WorkflowName:       positionals[0],
		WorkflowEntrypoint: strings.TrimSpace(entrypoint),
		Vars:               varMap,
		Debug:              debug,
		Volumes:            pvcs,
		SecretVolumes:      secrets,
	}
	var out run
	if err := c.do(ctx, http.MethodPost, "/runs", req, &out); err != nil {
		return err
	}
	if !wait {
		return writeOutput(stdout, cfg.Output, out)
	}
	start := time.Now()
	final, err := waitRun(ctx, c, cfg, out.RunID, stdout)
	if err != nil {
		return err
	}
	if cfg.Output == "json" || cfg.Output == "yaml" {
		return writeOutput(stdout, cfg.Output, waitResult{Status: final.Status, ElapsedSeconds: time.Since(start).Seconds(), Run: final})
	}
	fmt.Fprintf(stdout, "run %s %s in %.1fs\n", final.RunID, final.Status, time.Since(start).Seconds())
	return nil
}

func waitRun(ctx context.Context, c *apiClient, cfg cliConfig, runID string, stdout io.Writer) (run, error) {
	waitCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	last := ""
	for {
		var r run
		if err := c.do(waitCtx, http.MethodGet, "/runs/"+url.PathEscape(runID), nil, &r); err != nil {
			return r, err
		}
		if cfg.Output == "table" && r.Status != last {
			fmt.Fprintf(stdout, "run %s %s\n", r.RunID, r.Status)
			last = r.Status
		}
		switch r.Status {
		case "completed":
			return r, nil
		case "failed", "cancelled":
			return r, fmt.Errorf("run %s %s", r.RunID, r.Status)
		}
		select {
		case <-waitCtx.Done():
			return r, waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func parseVars(items []string, varsFile string) (map[string]string, error) {
	out := map[string]string{}
	if varsFile != "" {
		b, err := os.ReadFile(varsFile)
		if err != nil {
			return nil, err
		}
		var raw map[string]any
		if json.Unmarshal(b, &raw) != nil {
			if err := yaml.Unmarshal(b, &raw); err != nil {
				return nil, fmt.Errorf("parse vars file: %w", err)
			}
		}
		for k, v := range raw {
			out[k] = fmt.Sprint(v)
		}
	}
	for _, item := range items {
		k, v, ok := strings.Cut(item, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --var %q, expected KEY=VALUE", item)
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parsePVCMounts(items []string) ([]pvcMount, error) {
	seen := map[string]bool{}
	var out []pvcMount
	for _, item := range items {
		name, mount, err := parseMount(item)
		if err != nil {
			return nil, err
		}
		if seen[mount] {
			return nil, fmt.Errorf("duplicate mount path %q", mount)
		}
		seen[mount] = true
		out = append(out, pvcMount{Name: strings.ReplaceAll(name, "/", "-"), PVC: name, MountPath: mount})
	}
	return out, nil
}

func parseSecretMounts(items []string) ([]secretMount, error) {
	seen := map[string]bool{}
	var out []secretMount
	for _, item := range items {
		name, mount, err := parseMount(item)
		if err != nil {
			return nil, err
		}
		if seen[mount] {
			return nil, fmt.Errorf("duplicate mount path %q", mount)
		}
		seen[mount] = true
		out = append(out, secretMount{Name: strings.ReplaceAll(name, "/", "-"), Secret: name, MountPath: mount})
	}
	return out, nil
}

func parseMount(item string) (string, string, error) {
	if strings.Count(item, ":") != 1 {
		return "", "", fmt.Errorf("invalid mount %q, expected NAME:/absolute/path", item)
	}
	name, mount, _ := strings.Cut(item, ":")
	if name == "" || mount == "" {
		return "", "", fmt.Errorf("invalid mount %q, expected NAME:/absolute/path", item)
	}
	if !strings.HasPrefix(mount, "/") {
		return "", "", fmt.Errorf("mount path must be absolute: %q", mount)
	}
	return name, mount, nil
}

func runLogs(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("runs logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	follow := false
	fs.BoolVar(&follow, "follow", false, "")
	positionals, err := parseCommandFlags(fs, args)
	if err != nil {
		return err
	}
	if len(positionals) < 1 {
		return errors.New("usage: runs logs <run-id> [--follow]")
	}
	path := "/runs/" + url.PathEscape(positionals[0]) + "/logs"
	if follow {
		return streamSSE(ctx, c, path+"/stream", stdout)
	}
	var out any
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return err
	}
	return writeOutput(stdout, cfg.Output, out)
}

func streamSSE(ctx context.Context, c *apiClient, path string, stdout io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1"+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError{StatusCode: resp.StatusCode}
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			fmt.Fprintln(stdout, strings.TrimPrefix(line, "data: "))
		}
	}
	return scanner.Err()
}

func runSimpleList(ctx context.Context, c *apiClient, cfg cliConfig, args []string, path string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "list" {
		return errors.New("usage: list")
	}
	var out any
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return err
	}
	return writeOutput(stdout, cfg.Output, out)
}

func runPreferences(ctx context.Context, c *apiClient, cfg cliConfig, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("preferences command required")
	}
	switch args[0] {
	case "get":
		var out any
		if err := c.do(ctx, http.MethodGet, "/preferences", nil, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	case "set":
		if len(args) < 2 {
			return errors.New("usage: preferences set KEY=VALUE [KEY=VALUE...]")
		}
		body := map[string]any{}
		for _, item := range args[1:] {
			k, v, ok := strings.Cut(item, "=")
			if !ok || k == "" {
				return fmt.Errorf("invalid preference %q, expected KEY=VALUE", item)
			}
			parsed := any(v)
			trimmed := strings.TrimSpace(v)
			if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
				if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
					return fmt.Errorf("invalid JSON preference %s: %w", k, err)
				}
			}
			body[k] = parsed
		}
		var out any
		if err := c.do(ctx, http.MethodPut, "/preferences", body, &out); err != nil {
			return err
		}
		return writeOutput(stdout, cfg.Output, out)
	default:
		return fmt.Errorf("unknown preferences command %q", args[0])
	}
}

func writeOutput(w io.Writer, format string, v any) error {
	switch format {
	case "", "table":
		return writeTable(w, v)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case "yaml":
		b, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeTable(w io.Writer, v any) error {
	switch x := v.(type) {
	case []project:
		for _, p := range x {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Branch, p.SyncStatus, p.URL)
		}
	case project:
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", x.ID, x.Name, x.Branch, x.SyncStatus, x.URL)
	case []workflow:
		for _, wf := range x {
			fmt.Fprintf(w, "%s\t%s\t%s\n", wf.Name, wf.DefinitionKind, wf.SourcePath)
		}
	case workflow:
		fmt.Fprintf(w, "%s\t%s\t%s\n", x.Name, x.DefinitionKind, x.SourcePath)
	case []run:
		for _, r := range x {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.RunID, r.WorkflowName, r.Status)
		}
	case run:
		fmt.Fprintf(w, "%s\t%s\t%s\n", x.RunID, x.WorkflowName, x.Status)
	case map[string]string:
		for k, val := range x {
			fmt.Fprintf(w, "%s\t%s\n", k, val)
		}
	case map[string]any:
		for k, val := range x {
			fmt.Fprintf(w, "%s\t%v\n", k, val)
		}
	default:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	return nil
}
