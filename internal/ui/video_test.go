package ui

import (
	"testing"
	"time"
)

func TestNormalizarInicioLoteVideoNoPideFotogramasDespuesDelFinal(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			FotogramasPorSeg: 12,
		},
	}

	duracion := 5 * time.Second
	inicio := app.normalizarInicioLoteVideo(duracion, duracion)
	ultimoInicio := duracion - app.margenFinalLoteVideo()

	if inicio > ultimoInicio {
		t.Fatalf("el inicio del lote quedó después del último inicio válido: %v > %v", inicio, ultimoInicio)
	}
	if inicio < 0 {
		t.Fatalf("el inicio del lote no puede ser negativo: %v", inicio)
	}
}

func TestDebePrecargarSiguienteLoteSeDetieneCercaDelFinal(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Duracion:         5 * time.Second,
			Posicion:         4 * time.Second,
			FotogramasPorSeg: 12,
			FinBuffer:        5*time.Second - 150*time.Millisecond,
			Fotogramas: []fotogramaBufferVideo{
				{Instante: 4 * time.Second},
			},
		},
	}

	if app.debePrecargarSiguienteLote() {
		t.Fatal("no debería intentar precargar otro lote cuando el buffer ya cubre el tramo final del video")
	}
}

func TestNormalizarInstanteFotogramaVideoAlejaElCienPorCientoDelFinalExacto(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Duracion:         5 * time.Second,
			FotogramasPorSeg: 12,
		},
	}

	instante := app.normalizarInstanteFotogramaVideo(5*time.Second, 5*time.Second)
	if instante >= 5*time.Second {
		t.Fatalf("el instante del fotograma no debería quedar en el final exacto del video: %v", instante)
	}
	if instante < 0 {
		t.Fatalf("el instante del fotograma no puede ser negativo: %v", instante)
	}
}
