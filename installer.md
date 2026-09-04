# Deploying `50-a-org-documentation` as a Linux Background Service

**This guide runs everything as `root`, by request** — no dedicated service user. The systemd units are kept minimal: no memory/CPU limits, no filesystem sandboxing directives.

**Assumption used throughout:** the repo lives at `/50-a-org-documentation` (a subfolder directly under `/`).

**Runtime strategy:** unlike a Puppeteer/headless-browser scraper, `50-a-org-documentation` is a **plain Go program** (`main.go`) that walks pages with `net/http` + `golang.org/x/net/html` — no browser, no Chrome, no `xvfb`, no `zip`/`unzip` are required anywhere in this guide. The only runtime dependency is the Go toolchain itself.

**Two moving pieces in this repo, both covered below:**

1. **`main.go`** — the scraper. Fetches the NYPD units listing, walks every officer profile, downloads linked PDFs into `PDFs/<tax_id>/`, and updates the `CSVs/` datasets. It runs one full pass and **exits** — same shape as a cron job, not a long-running server. This guide compiles it once into a `50a-scraper` binary sitting in the same folder as `main.go` (step 8) and runs _that_ from systemd, rather than re-invoking `go run` (and therefore the Go compiler) on every hourly launch.
2. **`uploader.sh`** — a Bash script, **not Go**, that runs its own internal `while true` loop watching `git status`. It commits and pushes whatever the scraper produced whenever 25+ files have changed or 30 minutes have elapsed, whichever comes first. Unlike `main.go`, this one **never exits on its own** — it's already a daemon.

Because these two behave differently (one exits and needs re-launching, the other loops forever on its own), they get **two separate systemd units** in Part 2.

---

# Part 1: Installing the Go Toolchain and the Repo

## 1. Detect your CPU architecture

```bash
ARCH=$(dpkg --print-architecture)
echo "Detected architecture: $ARCH"
```

Returns `amd64` or `arm64`. Go's official toolchain has native, fully-supported builds for both — unlike the Puppeteer/Chrome situation in other projects on this org's account, there is **no ARM64 gap** to work around here. This is kept only in case you later want to cross-compile (`GOARCH=$ARCH`).

---

## 2. Update the package index

```bash
apt-get update && apt-get upgrade -y && apt-get install dist-upgrade -y
```

---

## 3. Install base system dependencies (including Go)

```bash
apt-get install -y --no-install-recommends ca-certificates curl git coreutils golang-go sudo
```

| Package           | Reason                                                                                                                                                                                                                                     |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ca-certificates` | Root certificates so Go's HTTP client and `git` can validate HTTPS/TLS connections.                                                                                                                                                        |
| `curl`            | General-purpose HTTPS fetch tool — handy for manual connectivity checks against `50-a.org`/GitHub while troubleshooting; not required by any command later in this guide since Go itself comes from `apt`, not a downloaded tarball.       |
| `git`             | Clones the repo (step 6) **and** is what `uploader.sh` shells out to on every cycle — required, not optional.                                                                                                                              |
| `coreutils`       | Provides the basic utilities this guide leans on throughout (`date`, `wc`, `tail`, `find`, etc.). Present on nearly every Ubuntu base image already, but installing explicitly guards against a minimal/stripped-down image that omits it. |
| `golang-go`       | The Go toolchain — compiles the scraper. Installed via `apt` rather than the upstream tarball, so it's patched the same way as every other package on the box (`apt-get upgrade golang-go`, see step 13).                                  |
| `sudo`            | Allows users to run commands with administrator/root privileges.                                                                                                                                                                           |
| `bash`            | Bourne Again SHell — a command-line shell used to execute commands and shell scripts.                                                                                                                                                      |

Nothing browser-related (`libnss3`, `libgbm1`, `fonts-liberation`, `xvfb`, etc.) is needed for this project — there's no headless Chrome in the dependency chain.

**Resolve the actual binary paths — don't assume a fixed location:**

```bash
BASH_BIN=$(command -v bash)
SUDO_BIN=$(command -v sudo)
GO_BIN=$(command -v go)
GIT_BIN=$(command -v git)

for var in BASH_BIN SUDO_BIN GO_BIN GIT_BIN; do
  if [ -z "${!var}" ]; then
    echo "ERROR: $var not found on PATH — install step for it failed or didn't complete."
  else
    echo "$var = ${!var}"
  fi
done

