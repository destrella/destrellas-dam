package metadatos

import (
	"image"
	"math"
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestRecalcularRegionesTrasRecorteMantieneProporciones(t *testing.T) {
	t.Parallel()

	archivo := modelo.Archivo{
		Ancho: 1000,
		Alto:  500,
		Metadatos: modelo.MetadatosArchivo{
			Regiones: []modelo.RegionEtiquetada{
				{Nombre: "Centro", X: 0.2, Y: 0.1, Ancho: 0.4, Alto: 0.6},
			},
		},
	}

	regiones := recalcularRegionesTrasRecorte(archivo, image.Rect(100, 50, 700, 350))
	if len(regiones) != 1 {
		t.Fatalf("se esperaba una región ajustada, se obtuvieron %d", len(regiones))
	}

	region := regiones[0]
	verificarFraccionCercana(t, region.X, 0.166667, "X")
	verificarFraccionCercana(t, region.Y, 0, "Y")
	verificarFraccionCercana(t, region.Ancho, 0.666667, "Ancho")
	verificarFraccionCercana(t, region.Alto, 1, "Alto")
}

func TestRecalcularRegionesTrasRecorteRespetaOrientacionVisual(t *testing.T) {
	t.Parallel()

	regionOriginal := modelo.RegionEtiquetada{Nombre: "Cara", X: 0.25, Y: 0.25, Ancho: 0.5, Alto: 0.5}
	archivo := modelo.Archivo{
		Ancho: 400,
		Alto:  800,
		Metadatos: modelo.MetadatosArchivo{
			Orientacion: 6,
			Regiones:    []modelo.RegionEtiquetada{regionOriginal},
		},
	}

	regionOrientada := modelo.TransformarRegionOrientada(regionOriginal, archivo.Metadatos.Orientacion)
	rect := image.Rect(
		int(regionOrientada.X*800),
		int(regionOrientada.Y*400),
		int((regionOrientada.X+regionOrientada.Ancho)*800),
		int((regionOrientada.Y+regionOrientada.Alto)*400),
	)

	regiones := recalcularRegionesTrasRecorte(archivo, rect)
	if len(regiones) != 1 {
		t.Fatalf("se esperaba una región ajustada, se obtuvieron %d", len(regiones))
	}

	region := regiones[0]
	verificarFraccionCercana(t, region.X, 0, "X")
	verificarFraccionCercana(t, region.Y, 0, "Y")
	verificarFraccionCercana(t, region.Ancho, 1, "Ancho")
	verificarFraccionCercana(t, region.Alto, 1, "Alto")
}

func verificarFraccionCercana(t *testing.T, obtenida, esperada float64, campo string) {
	t.Helper()
	if math.Abs(obtenida-esperada) > 0.00001 {
		t.Fatalf("%s inesperada: %.6f, se esperaba %.6f", campo, obtenida, esperada)
	}
}
