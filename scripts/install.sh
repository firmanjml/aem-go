#!/usr/bin/env bash
# Download and install a released AEM binary on macOS or Linux.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/install.sh [--version VERSION] [--aem-home PATH] [--android-home PATH] [--profile PATH] [--no-profile]

Downloads the latest AEM release (or VERSION) and installs it in AEM_HOME/bin.
By default, it also adds a small,
marked block to the detected Bash or Zsh profile so new shells receive AEM's
PATH, JAVA_HOME, ANDROID_HOME, and ANDROID_SDK_ROOT settings. An existing
ANDROID_HOME is retained; otherwise the installer asks for the Android SDK path.
EOF
}

quote_sh() {
  printf "'"
  printf '%s' "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

remove_profile_block() {
  local profile="$1" temp
  [ -f "$profile" ] || return 0
  temp="$(mktemp "${profile}.aem.XXXXXX")"
  awk '
    /^# >>> aem environment >>>$/ { skipping = 1; next }
    /^# <<< aem environment <<<$/{ skipping = 0; next }
    !skipping { print }
  ' "$profile" > "$temp"
  mv "$temp" "$profile"
}

aem_home="${AEM_HOME:-$HOME/.aem}"
android_home="${ANDROID_HOME:-}"
profile=""
update_profile=1
release_version="latest"
github_repository="Adaptive-Cloud/aem-go"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "--version requires a version" >&2; exit 2; }
      release_version="$2"
      shift 2
      ;;
    --aem-home)
      [ "$#" -ge 2 ] || { echo "--aem-home requires a path" >&2; exit 2; }
      aem_home="$2"
      shift 2
      ;;
    --android-home)
      [ "$#" -ge 2 ] || { echo "--android-home requires a path" >&2; exit 2; }
      android_home="$2"
      shift 2
      ;;
    --profile)
      [ "$#" -ge 2 ] || { echo "--profile requires a path" >&2; exit 2; }
      profile="$2"
      shift 2
      ;;
    --no-profile)
      update_profile=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$(uname -s)" in
  Darwin) release_os="darwin" ;;
  Linux) release_os="linux" ;;
  *)
    echo "This installer supports macOS and Linux. Use scripts/install.ps1 on Windows." >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) release_arch="amd64" ;;
  arm64|aarch64) release_arch="arm64" ;;
  *)
    echo "Unsupported CPU architecture: $(uname -m)." >&2
    exit 1
    ;;
esac

command -v curl >/dev/null 2>&1 || {
  echo "curl is required to download AEM." >&2
  exit 1
}
command -v tar >/dev/null 2>&1 || {
  echo "tar is required to extract the AEM release archive." >&2
  exit 1
}

if [ "$release_version" != "latest" ] && ! printf '%s' "$release_version" | grep -Eq '^[0-9A-Za-z._-]+$'; then
  echo "Invalid version: $release_version" >&2
  exit 2
fi

if [ -n "$android_home" ] && [ -d "$android_home" ]; then
  android_home="$(CDPATH= cd -- "$android_home" && pwd)"
  echo "Using existing Android SDK at $android_home."
else
  if [ -n "$android_home" ]; then
    echo "ANDROID_HOME does not exist: $android_home" >&2
  fi
  if [ ! -r /dev/tty ]; then
    echo "ANDROID_HOME is not configured. Re-run interactively or pass --android-home PATH." >&2
    exit 1
  fi
  while :; do
    read -r -p "Android SDK path (ANDROID_HOME): " android_home </dev/tty
    if [ -d "$android_home" ]; then
      android_home="$(CDPATH= cd -- "$android_home" && pwd)"
      break
    fi
    echo "That directory does not exist. Enter an existing Android SDK path." >&2
  done
fi

if [ "$update_profile" -eq 1 ] && [ -z "$profile" ]; then
  case "${SHELL:-}" in
    */zsh) profile="$HOME/.zshrc" ;;
    */bash|"") profile="$HOME/.bashrc" ;;
    *)
      echo "Could not determine a Bash or Zsh profile from SHELL. Re-run with --profile PATH or --no-profile." >&2
      exit 1
      ;;
  esac
fi

bin_dir="$aem_home/bin"
env_file="$aem_home/env.sh"
mkdir -p "$bin_dir"
temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/aem-install.XXXXXX")"
temporary_binary="$bin_dir/aem.$$.tmp"
trap 'rm -rf "$temporary_dir" "$temporary_binary"' EXIT

if [ "$release_version" = "latest" ]; then
  release_api="https://api.github.com/repos/$github_repository/releases/latest"
