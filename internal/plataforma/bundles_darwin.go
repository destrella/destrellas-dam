//go:build darwin

package plataforma

import (
	"path/filepath"
	"strings"
)

var extensionesBundle = map[string]struct{}{
	".app":            {},
	".appex":          {},
	".band":           {},
	".bundle":         {},
	".download":       {},
	".fcpbundle":      {},
	".framework":      {},
	".idocument":      {},
	".imovielibrary":  {},
	".key":            {},
	".kext":           {},
	".mailbundle":     {},
	".mpkg":           {},
	".numbers":        {},
	".pages":          {},
	".photolibrary":   {},
	".photoslibrary":  {},
	".pkg":            {},
	".playground":     {},
	".playgroundbook": {},
	".plugin":         {},
	".prefpane":       {},
	".qlgenerator":    {},
	".rtfd":           {},
	".savedsearch":    {},
	".scptd":          {},
	".sparsebundle":   {},
	".theater":        {},
	".wdgt":           {},
	".workflow":       {},
	".xcodeproj":      {},
}

// EsBundle informa si una ruta de macOS debe tratarse como paquete en lugar de carpeta navegable.
func EsBundle(ruta string) bool {
	extension := strings.ToLower(strings.TrimSpace(filepath.Ext(ruta)))
	_, existe := extensionesBundle[extension]
	return existe
}