"$GO_BIN" version
```

The README asks for **Go 1.21 or later**, and `go.mod` in the repo pins the actual minimum. `apt`'s `golang-go` version tracks the Ubuntu release, so it can lag behind upstream:

- **Ubuntu 24.04 ("noble")** ships Go 1.22.x — satisfies the `1.21+` requirement.
- **Older Ubuntu (22.04 and earlier)** ships Go 1.18.x — check this against `go.mod`'s `go` directive before relying on it; if it's too old, either add the [official Go PPA](https://go.dev/wiki/Ubuntu) (`longsleep/golang-backports`) before this step, or fall back to installing the upstream tarball from [go.dev/dl](https://go.dev/dl/) instead of `apt`.

Check what you actually got and compare against `go.mod` once the repo is cloned (step 6):

```bash
grep '^go ' "$APP_DIR/go.mod" 2>/dev/null || echo "APP_DIR not cloned yet — check after step 6"
```

The systemd unit in step 10 uses `$GO_BIN`'s resolved path directly, not an assumed fixed location — so whichever install method you use, this stays correct.

---

## 4. Verify available disk space

```bash
df -h /
```

Confirm meaningfully more headroom than a typical scraper box: the README documents **3,000+ officer subfolders under `PDFs/`** (with the org's full archive reportedly holding 40,000+), plus growing `CSVs/` datasets that get split into `_part_N.csv` files past 100 MB each. Budget generously — tens of GB, not a few — and revisit the disk-space alert in step 13 as the archive grows.

---

## 5. Set the application directory

```bash
APP_DIR=/50-a-org-documentation
```

At this point your shell should have `$ARCH`, `$GO_BIN`, and `$GIT_BIN` all set.

---

## 6. Clone the repository

```bash
"$GIT_BIN" clone https://github.com/CoreData-Labs/50-a-org-documentation.git "$APP_DIR"
```

(`git clone <url> /` fails outright — always give it a destination folder name, not `/` itself.)

---

## 7. Fetch Go module dependencies and build the binary

```bash
cd "$APP_DIR"
"$GO_BIN" mod download
```

- The repo already ships `go.mod` / `go.sum` pinning `golang.org/x/net/html` — `go mod download` (or the implicit download that `go build` triggers) pulls exactly what's specified, no separate `go get` needed on a fresh clone.
- Unlike the Puppeteer case, there's no multi-hundred-MB browser download hiding in this step — Go module downloads for this repo are small.

**Build a real binary in the same folder, instead of leaning on `go run` at service-launch time:**

```bash
"$GO_BIN" build -o "$APP_DIR/50a-scraper" "$APP_DIR/main.go"
chmod +x "$APP_DIR/50a-scraper"
echo "Build OK: $APP_DIR/50a-scraper"
```

Why build once instead of `go run`-ning from source on every launch:

- `go run` recompiles from scratch on every invocation — wasted CPU on every hourly re-scrape, and it means the box needs the full Go toolchain and an intact module cache forever just to start the scraper.
- A prebuilt binary starts instantly, has no compile step that can fail at 3 AM, and is exactly what `systemctl status`/`journalctl` are reporting on if something goes wrong — no ambiguity between "the build failed" and "the scraper failed."
- The binary is a plain file at `$APP_DIR/50a-scraper`, sitting right next to `main.go`, `PDFs/`, and `CSVs/` — it does **not** get committed to the repo (see the `.gitignore` step right below), so `uploader.sh`'s `git add -A` won't try to push a compiled binary into version control.

**Keep the binary out of the uploader's commits:**

```bash
grep -qxF '50a-scraper' "$APP_DIR/.gitignore" || echo '50a-scraper' >> "$APP_DIR/.gitignore"
```

The repo's existing `.gitignore` predates this binary (it doesn't exist until you build it), so this adds one idempotent line — safe to re-run on every deploy.

If the build fails on a missing package, re-run `go mod download`, or `go mod tidy` if `go.sum` is out of sync.

---

## 8. Manual test run before wiring up systemd

**Always verify interactively before automating** — far easier to debug a visible failure here than in `journalctl` later.

Run the binary you just built, not `go run` — this is also your first real check that the build in step 7 actually works end-to-end:

```bash
cd "$APP_DIR"
"$APP_DIR/50a-scraper"
```

- `cd "$APP_DIR"` — **required.** `main.go` writes to the relative paths `./PDFs/`, `./CSVs/`, and `./downloaded.txt`. Running the binary from the wrong directory (e.g. `/`) causes permission or "no such file" errors on those relative writes — this is true of the compiled binary exactly as much as it was of `go run`.
- No display server, no `xvfb-run`, no browser flags — this is a plain HTTP scraper, so the command is just the binary itself.

Let it run for a few minutes and confirm files are actually appearing:

```bash
find "$APP_DIR/PDFs" -name "*.pdf" | wc -l
tail -n 5 "$APP_DIR/downloaded.txt"
```

**Resuming a partial run:** if you stop the scraper partway through, `main.go` exposes a `startingPercentage` variable (see the README's "Resuming a Scrape" section) that skips ahead to a known checkpoint in the 481-unit list instead of re-scraping from the top. Leave it at `0.0` for a full run from the beginning.

This guide's systemd unit (step 10a) still points at the standalone `$APP_DIR/50a-scraper` binary either way, since that's the fastest, least failure-prone thing for systemd to launch directly.

---

## 9. Test the uploader separately

`uploader.sh` needs working `git` push credentials for whatever account owns commits to this fork — set those up (SSH deploy key or a stored HTTPS credential/PAT for `root`) before testing, or every push attempt will fail with an auth error that the script logs and simply retries on the next cycle.

```bash
"$GIT_BIN" config --global user.name  "your-bot-name"
"$GIT_BIN" config --global user.email "your-bot-email@example.com"
```

Then do a short interactive run and confirm you see it pull, stage, and (once real changes accumulate) commit/push:

```bash
cd "$APP_DIR"
timeout 90 bash uploader.sh
```

`uploader.sh` is already an infinite loop with its own `sleep 60` between checks — the `timeout 90` above just cuts the manual test short; don't add that to the systemd unit.

---

## 10. The systemd service files

### 10a. Scraper service — exits and needs periodic re-launch

Same shape as the Puppeteer-based project: `main.go` runs one pass and exits, so `Restart=always` + a `RestartSec` delay turns that into a recurring job instead of a server. This unit launches the **prebuilt binary from step 7**, not `go run` — systemd shouldn't be invoking a compiler on every cycle.

```bash
tee /etc/systemd/system/50a-scraper.service > /dev/null <<EOF
[Unit]
Description=50-a.org NYPD Records Scraper (50-a-org-documentation)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/50a-scraper
Restart=always
RestartSec=3600
StandardOutput=journal
StandardError=journal
SyslogIdentifier=50a-scraper

