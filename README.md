# gemini-web-proxy

High-performance, zero-dependency Go reverse proxy that translates Google Gemini's web backend into standard OpenAI (`/v1/chat/completions`, `/v1/responses`) and Google Native (`/v1beta`) REST APIs.

Designed for seamless integration with AI coding tools (e.g., OpenCode, OpenAI Codex CLI, Cherry Studio, Chatbox) without requiring paid API keys or complex setups.

## Features

- **Zero API Key Requirement**: Works out-of-the-box using public Gemini web endpoints or optional session cookies.
- **Dedicated Port**: Operates by default on port `58120` to prevent collisions with other local development services.
- **Automatic Cookie Discovery**: Automatically detects `cookies.json` in local working directory or `~/.config/gemini-web-proxy/cookies.json`.
- **OpenAI, OpenCode & Codex CLI Compatibility**: Fully compatible with OpenAI endpoints, including strict client validation rules (e.g., Codex v0.151.0 model metadata requirements).
- **Full Tool Execution Support**: Automatically translates function call schemas and tool executions (shell commands, file modifications) between OpenAI and Gemini.
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
./bin/gemini-web-proxy
```

By default, the proxy listens at `http://localhost:58120/v1`.

---

## Integration with OpenAI Codex CLI

### Important Setup Notice

Codex CLI requires custom providers (`model_provider`) to be set in your user-level configuration at `~/.codex/config.toml`. Setting `model_provider = "gemini-web"` globally routes Codex requests through your local Gemini proxy.

To switch back to native OpenAI models, comment out `model_provider = "gemini-web"` in `~/.codex/config.toml`.

### Step-by-Step Configuration

1. Edit your user-level configuration file at `~/.codex/config.toml`:

```toml
model = "gpt-5.5"
model_provider = "gemini-web"
approval = "never"
sandbox = "workspace-write"

[model_providers.gemini-web]
name = "Gemini Web Proxy"
base_url = "http://localhost:58120/v1"
env_key = "GEMINI_WEB_API_KEY"
wire_api = "responses"
```

2. Export the environment variable in your terminal session:

```bash
export GEMINI_WEB_API_KEY="sk-gemini"
```

3. Launch Codex:

```bash
codex
```

---

## In-Session Model Switching via `/model` Dropdown (Best DX)

Because Codex CLI v0.151.0 hardcodes its `/model` TUI dropdown menu to GPT model names, the proxy maps each TUI menu selection directly to a distinct Gemini operational mode:

| TUI Dropdown Selection | Mapped Gemini Backend Mode | Description |
| --- | --- | --- |
| **`gpt-5.5`** | **Gemini 3.7 Flash** | Standard all-around fast mode |
| **`gpt-5.6-sol`** | **Gemini 3.5 Flash Thinking** | **Deep Reasoning / Thinking Mode** |
| **`gpt-5.2`** | **Gemini 3.1 Pro** | **Pro Mode** (requires session cookies) |
| **`gpt-5.6-terra`** | **Gemini Flash Thinking Lite** | Adaptive depth thinking mode |
| **`gpt-5.6-luna`** | **Gemini Flash Lite** | Lightweight ultra-fast mode |

### How to switch modes in the middle of a session:
Press `/model` inside Codex CLI and pick:
- **`gpt-5.6-sol`** for **Deep Reasoning / Thinking Mode**
- **`gpt-5.2`** for **Gemini 3.1 Pro**
- **`gpt-5.5`** for **Gemini 3.7 Flash**

The proxy intercepts the selection instantly without needing to restart your session or type command line flags.

---

## Tool Execution Permissions

To ensure Codex permits Gemini to execute shell commands and read/modify files:
- Set `approval = "never"` (or `"auto"` / `"ask"`).
- Set `sandbox = "workspace-write"`.

---

## Integration with OpenCode

To connect OpenCode to `gemini-web-proxy`:

1. Open or create `~/.config/opencode/opencode.json` (or your project-level `opencode.json`).
2. Add the following provider configuration:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "autoupdate": false,
  "model": "gemini-web/gemini-3.7-flash",
  "provider": {
    "gemini-web": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Gemini Web Proxy",
      "options": {
        "baseURL": "http://localhost:58120/v1",
        "apiKey": "sk-gemini"
      },
      "models": {
        "gemini-3.7-flash": {
          "name": "Gemini 3.7 Flash"
        },
        "gemini-3.6-flash": {
          "name": "Gemini 3.6 Flash"
        },
        "gemini-3.1-pro": {
          "name": "Gemini 3.1 Pro"
        }
      }
    }
  }
}
```

3. Launch OpenCode and select `gemini-web/gemini-3.7-flash` from the model picker (`/models`).

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

The proxy will automatically sanitize and load `cookies.json` upon startup.

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
ExecStart=/home/YOUR_USERNAME/Desktop/gemini-web-proxy/bin/gemini-web-proxy --port 58120
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

## Configuration Reference

| Flag | Env Variable | Default | Description |
| --- | --- | --- | --- |
| `--port` | `GEMINI_PROXY_PORT` | `58120` | Port to listen on |
| `--host` | `GEMINI_PROXY_HOST` | `0.0.0.0` | Bind address |
| `--cookie-file` | `GEMINI_PROXY_COOKIE_FILE` | `""` | Path to cookies file for authenticated sessions |
| `--proxy` | `GEMINI_PROXY_HTTP_PROXY` | `""` | Outbound HTTP/HTTPS proxy URL |

---

## Project Structure

- `cmd/proxy/`: Main application entry point.
- `internal/config/`: Configuration loading and cookie auto-discovery.
- `internal/auth/`: Cookie parsing, XSRF token management, and SAPISID hashing.
- `internal/gemini/`: Upstream protocol framing, JSON payload generation, and streaming response parsing.
- `internal/api/`: OpenAI, Responses API, and Google Native API HTTP handlers.
- `internal/models/`: Model registry and strict client metadata mappings.
- `pkg/logger/`: Structured lightweight logging.

## License

MIT License.
