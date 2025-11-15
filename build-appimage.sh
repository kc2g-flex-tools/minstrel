#!/bin/bash
set -e

# Build AppImage for minstrel
# Usage: ./build-appimage.sh [amd64|arm64]

ARCH=${1:-amd64}
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
APP_NAME="minstrel"
APP_DIR="${APP_NAME}.AppDir"

echo "Building ${APP_NAME} AppImage for ${ARCH} (version: ${VERSION})"

# Clean previous build
rm -rf "${APP_DIR}" "${APP_NAME}-${ARCH}.AppImage"

# Build the binary
echo "Building binary..."
go build -o "${APP_NAME}" .

# Create AppDir structure
mkdir -p "${APP_DIR}/usr/bin"
mkdir -p "${APP_DIR}/usr/lib"
mkdir -p "${APP_DIR}/usr/share/applications"
mkdir -p "${APP_DIR}/usr/share/icons/hicolor/256x256/apps"

# Copy binary
cp "${APP_NAME}" "${APP_DIR}/usr/bin/"

# Copy required libraries
echo "Bundling libraries..."
copy_lib() {
    local lib=$1
    local libpath=$(ldd "${APP_NAME}" | grep "$lib" | awk '{print $3}')
    if [ -n "$libpath" ] && [ -f "$libpath" ]; then
        cp "$libpath" "${APP_DIR}/usr/lib/"
        echo "  Bundled: $libpath"
    else
        echo "  Warning: $lib not found"
    fi
}

copy_lib "libopus.so"
copy_lib "libopusfile.so"
copy_lib "libogg.so"

# Copy any additional dependencies of bundled libraries
for lib in "${APP_DIR}"/usr/lib/*.so*; do
    if [ -f "$lib" ]; then
        # Get dependencies and copy them if they're not system libraries
        ldd "$lib" 2>/dev/null | grep "libogg\|libopus" | awk '{print $3}' | while read dep; do
            if [ -f "$dep" ] && [ ! -f "${APP_DIR}/usr/lib/$(basename $dep)" ]; then
                cp "$dep" "${APP_DIR}/usr/lib/"
                echo "  Bundled dependency: $dep"
            fi
        done
    fi
done

# Create desktop file
cat > "${APP_DIR}/usr/share/applications/${APP_NAME}.desktop" << 'EOF'
[Desktop Entry]
Type=Application
Name=Minstrel
Comment=FlexRadio control and waterfall display
Exec=minstrel
Icon=minstrel
Categories=HamRadio;Network;
Terminal=false
EOF

# Create icon
mkdir -p "${APP_DIR}/usr/share/icons/hicolor/scalable/apps"
if [ -f "minstrel.svg" ]; then
    cp "minstrel.svg" "${APP_DIR}/usr/share/icons/hicolor/scalable/apps/${APP_NAME}.svg"
    cp "minstrel.svg" "${APP_DIR}/${APP_NAME}.svg"

    # Also create PNG version if rsvg-convert is available
    if command -v rsvg-convert &> /dev/null; then
        rsvg-convert -w 256 -h 256 "minstrel.svg" -o "${APP_DIR}/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png"
        cp "${APP_DIR}/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png" "${APP_DIR}/${APP_NAME}.png"
    else
        # Use SVG as fallback
        cp "minstrel.svg" "${APP_DIR}/${APP_NAME}.png"
    fi
else
    echo "Warning: No icon found at minstrel.svg"
fi

# Create AppRun script
cat > "${APP_DIR}/AppRun" << 'EOF'
#!/bin/bash
SELF=$(readlink -f "$0")
HERE=${SELF%/*}

# Set up library path to find bundled libraries
export LD_LIBRARY_PATH="${HERE}/usr/lib:${LD_LIBRARY_PATH}"

# Execute the application
exec "${HERE}/usr/bin/minstrel" "$@"
EOF

chmod +x "${APP_DIR}/AppRun"

# Copy desktop file and icon to root of AppDir
cp "${APP_DIR}/usr/share/applications/${APP_NAME}.desktop" "${APP_DIR}/"

# Download appimagetool if not present
APPIMAGETOOL="appimagetool-${ARCH}.AppImage"
if [ ! -f "$APPIMAGETOOL" ]; then
    echo "Downloading appimagetool..."
    if [ "$ARCH" = "arm64" ]; then
        wget -q "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-aarch64.AppImage" -O "$APPIMAGETOOL"
    else
        wget -q "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage" -O "$APPIMAGETOOL"
    fi
    chmod +x "$APPIMAGETOOL"
fi

# Build the AppImage
echo "Creating AppImage..."
ARCH=${ARCH} ./"$APPIMAGETOOL" "${APP_DIR}" "${APP_NAME}-${VERSION}-${ARCH}.AppImage"

echo ""
echo "AppImage created: ${APP_NAME}-${VERSION}-${ARCH}.AppImage"
echo ""
echo "To run: ./${APP_NAME}-${VERSION}-${ARCH}.AppImage"
