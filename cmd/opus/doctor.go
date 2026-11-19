package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/worker"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

const doctorTimeout = 3 * time.Second

type doctorStatus string

const (
	doctorPass doctorStatus = "PASS"
	doctorWarn doctorStatus = "WARN"
	doctorFail doctorStatus = "FAIL"
)

type doctorResult struct {
	Name    string       `json:"name"`
	Status  doctorStatus `json:"status"`
	Message string       `json:"message"`
	Hint    string       `json:"hint"`
}

type doctorOptions struct {
	jsonOut bool
}

type doctorDeps struct {
	out             io.Writer
	cfg             app.Config
	currentVersion  string
	goos            string
	goarch          string
	apiBaseURL      string
	telegramBaseURL string
	httpClient      *http.Client
	homeDir         func() (string, error)
	getenv          func(string) string
	stat            func(string) (os.FileInfo, error)
	lookPath        func(string) (string, error)
	runCmd          func(ctx context.Context, name string, args ...string) ([]byte, error)
	probeAgy        func(ctx context.Context, agyBin string) worker.ToolStatus
	dialTimeout     func(network, address string, timeout time.Duration) (net.Conn, error)
	listen          func(network, address string) (net.Listener, error)
	serviceStatus   func(ctx context.Context) (int, error)
}

func defaultDoctorDeps(ctx context.Context, cmd *cobra.Command) doctorDeps {
	cfg, _ := app.GetConfig(ctx)
	return doctorDeps{
		out:             cmd.OutOrStdout(),
		cfg:             cfg,
		currentVersion:  Version,
		goos:            runtime.GOOS,
		goarch:          runtime.GOARCH,
		apiBaseURL:      defaultGitHubAPIBaseURL,
		telegramBaseURL: "https://api.telegram.org",
		httpClient:      &http.Client{Timeout: doctorTimeout},
		homeDir:         opusHomeDir,
		getenv:          os.Getenv,
		stat:            os.Stat,
		lookPath:        exec.LookPath,
		runCmd: func(c context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(c, name, args...).CombinedOutput()
		},
		probeAgy: func(c context.Context, agyBin string) worker.ToolStatus {
			return worker.NewCapabilityProbe(agyBin)(c).Agy
		},
		dialTimeout: net.DialTimeout,
		listen:      net.Listen,
		serviceStatus: func(c context.Context) (int, error) {
			deps := defaultServiceDeps(cmd)
			deps.out = io.Discard
			return runServiceStatus(c, serviceOptions{role: serviceRoleServer}, deps)
		},
	}
}

func newDoctorCmd() *cobra.Command {
	var opts doctorOptions
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run read-only diagnostics on the opus environment",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			err := runDoctor(c.Context(), opts, defaultDoctorDeps(c.Context(), c))
			var exitErr exitCodeError
			if errors.As(err, &exitErr) {
				c.SilenceErrors = true
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output diagnostic results as JSON")
	return cmd
}

type doctorCheck struct {
	name string
	fn   func(ctx context.Context, deps doctorDeps) doctorResult
}

func runDoctor(ctx context.Context, opts doctorOptions, deps doctorDeps) error {
	checks := []doctorCheck{
		{name: "version", fn: checkDoctorVersion},
		{name: "git", fn: checkDoctorGit},
		{name: "gh", fn: checkDoctorGh},
		{name: "agy", fn: checkDoctorAgy},
		{name: "app_home", fn: checkDoctorAppHome},
		{name: "database", fn: checkDoctorDatabase},
		{name: "auth_secret", fn: checkDoctorAuthSecret},
		{name: "base_url_cookie", fn: checkDoctorBaseURLCookie},
		{name: "mailer", fn: checkDoctorMailer},
		{name: "telegram", fn: checkDoctorTelegram},
		{name: "port", fn: checkDoctorPort},
		{name: "worker", fn: checkDoctorWorker},
		{name: "setup", fn: checkDoctorSetup},
		{name: "service", fn: checkDoctorService},
	}

	results := make([]doctorResult, 0, len(checks))
	hasFail := false
	for _, chk := range checks {
		res := chk.fn(ctx, deps)
		res.Name = chk.name
		if res.Status == doctorFail {
			hasFail = true
		}
		results = append(results, res)
	}

	if opts.jsonOut {
		enc := json.NewEncoder(deps.out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return err
		}
	} else {
		for _, r := range results {
			if r.Hint != "" {
				_, _ = fmt.Fprintf(deps.out, "[%s] %s: %s (hint: %s)\n", r.Status, r.Name, r.Message, r.Hint)
			} else {
				_, _ = fmt.Fprintf(deps.out, "[%s] %s: %s\n", r.Status, r.Name, r.Message)
			}
		}
	}

	if hasFail {
		return exitCodeError{code: 1}
	}
	return nil
}

func checkDoctorVersion(ctx context.Context, deps doctorDeps) doctorResult {
	cur, err := parseSemver(deps.currentVersion)
	if err != nil {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("version %q (%s/%s) is not valid semver", deps.currentVersion, deps.goos, deps.goarch),
			Hint:    "build with -ldflags '-X main.Version=<semver>'",
		}
	}

	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	releases, err := fetchReleases(cctx, deps.httpClient, deps.apiBaseURL)
	if err != nil {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("%s (%s/%s); update check failed: %v", cur.String(), deps.goos, deps.goarch, err),
			Hint:    "check network connectivity or run 'opus self-update --check' later",
		}
	}
	_, latest, err := selectRelease(releases, "")
	if err != nil {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("%s (%s/%s); no valid releases found", cur.String(), deps.goos, deps.goarch),
		}
	}
	if compareSemver(cur, latest) < 0 {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("%s (%s/%s); newer version %s available", cur.String(), deps.goos, deps.goarch, latest.String()),
			Hint:    "run 'opus self-update' to upgrade",
		}
	}
	return doctorResult{
		Status:  doctorPass,
		Message: fmt.Sprintf("%s (%s/%s), up to date", cur.String(), deps.goos, deps.goarch),
	}
}

