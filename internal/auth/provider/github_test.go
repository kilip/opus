package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kilip/opus/internal/auth/provider"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func TestGitHubProvider_ConfigAndScopes(t *testing.T) {
	reg := provider.NewRegistry(provider.Config{
		BaseURL:            "http://localhost:7880",
		GitHubClientID:     "gh-id-123",
		GitHubClientSecret: "gh-secret-456",
	})

	p, ok := reg.Get("github")
	if !ok {
		t.Fatal("expected github provider to be registered")
	}

	ghp, ok := p.(*provider.GitHubProvider)
	if !ok {
		t.Fatalf("expected *provider.GitHubProvider, got %T", p)
	}

	if ghp.Name() != "github" {
		t.Errorf("got name %q, want %q", ghp.Name(), "github")
	}
	if ghp.Label() != "GitHub" {
		t.Errorf("got label %q, want %q", ghp.Label(), "GitHub")
	}

	cfg := ghp.OAuthConfig()
	// AC1: endpoint comes from golang.org/x/oauth2/github
	if cfg.Endpoint != github.Endpoint {
		t.Errorf("got endpoint %+v, want %+v", cfg.Endpoint, github.Endpoint)
	}

	// AC1: scopes are exactly read:user and user:email
	expectedScopes := []string{"read:user", "user:email"}
	if !reflect.DeepEqual(cfg.Scopes, expectedScopes) {
		t.Errorf("got scopes %v, want %v", cfg.Scopes, expectedScopes)
	}

	expectedRedirect := "http://localhost:7880/api/auth/oauth/github/callback"
	if cfg.RedirectURL != expectedRedirect {
		t.Errorf("got redirect url %q, want %q", cfg.RedirectURL, expectedRedirect)
	}
}

