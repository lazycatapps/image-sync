#!/bin/sh

# Build script with optional conditional build
# Usage:
#   ./build.sh        - Force rebuild (clean first)
#   ./build.sh fast   - Only build if dist doesn't exist

FORCE_BUILD=true

# Check if "fast" mode is enabled
if [ "$1" = "fast" ]; then
    if [ -d "./dist/web" ]; then
        echo "Dist directory exists, skipping frontend build..."
        exit 0
    fi
    FORCE_BUILD=false
fi

# Clean dist directory if force build
if [ "$FORCE_BUILD" = "true" ]; then
    echo "Cleaning dist directory..."
    rm -rf ./dist
fi

# Create dist directory
mkdir -p dist/web

# Get version from lzc-manifest.yml
APP_VERSION=$(grep '^version:' lzc-manifest.yml | sed 's/version: *//' | tr -d '\r')

# Get git commit information
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_COMMIT_FULL=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "App version: $APP_VERSION"
echo "Git commit: $GIT_COMMIT (branch: $GIT_BRANCH)"
echo "Build time: $BUILD_TIME"

# Build frontend with version and git commit info
echo "Building frontend..."
cd frontend
echo "Installing frontend dependencies..."
npm install
echo "Building frontend..."
REACT_APP_VERSION=$APP_VERSION \
REACT_APP_GIT_COMMIT=$GIT_COMMIT \
REACT_APP_GIT_COMMIT_FULL=$GIT_COMMIT_FULL \
REACT_APP_GIT_BRANCH=$GIT_BRANCH \
REACT_APP_BUILD_TIME=$BUILD_TIME \
npm run build

# Copy frontend build output to dist/web
cp -r build/* ../dist/web/

# Generate env-config.js for LPK deployment
# Use empty string to enable relative path, routes will proxy to backend
cd ..
cat > dist/web/env-config.js << 'EOF'
// Runtime configuration for LPK deployment
// Empty BACKEND_API_URL means using relative path
// The application routes will proxy /api/ to backend service
window._env_ = {
  BACKEND_API_URL: ""
};
EOF

echo "Frontend build completed successfully!"
echo "Generated env-config.js for LPK deployment (using relative path via routes)"