func checkDoctorGit(_ context.Context, deps doctorDeps) doctorResult {
	path, err := deps.lookPath("git")
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: "git binary not found in PATH",
			Hint:    "install git and ensure it is available in PATH",
		}
	}
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("git found at %s", path)}
}

func checkDoctorGh(ctx context.Context, deps doctorDeps) doctorResult {
	if _, err := deps.lookPath("gh"); err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: "gh CLI not found in PATH",
			Hint:    "install GitHub CLI from https://cli.github.com and run 'gh auth login'",
		}
	}
	if deps.getenv != nil && deps.getenv("GH_TOKEN") != "" {
		return doctorResult{Status: doctorPass, Message: "gh CLI installed and GH_TOKEN is set"}
	}
	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	if _, err := deps.runCmd(cctx, "gh", "auth", "status"); err == nil {
		return doctorResult{Status: doctorPass, Message: "gh CLI installed and authenticated"}
	}
	return doctorResult{
		Status:  doctorWarn,
		Message: "gh CLI installed but not authenticated",
		Hint:    "run 'gh auth login', set GH_TOKEN, or configure a GitHub token in Settings",
	}
}

func checkDoctorAgy(ctx context.Context, deps doctorDeps) doctorResult {
	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	switch deps.probeAgy(cctx, deps.cfg.WorkerAgyBin) {
	case worker.ToolStatusReady:
		return doctorResult{Status: doctorPass, Message: "agy CLI installed and authenticated"}
	case worker.ToolStatusUnauthenticated:
		return doctorResult{
			Status:  doctorWarn,
			Message: "agy CLI installed but not authenticated",
			Hint:    "run 'agy auth login'",
		}
	default:
		return doctorResult{
			Status:  doctorWarn,
			Message: "agy CLI not found in PATH",
			Hint:    "install agy or set WORKER_AGY_BIN",
		}
	}
}

func checkDoctorAppHome(_ context.Context, deps doctorDeps) doctorResult {
	appHome, err := deps.homeDir()
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("cannot resolve APP_HOME: %v", err),
			Hint:    "set APP_HOME to a writable directory",
		}
	}
	if _, err := deps.stat(appHome); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorResult{
				Status:  doctorWarn,
				Message: fmt.Sprintf("APP_HOME (%s) does not exist yet", appHome),
				Hint:    "it will be created automatically on first run",
			}
		}
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("cannot access APP_HOME (%s): %v", appHome, err),
			Hint:    "check directory permissions for APP_HOME",
		}
	}
	f, err := os.CreateTemp(appHome, ".opus-doctor-*")
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("APP_HOME (%s) is not writable: %v", appHome, err),
			Hint:    "ensure the user running opus has write permissions to APP_HOME",
		}
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("APP_HOME (%s) exists and is writable", appHome)}
}

