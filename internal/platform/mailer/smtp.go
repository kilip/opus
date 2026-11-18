package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig holds configuration for the SMTP mailer.
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	TLSConfig   *tls.Config                                                       // optional custom TLS configuration
	ImplicitTLS bool                                                              // forced implicit TLS (defaults to true when Port == 465)
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error) // optional custom dialer (e.g. for testing)
}

type smtpMailer struct {
	cfg SMTPConfig
}

// NewSMTP returns a Mailer that sends emails via SMTP.
func NewSMTP(cfg SMTPConfig) Mailer {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("%w: recipient address", ErrHeaderInjection)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("%w: subject", ErrHeaderInjection)
	}
	from := m.cfg.From
	if from == "" {
		if m.cfg.Host != "" {
			from = "noreply@" + m.cfg.Host
		} else {
			from = "noreply@localhost"
		}
	}
	if strings.ContainsAny(from, "\r\n") {
		return fmt.Errorf("%w: sender address", ErrHeaderInjection)
	}

	mailFrom := from
	if parsed, err := mail.ParseAddress(from); err == nil {
		mailFrom = parsed.Address
	}

	rcptTo := to
	if parsed, err := mail.ParseAddress(to); err == nil {
		rcptTo = parsed.Address
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	port := m.cfg.Port
	if port == 0 {
		port = 587
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(port))
	useImplicitTLS := port == 465 || m.cfg.ImplicitTLS

	tlsConfig := m.cfg.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{ServerName: m.cfg.Host}
	} else if tlsConfig.ServerName == "" {
		tlsConfig = tlsConfig.Clone()
		tlsConfig.ServerName = m.cfg.Host
	}

	// ponytail: net/smtp is frozen in Go stdlib; upgrade to modern mail client (e.g., github.com/wneessen/go-mail) when HTML templates, attachments, or DKIM are required
	var (
		c    *smtp.Client
		conn net.Conn
		err  error
	)

	const smtpTimeout = 10 * time.Second

	dialFunc := m.cfg.DialContext
	if dialFunc == nil {
		netDialer := &net.Dialer{Timeout: smtpTimeout}
		dialFunc = netDialer.DialContext
	}

	if useImplicitTLS {
		if m.cfg.DialContext != nil {
			rawConn, dialErr := m.cfg.DialContext(ctx, "tcp", addr)
			if dialErr != nil {
				return fmt.Errorf("smtp dial tls: %w", dialErr)
			}
			tlsConn := tls.Client(rawConn, tlsConfig)
			if handshakeErr := tlsConn.HandshakeContext(ctx); handshakeErr != nil {
				_ = rawConn.Close()
				return fmt.Errorf("smtp tls handshake: %w", handshakeErr)
			}
			conn = tlsConn
		} else {
			tlsDialer := &tls.Dialer{
				NetDialer: &net.Dialer{Timeout: smtpTimeout},
				Config:    tlsConfig,
			}
			conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return fmt.Errorf("smtp dial tls: %w", err)
			}
		}
		_ = conn.SetDeadline(time.Now().Add(smtpTimeout))
		c, err = smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp client: %w", err)
		}
	} else {
		conn, err = dialFunc(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		_ = conn.SetDeadline(time.Now().Add(smtpTimeout))
		c, err = smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp client: %w", err)
		}

		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				_ = c.Close()
				return fmt.Errorf("smtp starttls: %w", err)
			}
		} else if port == 587 {
			_ = c.Close()
			return errors.New("smtp: STARTTLS required on port 587 but not advertised by server")
		}
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	defer func() {
		_ = c.Close()
	}()

	hasCredentials := m.cfg.Username != "" || m.cfg.Password != ""
	_, isTLS := c.TLSConnectionState()

	if hasCredentials {
		if !isTLS {
			return ErrUnencryptedAuth
		}
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	normalizedBody := strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		from, to, subject, normalizedBody)

	if err := c.Mail(mailFrom); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(rcptTo); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := io.WriteString(w, msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	_ = c.Quit()
	return nil
}
