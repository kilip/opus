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
	"golang.org/x/oauth2/facebook"
)

func TestFacebookProvider_ConfigAndScopes(t *testing.T) {
	reg := provider.NewRegistry(provider.Config{
		BaseURL:              "http://localhost:7880",
		FacebookClientID:     "facebook-id-123",
		FacebookClientSecret: "facebook-secret-456",
	})

	p, ok := reg.Get("facebook")
	if !ok {
		t.Fatal("expected facebook provider to be registered")
	}

	fp, ok := p.(*provider.FacebookProvider)
	if !ok {
		t.Fatalf("expected *provider.FacebookProvider, got %T", p)
	}

	if fp.Name() != "facebook" {
		t.Errorf("got name %q, want %q", fp.Name(), "facebook")
	}
	if fp.Label() != "Facebook" {
		t.Errorf("got label %q, want %q", fp.Label(), "Facebook")
	}

	cfg := fp.OAuthConfig()
	// AC1: endpoint comes from golang.org/x/oauth2/facebook
	if cfg.Endpoint != facebook.Endpoint {
		t.Errorf("got endpoint %+v, want %+v", cfg.Endpoint, facebook.Endpoint)
	}

	// AC1: scopes are email and public_profile
	expectedScopes := []string{"email", "public_profile"}
	if !reflect.DeepEqual(cfg.Scopes, expectedScopes) {
		t.Errorf("got scopes %v, want %v", cfg.Scopes, expectedScopes)
	}

	expectedRedirect := "http://localhost:7880/api/auth/oauth/facebook/callback"
	if cfg.RedirectURL != expectedRedirect {
		t.Errorf("got redirect url %q, want %q", cfg.RedirectURL, expectedRedirect)
	}
}

