# qBit-Gluetun Sync Sidecar

A lightweight, event-driven sidecar proxy written in Go to synchronize the dynamic forwarded port from Gluetun (ProtonVPN) to qBitTorrent.

## Introduction
When using VPN providers like ProtonVPN with WireGuard through [Gluetun](https://github.com/qdm12/gluetun), the forwarded port changes dynamically. qBitTorrent needs to be updated with this new port to allow incoming connections. 

Instead of relying on heavy polling shell scripts, this project provides a **Go-based proxy** that:
- Sits in front of your qBitTorrent WebUI.
- Watches the `/tmp/gluetun/forwarded_port` file using `fsnotify` for instant, event-driven updates.
- Automatically calls the qBitTorrent API (`/api/v2/app/setPreferences`) to update the listening port whenever Gluetun changes it.
- Follows the single-process container container security pattern (runs as `nonroot` inside a minimal Google `distroless` image).
- Exposes useful sidecar endpoints like `/healthz` for Kubernetes probes.

## Configuration

The application is entirely driven by Environment Variables.

| Variable | Default | Description |
| :--- | :--- | :--- |
| `QBIT_ADDR` | `http://localhost:8080` | The actual address of your qBitTorrent Web UI. |
| `QBIT_USER` | *(empty)* | Username for qBitTorrent (if authentication bypass is disabled). |
| `QBIT_PASS` | *(empty)* | Password for qBitTorrent. |
| `PORT_FILE` | `/tmp/gluetun/forwarded_port` | Path to the port file created by Gluetun. |
| `LISTEN_PORT` | `9090` | The port this proxy wrapper will listen on. |

## Usage

### Docker Compose

In a compose file, you run `qbit-gluetun-sync` in the same network namespace as qBitTorrent, sharing the `/tmp/gluetun` volume.

```yaml
version: '3.8'

services:
  gluetun:
    image: qmcgaw/gluetun
    cap_add:
      - NET_ADMIN
    environment:
      - VPN_SERVICE_PROVIDER=protonvpn
      - VPN_TYPE=wireguard
      - WIREGUARD_PRIVATE_KEY=your_private_key
      - SERVER_COUNTRIES=Netherlands
      - VPN_PORT_FORWARDING=on
      - VPN_PORT_FORWARDING_STATUS_FILE=/tmp/gluetun/forwarded_port
      # This is an internal whitelist for the Gluetun container's firewall (iptables). 
      # It tells Gluetun: "Allow incoming traffic on port 8080 from the local network."
      - FIREWALL_INPUT_PORTS=8080,9090
    volumes:
      - gluetun_data:/tmp/gluetun
    ports:
      - "9090:9090" # Expose the Sync Proxy port, NOT the raw qBitTorrent port

  qbittorrent:
    image: lscr.io/linuxserver/qbittorrent:latest
    network_mode: "service:gluetun"
    environment:
      - WEBUI_PORT=8080
    volumes:
      - qbit_config:/config
      - qbit_downloads:/downloads

  sync-sidecar:
    image: ghcr.io/vehkiya/qbit-gluetun-sync:latest
    network_mode: "service:gluetun"
    environment:
      - QBIT_ADDR=http://localhost:8080
      - PORT_FILE=/tmp/gluetun/forwarded_port
      - LISTEN_PORT=9090
    volumes:
      - gluetun_data:/tmp/gluetun:ro
```

### Kubernetes (Sidecar Pattern)

If deploying in Kubernetes, deploy inside the same Pod as qBitTorrent with an `emptyDir` volume shared with Gluetun.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qbittorrent
spec:
  template:
    spec:
      containers:
        # 1. Gluetun Container
        - name: gluetun
          image: qmcgaw/gluetun
          env:
            - name: VPN_PORT_FORWARDING_STATUS_FILE
              value: /tmp/gluetun/forwarded_port
            # This is an internal whitelist for the Gluetun container's firewall (iptables). 
            # It tells Gluetun: "Allow incoming traffic on port 8080 from the local network."
            - name: FIREWALL_INPUT_PORTS
              value: "8080,9090"
          volumeMounts:
            - name: gluetun-sync
              mountPath: /tmp/gluetun

        # 2. qBitTorrent Container
        - name: qbittorrent
          image: lscr.io/linuxserver/qbittorrent:latest

        # 3. Sync Proxy Sidecar
        - name: qbit-sync
          image: ghcr.io/vehkiya/qbit-gluetun-sync:latest
          ports:
            - containerPort: 9090
              name: webui
          volumeMounts:
            - name: gluetun-sync
              mountPath: /tmp/gluetun
              readOnly: true
          livenessProbe:
            httpGet:
              path: /healthz
              port: 9090
      volumes:
        - name: gluetun-sync
          emptyDir: {}
```

## API Endpoints

Because `qbit-gluetun-sync` is a reverse proxy, accessing `http://<ip>:9090/` routes directly to your qBitTorrent WebUI. 

Additionally, we intercept the following endpoints for sidecar management:

- `GET /healthz` - Returns `200 OK` if the proxy wrapper is running. Useful for Docker health checks and Kubernetes probes.
- `GET /sync` - Manually triggers a read of the `PORT_FILE` and pushes it to qBitTorrent.

## Development

The project is built using Go completely natively. To develop locally:

```bash
git clone https://github.com/vehkiya/qbit-gluetun-sync.git
cd qbit-gluetun-sync
go test -v ./...
go build -o qbit-gluetun-sync ./cmd/sync
```
