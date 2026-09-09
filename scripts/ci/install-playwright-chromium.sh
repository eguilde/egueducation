#!/usr/bin/env bash

set -euo pipefail

# GitHub's Ubuntu image enables the Google Chrome APT repository even though
# these jobs install Playwright Chromium. During a repository publication the
# Release metadata and Packages file can briefly come from different CDN
# generations, making every unrelated apt operation fail with Hash Sum
# mismatch. Disable only that unused external source on the ephemeral runner.
for google_source in \
  /etc/apt/sources.list.d/google-chrome.list \
  /etc/apt/sources.list.d/google-chrome.sources
do
  if [[ -f "${google_source}" ]]; then
    sudo mv -- "${google_source}" "${google_source}.egueducation-disabled"
  fi
done

for attempt in 1 2 3 4; do
  if npx playwright install --with-deps chromium; then
    exit 0
  fi

  if [[ "${attempt}" -eq 4 ]]; then
    echo "Playwright Chromium installation failed after ${attempt} attempts." >&2
    exit 1
  fi

  sudo apt-get clean
  sleep "$((attempt * 10))"
done