func TestFacebookProvider_Exchange(t *testing.T) {
	tests := []struct {
		name              string
		userJSON          string
		userStatus        int
		tokenStatus       int
		verifier          string
		wantProfile       provider.Profile
		wantErr           bool
		wantErrSubstr     string
		checkAuthHeader   bool
		checkVerifierSent bool
		checkFieldsQuery  bool
	}{
		{
			name: "standard facebook profile with email (AC2, AC3)",
			userJSON: `{
				"id": "10987654321012345",
				"name": "Jane Doe",
				"email": "jane.doe@example.com"
			}`,
			userStatus:  http.StatusOK,
			tokenStatus: http.StatusOK,
			verifier:    "test-facebook-verifier-1",
			wantProfile: provider.Profile{
				ProviderUserID: "10987654321012345",
				Email:          "jane.doe@example.com",
				EmailVerified:  false, // AC3: always false
				Name:           "Jane Doe",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
			checkFieldsQuery:  true,
		},
		{
			name: "facebook profile without email (phone-only user) (AC2, AC3)",
			userJSON: `{
				"id": "20987654321012345",
				"name": "John Phone"
			}`,
			userStatus:  http.StatusOK,
			tokenStatus: http.StatusOK,
			verifier:    "test-facebook-verifier-2",
			wantProfile: provider.Profile{
				ProviderUserID: "20987654321012345",
				Email:          "",
				EmailVerified:  false, // AC3: always false
				Name:           "John Phone",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
			checkFieldsQuery:  true,
		},
		{
			name:          "oauth token exchange error",
			tokenStatus:   http.StatusBadRequest,
			verifier:      "test-verifier",
			wantErr:       true,
			wantErrSubstr: "facebook oauth exchange",
		},
		{
			name:          "graph api 500 error",
			tokenStatus:   http.StatusOK,
			userStatus:    http.StatusInternalServerError,
			wantErr:       true,
			wantErrSubstr: "facebook graph api returned status 500",
		},
		{
			name:          "graph api malformed JSON",
			tokenStatus:   http.StatusOK,
			userStatus:    http.StatusOK,
			userJSON:      "invalid json payload",
			wantErr:       true,
			wantErrSubstr: "decode facebook user response",
		},
		{
			name:        "missing facebook user id",
			tokenStatus: http.StatusOK,
			userStatus:  http.StatusOK,
			userJSON: `{
				"id": "",
				"name": "No ID",
				"email": "noid@example.com"
			}`,
			wantErr:       true,
			wantErrSubstr: "invalid or missing facebook user id",
		},
		{
			name:        "whitespace-only facebook user id",
			tokenStatus: http.StatusOK,
			userStatus:  http.StatusOK,
			userJSON: `{
				"id": "   ",
				"name": "Blank ID",
				"email": "blankid@example.com"
			}`,
			wantErr:       true,
			wantErrSubstr: "invalid or missing facebook user id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedVerifier string
			var receivedUserAuth string
			var receivedUserAccept string
			var receivedFieldsQuery string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/oauth/access_token":
					if tc.tokenStatus != 0 && tc.tokenStatus != http.StatusOK {
						http.Error(w, "oauth error", tc.tokenStatus)
						return
					}
					if err := r.ParseForm(); err == nil {
						receivedVerifier = r.Form.Get("code_verifier")
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"access_token": "mock-facebook-token-xyz",
						"token_type":   "Bearer",
					})
				case "/me":
					receivedUserAuth = r.Header.Get("Authorization")
					receivedUserAccept = r.Header.Get("Accept")
					receivedFieldsQuery = r.URL.Query().Get("fields")
					status := tc.userStatus
					if status == 0 {
						status = http.StatusOK
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(tc.userJSON))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := oauth2.Config{
				ClientID:     "fb-client-id",
				ClientSecret: "fb-client-secret",
				Endpoint: oauth2.Endpoint{
					AuthURL:  server.URL + "/dialog/oauth",
					TokenURL: server.URL + "/oauth/access_token",
				},
				RedirectURL: "http://localhost:7880/api/auth/oauth/facebook/callback",
				Scopes:      []string{"email", "public_profile"},
			}

			// AC5: injectable graph base URL
			p := provider.NewFacebookProvider(cfg, provider.WithGraphBaseURL(server.URL))

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
				expectedAuth := "Bearer mock-facebook-token-xyz"
				if receivedUserAuth != expectedAuth {
					t.Errorf("got Authorization header %q, want %q", receivedUserAuth, expectedAuth)
				}
				expectedAccept := "application/json"
				if receivedUserAccept != expectedAccept {
					t.Errorf("got Accept header %q, want %q", receivedUserAccept, expectedAccept)
				}
			}
			if tc.checkFieldsQuery {
				// AC2: calls /me with fields id,name,email
				expectedFields := "id,name,email"
				if receivedFieldsQuery != expectedFields {
					t.Errorf("got fields query %q, want %q", receivedFieldsQuery, expectedFields)
				}
			}
		})
	}
}

func TestFacebookProvider_WithFacebookHTTPClient(t *testing.T) {
	customTransportCalled := false
	customClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			customTransportCalled = true
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		}),
	}

	cfg := oauth2.Config{
		ClientID:     "facebook-client-id",
		ClientSecret: "facebook-client-secret",
		Endpoint: oauth2.Endpoint{
			TokenURL: "http://invalid-test-url/oauth/access_token",
		},
	}

	p := provider.NewFacebookProvider(cfg, provider.WithFacebookHTTPClient(customClient))

	_, err := p.Exchange(context.Background(), "code", "verifier")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !customTransportCalled {
		t.Error("expected custom HTTP client transport to be called")
	}
}

func TestFacebookProvider_WithGraphBaseURL_TrimSlash(t *testing.T) {
	// Verify using exchange against a mock server configured with trailing slash
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"token_type":   "Bearer",
			})
			return
		}
		if r.URL.Path == "/me" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"123","name":"Test"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := oauth2.Config{
		ClientID:     "facebook-client-id",
		ClientSecret: "facebook-client-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: server.URL + "/token"},
	}
	p := provider.NewFacebookProvider(cfg, provider.WithGraphBaseURL(server.URL+"/"))

	profile, err := p.Exchange(context.Background(), "code", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.ProviderUserID != "123" {
		t.Errorf("got ProviderUserID %q, want %q", profile.ProviderUserID, "123")
	}
}
