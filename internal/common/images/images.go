package images

import (
	_ "embed"
)

//go:embed opsicle-cat.png
var opsicleCat []byte

func GetOpsicleCat() (mimeType string, data []byte) {
	return "image/png", opsicleCat
}
