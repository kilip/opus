# Backup and Restore

Opus stores all persistent state in:

1. **SQLite database** (`APP_DATABASE_URL`, defaulting to `opus.db` in the working directory, or `/data/opus.db` in Docker/Helm).
2. **Opus home directory** (`$APP_HOME`, defaulting to `~/.opus`), which contains:
   - `auth_secret`: Auto-generated session signing secret (when `AUTH_SECRET` is unset).
   - `node.json`: Worker node credentials (on joined worker machines).
   - `repos/`: Cloned project Git repositories.
   - `wt/`: Active per-task Git worktrees.

## Online Backup (WAL-Safe)

Opus runs SQLite in WAL mode (`journal_mode=WAL`). Never copy a live `opus.db` file alone while the server is running. Use SQLite's online `.backup` command:

```bash
sqlite3 /path/to/opus.db ".backup '/path/to/opus-backup.db'"
```

Also back up `$APP_HOME/auth_secret` (if `AUTH_SECRET` is not set via environment variable) and `$APP_HOME/node.json` on worker nodes.

## Cold Backup

If the Opus server is stopped, you can copy the database and its WAL sidecars together:

```bash
cp opus.db opus.db-wal opus.db-shm /backup/dir/
```

## Restore

1. Stop the Opus server so no process holds open handles on the database.
2. Move the existing database files aside (including any `-wal` and `-shm` files).
3. Copy the verified backup into place and check integrity:

```bash
cp /path/to/opus-backup.db /path/to/opus.db
rm -f /path/to/opus.db-wal /path/to/opus.db-shm
sqlite3 /path/to/opus.db "PRAGMA integrity_check;"
```

4. Start Opus and verify `/api/readyz` returns `200`.

## Kubernetes Backup

Run an online SQLite backup from the running pod using `kubectl exec`:

```bash
kubectl exec -n opus deploy/opus -- \
  sqlite3 /data/opus.db ".backup '/tmp/opus-backup.db'"
kubectl cp opus/<pod-name>:/tmp/opus-backup.db ./opus-backup.db
kubectl exec -n opus deploy/opus -- rm -f /tmp/opus-backup.db
```

For automated backups, run a Kubernetes `CronJob` in the same namespace with the PVC mounted read-only or targeting a shared backup volume using `sqlite3 /data/opus.db ".backup '/backups/opus-$(date +%F).db'"`.
