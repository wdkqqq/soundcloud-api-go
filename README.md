# SoundCloud API (Unofficial)

A lightweight Go REST API for fetching direct SoundCloud track stream URLs.

## Features

- Service health check: `GET /health`
- Stream URL endpoint:
  - `GET /soundcloud/stream-url?url=<track_url>`
  - `POST /soundcloud/stream-url` with JSON body
- Request rate limiting
- Logging to both file and stdout
- Automatic port fallback: if `PORT` is busy, the server starts on a free port

## Requirements

- Go `1.25+`
- SoundCloud `AUTH_TOKEN` and `CLIENT_ID`

## Quick Start

1. Create or update `.env` in the project root:

```env
PORT=5000
LOG_FILE=SC_API.log
DEBUG=false
AUTH_TOKEN=your_soundcloud_oauth_token
CLIENT_ID=your_soundcloud_client_id
```

2. Run the API:

```bash
go run ./cmd/api
```

3. Check health:

```bash
curl http://localhost:5000/health
```

If port `5000` is already in use, check logs for the actual port:
`Port :5000 is busy, using :<port>` and `Server starting on :<port>`.

## Configuration

All values are loaded from environment variables (and from `.env` if present).

- `AUTH_TOKEN` (required): SoundCloud OAuth token
- `CLIENT_ID` (required): SoundCloud client ID
- `PORT` (default: `5000`): preferred server port
- `LOG_FILE` (default: `SC_API.log`): log file path
- `DEBUG` (default: `false`): verbose logging
- `RATE_LIMIT_REQUESTS` (default: `100`): max requests per window
- `RATE_LIMIT_WINDOW` (default: `1h`): rate limit window (`time.ParseDuration` format)
- `REQUEST_TIMEOUT` (default: `30s`): external request timeout
- `MAX_TRACK_URL_LEN` (default: `500`): maximum accepted track URL length

## API

### `GET /health`

Returns service status and token validation result.

Example:

```bash
curl -s http://localhost:5000/health
```

Example response:

```json
{
  "status": "healthy",
  "service": "soundcloud-api",
  "timestamp": "2026-02-14T12:00:00Z",
  "version": "1.0.0"
}
```

### `GET /soundcloud/stream-url`

Query parameter:
- `url` (required): SoundCloud track URL

Example:

```bash
curl -s "http://localhost:5000/soundcloud/stream-url?url=https://soundcloud.com/artist/track"
```

### `POST /soundcloud/stream-url`

Request body:

```json
{
  "track_url": "https://soundcloud.com/artist/track"
}
```

Example:

```bash
curl -s -X POST "http://localhost:5000/soundcloud/stream-url" \
  -H "Content-Type: application/json" \
  -d '{"track_url":"https://soundcloud.com/artist/track"}'
```

Success responses include:
- `stream_url`
- `track_info`
- `cache_info`

Error responses include:
- `error`
- `error_code`

## Run with Docker

```bash
docker compose up --build
```

## Tests

```bash
go test ./...
```

## Contributing

See `CONTRIBUTING.md` for development workflow and PR expectations.

## Security

See `SECURITY.md` for responsible vulnerability disclosure.

## License

This project is licensed under the MIT License.
See `LICENSE`.