[Install]
WantedBy=multi-user.target
EOF
```

| Directive                                 | Why it's set this way                                                                                                                                                                                                                                                       |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `After=`/`Wants=network-online.target`    | Don't start before the network is usable.                                                                                                                                                                                                                                   |
| `Type=simple`                             | The process in `ExecStart` _is_ the main process.                                                                                                                                                                                                                           |
| _(no `User=`/`Group=`)_                   | Runs as `root`, systemd's default — the deliberate simplification for this deployment.                                                                                                                                                                                      |
| `WorkingDirectory=`                       | The binary writes to the relative paths `./PDFs/`, `./CSVs/`, `./downloaded.txt` — must point at the repo root.                                                                                                                                                             |
| `ExecStart=`                              | Points directly at `$APP_DIR/50a-scraper`, the binary built in step 7 — no `$GO_BIN`, no `Environment=GOPATH=`, and no Go toolchain needed on the box at _run_ time at all (only at _build_ time). Starts in milliseconds instead of paying a compile cost on every launch. |
| `Restart=always` + `RestartSec=3600`      | The binary finishes and exits normally — this re-runs it an hour after each pass. **The one setting worth tuning** to your desired re-scrape frequency.                                                                                                                     |
| `StandardOutput=`/`StandardError=journal` | Logs go to `journalctl` — filterable, timestamped, rotated automatically.                                                                                                                                                                                                   |
| `SyslogIdentifier=`                       | Tags log lines for `journalctl -t 50a-scraper`.                                                                                                                                                                                                                             |
| `WantedBy=multi-user.target`              | Starts automatically at boot.                                                                                                                                                                                                                                               |

**If you ever need to fall back to `go run`** (e.g. quick one-off debugging without rebuilding), swap `ExecStart=` for `$GO_BIN run $APP_DIR/main.go` and add back `Environment=GOPATH=/root/go` — but that's a debugging exception, not how this unit should run day to day.

### 10b. Uploader service — already a daemon, no restart delay needed

`uploader.sh` never exits on its own (it's a `while true` loop with an internal `sleep 60`), so this unit is the opposite of the scraper's: `RestartSec` here is only a safety net if the script crashes, not a scheduling mechanism.

```bash
tee /etc/systemd/system/50a-uploader.service > /dev/null <<EOF
[Unit]
Description=50-a-org-documentation Auto Git Sync Uploader
After=network-online.target 50a-scraper.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$SUDO_BIN $BASH_BIN $APP_DIR/uploader.sh
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=50a-uploader

