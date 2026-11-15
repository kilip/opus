package mailer_test

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kilip/opus/internal/platform/mailer"
)

type fakeSMTPServer struct {
	listener     net.Listener
	addr         string
	serverTLS    *tls.Config
	implicitTLS  bool
	advertiseTLS bool
	user         string
	pass         string

	mu        sync.Mutex
	from      string
	to        []string
	data      []byte
	connCount int
}

func generateTestCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
}

func startFakeServer(t *testing.T, setup func(*fakeSMTPServer)) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := &fakeSMTPServer{
		listener:     ln,
		addr:         ln.Addr().String(),
		advertiseTLS: true,
	}
	if setup != nil {
		setup(server)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go server.handleConn(conn)
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
	})

	return server
}

func (s *fakeSMTPServer) handleConn(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()
	s.mu.Lock()
	s.connCount++
	s.mu.Unlock()

	var isTLS bool
	if s.implicitTLS {
		tlsConn := tls.Server(conn, s.serverTLS)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		conn = tlsConn
		isTLS = true
	}

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	writeLine := func(line string) error {
		if _, err := w.WriteString(line + "\r\n"); err != nil {
			return err
		}
		return w.Flush()
	}

	if err := writeLine("220 127.0.0.1 ESMTP FakeSMTP"); err != nil {
		return
	}

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		parts := strings.SplitN(line, " ", 2)
		cmd := strings.ToUpper(parts[0])
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch cmd {
		case "EHLO", "HELO":
			if s.serverTLS != nil && !isTLS && s.advertiseTLS {
				_ = writeLine("250-127.0.0.1")
				_ = writeLine("250-STARTTLS")
				_ = writeLine("250 8BITMIME")
			} else if s.user != "" {
				_ = writeLine("250-127.0.0.1")
				_ = writeLine("250-AUTH PLAIN")
				_ = writeLine("250 8BITMIME")
			} else {
				_ = writeLine("250 127.0.0.1")
			}
		case "STARTTLS":
			if s.serverTLS == nil || isTLS || !s.advertiseTLS {
				_ = writeLine("502 Command not implemented")
				continue
			}
			if err := writeLine("220 Ready to start TLS"); err != nil {
				return
			}
			tlsConn := tls.Server(conn, s.serverTLS)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			isTLS = true
			r = bufio.NewReader(conn)
			w = bufio.NewWriter(conn)
		case "AUTH":
			authParts := strings.SplitN(arg, " ", 2)
			if len(authParts) == 2 && strings.ToUpper(authParts[0]) == "PLAIN" {
				decoded, err := base64.StdEncoding.DecodeString(authParts[1])
				if err != nil {
					_ = writeLine("501 Cannot decode auth")
					continue
				}
				tokens := strings.Split(string(decoded), "\x00")
				var u, p string
				if len(tokens) == 3 {
					u, p = tokens[1], tokens[2]
				}
				if s.user != "" && (u != s.user || p != s.pass) {
					_ = writeLine("535 Authentication credentials invalid")
					continue
				}
				_ = writeLine("235 2.7.0 Authentication successful")
			} else {
				_ = writeLine("504 Unrecognized authentication type")
			}
		case "MAIL":
			from := arg
			if start := strings.Index(arg, "<"); start != -1 {
				if end := strings.Index(arg[start:], ">"); end != -1 {
					from = arg[start+1 : start+end]
				}
			}
			s.mu.Lock()
			s.from = from
			s.mu.Unlock()
			_ = writeLine("250 OK")
		case "RCPT":
			to := arg
			if start := strings.Index(arg, "<"); start != -1 {
				if end := strings.Index(arg[start:], ">"); end != -1 {
					to = arg[start+1 : start+end]
				}
			}
			s.mu.Lock()
			s.to = append(s.to, to)
			s.mu.Unlock()
			_ = writeLine("250 OK")
		case "DATA":
			_ = writeLine("354 Start mail input; end with <CRLF>.<CRLF>")
			var body strings.Builder
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" || dataLine == ".\n" {
					break
				}
				body.WriteString(dataLine)
			}
			s.mu.Lock()
			s.data = []byte(body.String())
			s.mu.Unlock()
			_ = writeLine("250 OK: message queued")
		case "QUIT":
			_ = writeLine("221 Bye")
			return
		case "RSET", "NOOP":
			_ = writeLine("250 OK")
		default:
			_ = writeLine("500 Command unrecognized")
		}
	}
}

func parsePort(t *testing.T, addr string) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("failed to split host port: %v", err)
	}
	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}
	return port
}

func TestSMTPMailer_Send_Plain(t *testing.T) {
	server := startFakeServer(t, nil)
	port := parsePort(t, server.addr)

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: port,
		From: "sender@example.com",
	})

	err := m.Send(context.Background(), "pak.bos@example.com", "Plain Subject", "Hello from plain SMTP")
	if err != nil {
		t.Fatalf("unexpected Send error: %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.from != "sender@example.com" {
		t.Errorf("expected from sender@example.com, got %q", server.from)
	}
	if len(server.to) != 1 || server.to[0] != "pak.bos@example.com" {
		t.Errorf("expected to [pak.bos@example.com], got %v", server.to)
	}

	data := string(server.data)
	if !strings.Contains(data, "Subject: Plain Subject") {
		t.Errorf("expected subject in data, got: %s", data)
	}
	if !strings.Contains(data, "Hello from plain SMTP") {
		t.Errorf("expected body in data, got: %s", data)
	}
	if !strings.Contains(data, "From: sender@example.com") {
		t.Errorf("expected From header, got: %s", data)
	}
	if !strings.Contains(data, "To: pak.bos@example.com") {
		t.Errorf("expected To header, got: %s", data)
	}
}

