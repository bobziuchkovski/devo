#!/bin/sh

# This script cross-compiles binaries for all supported Go platforms and deploys
# the resulting binaries to the DeVo releases page.  The heavy-liftin is done by
# goxc.

die() {
  echo "Error: $*" 1>&2 && exit 1
}

if [ -z "$TRAVIS_TAG" ]; then
  echo "Skipping.  Build is not a tagged release."
  exit 0
else
  DEVO_VERSION=$(echo "$TRAVIS_TAG" | sed 's/^v//')
fi

[ -n "$GITHUB_TOKEN" ] || die "GITHUB_TOKEN must be set"
[ -n "$TARGET_GO_VERSION" ] || die "TARGET_GO_VERSION must be set"
if [ "$TRAVIS_GO_VERSION" != "$TARGET_GO_VERSION" ]; then
  echo "Skipping.  Travis Go $TRAVIS_GO_VERSION doesn't match target Go $TARGET_GO_VERSION"
  exit 0
fi

set -eu
echo "Installing goxc..."
export GOROOT_BOOTSTRAP="$GOROOT"
which goxc || go get github.com/laher/goxc
goxc -wlc default publish-github -apikey="$GITHUB_TOKEN"

# We set -x only *after* the secret GITHUB_TOKEN is set above
echo "Building and deploying with goxc..."
set -x
goxc -t
goxc -pv="$DEVO_VERSION"
