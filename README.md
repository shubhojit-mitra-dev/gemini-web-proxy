# gemini-web-proxy

High-performance, zero-dependency Go reverse proxy that translates Google Gemini's web backend into standard OpenAI (`/v1/chat/completions`, `/v1/responses`) and Google Native (`/v1beta`) REST APIs.

Designed for seamless integration with AI coding assistants (e.g., OpenAI Codex CLI, Cherry Studio, Chatbox) without requiring paid API keys or complex setups.

## Features

- **Zero API Key Requirement**: Works using public Gemini web endpoints or optional session cookies.
- **OpenAI & Codex CLI Compatibility**: Fully compatible with OpenAI endpoints, including strict client validation rules (e.g., Codex v0.151.0 model metadata requirements).
- **Real-Time SSE Streaming**: Native Server-Sent Events streaming powered by Go goroutines for low latency.
- **Auto Token & BL Refresh**: Automatically fetches and refreshes `SNlM0e` XSRF tokens and backend build labels (`gemini_bl`).
- **Cross-Platform**: Compiles to a single binary with zero external runtime dependencies on Linux, macOS, and Windows.

## Quick Start

### Installation

Ensure Go 1.21+ is installed on your system.

```bash
git clone https://github.com/blackknight05/gemini-web-proxy.git
cd gemini-web-proxy
go build -o bin/gemini-web-proxy ./cmd/proxy
```

### Running the Proxy

```bash
./bin/gemini-web-proxy --port 8081
```

By default, the proxy listens at `http://localhost:8081/v1`.

### Integration with Codex CLI

Add the following to your `~/.codex/config.toml`:

```toml
model = "gpt-5.5"
model_provider = "gemini-web"

[model_providers.gemini-web]
name = "Gemini Web Proxy"
base_url = "http://localhost:8081/v1"
env_key = "GEMINI_WEB_API_KEY"
wire_api = "responses"
```

Then export a placeholder key and launch Codex:

```bash
export GEMINI_WEB_API_KEY="sk-gemini"
codex
```

## Configuration

Configuration can be specified via JSON configuration file, environment variables, or CLI flags.

| Flag | Env Variable | Default | Description |
| --- | --- | --- | --- |
| `--port` | `GEMINI_PROXY_PORT` | `8081` | Port to listen on |
| `--host` | `GEMINI_PROXY_HOST` | `0.0.0.0` | Bind address |
| `--cookie-file` | `GEMINI_PROXY_COOKIE_FILE` | `""` | Path to cookies file for authenticated sessions |
| `--proxy` | `GEMINI_PROXY_HTTP_PROXY` | `""` | Outbound HTTP/HTTPS proxy URL |

## Architecture

The project follows standard Go package layout guidelines:

- `cmd/proxy/`: Application entry point.
- `internal/config/`: Configuration loading and validation.
- `internal/auth/`: Cookie parsing, XSRF token management, and SAPISID hashing.
- `internal/gemini/`: Upstream protocol framing, JSON payload generation, and streaming response parsing.
- `internal/api/`: OpenAI and Google Native API HTTP handlers.
- `pkg/logger/`: Structured lightweight logging.

## License

MIT License.
