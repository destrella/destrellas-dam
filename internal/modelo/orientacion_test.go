package modelo

import (
	"math"
	"testing"
)

func TestTransformarEInvertirRegionOrientada(t *testing.T) {
	t.Parallel()

	original := RegionEtiquetada{
		Nombre: "rostro",
		X:      0.12,
		Y:      0.18,
		Ancho:  0.34,
		Alto:   0.27,
	}

	for orientacion := 1; orientacion <= 8; orientacion++ {
		orientacion := orientacion
		t.Run(string(rune('0'+orientacion)), func(t *testing.T) {
			t.Parallel()

			transformada := TransformarRegionOrientada(original, orientacion)
			revertida := InvertirRegionOrientada(transformada, orientacion)

			if !regionesCasiIguales(original, revertida) {
				t.Fatalf("la orientación %d no revierte correctamente: original=%+v revertida=%+v", orientacion, original, revertida)
			}
		})
	}
}

func TestNormalizarRotacionCuartos(t *testing.T) {
	t.Parallel()

	casos := map[int]int{
		0:   0,
		90:  90,
		180: 180,
		270: 270,
		360: 0,
		-90: 270,
		89:  90,
		181: 180,
		359: 0,
	}

	for entrada, esperado := range casos {
		if obtenido := NormalizarRotacionCuartos(entrada); obtenido != esperado {
			t.Fatalf("rotación %d: esperado %d, obtenido %d", entrada, esperado, obtenido)
		}
	}
}

func regionesCasiIguales(izquierda, derecha RegionEtiquetada) bool {
	return math.Abs(izquierda.X-derecha.X) < 0.000001 &&
		math.Abs(izquierda.Y-derecha.Y) < 0.000001 &&
		math.Abs(izquierda.Ancho-derecha.Ancho) < 0.000001 &&
		math.Abs(izquierda.Alto-derecha.Alto) < 0.000001
}
