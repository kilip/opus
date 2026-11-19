# 🛡️ Security Policy

## 📌 Supported Versions

Security updates are applied to the latest release and the `main` branch.

| Version | Supported |
| :--- | :--- |
| `0.3.x` / `main` | ✅ |

---

## 🚨 Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately via [GitHub Security Advisories](https://github.com/kilip/opus/security/advisories/new).

When reporting, please include:

- A clear description of the vulnerability and its potential impact
- Steps to reproduce or a minimal proof of concept
- Any suggested fix or mitigation, if you have one

We will acknowledge your report as soon as possible and work with you to validate and address the issue before any public disclosure.

---

## 🔐 Deployment Security Notes

- Always set a strong, random `AUTH_SECRET` in production.
- Treat worker node join tokens and credentials (`node.json`) as secrets.
- Restrict `APP_TRUSTED_ORIGINS` and `APP_BASE_URL` to your actual deployment domain.
