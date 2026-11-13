#!/bin/bash
set -euo pipefail

info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

success() {
  echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

warn() {
  echo -e "\033[1;33m[WARNING]\033[0m $1"
}

for tool in wails3 codesign xcrun ditto plutil security shasum spctl zipinfo; do
  if ! command -v "$tool" &> /dev/null; then
    error "$tool is required but not installed. Exiting."
    exit 1
  fi
done

if [ $# -ge 1 ] && [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION=$(git describe --tags --exact-match 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
fi
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
CLEAN_VERSION="${VERSION#v}"
CLEAN_VERSION="${CLEAN_VERSION#V}"

info "Starting macOS Release Build for twsnmpmw (arm64)"
info "Version: ${VERSION} (SemVer: ${CLEAN_VERSION}, Commit: ${COMMIT})"

# Step 1: Build and package .app bundle via Wails v3
info "Building twsnmpmw.app for darwin/arm64..."
wails3 task darwin:package ARCH=arm64 VERSION="${VERSION}" COMMIT="${COMMIT}"

ORIG_APP_PATH="bin/twsnmpmw.app"
if [ ! -d "$ORIG_APP_PATH" ]; then
  error "Build failed: $ORIG_APP_PATH not found."
  exit 1
fi

mkdir -p dist
APP_PATH="dist/twsnmpmw.app"
rm -rf "$APP_PATH"
ditto "$ORIG_APP_PATH" "$APP_PATH"

# Sync bundle Info.plist metadata to match release tag
PLIST_PATH="$APP_PATH/Contents/Info.plist"
plutil -replace CFBundleIdentifier -string "jp.co.twise.twsnmpmw" "$PLIST_PATH"
plutil -replace CFBundleShortVersionString -string "${CLEAN_VERSION}" "$PLIST_PATH"
plutil -replace CFBundleVersion -string "${CLEAN_VERSION}" "$PLIST_PATH"

# Clear Finder attributes and ad-hoc signature before Developer ID signing
xattr -cr "$APP_PATH"

# Step 2: Select Developer ID Application certificate
APP_IDENTITY="${MACOS_APP_IDENTITY:-}"
if [ -z "$APP_IDENTITY" ] && [ -f "$HOME/.config/wails/defaults.yaml" ]; then
  APP_IDENTITY=$(awk '/^[[:space:]]*identity:/ {sub(/^[[:space:]]*identity:[[:space:]]*/, ""); gsub(/^["'\'']|["'\'']$/, ""); print; exit}' "$HOME/.config/wails/defaults.yaml" || true)
fi

if [ -z "$APP_IDENTITY" ]; then
  info "Scanning Keychain for 'Developer ID Application' certificates..."
  identities=$(security find-identity -v -p codesigning)
  app_certs=()

  while IFS= read -r line; do
    if [[ "$line" =~ \"(Developer\ ID\ Application:[^\"]+)\" ]]; then
      app_certs+=("${BASH_REMATCH[1]}")
    fi
  done <<< "$identities"

  if [ ${#app_certs[@]} -eq 0 ]; then
    error "No valid 'Developer ID Application' certificate found in Keychain."
    error "Set MACOS_APP_IDENTITY or import a valid certificate."
    exit 1
  elif [ ${#app_certs[@]} -eq 1 ]; then
    APP_IDENTITY="${app_certs[0]}"
  else
    info "Multiple certificates found. Please select one:"
    for i in "${!app_certs[@]}"; do
      echo "  $((i+1))) ${app_certs[i]}"
    done
    while true; do
      read -r -p "Select certificate number (1-${#app_certs[@]}): " cert_idx < /dev/tty
      if [[ "$cert_idx" =~ ^[0-9]+$ ]] && [ "$cert_idx" -ge 1 ] && [ "$cert_idx" -le "${#app_certs[@]}" ]; then
        APP_IDENTITY="${app_certs[$((cert_idx-1))]}"
        break
      fi
      warn "Invalid selection. Enter a number between 1 and ${#app_certs[@]}."
    done
  fi
fi

info "Selected Signing Identity: $APP_IDENTITY"

# Step 3: Sign .app bundle with Hardened Runtime + Timestamp
info "Signing $APP_PATH with Hardened Runtime..."
codesign --force --deep --options runtime --timestamp --sign "$APP_IDENTITY" "$APP_PATH"
codesign --verify --deep --strict "$APP_PATH"
success "Code signing verified."

# Step 4: Determine Notarization keychain profile
NOTARY_PROFILE="${MACOS_NOTARY_PROFILE:-}"
if [ -z "$NOTARY_PROFILE" ] && [ -f "$HOME/.config/wails/defaults.yaml" ]; then
  NOTARY_PROFILE=$(awk '/^[[:space:]]*keychainProfile:/ {sub(/^[[:space:]]*keychainProfile:[[:space:]]*/, ""); gsub(/^["'\'']|["'\'']$/, ""); print; exit}' "$HOME/.config/wails/defaults.yaml" || true)
fi

if [ -z "$NOTARY_PROFILE" ]; then
  read -r -p "Enter Apple Notarization keychain profile name (e.g. WAILS3): " NOTARY_PROFILE < /dev/tty
  if [ -z "$NOTARY_PROFILE" ]; then
    error "Notarization keychain profile is required."
    exit 1
  fi
fi

# Step 5: Notarize and staple the .app bundle
NOTARIZE_ZIP="dist/twsnmpmw-notarize.zip"
rm -f "$NOTARIZE_ZIP"
info "Creating archive for notarization..."
ditto -c -k --keepParent "$APP_PATH" "$NOTARIZE_ZIP"

info "Submitting to Apple Notary service using profile '$NOTARY_PROFILE'..."
xcrun notarytool submit "$NOTARIZE_ZIP" --keychain-profile "$NOTARY_PROFILE" --wait
rm -f "$NOTARIZE_ZIP"

info "Stapling notarization ticket to $APP_PATH..."
xcrun stapler staple "$APP_PATH"
xcrun stapler validate "$APP_PATH"
codesign --verify --deep --strict "$APP_PATH"
spctl --assess -a -t exec -vv "$APP_PATH"
success "Notarization, stapling, and Gatekeeper checks passed."

# Step 6: Package stapled .app into release zip for Wails v3 Updater
FINAL_ZIP_NAME="twsnmpmw-darwin-arm64.zip"
FINAL_ZIP="dist/${FINAL_ZIP_NAME}"
rm -f "$FINAL_ZIP" "${FINAL_ZIP}.sha256"

ditto -c -k --keepParent "$APP_PATH" "$FINAL_ZIP"

# Ensure the archive has only a single top-level entry (twsnmpmw.app/) required by Wails v3 updater
TOP_LEVEL_ENTRIES=$(zipinfo -1 "$FINAL_ZIP" | awk -F/ 'NF>=1 && $1 != "" {print $1}' | sort -u)
if [ "$TOP_LEVEL_ENTRIES" != "twsnmpmw.app" ]; then
  error "Unexpected top-level archive entries in $FINAL_ZIP: $TOP_LEVEL_ENTRIES"
  exit 1
fi

(cd dist && shasum -a 256 "$FINAL_ZIP_NAME" > "${FINAL_ZIP_NAME}.sha256")
MAC_SHA256=$(cat "${FINAL_ZIP}.sha256")

success "macOS release artifact created: $FINAL_ZIP"
echo ""
echo "=== Next Manual Release Steps ==="
echo "1. Open the Draft GitHub Release for ${VERSION}."
echo "2. Upload ${FINAL_ZIP}."
echo "3. Download SHA256SUMS from the Draft Release, append the line below, and re-upload SHA256SUMS:"
echo "   ${MAC_SHA256}"
echo "4. Publish the Release."
