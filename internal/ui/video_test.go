package ui

import (
	"testing"
	"time"

	"destrellas-dam/internal/modelo"
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

func TestAlternarReproductorVideoReiniciaSiYaHabiaTerminado(t *testing.T) {
	t.Parallel()

	ruta := "/tmp/video-prueba.mp4"
	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo: modelo.Archivo{
			Ruta:     ruta,
			Tipo:     modelo.TipoVideo,
			Duracion: 5 * time.Second,
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:            ruta,
			Duracion:        5 * time.Second,
			Posicion:        5 * time.Second,
			MaximoFotograma: 960,
		},
	}

	app.alternarReproductorVideo()

	if !app.reproductorVideo.Reproduciendo {
		t.Fatal("la reproducción debería reanudarse al pulsar play desde el final")
	}
	if app.reproductorVideo.Posicion != 0 {
		t.Fatalf("la reproducción debería reiniciarse desde el inicio, se obtuvo %v", app.reproductorVideo.Posicion)
	}
}

func TestPrepararInicioReproductorVideoInvalidaSolicitudesPendientes(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Cargando:          true,
			TienePendiente:    true,
			InstantePendiente: 3 * time.Second,
			MaximoPendiente:   640,
			VersionSolicitud:  4,
			Posicion:          5 * time.Second,
		},
	}

	app.prepararInicioReproductorVideo(false)

	if app.reproductorVideo.Cargando {
		t.Fatal("el reinicio debería cancelar la solicitud de fotogramas activa")
	}
	if app.reproductorVideo.TienePendiente {
		t.Fatal("el reinicio debería descartar la solicitud pendiente")
	}
	if app.reproductorVideo.Posicion != 0 {
		t.Fatalf("la posición debería volver al inicio, se obtuvo %v", app.reproductorVideo.Posicion)
	}
	if app.reproductorVideo.VersionSolicitud != 5 {
		t.Fatalf("la versión de la solicitud debería invalidarse, se obtuvo %d", app.reproductorVideo.VersionSolicitud)
	}
}

func TestResolverPosicionFinReproductorVideoSinLoopSeDetiene(t *testing.T) {
	t.Parallel()

	posicion, sigue := resolverPosicionFinReproductorVideo(5300*time.Millisecond, 5*time.Second, false)
	if sigue {
		t.Fatal("sin loop la reproducción debería detenerse al llegar al final")
	}
	if posicion != 5*time.Second {
		t.Fatalf("la posición final sin loop debería quedar al final exacto, se obtuvo %v", posicion)
	}
}

func TestResolverPosicionFinReproductorVideoConLoopReinicia(t *testing.T) {
	t.Parallel()

	posicion, sigue := resolverPosicionFinReproductorVideo(5300*time.Millisecond, 5*time.Second, true)
	if !sigue {
		t.Fatal("con loop la reproducción debería continuar")
	}
	if posicion != 0 {
		t.Fatalf("la posición en loop debería reiniciarse en el inicio exacto, se obtuvo %v", posicion)
	}
}

func TestControlVideoFueManipuladoPorUsuario(t *testing.T) {
	t.Parallel()

	if controlVideoFueManipuladoPorUsuario(1, 0, false) {
		t.Fatal("un cambio programático del slider no debería tratarse como una interacción del usuario")
	}
	if !controlVideoFueManipuladoPorUsuario(0.2, 0.4, true) {
		t.Fatal("un cambio mientras se arrastra el slider sí debería tratarse como una interacción del usuario")
	}
}

func TestCambiarVistaDetieneReproduccionAlSalirDelVisor(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		vistaActual: vistaElementoUnico,
		reproductorVideo: estadoReproductorVideo{
			Reproduciendo: true,
			UltimoTick:    time.Now(),
		},
	}

	app.cambiarVista(vistaPrincipal)

	if app.vistaActual != vistaPrincipal {
		t.Fatalf("la vista actual debería cambiar a explorador, se obtuvo %q", app.vistaActual)
	}
	if app.reproductorVideo.Reproduciendo {
		t.Fatal("la reproducción debería detenerse al salir del visor")
	}
	if !app.reproductorVideo.UltimoTick.IsZero() {
		t.Fatal("el reloj interno del reproductor debería reiniciarse al salir del visor")
	}
}