[Install]
WantedBy=multi-user.target
EOF
```

| Directive                          | Why it's set this way                                                                                                                                             |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `After=50a-scraper.service`        | Purely ordering — the uploader can start before the scraper has produced anything (it just finds nothing to commit yet), but starting it after is tidier on boot. |
| `WorkingDirectory=`                | `uploader.sh` runs `git status`/`git pull`/`git add`/`git commit`/`git push` against the current directory — must be the repo root.                               |
| `ExecStart=`                       | Runs the script directly with `bash`; it's already self-looping, so nothing wraps it.                                                                             |
| `Restart=always` + `RestartSec=30` | Short restart delay — this only fires if the script itself crashes (e.g. an unhandled `git` failure path), not as part of normal operation.                       |
| `SyslogIdentifier=`                | Tags log lines for `journalctl -t 50a-uploader`.                                                                                                                  |

**On the "same code" note:** if you're pointing `uploader.sh` at another one of this org's projects too, it's the identical script — nothing project-specific is hardcoded inside it beyond "the current git working directory," so the only thing that changes between deployments is `WorkingDirectory=` in the unit file above.

---

## 11. Reload systemd and start both services

```bash
systemctl daemon-reload
systemctl enable --now 50a-scraper.service
systemctl enable --now 50a-uploader.service
```

---

## 12. Verifying it's actually working

```bash
systemctl status 50a-scraper.service
systemctl status 50a-uploader.service
journalctl -u 50a-scraper.service -f
journalctl -u 50a-uploader.service -f
watch -n 30 'find /50-a-org-documentation/PDFs -name "*.pdf" | wc -l'
```

---

## 13. Production monitoring (independent of the service files themselves)

**Log rotation:** `journalctl --disk-usage` periodically; cap with `SystemMaxUse=500M` under `[Journal]` in `/etc/systemd/journald.conf` if needed.

**Disk space alerting**, since `./PDFs/` and `./CSVs/` grow unbounded and this archive is documented as being considerably larger than a typical scrape target:

```bash
# /etc/cron.d/disk-space-check
0 * * * * root df / | awk 'NR==2 && $5+0 > 85 {print "Disk usage high: "$5}' | logger -t disk-check
```

**Keeping Go current:** since `golang-go` came from `apt` (step 3), patching it is a normal `apt-get upgrade golang-go`, same as any other system package — no separate download-and-swap dance needed.

**Push auth expiry:** if `uploader.sh` starts logging `[ERROR] Failed to push changes` repeatedly, check `git` credentials for the `root` user first (expired PAT, revoked SSH key) before assuming anything is wrong with the scraper.

**Cron-style schedule instead of a restarting daemon (scraper only):** for a fixed time (e.g. nightly 2 AM) instead of "an hour after the last run," switch `50a-scraper.service`'s `Type=simple`+`Restart=always` to `Type=oneshot` (drop `Restart=`/`RestartSec=`) and pair with a `.timer` unit using `OnCalendar=*-*-* 02:00:00`. This doesn't apply to the uploader, which is designed to run continuously.

---

## 14. Troubleshooting

| Symptom                                                                      | Likely cause / fix                                                                                                                                                                                            |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `systemctl start 50a-scraper.service` fails with "No such file or directory" | The binary at `$APP_DIR/50a-scraper` was never built, or was built somewhere else — re-run step 7's `go build` and confirm `ls -la $APP_DIR/50a-scraper` shows an executable file.                            |
| Scraper writes nothing to `PDFs/`/`CSVs/`                                    | Check `WorkingDirectory=` matches `$APP_DIR` exactly; a relative-path mismatch fails silently into the wrong directory or errors on missing folders.                                                          |
| Uploader keeps committing `50a-scraper` as a binary blob                     | The `.gitignore` line from step 7 wasn't added before the binary was built, or was added after a first accidental commit — add the line, then `git rm --cached 50a-scraper` once to untrack it going forward. |
| Uploader logs `[ERROR] Failed to pull/rebase`                                | Local uncommitted changes conflicting with upstream, or missing credentials — resolve manually with `git status`/`git pull` as `root` in `$APP_DIR`, then let the service resume.                             |
| Uploader logs `[ERROR] Failed to push`                                       | Auth failure (expired token/key) or a protected branch — verify `git push` works manually as `root` first.                                                                                                    |
| `go build` complains about Go version                                        | `apt`'s `golang-go` (step 3) is older than what `go.mod` requires — add the official Go PPA or install from `go.dev/dl` instead, then re-run step 7.                                                          |
