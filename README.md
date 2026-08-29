# gemini-web-proxy

High-performance, zero-dependency Go reverse proxy that translates Google Gemini's web backend into standard OpenAI (`/v1/chat/completions`, `/v1/responses`) and Google Native (`/v1beta`) REST APIs.

Designed for seamless integration with AI coding tools (e.g., OpenAI Codex CLI, Cherry Studio, Chatbox) without requiring paid API keys or complex setups.

## Features

- **Zero API Key Requirement**: Works out-of-the-box using public Gemini web endpoints or optional session cookies.
- **Automatic Cookie Discovery**: Automatically detects `cookies.json` in local working directory or `~/.config/gemini-web-proxy/cookies.json`.
- **OpenAI & Codex CLI Compatibility**: Fully compatible with OpenAI endpoints, including strict client validation rules (e.g., Codex v0.151.0 model metadata requirements).
- **Real-Time SSE Streaming**: Native Server-Sent Events streaming powered by Go goroutines for low latency.
- **Auto Token & BL Refresh**: Automatically fetches and refreshes `SNlM0e` XSRF tokens and backend build labels (`gemini_bl`).
- **Cross-Platform**: Compiles to a single binary with zero external runtime dependencies on Linux, macOS, and Windows.

## Quick Start

### Installation

Ensure Go 1.21+ is installed on your system.

```bash
git clone https://github.com/shubhojit-mitra-dev/gemini-web-proxy.git
cd gemini-web-proxy
make build
```

The compiled binary will be placed at `bin/gemini-web-proxy`.

### Running the Proxy

```bash
./bin/gemini-web-proxy --port 8081
```

By default, the proxy listens at `http://localhost:8081/v1`.

---

## Authentication & Cookies Setup (Optional)

While basic public queries work anonymously, adding session cookies allows access to model persistence, higher quota limits, and Pro routing.

### How to Add Your Cookies

1. Export your cookies from `https://gemini.google.com` using a browser extension (such as "Cookie-Editor" or "Export cookies as JSON").
2. Save the file as `cookies.json` inside your working directory or at `~/.config/gemini-web-proxy/cookies.json`.

Alternatively, copy `cookies.json.example` and populate it:

```json
{
  "cookie": "__Secure-1PSID=...; __Secure-3PSID=...; SAPISID=...;",
  "sapisid": "..."
}
```

The proxy will automatically detect and load `cookies.json` upon startup.

---

## Systemd User Service Setup (Linux)

To run `gemini-web-proxy` automatically as a background daemon on Linux:

1. Create a systemd user service file at `~/.config/systemd/user/gemini-web-proxy.service`:

```ini
[Unit]
Description=Gemini Web Proxy Daemon
After=network.target

[Service]
Type=simple
WorkingDirectory=/home/YOUR_USERNAME/Desktop/gemini-web-proxy
ExecStart=/home/YOUR_USERNAME/Desktop/gemini-web-proxy/bin/gemini-web-proxy --port 48392
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
```

2. Enable and start the service:

```bash
systemctl --user daemon-reload
systemctl --user enable --now gemini-web-proxy.service
```

3. Check service status:

```bash
systemctl --user status gemini-web-proxy.service
```

---

## Integration with OpenAI Codex CLI

1. Add the following provider configuration to `~/.codex/config.toml`:

```toml
model = "gpt-5.5"
model_provider = "gemini-web"

[model_providers.gemini-web]
name = "Gemini Web Proxy"
base_url = "http://localhost:48392/v1"
env_key = "GEMINI_WEB_API_KEY"
wire_api = "responses"
```

2. Export a placeholder API key and launch Codex:

```bash
export GEMINI_WEB_API_KEY="sk-gemini"
codex
```

---

## Configuration Reference

Configuration can be specified via JSON configuration file, environment variables, or CLI flags.

| Flag | Env Variable | Default | Description |
| --- | --- | --- | --- |
| `--port` | `GEMINI_PROXY_PORT` | `8081` | Port to listen on |
| `--host` | `GEMINI_PROXY_HOST` | `0.0.0.0` | Bind address |
| `--cookie-file` | `GEMINI_PROXY_COOKIE_FILE` | `""` | Path to cookies file for authenticated sessions |
| `--proxy` | `GEMINI_PROXY_HTTP_PROXY` | `""` | Outbound HTTP/HTTPS proxy URL |

---

## Project Structure

The project follows standard Go package layout guidelines:

- `cmd/proxy/`: Main application entry point.
- `internal/config/`: Configuration loading and cookie auto-discovery.
- `internal/auth/`: Cookie parsing, XSRF token management, and SAPISID hashing.
- `internal/gemini/`: Upstream protocol framing, JSON payload generation, and streaming response parsing.
- `internal/api/`: OpenAI, Responses API, and Google Native API HTTP handlers.
- `internal/models/`: Model registry and strict client metadata mappings.
- `pkg/logger/`: Structured lightweight logging.

## License

MIT License.
