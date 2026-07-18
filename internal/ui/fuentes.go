package ui

import (
	"os"
	"sync"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

var (
	coleccionFuentesInterfazOnce sync.Once
	coleccionFuentesInterfazMemo []font.FontFace
)

// cargarColeccionFuentesInterfaz prepara una colección reutilizable con las
// fuentes preferidas del sistema. Si alguna no está disponible, se omite sin
// interrumpir el resto de la interfaz.
func cargarColeccionFuentesInterfaz() []font.FontFace {
	var coleccion []font.FontFace
	for _, ruta := range rutasFuentesSistemaPreferidas() {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			continue
		}
		faces, err := opentype.ParseCollection(datos)
		if err != nil {
			continue
		}
		coleccion = append(coleccion, faces...)
	}
	return coleccion
}

// coleccionFuentesInterfaz devuelve una copia para evitar que otros appends
// reutilicen accidentalmente la memoria interna cacheada.
func coleccionFuentesInterfaz() []font.FontFace {
	coleccionFuentesInterfazOnce.Do(func() {
		coleccionFuentesInterfazMemo = cargarColeccionFuentesInterfaz()
	})
	return append([]font.FontFace(nil), coleccionFuentesInterfazMemo...)
}

// familiaFuenteInterfaz define un stack base legible con fallback explícito de
// emoji para nombres de archivos y carpetas que usan pictogramas Unicode.
func familiaFuenteInterfaz() font.Typeface {
	familiaSistema := familiaFuenteSistemaPreferida()
	if familiaSistema == "" {
		return font.Typeface("sans-serif, emoji")
	}
	return font.Typeface(familiaSistema + ", sans-serif, emoji")
}