else
  release_api="https://api.github.com/repos/$github_repository/releases/tags/v$release_version"
fi
release_json="$(curl --fail --location --retry 3 --silent --show-error \
  -H 'Accept: application/vnd.github+json' "$release_api")" || {
  echo "Could not find AEM release '$release_version'." >&2
  exit 1
}
release_tag="$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[ -n "$release_tag" ] || { echo "The release response did not include a tag name." >&2; exit 1; }
archive_version="${release_tag#v}"
archive_name="aem_${archive_version}_${release_os}_${release_arch}.tar.gz"
archive_url="$(printf '%s\n' "$release_json" | sed -n 's|.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*/'"$archive_name"'\)".*|\1|p')"
checksums_url="$(printf '%s\n' "$release_json" | sed -n 's|.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*/checksums.txt\)".*|\1|p')"

if [ -z "$archive_url" ] || [ -z "$checksums_url" ]; then
  echo "Release $release_tag does not contain $archive_name and checksums.txt." >&2
  exit 1
fi

archive_path="$temporary_dir/$archive_name"
checksums_path="$temporary_dir/checksums.txt"
echo "Downloading AEM $release_tag for $release_os/$release_arch..."
curl --fail --location --retry 3 --silent --show-error "$archive_url" -o "$archive_path"
curl --fail --location --retry 3 --silent --show-error "$checksums_url" -o "$checksums_path"
expected_checksum="$(awk -v name="$archive_name" '$2 == name || $2 == "*" name { print $1; exit }' "$checksums_path")"
[ -n "$expected_checksum" ] || { echo "No checksum found for $archive_name." >&2; exit 1; }

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "$archive_path" | awk '{ print $1 }')"
else
  command -v shasum >/dev/null 2>&1 || { echo "sha256sum or shasum is required to verify the download." >&2; exit 1; }
  actual_checksum="$(shasum -a 256 "$archive_path" | awk '{ print $1 }')"
fi
if [ "$actual_checksum" != "$expected_checksum" ]; then
  echo "Checksum verification failed for $archive_name." >&2
  exit 1
fi

mkdir "$temporary_dir/unpacked"
tar -xzf "$archive_path" -C "$temporary_dir/unpacked"
released_binary="$temporary_dir/unpacked/aem"
[ -f "$released_binary" ] || { echo "Release archive did not contain an aem binary." >&2; exit 1; }
install -m 0755 "$released_binary" "$temporary_binary"
mv -f "$temporary_binary" "$bin_dir/aem"

cat > "$env_file" <<EOF
# Managed by AEM's install.sh. It is sourced by your shell profile.
export AEM_HOME=$(quote_sh "$aem_home")

aem_prepend_path() {
  case ":\${PATH-}:" in
    *":\$1:"*) ;;
    *) PATH="\$1\${PATH:+:\$PATH}" ;;
  esac
}

aem_prepend_path "\$AEM_HOME/bin"
aem_prepend_path "\$AEM_HOME/current/node/bin"

# AEM normalizes macOS JDK archives so their Java home is always available at
# Contents/Home. Keep that stable path even before a JDK is selected: aem use
# only swaps the current/java link, so the shell sees the new JDK immediately.
aem_java_home="\$AEM_HOME/current/java"
if [ "\$(uname -s)" = "Darwin" ]; then
  aem_java_home="\$aem_java_home/Contents/Home"
fi
export JAVA_HOME="\$aem_java_home"
aem_prepend_path "\$JAVA_HOME/bin"
unset aem_java_home

aem_prepend_path $(quote_sh "$android_home/platform-tools")
aem_prepend_path $(quote_sh "$android_home/cmdline-tools/latest/bin")
unset -f aem_prepend_path
export PATH

export ANDROID_HOME=$(quote_sh "$android_home")
export ANDROID_SDK_ROOT=$(quote_sh "$android_home")
EOF
chmod 0644 "$env_file"

if [ "$update_profile" -eq 1 ]; then
  mkdir -p "$(dirname "$profile")"
  touch "$profile"
  remove_profile_block "$profile"
  {
    printf '\n# >>> aem environment >>>\n'
    printf 'if [ -f %s ]; then\n' "$(quote_sh "$env_file")"
    printf '  . %s\n' "$(quote_sh "$env_file")"
    printf 'fi\n# <<< aem environment <<<\n'
  } >> "$profile"
  echo "Installed AEM to $bin_dir/aem and updated $profile."
  echo "Open a new terminal or run: . $(quote_sh "$profile")"
else
  echo "Installed AEM to $bin_dir/aem."
  echo "To configure this shell, run: . $(quote_sh "$env_file")"
fi
