# Kubernetes Admin Credentials

markovd creates an `admin` user on first startup when the database has no users. By default it generates a random password and logs it to stdout. In Kubernetes, you can provide the password via a Secret instead.

Two environment variables control this (checked in order):

| Variable | Description |
|----------|-------------|
| `MARKOVD_ADMIN_PASSWORD` | Admin password value (e.g., from a Secret `secretKeyRef`) |
| `MARKOVD_ADMIN_PASSWORD_FILE` | Path to a file containing the password (e.g., a mounted Secret) |

If both are set, `MARKOVD_ADMIN_PASSWORD` takes precedence. If neither is set, a random password is generated and logged.

## Option 1: Environment Variable

Create a Secret and reference it in the Deployment:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: markovd-admin
type: Opaque
stringData:
  password: "your-secure-password"
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: markovd
spec:
  template:
    spec:
      containers:
        - name: markovd
          env:
            - name: MARKOVD_ADMIN_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: markovd-admin
                  key: password
```

## Option 2: Mounted File

Create the same Secret, but mount it as a volume:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: markovd-admin
type: Opaque
stringData:
  password: "your-secure-password"
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: markovd
spec:
  template:
    spec:
      containers:
        - name: markovd
          env:
            - name: MARKOVD_ADMIN_PASSWORD_FILE
              value: /etc/markovd/secrets/password
          volumeMounts:
            - name: admin-secret
              mountPath: /etc/markovd/secrets
              readOnly: true
      volumes:
        - name: admin-secret
          secret:
            secretName: markovd-admin
```

This approach is preferred in production because environment variables can leak through `/proc`, crash dumps, or container inspection.

## Notes

- The admin user is only created when the database has zero users. Subsequent restarts skip creation entirely.
- When a password is provided via env var or file, markovd does **not** log the password to stdout.
- Whitespace is trimmed from file contents, so a trailing newline in the Secret value is fine.
