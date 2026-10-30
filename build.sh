#!/bin/sh
set -e

# Windows (amd64) ビルド
task windows:build ARCH=amd64
cp bin/twsnmpmw.exe bin/twsnmpmw-amd64.exe

# macOS (.app bundle) パッケージング
task darwin:package

# 配布用アーカイブ作成
cd bin
zip -r twsnmpmw.zip twsnmpmw twsnmpmw.app twsnmpmw*.exe