func TestGitHubProvider_Exchange(t *testing.T) {
	tests := []struct {
		name              string
		userJSON          string
		userStatus        int
		emailsJSON        string
		emailsStatus      int
		tokenStatus       int
		verifier          string
		wantProfile       provider.Profile
		wantErr           bool
		wantErrSubstr     string
		checkAuthHeader   bool
		checkVerifierSent bool
	}{
		{
			name: "verified primary email",
			userJSON: `{
				"id": 1234567,
				"login": "octocat",
				"name": "Mona Lisa Octocat",
				"avatar_url": "https://github.com/images/octocat.png"
			}`,
			userStatus: http.StatusOK,
			emailsJSON: `[
				{"email": "unverified@example.com", "primary": false, "verified": false},
				{"email": "primary-verified@example.com", "primary": true, "verified": true},
				{"email": "verified-secondary@example.com", "primary": false, "verified": true}
			]`,
			emailsStatus: http.StatusOK,
			tokenStatus:  http.StatusOK,
			verifier:     "test-code-verifier",
			wantProfile: provider.Profile{
				ProviderUserID: "1234567",
				Email:          "primary-verified@example.com",
				EmailVerified:  true,
				Name:           "Mona Lisa Octocat",
				AvatarURL:      "https://github.com/images/octocat.png",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
		},
		{
			name: "unverified primary email falls back to empty email",
			userJSON: `{
				"id": 7654321,
				"login": "unverified_dev",
				"name": "",
				"avatar_url": ""
			}`,
			userStatus: http.StatusOK,
			emailsJSON: `[
				{"email": "primary-unverified@example.com", "primary": true, "verified": false},
				{"email": "other-verified@example.com", "primary": false, "verified": true}
			]`,
			emailsStatus: http.StatusOK,
			tokenStatus:  http.StatusOK,
			wantProfile: provider.Profile{
				ProviderUserID: "7654321",
				Email:          "",
				EmailVerified:  false,
				Name:           "unverified_dev",
				AvatarURL:      "",
			},
		},
		{
			name: "no email falls back to empty email",
			userJSON: `{
				"id": 999999,
				"login": "secretuser",
				"name": "Secret Dev",
				"avatar_url": "https://github.com/images/secret.png"
			}`,
			userStatus:   http.StatusOK,
			emailsJSON:   `[]`,
			emailsStatus: http.StatusOK,
			tokenStatus:  http.StatusOK,
			wantProfile: provider.Profile{
				ProviderUserID: "999999",
				Email:          "",
				EmailVerified:  false,
				Name:           "Secret Dev",
				AvatarURL:      "https://github.com/images/secret.png",
			},
		},
		{
			name:          "token exchange failure",
			tokenStatus:   http.StatusBadRequest,
			wantErr:       true,
			wantErrSubstr: "github oauth exchange",
		},
		{
			name:          "user api returns error status",
			tokenStatus:   http.StatusOK,
			userStatus:    http.StatusUnauthorized,
			userJSON:      `{"message": "Bad credentials"}`,
			wantErr:       true,
			wantErrSubstr: "github api /user returned status 401",
		},
		{
			name:        "emails api returns error status",
			tokenStatus: http.StatusOK,
			userStatus:  http.StatusOK,
			userJSON: `{
				"id": 111111,
				"login": "dev",
				"name": "Dev",
				"avatar_url": ""
			}`,
			emailsStatus:  http.StatusInternalServerError,
			emailsJSON:    `{"message": "Internal error"}`,
			wantErr:       true,
			wantErrSubstr: "github api /user/emails returned status 500",
		},
		{
			name:        "invalid or missing user id",
			tokenStatus: http.StatusOK,
			userStatus:  http.StatusOK,
			userJSON: `{
				"id": 0,
				"login": "invalid"
			}`,
			wantErr:       true,
			wantErrSubstr: "invalid or missing github user id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedVerifier string
			var receivedUserAuth string
			var receivedUserAccept string
			var receivedEmailsAuth string
			var receivedEmailsAccept string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/login/oauth/access_token":
					if tc.tokenStatus != 0 && tc.tokenStatus != http.StatusOK {
						http.Error(w, "oauth error", tc.tokenStatus)
						return
					}
					if err := r.ParseForm(); err == nil {
						receivedVerifier = r.Form.Get("code_verifier")
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"access_token": "mock-token-xyz",
						"token_type":   "bearer",
					})
				case "/user":
					receivedUserAuth = r.Header.Get("Authorization")
					receivedUserAccept = r.Header.Get("Accept")
					status := tc.userStatus
					if status == 0 {
						status = http.StatusOK
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(tc.userJSON))
				case "/user/emails":
					receivedEmailsAuth = r.Header.Get("Authorization")
					receivedEmailsAccept = r.Header.Get("Accept")
					status := tc.emailsStatus
					if status == 0 {
						status = http.StatusOK
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(tc.emailsJSON))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := oauth2.Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Endpoint: oauth2.Endpoint{
					AuthURL:  server.URL + "/login/oauth/authorize",
					TokenURL: server.URL + "/login/oauth/access_token",
				},
				RedirectURL: "http://localhost:7880/api/auth/oauth/github/callback",
				Scopes:      []string{"read:user", "user:email"},
			}

			p := provider.NewGitHubProvider(cfg, provider.WithAPIBaseURL(server.URL))

			profile, err := p.Exchange(context.Background(), "auth-code-123", tc.verifier)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil profile: %+v", profile)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("got error %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if profile != tc.wantProfile {
				t.Errorf("got profile %+v, want %+v", profile, tc.wantProfile)
			}

			if tc.checkVerifierSent && receivedVerifier != tc.verifier {
				t.Errorf("got code_verifier %q, want %q", receivedVerifier, tc.verifier)
			}
			if tc.checkAuthHeader {
				expectedAuth := "Bearer mock-token-xyz"
				if receivedUserAuth != expectedAuth {
					t.Errorf("got user Authorization header %q, want %q", receivedUserAuth, expectedAuth)
				}
				if receivedEmailsAuth != expectedAuth {
					t.Errorf("got emails Authorization header %q, want %q", receivedEmailsAuth, expectedAuth)
				}
				expectedAccept := "application/vnd.github+json"
				if receivedUserAccept != expectedAccept {
					t.Errorf("got user Accept header %q, want %q", receivedUserAccept, expectedAccept)
				}
				if receivedEmailsAccept != expectedAccept {
					t.Errorf("got emails Accept header %q, want %q", receivedEmailsAccept, expectedAccept)
				}
			}
		})
	}
}
