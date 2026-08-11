# Adaptive Environment Manager (AEM)

Adaptive Environment Manager (AEM) is a tool designed to simplify and automate the management of development environment dependencies such as Node.js, Java, and Android SDK components.

AEM is especially useful for developers working with frameworks such as React Native, where managing multiple versions of development tools across projects can become cumbersome and time-consuming.

By automating environment setup and version management, AEM enables developers to spend less time configuring development environments and more time building software.

[AEM Demo](https://github.com/user-attachments/assets/216dd516-6744-480b-aecd-a063334166c1)

## Table of Contents

* [Overview](#overview)
* [Features](#features)
* [Supported Packages](#supported-packages)
* [Installation](#installation)
* [Usage](#usage)
* [Project Configuration](#project-configuration)
* [Android Configuration](#android-configuration)
* [Setup Hooks](#setup-hooks)
* [Environment Setup](#environment-setup)
* [Shell Environment Integration](#shell-environment-integration)
* [Providers, Cache, and Upgrades](#providers-cache-and-upgrades)
* [Troubleshooting](#troubleshooting)
* [Releases](#releases)
* [Contribution](#contribution)
* [Motivation](#motivation)
* [Future Plans](#future-plans)
* [Contact](#contact)

## Overview

Modern development environments often require multiple versions of Node.js, Java JDK, Android SDK platforms, Android NDK, Build Tools, and other development dependencies.

Different projects may require different versions of these tools. Manually installing, configuring, and switching between them can be error-prone and time-consuming.

AEM provides a centralized way to install, manage, and switch between development environment dependencies.

AEM can:

* Install and manage multiple Node.js versions.
* Download and manage Java JDK versions.
* Install and manage Android SDK components.
* Manage Android NDK versions.
* Manage Android Build Tools versions.
* Manage CMake versions.
* Configure project-specific development environments.
* Automatically install missing dependencies from `aem.json`.
* Switch active tool versions.
* Provide a consistent environment across macOS, Linux, and Windows.

Although React Native is one of the primary use cases, **AEM itself is framework-agnostic**.

AEM does not determine which versions a framework requires. Instead, the developer explicitly defines the required environment in the project's `aem.json`.

## Features

### Node.js Version Management

Install, list, switch, and manage multiple Node.js versions.

```bash
aem install node 20
aem use node 20.11.1
aem list node
```

### Java JDK Management

Download and install Java JDK distributions and manage multiple JDK versions.

```bash
aem install java 17
aem use java 17.0.15
aem list java
```

AEM currently supports downloading JDKs through the Azul Zulu API.

### Android SDK Management

AEM can automate Android development environment setup, including:

* Android SDK platforms
* Android Build Tools
* Android NDK
* CMake
* Android system images
* Android SDK command-line tools
* Android Platform Tools

### Project-Aware Setup

Projects can define their required environment in an `aem.json` file.

Running:

```bash
aem setup
```

automatically installs and activates the dependencies declared by the project.

### Project Initialization

Create a project-specific canonical `aem.json`:

```bash
aem init
```

`aem init` opens a terminal menu that scans AEM-managed and `PATH`-available
Node.js and Java runtimes, plus configurable packages from the local Android
SDK. Choose one runtime version with the arrow keys, and use Space to toggle
Android platforms, Build Tools, NDKs, CMake versions, and system images. Android
components start unchecked, so select only what the project requires.
Use `aem init --force` only when you deliberately want to replace an existing
configuration.

### Cross-Platform

AEM is designed to support:

* Windows
* macOS
* Linux

### Lightweight and Fast

AEM is written in Go, providing a lightweight CLI with minimal runtime dependencies.

### Extensible

AEM uses a provider-based architecture so additional runtimes and development tools can be added in the future.

## Supported Packages

### Node.js

Manage multiple Node.js distributions and versions.

```bash
aem install node 20
aem list node
aem use node 20.11.1
aem available node
aem uninstall node 20.11.1
```

### Java JDK

Manage multiple JDK versions.

```bash
aem install java 17
aem list java
aem use java 17.0.15
```

JDK downloads are currently provided through the Azul Zulu API.

### Android

AEM installs and reports these managed Android SDK components:

* SDK platforms
* Build Tools
* NDK
* CMake
* System images
* Platform Tools
* Command-line Tools

## Installation

### Prerequisites

* Internet connection for downloading SDKs and runtimes

### Install

```bash
curl -fsSL https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/refs/heads/main/scripts/install.sh | bash
```

On macOS and Linux, the installer downloads the matching released `aem` binary,
verifies it against the published checksum, installs it to
`$AEM_HOME/bin/aem` (default: `~/.aem/bin/aem`), and updates the detected Bash
or Zsh profile. Open a new terminal after it completes.

On Windows, run this in PowerShell:

```powershell
irm https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/refs/heads/main/scripts/install.ps1 | iex
```

The Windows installer downloads and verifies the matching released binary, then
sets persistent per-user environment variables; open a new terminal afterwards.

The macOS/Linux installer retains an existing `ANDROID_HOME`; if it is not
set, it asks for an existing Android SDK directory and configures
`ANDROID_HOME` and `ANDROID_SDK_ROOT` to that location. Pass
`--android-home PATH` for non-interactive installation. It adds the SDK's
`platform-tools` and command-line-tools directories to `PATH`. Use
`--aem-home PATH` on macOS/Linux or `-AemHome PATH` on Windows to use another
AEM directory. By default, both installers use the latest release. Pass
`--version 1.2.3` on macOS/Linux or `-Version 1.2.3` on Windows to install a
specific release.

### Uninstall

The uninstallers remove the AEM executable and the environment integration but
retain installed runtimes and SDKs by default:

```bash
./scripts/uninstall.sh
```

```powershell
.\scripts\uninstall.ps1
```

Pass `--remove-data` (macOS/Linux) or `-RemoveData` (Windows) to also delete
the selected `AEM_HOME` directory and every AEM-managed runtime and SDK.

### Release Archives

Tagged releases publish standalone archives for Linux, macOS, and Windows
(AMD64 and ARM64 where supported). Download the archive matching your machine
from the [GitHub Releases page](https://github.com/Adaptive-Cloud/aem-go/releases),
then verify it against the published `checksums.txt` before adding the extracted
directory to your `PATH`. Use `aem version` to see the installed build version,
commit, and build date.

## Usage

Run:

```bash
aem --help
```

to see available commands.

### Initialize a Project

Create an `aem.json` template:

```bash
aem init
```

The command opens a keyboard-driven terminal UI. Use ↑/↓ (or `j`/`k`) to move,
Enter to select a Node.js or Java version, and Space to toggle detected Android
SDK components before confirming them with Enter. It then prints the path it
created. If you skip every option, the initial contents are valid and
intentionally minimal:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {},
  "android": {},
  "hooks": {}
}
```

The generated configuration can then be reviewed and committed to source control.

`aem init` only creates the project configuration. It does not install the dependencies.
For automation, use `aem init --non-interactive` to create the minimal template
without prompts.

Use:

```bash
aem setup
```

to install and configure the environment defined by the generated `aem.json`.

### List Available Versions

```bash
aem available node
aem available java
```

### Install a Runtime

```bash
aem install node 20
aem install java 17
```

### Switch the Active Runtime

```bash
aem use node 20.11.1
aem use java 17.0.15
```

### Show Installed Versions

```bash
aem list node
aem list java
aem list android
```

### Remove an Inactive Runtime

```bash
aem uninstall node 20.11.1
aem uninstall java 17.0.15
```

For safety, AEM refuses to remove a runtime selected through `aem use` or
`aem setup`. Switch to another version first.

### Show Currently Active Runtimes

```bash
aem current
```

### Inspect Local Environment

```bash
aem doctor
```

### Set Up the Current Project

```bash
aem setup
```

AEM searches for the nearest `aem.json` in the current directory or one of its parent directories.

If a requested dependency is missing, AEM downloads and installs it automatically.

### Command Summary

| Command         | Description                                                 |
| --------------- | ----------------------------------------------------------- |
| `aem init`      | Interactively create and validate a canonical `aem.json`    |
| `aem setup`     | Install and configure the environment defined by `aem.json` |
| `aem install`   | Install a specific version                                  |
| `aem list`      | List installed versions                                     |
| `aem use`       | Switch the active version                                   |
| `aem available` | List available remote versions                              |
| `aem uninstall` | Remove an inactive AEM-managed runtime                      |
| `aem current`   | Show currently active runtimes                              |
| `aem doctor`    | Inspect local AEM environment                               |

> Commands and flags may evolve during development. Run `aem --help` for the latest usage information.

## Project Configuration

A project can define its development environment using an `aem.json` file.

The configuration is **declarative**:

> The developer defines what versions the project needs. AEM determines how to install and configure them.

AEM does not attempt to determine the React Native version or infer dependency versions from the framework.

### Example

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4",
    "java": "17"
  },
  "android": {
    "platforms": [
      "35"
    ],
    "buildTools": [
      "35.0.0"
    ],
    "ndk": [
      "27.0.12077973"
    ],
    "cmake": [
      "3.22.1"
    ],
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

The `aem.json` file should normally be committed to source control so that the project's environment requirements can be reproduced by other developers and CI systems.

`$schema` points editors and JSON tooling to the AEM schema. `runtime`,
`android`, and `hooks` are optional, so a project can declare only the pieces
it needs.

### Migrating Legacy Configuration

Older configuration files used top-level `node` and `jdk`, plus Android
`sdk` and `build-tool`. AEM continues to read those fields and reports a
migration notice during setup, but new configurations must use the canonical
shape:

| Legacy field | Canonical field |
| --- | --- |
| `node` | `runtime.node` |
| `jdk` | `runtime.java` |
| `android.sdk` | `android.platforms` |
| `android.build-tool` | `android.buildTools` |

Do not mix legacy and canonical names in the same configuration; that is
rejected to avoid ambiguous setup results.

### Optional Components

Projects do not need to define every available dependency.

For example, a project may only require Node.js:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4"
  }
}
```

Or Node.js and Java:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4",
    "java": "17"
  }
}
```

Android components can be added when required:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4",
    "java": "17"
  },
  "android": {
    "platforms": [
      "35"
    ],
    "buildTools": [
      "35.0.0"
    ],
    "ndk": [
      "27.0.12077973"
    ]
  }
}
```

AEM should only configure the components explicitly declared by the developer, along with any underlying prerequisites required to install them.

## Android Configuration

Android components are defined using human-readable configuration rather than raw Android `sdkmanager` package identifiers.

### SDK Platforms

Specify Android API levels:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "android": {
    "platforms": [
      "34",
      "35"
    ]
  }
}
```

AEM translates:

```text
"34"
```

into the Android package:

```text
platforms;android-34
```

### Android Build Tools

Specify Build Tools versions:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "android": {
    "buildTools": [
      "34.0.0",
      "35.0.0"
    ]
  }
}
```

AEM translates:

```text
"34.0.0"
```

into:

```text
build-tools;34.0.0
```

### Android NDK

Specify NDK versions:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "android": {
    "ndk": [
      "25.1.8937393",
      "27.0.12077973"
    ]
  }
}
```

AEM translates:

```text
"25.1.8937393"
```

into:

```text
ndk;25.1.8937393
```

### CMake

Specify CMake versions:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "android": {
    "cmake": [
      "3.22.1"
    ]
  }
}
```

AEM translates:

```text
"3.22.1"
```

into:

```text
cmake;3.22.1
```

### Android System Images

System images use structured configuration because the Android package identifier contains multiple pieces of information.

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "android": {
    "systemImages": [
      {
        "apiLevel": 34,
        "variant": "google_apis",
        "architecture": "arm64-v8a"
      }
    ]
  }
}
```

AEM translates this configuration into:

```text
system-images;android-34;google_apis;arm64-v8a
```

The configuration fields are:

| Field          | Description                                 |
| -------------- | ------------------------------------------- |
| `apiLevel`     | Android API level                           |
| `variant`      | System image variant, such as `google_apis` |
| `architecture` | System image CPU architecture               |
| `package`      | Optional exact SDK package ID for preview versions, such as `android-37.0` |

The raw `sdkmanager` identifier is normally an implementation detail of the Android provider. `aem init` adds `package` only when an Android preview image needs an exact identifier, such as `system-images;android-37.0;...`.

### Android Command-Line Tools

AEM automatically ensures that the Android command-line tooling required to perform SDK setup is available.

This may include:

* Android SDK Command-line Tools
* `sdkmanager`
* Android Platform Tools

These are considered infrastructure required by the Android provider rather than project-specific dependencies that developers need to declare in every `aem.json`.

During setup, AEM uses the Android SDK command-line tooling to install the requested Android components.
When any Android component is requested, AEM also installs `platform-tools` as
the shared SDK prerequisite. Re-running setup only passes locally missing
components to `sdkmanager`; it does not use the Android repository to resolve
or validate already installed local package directories.

### Inspecting Android Components

Use the same list command used for runtimes to see locally installed Android
package identifiers:

```bash
aem list android
```

`aem list android` reads the SDK selected through `ANDROID_HOME` (or
`ANDROID_SDK_ROOT`) and prints every locally installed package. It reads Android
package metadata, so packages installed outside AEM—such as emulator, sources,
and extras—are included as well.
`aem doctor` includes that SDK path, an installed-component count, and the
component inventory. Android is not an active runtime, so it is intentionally
absent from `aem current`.

## Setup Hooks

The optional `hooks` object defines lifecycle hooks around `aem setup`:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "hooks": {
    "preSetup": [],
    "postSetup": []
  }
}
```

`preSetup` runs before dependency installation and activation; `postSetup` runs
after setup completes. Both fields are ordered arrays of shell commands that
run from the directory containing `aem.json`. A failed hook stops setup and is
reported as an error. Leave a field as an empty array when no hook is required.

## Environment Setup

When running:

```bash
aem setup
```

AEM:

1. Searches for the nearest `aem.json`.
2. Loads and validates the configuration.
3. Detects the host operating system.
4. Detects the host architecture.
5. Checks existing runtime installations and the configured Android SDK.
6. Installs missing runtimes.
7. Installs missing Android components.
8. Activates the requested runtime versions.
9. Configures the environment.
10. Reports the resulting environment.

The setup operation should be idempotent.

Running:

```bash
aem setup
```

multiple times should not reinstall components that are already correctly installed.

### AEM Home

AEM stores managed Node.js and Java runtimes inside a dedicated directory.
Android packages are installed in the SDK selected through `ANDROID_HOME`.

If `AEM_HOME` is not configured, the default location is:

**macOS / Linux**

```text
~/.aem
```

**Windows**

```text
%USERPROFILE%\.aem
```

AEM-managed installations may be organized approximately as:

```text
~/.aem/
├── bin/
│   └── aem
├── current/
│   ├── node
│   └── java
├── sys_installed/
│   ├── node/
│   └── java/
├── tmp/
└── versions.json (legacy activation state, when present)
```

The exact internal structure may change as the project evolves.

### Active Versions

AEM uses stable paths for active versions.

For example:

```text
~/.aem/current/node
~/.aem/current/java
```

These paths point to the currently selected versions.

This allows the user's shell configuration to remain stable even when the active version changes.

For example:

```text
~/.aem/current/node
        ↓
~/.aem/versions/node/20.11.1
```

After switching versions:

```text
~/.aem/current/node
        ↓
~/.aem/versions/node/22.14.0
```

### Environment Variables

The platform installers above configure these values automatically. For a
manual setup, add the following to your shell profile:

```bash
export AEM_HOME="${AEM_HOME:-$HOME/.aem}"
export JAVA_HOME="$AEM_HOME/current/java"
if [ "$(uname -s)" = "Darwin" ]; then
  export JAVA_HOME="$JAVA_HOME/Contents/Home"
fi
export ANDROID_HOME="/path/to/your/Android/sdk"
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export PATH="$AEM_HOME/bin:$AEM_HOME/current/node/bin:$JAVA_HOME/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/cmdline-tools/latest/bin:$PATH"
```

The macOS/Linux installer adds a clearly marked, removable block to the chosen
shell profile; `scripts/uninstall.sh` removes only that block. The Windows
installer changes only per-user environment variables and records their prior
values so `scripts/uninstall.ps1` can restore them.

## Shell Environment Integration

`aem use` and `aem setup` change stable links below `$AEM_HOME/current`; they
do not modify a terminal that is already open. Configure the shell once to use
those stable paths, then opening a new shell (or sourcing its configuration)
picks up every subsequent runtime switch.

On macOS and Linux, `./scripts/install.sh` writes `$AEM_HOME/env.sh` and adds a
marked source block to the selected Bash or Zsh profile. It exports:

```text
PATH=$AEM_HOME/bin:$AEM_HOME/current/node/bin:$JAVA_HOME/bin:...
JAVA_HOME=$AEM_HOME/current/java
ANDROID_HOME=<your existing Android SDK>
ANDROID_SDK_ROOT=<your existing Android SDK>
```

On macOS, Azul archives may be app bundles, so the managed script sets
`JAVA_HOME` to `$AEM_HOME/current/java/Contents/Home`. The installer retains an
existing Android SDK rather than moving it. For another shell, source the file
from a POSIX-compatible startup wrapper, or source it manually before running
tools that require Node or Java.

On Windows, `scripts/install.ps1` persists the same values for the current user
and adds `%AEM_HOME%\bin`, `%AEM_HOME%\current\node`,
`%AEM_HOME%\current\java\bin`, and Android tool directories to the user
`Path`. Restart PowerShell, Command Prompt, Windows Terminal, and IDE terminals
after running it. The Windows installer uses `%AEM_HOME%\current\android` as
the managed SDK directory unless `ANDROID_HOME` is overridden.

If you configure links manually, point `AEM_NODE_SYMLINK` and
`AEM_JAVA_SYMLINK` at safe, dedicated directory-link paths. AEM refuses to
replace a regular file or directory at either path. On Windows it falls back to
a directory junction when symbolic links are unavailable.

## Providers, Cache, and Upgrades

Node.js binaries and release metadata come from the official Node.js
distribution. Java JDKs come from Azul Zulu's public metadata API. Android
components are installed through Google's Android command-line tools. AEM
selects artifacts for the detected operating system and architecture; an
unsupported target fails instead of silently choosing another platform.

Downloads are staged in `$AEM_HOME/tmp` and removed after an installation.
AEM has no persistent offline artifact cache yet, so a missing runtime or
Android package requires network access. Valid installed versions are reused,
making `aem setup` safe to repeat without re-downloading them.

AEM never upgrades a project runtime merely because a newer release exists.
Update the version in `aem.json` (or explicitly run `aem install`) and then
activate or set up that version. Update AEM itself by installing a newer
verified release archive or rebuilding a newer source checkout; verify the
result with `aem version`.

## Troubleshooting

Run `aem doctor` first. It reports the AEM home, install location, active
runtime links, and discovered Android components.

* **A runtime is installed but a command is not found:** start a new terminal,
  or source `$AEM_HOME/env.sh` on macOS/Linux. Confirm the active runtime with
  `aem current` and ensure the relevant `current` path precedes a system
  installation in `PATH`.
* **Java is selected but build tools cannot find it:** check `JAVA_HOME`. On
  macOS it normally ends in `current/java/Contents/Home`; on Linux and Windows
  it normally ends in `current/java`.
* **Android tooling is missing:** set `ANDROID_HOME` and `ANDROID_SDK_ROOT` to
  the SDK AEM manages or the SDK selected during installation, then ensure
  `platform-tools` and `cmdline-tools/latest/bin` are in `PATH`.
* **Windows activation cannot create a link:** use a dedicated link path and
  rerun the command. AEM attempts a symbolic link and then a directory junction;
  enable Developer Mode or use an elevated shell only if both fail.
* **A download fails:** check network/proxy settings, rerun with `aem --debug`,
  and retry. Staging files in `$AEM_HOME/tmp` are safe to remove after AEM is
  no longer running.

## Configuration Philosophy

AEM is a **project-defined environment manager**, not a framework compatibility manager.

For example, if a developer specifies:

```json
{
  "$schema": "https://raw.githubusercontent.com/Adaptive-Cloud/aem-go/main/aem.schema.json",
  "runtime": {
    "node": "20.19.4",
    "java": "17"
  },
  "android": {
    "platforms": [
      "35"
    ],
    "buildTools": [
      "35.0.0"
    ],
    "ndk": [
      "27.0.12077973"
    ]
  }
}
```

AEM should install those requested versions.

AEM should **not**:

* Detect the React Native version.
* Inspect `package.json` to determine framework requirements.
* Automatically upgrade versions.
* Automatically downgrade versions.
* Replace requested versions with versions it considers more compatible.
* Infer dependency versions from a framework.

The project configuration is the source of truth.

## Contribution

Contributions are welcome!

This project is a personal initiative to learn Go while building a practical development environment management tool.

If you'd like to contribute:

1. Fork the repository.
2. Create a feature branch:

```bash
git checkout -b feature/your-feature
```

3. Make your changes.
4. Add or update tests.
5. Commit your changes:

```bash
git commit -m "Add some feature"
```

6. Push your branch:

```bash
git push origin feature/your-feature
```

7. Open a Pull Request.

Please make sure to write tests and document significant changes.

## Motivation

AEM has two primary goals.

### Learning Go

This project provides a practical way to learn and improve skills in:

* Go
* CLI development
* Cross-platform development
* API integration
* Package management
* Environment management
* Software distribution

### Reducing Development Environment Setup Time

Setting up development environments often involves repetitive manual work.

Developers may need to:

* Install a specific Node.js version.
* Install a specific JDK.
* Configure `JAVA_HOME`.
* Install Android SDK platforms.
* Install Build Tools.
* Install a particular NDK.
* Install CMake.
* Configure Android environment variables.
* Switch between different project environments.

AEM aims to automate this process.

A developer should be able to clone a project containing `aem.json` and run:

```bash
aem setup
```

to have the environment described by the project configuration ready to use.

## Future Plans

Potential future improvements include:

* CI/CD environment setup.
* Environment profiles.
* Improved configuration management.
* Additional runtime providers.
* More Android SDK components.
* Android Emulator management.
* Better cross-platform installers.
* Package manager integrations.
* Shell integration.
* Environment export/import.
* Improved caching and download management.
* Checksums and artifact verification.
* More sophisticated version management.
* Reproducible development environments.

## Contact

Created by **firmajml**.

Feel free to open an issue for bugs, feature requests, suggestions, or questions.

Thank you for using Adaptive Environment Manager! 🚀