func sqliteFilePath(databaseURL string) string {
	dsn := strings.TrimPrefix(databaseURL, "sqlite:")
	dsn = strings.TrimPrefix(dsn, "file:")
	base, _, _ := strings.Cut(dsn, "?")
	return base
}

func checkDoctorDatabase(ctx context.Context, deps doctorDeps) doctorResult {
	dbURL := deps.cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	}
	if !strings.HasPrefix(dbURL, "file:") && !strings.HasPrefix(dbURL, "sqlite:") {
		return doctorResult{
			Status:  doctorFail,
			Message: "unsupported APP_DATABASE_URL scheme (only sqlite/file is supported)",
			Hint:    "set APP_DATABASE_URL=file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		}
	}

	dbPath := sqliteFilePath(dbURL)
	isMemory := dbPath == "" || strings.Contains(strings.ToLower(dbPath), ":memory:")
	if !isMemory {
		if _, err := deps.stat(dbPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return doctorResult{
					Status:  doctorWarn,
					Message: fmt.Sprintf("SQLite database %s does not exist yet", dbPath),
					Hint:    "start 'opus serve' to initialize the database",
				}
			}
			return doctorResult{
				Status:  doctorFail,
				Message: fmt.Sprintf("cannot stat database %s: %v", dbPath, err),
				Hint:    "check APP_DATABASE_URL and file permissions",
			}
		}
	}

	db, err := sql.Open("sqlite", strings.TrimPrefix(dbURL, "sqlite:"))
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("open sqlite database: %v", err),
			Hint:    "check APP_DATABASE_URL",
		}
	}
	defer func() { _ = db.Close() }()

	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	var journalMode string
	if err := db.QueryRowContext(cctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("query journal_mode: %v", err),
			Hint:    "verify database file is a valid SQLite database",
		}
	}
	var busyTimeout int
	if err := db.QueryRowContext(cctx, "PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("query busy_timeout: %v", err),
			Hint:    "verify database file is a valid SQLite database",
		}
	}

	if !isMemory && !strings.EqualFold(journalMode, "wal") {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("SQLite journal_mode is %q (expected wal), busy_timeout=%dms", journalMode, busyTimeout),
			Hint:    "enable WAL mode or let 'opus serve' configure journal_mode(WAL)",
		}
	}
	if busyTimeout <= 0 {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("SQLite journal_mode=%s, busy_timeout=%dms (expected > 0)", journalMode, busyTimeout),
			Hint:    "add _pragma=busy_timeout(5000) to APP_DATABASE_URL",
		}
	}
	return doctorResult{
		Status:  doctorPass,
		Message: fmt.Sprintf("SQLite %s ready (journal_mode=%s, busy_timeout=%dms)", dbPath, journalMode, busyTimeout),
	}
}

func checkDoctorAuthSecret(_ context.Context, deps doctorDeps) doctorResult {
	if s := deps.cfg.SessionSecret; s != "" {
		if len(s) < 32 {
			return doctorResult{
				Status:  doctorFail,
				Message: fmt.Sprintf("AUTH_SECRET is too short (%d bytes, minimum 32)", len(s)),
				Hint:    "set AUTH_SECRET to at least 32 random characters",
			}
		}
		return doctorResult{Status: doctorPass, Message: "AUTH_SECRET is configured (>= 32 bytes)"}
	}

	appHome, err := deps.homeDir()
	if err == nil && appHome != "" {
		secretPath := filepath.Join(appHome, "auth_secret")
		if raw, err := os.ReadFile(secretPath); err == nil {
			s := strings.TrimSpace(string(raw))
			if len(s) < 32 {
				return doctorResult{
					Status:  doctorFail,
					Message: fmt.Sprintf("persisted %s is too short (%d bytes, minimum 32)", secretPath, len(s)),
					Hint:    "remove or regenerate the auth_secret file",
				}
			}
			return doctorResult{Status: doctorPass, Message: fmt.Sprintf("persisted auth_secret at %s is valid (>= 32 bytes)", secretPath)}
		}
	}

	return doctorResult{
		Status:  doctorWarn,
		Message: "AUTH_SECRET is not set and no auth_secret file exists yet",
		Hint:    "set AUTH_SECRET (>= 32 chars) or start 'opus serve' to auto-generate auth_secret",
	}
}

