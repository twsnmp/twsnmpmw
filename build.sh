#!/bin/sh
wails3 package windows:amd64
wails3 task package:darwin:amd64:test
cd bin
zip -r twsnmpmw.zip twsnmpmw twsnmpmw.app twsnmpmw*.exe