func TestSMTPMailer_Send_STARTTLS(t *testing.T) {
	cert := generateTestCert(t)
	server := startFakeServer(t, func(s *fakeSMTPServer) {
		s.serverTLS = &tls.Config{Certificates: []tls.Certificate{cert}}
		s.user = "opususer"
		s.pass = "opuspass"
	})
	port := parsePort(t, server.addr)

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host:      "127.0.0.1",
		Port:      port,
		Username:  "opususer",
		Password:  "opuspass",
		From:      "secure@example.com",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})

	err := m.Send(context.Background(), "pak.bos@example.com", "STARTTLS Subject", "Secure message body")
	if err != nil {
		t.Fatalf("unexpected Send error: %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.from != "secure@example.com" {
		t.Errorf("expected from secure@example.com, got %q", server.from)
	}
	if len(server.to) != 1 || server.to[0] != "pak.bos@example.com" {
		t.Errorf("expected to [pak.bos@example.com], got %v", server.to)
	}
	data := string(server.data)
	if !strings.Contains(data, "STARTTLS Subject") {
		t.Errorf("expected subject in message, got: %s", data)
	}
	if !strings.Contains(data, "Secure message body") {
		t.Errorf("expected body in message, got: %s", data)
	}
}

func TestSMTPMailer_Send_ImplicitTLS(t *testing.T) {
	cert := generateTestCert(t)
	server := startFakeServer(t, func(s *fakeSMTPServer) {
		s.serverTLS = &tls.Config{Certificates: []tls.Certificate{cert}}
		s.implicitTLS = true
		s.user = "opususer"
		s.pass = "opuspass"
	})
	serverAddr := server.addr

	// Test Port == 465 with custom dialer redirecting to test fake server
	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host:      "127.0.0.1",
		Port:      465,
		Username:  "opususer",
		Password:  "opuspass",
		From:      "implicit@example.com",
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, network, serverAddr)
		},
	})

	err := m.Send(context.Background(), "pak.bos@example.com", "Implicit TLS Subject", "Implicit TLS Body")
	if err != nil {
		t.Fatalf("unexpected Send error: %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.from != "implicit@example.com" {
		t.Errorf("expected from implicit@example.com, got %q", server.from)
	}
	data := string(server.data)
	if !strings.Contains(data, "Implicit TLS Subject") {
		t.Errorf("expected subject in message, got: %s", data)
	}
}

func TestSMTPMailer_RefuseUnencryptedAuth(t *testing.T) {
	server := startFakeServer(t, func(s *fakeSMTPServer) {
		s.serverTLS = nil // unencrypted server
		s.user = "opususer"
		s.pass = "opuspass"
	})
	port := parsePort(t, server.addr)

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "opususer",
		Password: "opuspass",
	})

	err := m.Send(context.Background(), "pak.bos@example.com", "Plain with Auth", "Should fail")
	if err == nil {
		t.Fatal("expected error when sending credentials over unencrypted connection, got nil")
	}
	if !errors.Is(err, mailer.ErrUnencryptedAuth) {
		t.Errorf("expected ErrUnencryptedAuth, got: %v", err)
	}
}

func TestSMTPMailer_HeaderInjection(t *testing.T) {
	server := startFakeServer(t, nil)
	port := parsePort(t, server.addr)

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: port,
	})

	tests := []struct {
		name    string
		to      string
		subject string
	}{
		{"to with CR", "pak.bos@example.com\rBcc: victim@example.com", "Normal Subject"},
		{"to with LF", "pak.bos@example.com\nBcc: victim@example.com", "Normal Subject"},
		{"to with CRLF", "pak.bos@example.com\r\nBcc: victim@example.com", "Normal Subject"},
		{"subject with CR", "pak.bos@example.com", "Subject\rInjected Header: value"},
		{"subject with LF", "pak.bos@example.com", "Subject\nInjected Header: value"},
		{"subject with CRLF", "pak.bos@example.com", "Subject\r\nInjected Header: value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := m.Send(context.Background(), tc.to, tc.subject, "Body content")
			if err == nil {
				t.Fatalf("expected header injection error for %s, got nil", tc.name)
			}
			if !errors.Is(err, mailer.ErrHeaderInjection) {
				t.Errorf("expected ErrHeaderInjection, got: %v", err)
			}
		})
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.connCount != 0 {
		t.Errorf("expected zero connections to server on header injection, got %d", server.connCount)
	}
}

func TestSMTPMailer_Port587_RequiresSTARTTLS(t *testing.T) {
	server := startFakeServer(t, func(s *fakeSMTPServer) {
		s.serverTLS = nil
		s.advertiseTLS = false
	})
	serverAddr := server.addr

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: 587,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, network, serverAddr)
		},
	})

	err := m.Send(context.Background(), "pak.bos@example.com", "Port 587 Test", "Body")
	if err == nil {
		t.Fatal("expected error on port 587 when server does not advertise STARTTLS")
	}
	if !strings.Contains(err.Error(), "STARTTLS required on port 587") {
		t.Errorf("expected error about STARTTLS required on port 587, got: %v", err)
	}
}

func TestSMTPMailer_ContextCancellation(t *testing.T) {
	server := startFakeServer(t, nil)
	port := parsePort(t, server.addr)

	m := mailer.NewSMTP(mailer.SMTPConfig{
		Host: "127.0.0.1",
		Port: port,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := m.Send(ctx, "pak.bos@example.com", "Cancelled", "Body")
	if err == nil {
		t.Fatal("expected error with cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}
