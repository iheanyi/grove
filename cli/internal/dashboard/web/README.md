# Grove dashboard

This is the embedded dashboard, served by the CLI.

## Development

```sh
npm run dev
```

In another terminal, from `cli/`:

```sh
grove dashboard --dev
```

Or `make dev-dashboard` (builds the Go binary without the Vite bundle).

## Production embed

`npm run build` writes `build/`. Then, from `cli/`:

```sh
go build -tags embedui
```

`make build` does both (`npm install && npm run build`, then `go build -tags embedui`).

## go install

`go install` embeds `web/stub` only (no Node, no stale Vite bundle). GitHub release and Homebrew binaries embed the real UI via `-tags embedui` after `npm run build`.

Never commit `web/build`.
