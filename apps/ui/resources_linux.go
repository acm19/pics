//go:build linux

package main

import "embed"

//go:embed build/resources/linux/exiftool
//go:embed build/resources/linux/jpegoptim
//go:embed build/resources/linux/lib
var resources embed.FS

const platformDir = "linux"
const platformExt = ""
const hasLib = true
