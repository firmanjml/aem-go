# AEM contributor guide and roadmap

## Project purpose

Adaptive Environment Manager (AEM) is a Go CLI for installing and switching
project-specific development dependencies. Its initial focus is Node.js, Java
JDKs, and Android SDK components, while remaining framework-agnostic. Project
requirements belong in `aem.json`; AEM should install only what that file
declares and make repeated setup runs safe.

## Repository map

- `main.go` starts the CLI.
- `cmd/` defines Cobra commands and CLI output.
- `internal/setup/` orchestrates project setup.
- `internal/node/`, `internal/java/`, and `internal/android/` implement
  runtime/provider behavior.
- `internal/config/` finds and reads `aem.json`.
- `extensions/` provides remote-version discovery.
- `pkg/` contains shared filesystem, download, archive, process, state, log,
  error, and progress helpers.
- `assets/` contains CLI presentation assets.

## Development workflow

- Use Go and keep the module buildable with the Go version declared in
  `go.mod`.
- Format changed Go files with `gofmt -w <files>`.
- Verify changes with `go test ./...` and `go build ./...`.
- Do not make real installs against a developer's default AEM directory during
  automated checks. Set `AEM_HOME` and any `AEM_*_SYMLINK` overrides to a
  temporary test directory.
- Keep provider-specific download URLs, archive layouts, and host/architecture
  handling out of command handlers. Commands should delegate to services.
- Preserve cancellable work through `pkg/process` and return contextual errors
  instead of exiting from service code.

## Configuration contract

`aem setup` searches upward from the working directory for the nearest
`aem.json`. This is the canonical project configuration shape:

```json
{
  "$schema": "https://raw.githubusercontent.com/firmanjml/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4",
    "java": "17"
  },
  "android": {
    "platforms": ["35"],
    "buildTools": ["35.0.0"],
    "ndk": ["27.0.12077973"],
    "cmake": ["3.22.1"],
    "systemImages": [
      {
        "apiLevel": 35,
        "variant": "google_apis",
        "architecture": "arm64-v8a"
      }
    ]
  },
  "hooks": {
    "preSetup": [],
    "postSetup": []
  }
}
```

`$schema` enables editor validation. `runtime`, `android`, and `hooks` are
optional. Runtime versions belong in `runtime.node` and `runtime.java`;
Android package versions use the arrays in `android`; lifecycle hooks belong in
`hooks.preSetup` and `hooks.postSetup`. Keep the parser, setup implementation,
schema, tests, and README aligned whenever this contract changes.

## Environment behavior

- `AEM_HOME` defaults to `~/.aem` and stores managed installs.
- Active Node.js and Java runtimes are exposed through stable links below
  `$AEM_HOME/current/`. Android packages use the SDK selected by
  `ANDROID_HOME`.
- `AEM_NODE_SYMLINK` and `AEM_JAVA_SYMLINK` override the corresponding
  active-link paths.
- Keep setup idempotent: already valid installations should not be downloaded
  or reinstalled.

## Roadmap

Use this checklist as the project progress tracker. Mark an item complete only
when it is implemented, tested, and its user-facing documentation is accurate.

### Foundation

- [x] Build a Cobra-based Go CLI with shared filesystem, download, archive,
  logging, state, and process helpers.
- [x] Implement Node.js installation, activation, listing, current-state, and
  project setup support.
- [x] Implement Azul Zulu JDK installation, activation, listing, current-state,
  and project setup support.
- [x] Find the nearest `aem.json` and set up Node and Java concurrently.
- [x] Add `current`, `doctor`, `--debug`, and configurable active-link paths.
- [x] Add unit tests for configuration discovery/parsing, version resolution,
  filesystem state, and symlink behavior.
- [x] Add integration tests using temporary `AEM_HOME` directories and mocked
  remote downloads.

### Configuration and CLI parity

- [x] Implement `aem init` to generate and validate a project `aem.json`.
- [x] Implement `aem available <module>` as the documented remote-version
  command, or document `aem list` as its replacement.
- [x] Implement `aem uninstall <module> <version>` with safe checks that the
  selected runtime is not active.
- [x] Implement and validate the canonical nested `runtime` configuration,
  `$schema` reference, and `hooks.preSetup`/`hooks.postSetup` lifecycle fields.
- [x] Provide migration guidance from the legacy top-level `node`/`jdk` and
  Android `sdk`/`build-tool` fields.
- [x] Make command help, README examples, and supported command registration
  agree.

### Android support

- [x] Download Android command-line tools and install declared SDK, build-tool,
  and NDK packages during setup.
- [x] Accept SDK licenses and expose the managed SDK through the active Android
  link.
- [x] Add CMake, platform-tools, and system-image support.
- [x] Support the canonical `platforms`, `buildTools`, `cmake`, and
  `systemImages` Android configuration fields.
- [x] Add Android package resolution and setup tests without relying on a live
  Android repository.
- [x] Report installed Android components in `list`, `current`, and `doctor`.

### Cross-platform and release readiness

- [x] Make Node and Java remote metadata/download selection use the detected OS
  and architecture (the extension layer previously contained Windows/x64
  assumptions).
- [x] Verify archive extraction, executable permissions, symlinks/junctions,
  and environment activation on macOS, Linux, and Windows.
- [x] Document shell-specific `PATH`, `JAVA_HOME`, and `ANDROID_HOME`/SDK
  environment integration.
- [x] Add CI for formatting, tests, builds, and platform smoke tests.
- [x] Add versioning, release artifacts, checksums, and installation guidance.
- [x] Document supported providers, offline/cache behavior, upgrade policy, and
  troubleshooting.

## Definition of done

For each roadmap item, include focused tests, run the validation commands above,
update user documentation and command help, and avoid regressions to existing
managed installations.