func checkDoctorBaseURLCookie(_ context.Context, deps doctorDeps) doctorResult {
	baseURL := deps.cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:7880"
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("invalid APP_BASE_URL %q", baseURL),
			Hint:    "set APP_BASE_URL to a valid http:// or https:// URL",
		}
	}

	mode := strings.ToLower(strings.TrimSpace(deps.cfg.AuthCookieSecure))
	switch mode {
	case "", "auto", "true", "false":
	default:
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("invalid AUTH_COOKIE_SECURE %q", deps.cfg.AuthCookieSecure),
			Hint:    "set AUTH_COOKIE_SECURE to auto, true, or false",
		}
	}

	if mode == "true" && u.Scheme == "http" {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("AUTH_COOKIE_SECURE=true with %s will block session cookies over HTTP", baseURL),
			Hint:    "use https:// APP_BASE_URL or set AUTH_COOKIE_SECURE=auto",
		}
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("APP_BASE_URL (%s) uses plain HTTP on a non-loopback host", baseURL),
			Hint:    "serve behind HTTPS in production and set APP_BASE_URL=https://...",
		}
	}
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("APP_BASE_URL (%s) and cookie policy (%s) are compatible", baseURL, modeOrDefault(mode))}
}

func modeOrDefault(mode string) string {
	if mode == "" {
		return "auto"
	}
	return mode
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func checkDoctorMailer(_ context.Context, deps doctorDeps) doctorResult {
	if deps.cfg.SMTPHost == "" {
		return doctorResult{
			Status:  doctorWarn,
			Message: "SMTP not configured (using console mailer)",
			Hint:    "set APP_SMTP_HOST and APP_SMTP_PORT for email OTP delivery",
		}
	}
	port := deps.cfg.SMTPPort
	if port <= 0 {
		port = 587
	}
	addr := net.JoinHostPort(deps.cfg.SMTPHost, strconv.Itoa(port))
	conn, err := deps.dialTimeout("tcp", addr, doctorTimeout)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("SMTP server %s unreachable: %v", addr, err),
			Hint:    "verify APP_SMTP_HOST, APP_SMTP_PORT, and network connectivity",
		}
	}
	_ = conn.Close()
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("SMTP server %s reachable", addr)}
}

func scrubToken(msg, token string) string {
	if token == "" {
		return msg
	}
	return strings.ReplaceAll(msg, token, "[REDACTED]")
}

func checkDoctorTelegram(ctx context.Context, deps doctorDeps) doctorResult {
	token := deps.cfg.NotifyTelegramBotToken
	if token == "" {
		return doctorResult{Status: doctorPass, Message: "Telegram notifications not configured (optional)"}
	}
	baseURL := strings.TrimRight(deps.telegramBaseURL, "/")
	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, fmt.Sprintf("%s/bot%s/getMe", baseURL, token), nil)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("invalid Telegram request: %s", scrubToken(err.Error(), token)),
			Hint:    "verify NOTIFY_TELEGRAM_BOT_TOKEN",
		}
	}
	resp, err := deps.httpClient.Do(req)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("Telegram API check failed: %s", scrubToken(err.Error(), token)),
			Hint:    "verify NOTIFY_TELEGRAM_BOT_TOKEN and network connectivity",
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("Telegram API returned status %d", resp.StatusCode),
			Hint:    "verify NOTIFY_TELEGRAM_BOT_TOKEN",
		}
	}
	return doctorResult{Status: doctorPass, Message: "Telegram bot token verified"}
}

func checkDoctorPort(ctx context.Context, deps doctorDeps) doctorResult {
	port := deps.cfg.Port
	if port == "" {
		port = "7880"
	}
	ln, err := deps.listen("tcp", "127.0.0.1:"+port)
	if err == nil {
		_ = ln.Close()
		return doctorResult{Status: doctorPass, Message: fmt.Sprintf("port %s is available", port)}
	}
	if deps.serviceStatus != nil {
		if code, svcErr := deps.serviceStatus(ctx); svcErr == nil && code == serviceStatusRunning {
			return doctorResult{Status: doctorPass, Message: fmt.Sprintf("port %s is in use by running opus service", port)}
		}
	}
	return doctorResult{
		Status:  doctorWarn,
		Message: fmt.Sprintf("port %s is already in use", port),
		Hint:    "set APP_PORT to an available port or stop the process using it",
	}
}

