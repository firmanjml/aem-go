#!/usr/bin/env bash
# Remove AEM's shell integration and installed CLI on macOS or Linux.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/uninstall.sh [--aem-home PATH] [--profile PATH] [--remove-data]

Removes the AEM executable and its marked shell-profile integration. Managed
runtimes in AEM_HOME are retained unless --remove-data is supplied.
EOF
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
profile=""
remove_data=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --aem-home)
      [ "$#" -ge 2 ] || { echo "--aem-home requires a path" >&2; exit 2; }
      aem_home="$2"
      shift 2
      ;;
    --profile)
      [ "$#" -ge 2 ] || { echo "--profile requires a path" >&2; exit 2; }
      profile="$2"
      shift 2
      ;;
    --remove-data) remove_data=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

case "$(uname -s)" in
  Darwin|Linux) ;;
  *) echo "This uninstaller supports macOS and Linux. Use scripts/uninstall.ps1 on Windows." >&2; exit 1 ;;
esac

if [ -z "$profile" ]; then
  case "${SHELL:-}" in
    */zsh) profile="$HOME/.zshrc" ;;
    */bash|"") profile="$HOME/.bashrc" ;;
    *) profile="" ;;
  esac
fi

if [ -n "$profile" ]; then
  remove_profile_block "$profile"
  echo "Removed AEM shell integration from $profile."
fi

rm -f "$aem_home/bin/aem" "$aem_home/env.sh"
rmdir "$aem_home/bin" 2>/dev/null || true

if [ "$remove_data" -eq 1 ]; then
  aem_home="$(cd "$aem_home" 2>/dev/null && pwd || true)"
  [ -n "$aem_home" ] && [ "$aem_home" != "/" ] || { echo "Refusing to remove an unsafe AEM_HOME path." >&2; exit 1; }
  rm -rf "$aem_home"
  echo "Removed AEM and all managed data from $aem_home."
else
  echo "Removed the AEM executable. Managed data in $aem_home was retained."
  echo "Use --remove-data to delete managed runtimes and SDKs as well."
fi
