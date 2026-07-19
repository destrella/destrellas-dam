package ui

import (
	"image"
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestGuardarPreviewPodaEntradasMasAntiguas(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		previews:              make(map[string]*estadoPreview),
		limiteMemoriaPreviews: 1800,
		limiteCantidadPreview: 8,
	}

	app.guardarPreview("uno", &estadoPreview{Imagen: image.NewNRGBA(image.Rect(0, 0, 20, 20))})
	app.guardarPreview("dos", &estadoPreview{Imagen: image.NewNRGBA(image.Rect(0, 0, 20, 20))})

	if _, existe := app.previews["uno"]; existe {
		t.Fatal("la preview menos reciente debería expulsarse cuando la caché supera el presupuesto")
	}
	if _, existe := app.previews["dos"]; !existe {
		t.Fatal("la preview más reciente debería permanecer en caché")
	}
}

func TestGuardarPreviewProtegeElArchivoActivo(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		previews:              make(map[string]*estadoPreview),
		limiteMemoriaPreviews: 1800,
		limiteCantidadPreview: 8,
		tieneArchivoActivo:    true,
		archivoActivo:         modelo.Archivo{Ruta: "uno"},
	}

	app.guardarPreview("uno", &estadoPreview{Imagen: image.NewNRGBA(image.Rect(0, 0, 20, 20))})
	app.guardarPreview("dos", &estadoPreview{Imagen: image.NewNRGBA(image.Rect(0, 0, 20, 20))})

	if _, existe := app.previews["uno"]; !existe {
		t.Fatal("la preview del archivo activo no debería expulsarse")
	}
	if _, existe := app.previews["dos"]; existe {
		t.Fatal("la preview no protegida debería expulsarse primero")
	}
}

func TestCompactarPreviewParaCacheReduceLaResolucion(t *testing.T) {
	t.Parallel()

	imagenOriginal := image.NewNRGBA(image.Rect(0, 0, 2000, 1000))
	compactada := compactarPreviewParaCache(imagenOriginal, 500)

	if compactada.Bounds().Dx() != 500 {
		t.Fatalf("ancho inesperado tras compactar la preview: %d", compactada.Bounds().Dx())
	}
	if compactada.Bounds().Dy() != 250 {
		t.Fatalf("alto inesperado tras compactar la preview: %d", compactada.Bounds().Dy())
	}
}
