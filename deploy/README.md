# Deploy

## Install binary from the latest GitHub release

Releases: [github.com/aredoff/whoiscached/releases](https://github.com/aredoff/whoiscached/releases). Asset names: `whoiscached-<tag>-linux-amd64.tar.gz` and `...-linux-arm64.tar.gz`, plus `whoiscached-<tag>_checksums.txt`.

```bash
REPO=aredoff/whoiscached
VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
case "$(uname -m)" in
  x86_64)  ARCH=amd64 ;;
  aarch64) ARCH=arm64 ;;
  *) echo "unsupported arch: $(uname -m)"; exit 1 ;;
esac
cd /tmp
curl -sLO "https://github.com/${REPO}/releases/download/${VERSION}/whoiscached-${VERSION}-linux-${ARCH}.tar.gz"
curl -sLO "https://github.com/${REPO}/releases/download/${VERSION}/whoiscached-${VERSION}_checksums.txt"
grep "linux-${ARCH}.tar.gz" "whoiscached-${VERSION}_checksums.txt" | sha256sum -c  # optional
tar -xzf "whoiscached-${VERSION}-linux-${ARCH}.tar.gz"
sudo install -m 755 "whoiscached-${VERSION}-linux-${ARCH}" /usr/bin/whoiscached
```

Omit the `grep ... | sha256sum -c` line if you skip verification. Remove downloaded files from `/tmp` when done.

## systemd

1. Install `whoiscached` to `/usr/bin` as described in [Install binary from the latest GitHub release](#install-binary-from-the-latest-github-release).

2. Create user and directories:

   ```bash
   sudo useradd --system --home-dir /var/lib/whoiscache --shell /usr/sbin/nologin whoiscache
   sudo mkdir -p /etc/whoiscache /var/lib/whoiscache
   sudo chown whoiscache:whoiscache /var/lib/whoiscache
   ```

3. Copy config and set `snapshot_path` under `/var/lib/whoiscache/` (e.g. `snapshot_path = /var/lib/whoiscache/whoiscache.snap`).

   ```bash
   sudo cp deploy/config.ini.example /etc/whoiscache/config.ini
   sudo chmod 640 /etc/whoiscache/config.ini
   sudo chown root:whoiscache /etc/whoiscache/config.ini
   ```

   The snapshot file does **not** exist until the first time the store flushes to disk: after there is something to save (cached queries) and `snapshot_interval` elapses, or when the service stops. A missing file right after install is expected.

4. Install and enable the unit:

   ```bash
   sudo cp deploy/whoiscached.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now whoiscached
   ```

5. CLI maintenance (stop the service first if you need a consistent snapshot, or accept a short race):

   ```bash
   sudo -u whoiscache /usr/bin/whoiscached -config /etc/whoiscache/config.ini -dump-keys
   ```

`listen tcp :43: permission denied` happens because **ports below 1024** need `CAP_NET_BIND_SERVICE`. The shipped `whoiscached.service` sets `AmbientCapabilities=CAP_NET_BIND_SERVICE` so the daemon can listen on `:43` as `whoiscache`. Copy the updated unit, then `systemctl daemon-reload` and **`systemctl restart whoiscached`**.

To run the **server** manually as `whoiscache` (e.g. `sudo -u whoiscache /usr/bin/whoiscached ...`) without systemd, run once: **`sudo setcap 'cap_net_bind_service=+ep' /usr/bin/whoiscached`**, or set `[server] listen_addr` in `config.ini` to a high port (e.g. `:4343`) for ad hoc tests.
