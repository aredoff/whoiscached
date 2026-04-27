# Deploy

## systemd

1. Build and install the binary (example):

   ```bash
   install -m 755 -D ./bin/whoiscached /usr/bin/whoiscached
   ```

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
