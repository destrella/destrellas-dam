package metadatos

import (
	"image"
	"image/color"
	"testing"
)

func TestSugerirRecorteMateDetectaBarrasHorizontales(t *testing.T) {
	t.Parallel()

	imagen := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	rellenarRectangulo(imagen, image.Rect(0, 0, 320, 20), color.NRGBA{A: 255})
	rellenarRectangulo(imagen, image.Rect(0, 180, 320, 200), color.NRGBA{A: 255})
	rellenarPatronContenido(imagen, image.Rect(0, 20, 320, 180))

	rectangulo, ok := SugerirRecorteMate(imagen)
	if !ok {
		t.Fatal("se esperaba una sugerencia de recorte")
	}

	if rectangulo.Min.Y < 14 || rectangulo.Min.Y > 26 {
		t.Fatalf("inicio Y inesperado: %d", rectangulo.Min.Y)
	}
	if rectangulo.Max.Y < 174 || rectangulo.Max.Y > 186 {
		t.Fatalf("fin Y inesperado: %d", rectangulo.Max.Y)
	}
	if rectangulo.Min.X != 0 || rectangulo.Max.X != 320 {
		t.Fatalf("no se esperaban recortes laterales: %+v", rectangulo)
	}
}

func TestSugerirRecorteMateDetectaBarrasVerticalesConDegradado(t *testing.T) {
	t.Parallel()

	imagen := image.NewNRGBA(image.Rect(0, 0, 220, 320))
	rellenarBarrasVerticalesConDegradado(imagen, 26)
	rellenarPatronContenido(imagen, image.Rect(26, 0, 194, 320))

	rectangulo, ok := SugerirRecorteMate(imagen)
	if !ok {
		t.Fatal("se esperaba una sugerencia de recorte")
	}

	if rectangulo.Min.X < 18 || rectangulo.Min.X > 34 {
		t.Fatalf("inicio X inesperado: %d", rectangulo.Min.X)
	}
	if rectangulo.Max.X < 186 || rectangulo.Max.X > 202 {
		t.Fatalf("fin X inesperado: %d", rectangulo.Max.X)
	}
	if rectangulo.Min.Y != 0 || rectangulo.Max.Y != 320 {
		t.Fatalf("no se esperaban recortes superiores ni inferiores: %+v", rectangulo)
	}
}

func rellenarRectangulo(imagen *image.NRGBA, rectangulo image.Rectangle, tono color.NRGBA) {
	rectangulo = rectangulo.Intersect(imagen.Bounds())
	for y := rectangulo.Min.Y; y < rectangulo.Max.Y; y++ {
		for x := rectangulo.Min.X; x < rectangulo.Max.X; x++ {
			imagen.SetNRGBA(x, y, tono)
		}
	}
}

func rellenarPatronContenido(imagen *image.NRGBA, rectangulo image.Rectangle) {
	rectangulo = rectangulo.Intersect(imagen.Bounds())
	colores := []color.NRGBA{
		{R: 245, G: 94, B: 80, A: 255},
		{R: 251, G: 191, B: 36, A: 255},
		{R: 34, G: 197, B: 94, A: 255},
		{R: 59, G: 130, B: 246, A: 255},
	}
	for y := rectangulo.Min.Y; y < rectangulo.Max.Y; y++ {
		for x := rectangulo.Min.X; x < rectangulo.Max.X; x++ {
			bloque := ((x - rectangulo.Min.X) / 12) + ((y - rectangulo.Min.Y) / 12)
			tono := colores[bloque%len(colores)]
			if (x+y)%7 == 0 {
				tono.R = 255 - tono.R
				tono.G = 255 - tono.G
				tono.B = 255 - tono.B
			}
			imagen.SetNRGBA(x, y, tono)
		}
	}
}

func rellenarBarrasVerticalesConDegradado(imagen *image.NRGBA, grosor int) {
	limites := imagen.Bounds()
	for y := limites.Min.Y; y < limites.Max.Y; y++ {
		proporcion := float64(y-limites.Min.Y) / float64(maximoEnteroRecorte(1, limites.Dy()-1))
		base := uint8(12 + int(24*proporcion))
		for x := 0; x < grosor; x++ {
			imagen.SetNRGBA(x, y, color.NRGBA{R: base, G: base, B: base, A: 255})
			imagen.SetNRGBA(limites.Max.X-1-x, y, color.NRGBA{R: base, G: base, B: base, A: 255})
		}
	}
}
