# Adaptive Environment Manager (AEM) [WORK IN PROGRESS 🏗️]

Adaptive Environment Manager (AEM) is a powerful tool designed to simplify and automate the management of development environment dependencies, such as Node.js, Java, and Android SDK. AEM is especially tailored for developers working with frameworks like React Native, where managing multiple versions of these tools can become cumbersome and time-consuming.

By automating the setup and version management of essential SDKs and runtimes, AEM enables developers to focus more on coding and less on environment configuration — accelerating project setup and boosting productivity.


https://github.com/user-attachments/assets/df05cc1b-31c7-49d9-b70b-e574b5182a39


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
./aem --help

# Install a specific Node.js version
./aem node install 16.14.0

# List installed Node.js versions
./aem node list

# Use a specific Node.js version for your project
./aem node use 16.14.0

# Install a Java JDK version via Azul Zulu API
./aem java install 11.0.15

# List installed Java versions
./aem java list

# Install Android SDK components
./aem android install

# Setup environment by retrieving configuration from aem.json
./aem setup
```

> **Note:** Commands and flags may evolve; run `./aem --help` for the latest usage information.

---

## Configuration

AEM stores configurations and installed SDKs in a user-specific directory:

c:/aem/

Create an environmental variables:
- AEM_HOME -> C:\Users\haru1\OneDrive\Desktop\Projects\aem-go
- AEM_JAVA_SYMLINK -> C:\aem\jdk
- AEM_NODE_SYMLINK -> C:\aem\nodejs

Add these ENV into PATH
- %AEM_HOME%
- %AEM_NODE_SYMLINK%
- %AEM_JAVA_SYMLINK%
- %AEM_JAVA_SYMLINK%\bin
- 
![Editing Env Path](https://github.com/user-attachments/assets/2305fe63-b2c3-42d2-82d3-9f6d8ad9969f)

Example of aem.json
```
{
  "node": "16.20.2",
  "jdk": "17.0.15",
  "android": {
    "sdk": "",
    "ndk": "",
    "build-tool": ""
  }
}
```

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

