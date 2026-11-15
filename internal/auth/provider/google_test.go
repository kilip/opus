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
	"golang.org/x/oauth2/google"
)

func TestGoogleProvider_ConfigAndScopes(t *testing.T) {
	reg := provider.NewRegistry(provider.Config{
		BaseURL:            "http://localhost:7880",
		GoogleClientID:     "google-id-123",
		GoogleClientSecret: "google-secret-456",
	})

	p, ok := reg.Get("google")
	if !ok {
		t.Fatal("expected google provider to be registered")
	}

	gp, ok := p.(*provider.GoogleProvider)
	if !ok {
		t.Fatalf("expected *provider.GoogleProvider, got %T", p)
	}

	if gp.Name() != "google" {
		t.Errorf("got name %q, want %q", gp.Name(), "google")
	}
	if gp.Label() != "Google" {
		t.Errorf("got label %q, want %q", gp.Label(), "Google")
	}

	cfg := gp.OAuthConfig()
	// AC1: endpoint comes from golang.org/x/oauth2/google
	if cfg.Endpoint != google.Endpoint {
		t.Errorf("got endpoint %+v, want %+v", cfg.Endpoint, google.Endpoint)
	}

	// AC1: scopes are exactly openid, email, and profile
	expectedScopes := []string{"openid", "email", "profile"}
	if !reflect.DeepEqual(cfg.Scopes, expectedScopes) {
		t.Errorf("got scopes %v, want %v", cfg.Scopes, expectedScopes)
	}

	expectedRedirect := "http://localhost:7880/api/auth/oauth/google/callback"
	if cfg.RedirectURL != expectedRedirect {
		t.Errorf("got redirect url %q, want %q", cfg.RedirectURL, expectedRedirect)
	}
}

