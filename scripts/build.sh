#!/usr/bin/env bash
# Build and install AEM from this checked-out source tree. It never downloads
# an AEM release or clones source from source control.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/build.sh [--aem-home PATH] [--android-home PATH] [--profile PATH] [--no-profile]

Builds the current AEM source checkout and installs it in AEM_HOME/bin. By
default, it also adds a marked block to the detected Bash or Zsh profile so new
shells receive AEM's PATH, JAVA_HOME, ANDROID_HOME, and ANDROID_SDK_ROOT.
EOF
}

quote_sh() {
  printf "'"
  printf '%s' "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

remove_profile_block() {
  local profile="$1" temporary_file
  [ -f "$profile" ] || return 0
  temporary_file="$(mktemp "${profile}.aem.XXXXXX")"
  awk '
    /^# >>> aem environment >>>$/ { skipping = 1; next }
    /^# <<< aem environment <<</ { skipping = 0; next }
    !skipping { print }
  ' "$profile" > "$temporary_file"
  mv "$temporary_file" "$profile"
}

repo_root="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
aem_home="${AEM_HOME:-$HOME/.aem}"
android_home="${ANDROID_HOME:-}"
profile=""
update_profile=1

while [ "$#" -gt 0 ]; do
  case "$1" in
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

command -v go >/dev/null 2>&1 || {
  echo "Go is required to build AEM from source." >&2
  exit 1
}
[ -f "$repo_root/go.mod" ] || {
  echo "Could not find go.mod in the AEM source tree: $repo_root" >&2
  exit 1
}

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
temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/aem-build.XXXXXX")"
temporary_binary="$temporary_dir/aem"
trap 'rm -rf "$temporary_dir"' EXIT

build_version="dev"
build_commit="none"
if command -v git >/dev/null 2>&1 && git -C "$repo_root" rev-parse --git-dir >/dev/null 2>&1; then
  if described="$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null)"; then
    build_version="${described#v}"
  fi
  build_commit="$(git -C "$repo_root" rev-parse --short HEAD 2>/dev/null || printf 'none')"
fi
build_date="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
build_ldflags=(
  -s -w
  -X "aem/cmd.Version=${build_version}"
  -X "aem/cmd.Commit=${build_commit}"
  -X "aem/cmd.BuildDate=${build_date}"
)

echo "Building AEM ${build_version} from $repo_root..."
(
  cd "$repo_root"
  go build -trimpath -ldflags "${build_ldflags[*]}" -o "$temporary_binary" .
)
chmod 0755 "$temporary_binary"
mv -f "$temporary_binary" "$bin_dir/aem"

cat > "$env_file" <<EOF
# Managed by AEM's build.sh. It is sourced by your shell profile.
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
# Contents/Home, including wrapped Zulu 8 bundles.
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
  echo "Built and installed AEM to $bin_dir/aem and updated $profile."
  echo "Open a new terminal or run: . $(quote_sh "$profile")"
else
  echo "Built and installed AEM to $bin_dir/aem."
  echo "To configure this shell, run: . $(quote_sh "$env_file")"
fi
