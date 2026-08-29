# gemini-web-proxy

High-performance, zero-dependency Go reverse proxy that translates Google Gemini's web backend into standard OpenAI (`/v1/chat/completions`, `/v1/responses`) and Google Native (`/v1beta`) REST APIs.

Designed for seamless integration with AI coding tools (e.g., OpenCode, OpenAI Codex CLI, Cherry Studio, Chatbox) without requiring paid API keys or complex setups.

## Features

- **Zero API Key Requirement**: Works out-of-the-box using public Gemini web endpoints or optional session cookies.
- **Dedicated Default Port**: Operates by default on port `58120` to prevent collisions with other local development services.
- **Automatic Cookie Discovery**: Automatically detects `cookies.json` in the local working directory or at `~/.config/gemini-web-proxy/cookies.json`.
- **OpenAI, OpenCode & Codex CLI Compatibility**: Fully compatible with OpenAI endpoints, including strict client metadata validation rules (e.g., Codex v0.151.0).
- **Full Tool Execution Support**: Automatically translates function call schemas and tool executions (shell commands, file modifications) between OpenAI and Gemini.
- **Real-Time SSE Streaming**: Native Server-Sent Events streaming powered by Go goroutines for low latency.
- **Auto Token & BL Refresh**: Automatically fetches and refreshes `SNlM0e` XSRF tokens and backend build labels (`gemini_bl`).
- **Production-Grade Tooling**: Includes Docker containerization, systemd user service manifests, and GitHub Actions CI pipelines.

---

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

### Docker Usage

```bash
# Build Docker Image
docker build -t gemini-web-proxy .

# Run Container
docker run -d -p 58120:58120 --name gemini-web-proxy gemini-web-proxy
```

---

## Integration with OpenAI Codex CLI

### User-Level Configuration Requirement

Codex CLI security policy **explicitly ignores** `model_provider` and `model_providers` keys in project-local `.codex/config.toml` files. Custom provider settings **must** be declared in your user-level configuration at `~/.codex/config.toml`.

### Step-by-Step Configuration

1. Edit your user-level configuration file at `~/.codex/config.toml`:

```toml
model = "gpt-5.6-sol"
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

3. Launch Codex CLI:

```bash
codex
```

---

## In-Session Model Switching via `/model` Dropdown

In Codex CLI v0.151.0, pressing `/model` inside the TUI displays a hardcoded list of model choices. The proxy maps each TUI menu item directly to its corresponding Gemini operational backend:

| Codex TUI Choice | Description in Codex | Mapped Gemini Backend | Purpose / Capability |
| :--- | :--- | :--- | :--- |
| **`1. gpt-5.6-sol`** | Latest frontier agentic coding model | **Gemini 3.1 Pro** | **Pro Mode** (requires session cookies) |
| **`2. gpt-5.6-terra`** | Balanced agentic coding model for everyday work | **Gemini 3.5 Flash Thinking** | **Deep Reasoning / Thinking Mode** |
| **`3. gpt-5.6-luna`** | Fast and affordable agentic coding model | **Gemini Flash Thinking Lite** | **Adaptive Depth Thinking** |
| **`4. gpt-5.5`** | Frontier model for complex coding, research | **Gemini 3.7 Flash** | **Standard All-Around Fast Mode** |
| **`5. gpt-5.4`** | Strong model for everyday coding | **Gemini 3.6 Flash** | **Reliable Flash Mode** |
| **`6. gpt-5.4-mini`** | Small, fast, and cost-efficient model | **Gemini Flash Lite** | **Lightweight Ultra-Fast Mode** |

### How to switch models mid-session:
While inside an active Codex CLI session, press `/model` and pick:
- **`1. gpt-5.6-sol`** to switch instantly to **Gemini 3.1 Pro Mode**.
- **`2. gpt-5.6-terra`** to switch instantly to **Gemini 3.5 Flash Thinking (Deep Reasoning)**.
- **`4. gpt-5.5`** to switch instantly to **Gemini 3.7 Flash**.

The proxy intercepts your choice dynamically without requiring session restarts!

---

## Toggling Between Gemini & Native OpenAI Models

- **To use free Gemini models**: Keep `model_provider = "gemini-web"` active in `~/.codex/config.toml`.
- **To use native OpenAI models**: Comment out `# model_provider = "gemini-web"` in `~/.codex/config.toml`.

---

## Tool Execution Permissions

To ensure Codex permits Gemini to execute shell commands and read/modify files:
- Set `approval = "never"` (or `"auto"` / `"ask"` depending on security preferences).
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

While basic public queries work anonymously, adding session cookies allows access to model persistence, higher quota limits, and Pro routing (`gemini-3.1-pro`).

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
| :--- | :--- | :--- | :--- |
| `--port` | `GEMINI_PROXY_PORT` | `58120` | Port to listen on |
| `--host` | `GEMINI_PROXY_HOST` | `0.0.0.0` | Bind address |
| `--cookie-file` | `GEMINI_PROXY_COOKIE_FILE` | `""` | Path to cookies file for authenticated sessions |
| `--proxy` | `GEMINI_PROXY_HTTP_PROXY` | `""` | Outbound HTTP/HTTPS proxy URL |

---

## Development & Quality Assurance

```bash
# Run tests
make test

# Format code
make fmt

# Run linters
golangci-lint run
```

---

## License

[MIT License](LICENSE) &copy; 2026 Shubhojit Mitra.
