#!/usr/bin/env bash
set -euo pipefail

PIN=4ad545491e8a141a72bb52b8d719b1842c5a7d1b

if [[ -d lizard-upstream/.git ]]; then
    git -C lizard-upstream fetch --depth 1 origin "$PIN"
    git -C lizard-upstream checkout "$PIN"
else
    git clone --depth 1 https://github.com/terryyin/lizard.git lizard-upstream
    git -C lizard-upstream fetch --depth 1 origin "$PIN"
    git -C lizard-upstream checkout "$PIN"
fi

# Mirror testdata into our tree so per-language tests don't need the upstream tree at runtime.
rsync -a --delete lizard-upstream/test/test_languages/testdata/ testdata/
