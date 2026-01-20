# Security Policy

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to security@skyhook.io.

Include as much of the following information as possible:

- Type of issue (e.g., privilege escalation, information disclosure)
- Full paths of source file(s) related to the issue
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

## Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 7 days
- **Fix timeline**: Depends on severity, typically within 30-90 days

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Security Considerations

Skyhook Explorer is designed with security in mind:

- **Read-only access**: Explorer only reads from the Kubernetes API; it never modifies resources
- **Local execution**: Runs on your machine, no data sent to external servers
- **Kubeconfig respect**: Uses your existing kubeconfig and RBAC permissions
- **No persistent storage**: No data is stored between sessions

## Best Practices

When using Explorer:

1. Use a kubeconfig with minimal required permissions (read-only)
2. Run Explorer locally, not exposed to the internet
3. Keep Explorer updated to the latest version
