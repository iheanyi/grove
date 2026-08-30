# Releasing

The Git tag is the CLI release trigger. Do not start `release.yml` by hand to mint a tag.

From a clean, up-to-date `main`, run:

```sh
make release V=0.10.3
```

That creates and pushes annotated tag `v0.10.3`. GitHub Actions then builds the GitHub Release binaries and GoReleaser updates the Homebrew tap automatically.

For the next patch from the latest `v*` tag, you can run:

```sh
make release-patch
```

Menubar releases use the separate Swift pipeline:

```sh
make release-menubar
```

Do not commit generated Vite output; only the dashboard stub belongs in git.
