# Grove dashboard

This SvelteKit UI is embedded in the Grove CLI binary.

## Local development

Start the Vite dev server:

```sh
npm run dev
```

Then from `cli/`, run the dashboard against it:

```sh
grove dashboard --dev
```

## Production embed

CI and GitHub Releases run `npm run build`, then Go embeds `web/build` via `//go:embed all:web/build`. Goreleaser `before.hooks` do the same npm ci/build so Homebrew and GitHub release binaries ship the full UI.

Git tracks only the stub `build/index.html` so `go install`, `go test`, and `go build` compile without Node. Never commit `_app` or hashed Vite files.

To package the full dashboard into a local binary:

```sh
make build
```

from `cli/`.