func checkDoctorWorker(ctx context.Context, deps doctorDeps) doctorResult {
	appHome, err := deps.homeDir()
	if err != nil || appHome == "" {
		return doctorResult{Status: doctorPass, Message: "worker mode not configured"}
	}
	cfgPath := nodeConfigPath(appHome)
	if _, err := deps.stat(cfgPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorResult{Status: doctorPass, Message: "worker mode not configured (no node.json)"}
		}
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("cannot stat %s: %v", cfgPath, err),
			Hint:    "check permissions on node.json",
		}
	}
	nodeCfg, err := readNodeConfig(cfgPath)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("invalid node.json: %v", err),
			Hint:    "re-run 'opus join <url>' to regenerate node.json",
		}
	}

	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	healthURL := strings.TrimRight(nodeCfg.MasterURL, "/") + "/api/health"
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("invalid masterUrl in node.json: %v", err),
			Hint:    "re-run 'opus join <url>'",
		}
	}
	resp, err := deps.httpClient.Do(req)
	if err != nil {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("master %s unreachable: %s", nodeCfg.MasterURL, scrubToken(err.Error(), nodeCfg.Credential)),
			Hint:    "ensure the master server is running and reachable",
		}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return doctorResult{
			Status:  doctorFail,
			Message: fmt.Sprintf("master %s returned status %d", nodeCfg.MasterURL, resp.StatusCode),
			Hint:    "check master server health",
		}
	}
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("worker node %s configured and master %s reachable", nodeCfg.NodeID, nodeCfg.MasterURL)}
}

func checkDoctorSetup(ctx context.Context, deps doctorDeps) doctorResult {
	dbURL := deps.cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	}
	dbPath := sqliteFilePath(dbURL)
	if dbPath == "" {
		return doctorResult{Status: doctorPass, Message: "setup check skipped (no file database)"}
	}
	if _, err := deps.stat(dbPath); err != nil {
		return doctorResult{Status: doctorPass, Message: "database not initialized yet"}
	}
	db, err := sql.Open("sqlite", strings.TrimPrefix(dbURL, "sqlite:"))
	if err != nil {
		return doctorResult{Status: doctorPass, Message: "setup check skipped (database unavailable)"}
	}
	defer func() { _ = db.Close() }()

	cctx, cancel := context.WithTimeout(ctx, doctorTimeout)
	defer cancel()
	var count int
	if err := db.QueryRowContext(cctx, "SELECT COUNT(*) FROM users;").Scan(&count); err != nil {
		return doctorResult{Status: doctorPass, Message: "users table not initialized yet"}
	}
	if count == 0 {
		u, _ := url.Parse(deps.cfg.BaseURL)
		nonLoopbackURL := u != nil && u.Hostname() != "" && !isLoopbackHost(u.Hostname())
		nonLoopbackHost := deps.cfg.Host == "" || deps.cfg.Host == "0.0.0.0" || deps.cfg.Host == "::" || !isLoopbackHost(deps.cfg.Host)
		if nonLoopbackURL || nonLoopbackHost {
			return doctorResult{
				Status:  doctorWarn,
				Message: "no owner user exists yet while server is exposed on non-loopback address",
				Hint:    "complete onboarding at /onboarding promptly or bind APP_HOST=127.0.0.1",
			}
		}
	}
	return doctorResult{Status: doctorPass, Message: fmt.Sprintf("owner setup complete (%d user(s))", count)}
}

func checkDoctorService(ctx context.Context, deps doctorDeps) doctorResult {
	if deps.serviceStatus == nil {
		return doctorResult{Status: doctorPass, Message: "service check skipped"}
	}
	code, err := deps.serviceStatus(ctx)
	if err != nil {
		return doctorResult{
			Status:  doctorWarn,
			Message: fmt.Sprintf("could not check service status: %v", err),
		}
	}
	switch code {
	case serviceStatusRunning:
		return doctorResult{Status: doctorPass, Message: "opus service is running"}
	case serviceStatusStopped:
		return doctorResult{
			Status:  doctorWarn,
			Message: "opus service is installed but stopped",
			Hint:    "run 'opus service start' to start it",
		}
	default:
		return doctorResult{Status: doctorPass, Message: "opus service is not installed (optional)"}
	}
}
