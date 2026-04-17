# Adaptive Environment Manager (AEM) [WORK IN PROGRESS 🏗️]
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/firmanjml/aem-go)

Adaptive Environment Manager (AEM) is a powerful tool designed to simplify and automate the management of development environment dependencies, such as Node.js, Java, and Android SDK. AEM is especially tailored for developers working with frameworks like React Native, where managing multiple versions of these tools can become cumbersome and time-consuming.

By automating the setup and version management of essential SDKs and runtimes, AEM enables developers to focus more on coding and less on environment configuration — accelerating project setup and boosting productivity.



https://github.com/user-attachments/assets/3c0c3acb-6317-408e-b37e-75eba5ea69b5







https://github.com/user-attachments/assets/e0c73c67-5a52-4be0-a61f-25a8de93051d



---

## Table of Contents

- [Overview](#overview)  
- [Features](#features)  
- [Supported Packages](#supported-packages)  
- [Installation](#installation)  
- [Usage](#usage)  
- [Configuration](#configuration)  
- [Contribution](#contribution)  
- [Motivation](#motivation)
- [Future Plans](#future-plans)
- [Contact](#contact)

---

## Overview

Modern development environments often require managing multiple versions of Node.js, Java JDK, and Android SDKs to maintain compatibility with various projects. Manually installing, configuring, and switching between these versions can be error-prone and slow down the development workflow.

**Adaptive Environment Manager (AEM)** solves this by providing an intuitive command-line interface and automation for:

- Installing and managing Node.js versions  
- Downloading and configuring Java JDKs via Azul Zulu API  
- Managing Android SDK versions and setup

AEM abstracts away the complexity involved in environment management, streamlining your project setup process.

---

## Features

- **Managing Node.js Version**  
  Install, list, switch, and manage multiple Node.js versions effortlessly.

- **Managing Java JDK**  
  Download and install Java JDKs using the Azul Zulu API for a wide range of versions and platforms.

- **Android SDK Setup**  
  Automated setup and configuration of Android SDK components required for React Native and Android development.

- **Cross-platform Support**  
  Works on major platforms: Windows, macOS, and Linux.

- **Lightweight and Fast**  
  Written in Go for speedy execution with minimal dependencies.

- **Extensible**  
  Designed to be easily extended to support other SDKs or tools.

---

## Supported Packages

- **Node.js**  
  Manage different Node.js distributions and versions.

- **Java JDK**  
  Download and configure Java JDKs from the official [Azul Zulu API](https://www.azul.com/downloads/zulu/).

- **Android SDK**  (WORK IN PROGRESS)
  Automate Android SDK installation and configuration for mobile app development.

---

## Installation

### Pre-requisites

- Go (version 1.18+) installed on your system  
- Internet connection to download SDKs and runtimes

### Build from Source

```bash
git clone https://github.com/firmanjml/aem-go.git
cd aem-go
go build -o aem
```

This will generate a binary named `aem` in your current directory.

---

## Usage

Run `aem` with commands to manage your environment:

```bash
# Show help and available commands
aem --help

# List available remote versions for a module
aem list node
aem list java 17

# Install a runtime version
aem install node 20
aem install java 17

# Switch the active runtime version
aem use node 20.11.1
aem use java 17.0.15

# Show the currently active runtimes
aem current

# Inspect local state and health
aem doctor

# Setup the current project from the nearest aem.json
aem setup
```

> **Note:** Commands and flags may evolve; run `aem --help` for the latest usage information.

---

## Configuration

`aem setup` now behaves like a project-aware switcher:

- It searches for the nearest `aem.json` in the current directory or any parent directory.
- It uses cached installs from `AEM_HOME` when available.
- If a requested runtime is missing, it downloads and installs it automatically.
- It switches the active toolchain by updating stable symlinks, so your shell only needs to be configured once.
- The active version is resolved from those symlinks, not from parsing `versions.json`.

If `AEM_HOME` is not set, AEM defaults to:

- macOS/Linux: `~/.aem`
- Windows: `%USERPROFILE%\.aem`

Default active symlinks are created under:

- `~/.aem/current/node`
- `~/.aem/current/java`
- `~/.aem/current/android`

You can override them with:

- `AEM_HOME`
- `AEM_NODE_SYMLINK`
- `AEM_JAVA_SYMLINK`
- `AEM_ANDROID_SYMLINK`

Recommended shell setup:

- Add your `aem` binary to `PATH`
- Add `~/.aem/current/node/bin` to `PATH`
- Add `~/.aem/current/java/bin` to `PATH`
- Add `~/.aem/current/android/platform-tools` to `PATH`
- Add `~/.aem/current/android/cmdline-tools/latest/bin` to `PATH`
- Set `JAVA_HOME=~/.aem/current/java`
- Set `ANDROID_HOME=~/.aem/current/android`
- Set `ANDROID_SDK_ROOT=~/.aem/current/android`

Example `aem.json`
```
{
  "node": "16.20.2",
  "jdk": "17.0.15",
  "android": {
    "sdk": ["34"],
    "ndk": ["25.1.8937393"],
    "build-tool": ["34.0.0"]
  }
}
```

Android values can be either arrays or single strings. During `aem setup`, AEM ensures Android command-line tools are installed, accepts SDK licenses, and installs the requested packages through `sdkmanager`.

---

## Contribution

Contributions are welcome! This project is a personal initiative to learn Go and improve development workflow automation. If you'd like to contribute:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -m 'Add some feature'`)
4. Push to your branch (`git push origin feature/your-feature`)
5. Open a Pull Request

Please make sure to write tests and document your changes.

---

## Motivation

The motivation behind AEM is twofold:

* **Learning Go:**
  This project serves as a practical and hands-on way to deepen knowledge of Go programming, CLI design, and API integration.

* **Reduce Setup Time:**
  Setting up environments for React Native and similar frameworks often involves repetitive manual work. AEM aims to reduce that overhead by automating environment setup, so developers spend more time coding and less time configuring.

---

## Future Plans

* Integration with CI/CD pipelines for project building
* Enhanced configuration management with environment profiles
* Better cross-platform installers and package manager support

---

## Contact

Created by firmajml
Feel free to reach out or create issues for any bugs, suggestions, or questions.

---

Thank you for using Adaptive Environment Manager! 🚀