func TestGoogleProvider_Exchange(t *testing.T) {
	tests := []struct {
		name              string
		userInfoJSON      string
		userInfoStatus    int
		tokenStatus       int
		verifier          string
		wantProfile       provider.Profile
		wantErr           bool
		wantErrSubstr     string
		checkAuthHeader   bool
		checkVerifierSent bool
	}{
		{
			name: "verified gmail email (AC2, AC4)",
			userInfoJSON: `{
				"sub": "109876543210123456789",
				"name": "Jane Doe",
				"given_name": "Jane",
				"family_name": "Doe",
				"picture": "https://lh3.googleusercontent.com/a/abc",
				"email": "jane.doe@gmail.com",
				"email_verified": true
			}`,
			userInfoStatus: http.StatusOK,
			tokenStatus:    http.StatusOK,
			verifier:       "test-google-verifier-1",
			wantProfile: provider.Profile{
				ProviderUserID: "109876543210123456789",
				Email:          "jane.doe@gmail.com",
				EmailVerified:  true,
				Name:           "Jane Doe",
				AvatarURL:      "https://lh3.googleusercontent.com/a/abc",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
		},
		{
			name: "unverified email (AC2, AC4)",
			userInfoJSON: `{
				"sub": "209876543210123456789",
				"name": "Unverified User",
				"picture": "https://lh3.googleusercontent.com/a/unverified",
				"email": "unverified@example.com",
				"email_verified": false
			}`,
			userInfoStatus: http.StatusOK,
			tokenStatus:    http.StatusOK,
			verifier:       "test-google-verifier-2",
			wantProfile: provider.Profile{
				ProviderUserID: "209876543210123456789",
				Email:          "unverified@example.com",
				EmailVerified:  false,
				Name:           "Unverified User",
				AvatarURL:      "https://lh3.googleusercontent.com/a/unverified",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
		},
		{
			name: "non-gmail workspace account with verified email",
			userInfoJSON: `{
				"sub": "309876543210123456789",
				"name": "Alice Workspace",
				"given_name": "Alice",
				"family_name": "Smith",
				"picture": "https://lh3.googleusercontent.com/a/corp",
				"email": "alice@customcompany.org",
				"email_verified": true
			}`,
			userInfoStatus: http.StatusOK,
			tokenStatus:    http.StatusOK,
			verifier:       "test-google-verifier-3",
			wantProfile: provider.Profile{
				ProviderUserID: "309876543210123456789",
				Email:          "alice@customcompany.org",
				EmailVerified:  true,
				Name:           "Alice Workspace",
				AvatarURL:      "https://lh3.googleusercontent.com/a/corp",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
		},
		{
			name: "fallback to given and family name when name empty",
			userInfoJSON: `{
				"sub": "409876543210123456789",
				"name": "",
				"given_name": "Bob",
				"family_name": "Ross",
				"picture": "",
				"email": "bob@paintings.net",
				"email_verified": true
			}`,
			userInfoStatus: http.StatusOK,
			tokenStatus:    http.StatusOK,
			verifier:       "test-google-verifier-4",
			wantProfile: provider.Profile{
				ProviderUserID: "409876543210123456789",
				Email:          "bob@paintings.net",
				EmailVerified:  true,
				Name:           "Bob Ross",
				AvatarURL:      "",
			},
			checkAuthHeader:   true,
			checkVerifierSent: true,
		},
		{
			name:          "oauth token exchange error",
			tokenStatus:   http.StatusBadRequest,
			verifier:      "test-verifier",
			wantErr:       true,
			wantErrSubstr: "google oauth exchange",
		},
		{
			name:           "userinfo endpoint 500 error",
			tokenStatus:    http.StatusOK,
			userInfoStatus: http.StatusInternalServerError,
			wantErr:        true,
			wantErrSubstr:  "google api userinfo returned status 500",
		},
		{
			name:           "userinfo malformed JSON",
			tokenStatus:    http.StatusOK,
			userInfoStatus: http.StatusOK,
			userInfoJSON:   "invalid json payload",
			wantErr:        true,
			wantErrSubstr:  "decode google userinfo response",
		},
		{
			name:           "missing user sub",
			tokenStatus:    http.StatusOK,
			userInfoStatus: http.StatusOK,
			userInfoJSON: `{
				"sub": "",
				"name": "No Sub",
				"email": "nosub@example.com",
				"email_verified": true
			}`,
			wantErr:       true,
			wantErrSubstr: "invalid or missing google user sub",
		},
		{
			name:           "whitespace-only user sub",
			tokenStatus:    http.StatusOK,
			userInfoStatus: http.StatusOK,
			userInfoJSON: `{
				"sub": "   ",
				"name": "Blank Sub",
				"email": "blanksub@example.com",
				"email_verified": true
			}`,
			wantErr:       true,
			wantErrSubstr: "invalid or missing google user sub",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedVerifier string
			var receivedUserAuth string
			var receivedUserAccept string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/oauth2/token":
					if tc.tokenStatus != 0 && tc.tokenStatus != http.StatusOK {
						http.Error(w, "oauth error", tc.tokenStatus)
						return
					}
					if err := r.ParseForm(); err == nil {
						receivedVerifier = r.Form.Get("code_verifier")
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"access_token": "mock-google-token-xyz",
						"token_type":   "Bearer",
					})
				case "/v1/userinfo":
					receivedUserAuth = r.Header.Get("Authorization")
					receivedUserAccept = r.Header.Get("Accept")
					status := tc.userInfoStatus
					if status == 0 {
						status = http.StatusOK
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(tc.userInfoJSON))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := oauth2.Config{
				ClientID:     "google-client-id",
				ClientSecret: "google-client-secret",
				Endpoint: oauth2.Endpoint{
					AuthURL:  server.URL + "/o/oauth2/auth",
					TokenURL: server.URL + "/oauth2/token",
				},
				RedirectURL: "http://localhost:7880/api/auth/oauth/google/callback",
				Scopes:      []string{"openid", "email", "profile"},
			}

			// AC3: Userinfo URL is injectable
			p := provider.NewGoogleProvider(cfg, provider.WithUserInfoURL(server.URL+"/v1/userinfo"))

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
				expectedAuth := "Bearer mock-google-token-xyz"
				if receivedUserAuth != expectedAuth {
					t.Errorf("got user Authorization header %q, want %q", receivedUserAuth, expectedAuth)
				}
				expectedAccept := "application/json"
				if receivedUserAccept != expectedAccept {
					t.Errorf("got user Accept header %q, want %q", receivedUserAccept, expectedAccept)
				}
			}
		})
	}
}

func TestGoogleProvider_WithGoogleHTTPClient(t *testing.T) {
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
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		Endpoint: oauth2.Endpoint{
			TokenURL: "http://invalid-test-url/token",
		},
	}

	p := provider.NewGoogleProvider(cfg, provider.WithGoogleHTTPClient(customClient))

	_, err := p.Exchange(context.Background(), "code", "verifier")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !customTransportCalled {
		t.Error("expected custom HTTP client transport to be called")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
