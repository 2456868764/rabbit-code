# Regenerate `allowlist_shared.json`

Upstream: `claude-code-sourcemap/restored-src/src/utils/shell/readOnlyCommandValidation.ts`. For bundling without pulling `../platform.js`, keep a vendored copy with a local `getPlatform` shim (path may vary; point esbuild at that file).

```bash
cd rabbit-code
# Example when vendor copy exists:
# npx esbuild vendor/readonly_validation/readOnlyCommandValidation.ts \
#   --bundle --platform=node --format=esm --outfile=/tmp/rov_vendor.mjs
node tools/dump_allowlist_from_bundle.mjs
```

`dump_allowlist_from_bundle.mjs` writes **`internal/readonlycmd/allowlist_shared.json`** (consumed by **`go:embed`** in **`internal/readonlycmd/load.go`**).

Alternatively run **`npx tsx tools/dump_readonly_inner.ts`** (imports directly from **`claude-code-sourcemap/restored-src/.../readOnlyCommandValidation.ts`**) to write the same path.
