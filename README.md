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

## Model Selection & Compatibility Mappings

When querying the proxy root status (`http://localhost:58120/`), the proxy lists native Gemini models:

- `gemini-3.7-flash` (Default all-around fast model)
- `gemini-3.6-flash`
- `gemini-3.5-flash-thinking` (Deep thinking mode)
- `gemini-3.1-pro` (Requires session cookies for routing)
- `gemini-flash-lite`

For compatibility with strict AI coding tools (such as OpenAI Codex CLI v0.151.0) that enforce hardcoded OpenAI model names, the proxy also accepts mock model aliases (`gpt-5.5`, `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`, `gpt-5.2`) and routes them directly to Gemini 3.7 Flash.

---

## Integration with OpenAI Codex CLI

### Tool Access & Sandbox Permissions

For Codex CLI to execute shell commands and modify files using Gemini, ensure your Codex sandbox and approval settings permit tool execution.

Add the following to your Codex configuration file:

```toml
approval = "never"  # Options: "never", "auto", or "ask"
sandbox = "workspace-write"
```

If tool access is disabled by Codex's sandbox, Codex will notify the model that shell tools are unavailable.

---

### Project-Level Setup (Recommended DX)

Rather than overriding your global Codex configuration for all projects, you can enable Gemini on a per-project basis.

Place a `.codex/config.toml` file inside your project directory:

```toml
model = "gemini-3.7-flash"
model_provider = "gemini-web"

[model_providers.gemini-web]
name = "Gemini Web Proxy"
base_url = "http://localhost:58120/v1"
env_key = "GEMINI_WEB_API_KEY"
wire_api = "responses"
```

Export the placeholder environment variable and launch Codex inside that folder:

```bash
export GEMINI_WEB_API_KEY="sk-gemini"
codex
```

This ensures outside your project folder, Codex continues using your default OpenAI setup, while inside your project folder, Codex routes through your free local Gemini proxy.

---

### Global Setup (Optional)

If you prefer to route all Codex CLI sessions globally through Gemini, add the provider block to `~/.codex/config.toml`:

```toml
model = "gemini-3.7-flash"
model_provider = "gemini-web"

[model_providers.gemini-web]
name = "Gemini Web Proxy"
base_url = "http://localhost:58120/v1"
env_key = "GEMINI_WEB_API_KEY"
wire_api = "responses"
```

---

### Switching Models in Codex CLI

#### Method 1: Using the `-m` CLI Flag (Recommended)

You can launch Codex with any model directly from the command line:

```bash
# Gemini 3.7 Flash
codex -m gemini-3.7-flash

# Gemini 3.5 Flash Thinking (Deep Thinking Mode)
codex -m gemini-3.5-flash-thinking

# Gemini 3.1 Pro (Requires session cookies)
codex -m gemini-3.1-pro

# GPT Compatibility Alias
codex -m gpt-5.5
```

#### Method 2: TUI Behavior Note

In Codex v0.151.0, pressing `/model` in the TUI opens a hardcoded list of OpenAI models (`gpt-5.6-sol`, `gpt-5.5`). If you select a model from that dropdown, Codex overwrites your selected model in its local state. To return to Gemini 3.7 Flash, relaunch Codex using `codex -m gemini-3.7-flash` or set `model = "gemini-3.7-flash"` in your `.codex/config.toml`.

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
