#!/usr/bin/env bash
# Downloads the twelve local dharma-talk recordings used by the seed.
# Files are gitignored (about 325 MB); run once per machine.
set -euo pipefail

dest="${1:-assets/audio/dharma-talks}"
base="https://archive.org/download/phap-thoai-ht-tri-quang-moi"
files=(
  001-an-lac 002-an-toan-giao-thong 003-anh-sang-cuoi-cung 004-ban-nga-sai-lam
  005-bao-ve-moi-truong 006-bo-tat-kim-than 007-bo-tat-muoi-phuong 008-bo-tat-nguyet-quang
  009-bo-tat-so-tru 010-bon-tam-vo-luong 011-cau-bo-tat 012-cau-nguyen-hoa-binh
)

mkdir -p "$dest"
for f in "${files[@]}"; do
  out="$dest/$f.mp3"
  if [ -s "$out" ]; then
    echo "skip $f"
    continue
  fi
  echo "fetch $f"
  curl -fL --retry 3 --max-time 600 -o "$out" "$base/$f.mp3"
done
du -sh "$dest"
