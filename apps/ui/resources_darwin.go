//go:build darwin

package main

import "embed"

//go:embed build/resources/darwin/exiftool
//go:embed build/resources/darwin/jpegoptim
//go:embed build/resources/darwin/lib
var resources embed.FS

const platformDir = "darwin"
const platformExt = ""
const hasLib = true
