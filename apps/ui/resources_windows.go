//go:build windows

package main

import "embed"

//go:embed build/resources/windows/exiftool.exe
//go:embed build/resources/windows/jpegoptim.exe
var resources embed.FS

const platformDir = "windows"
const platformExt = ".exe"
const hasLib = false
