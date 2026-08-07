package ui

import (
	"image"
	"math"
	"testing"
	"time"

	"gioui.org/widget"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/metadatos"
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

func TestSincronizarReproductorVideoUsaFPSDelArchivo(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			FotogramasPorSeg: 12,
		},
	}

	app.sincronizarReproductorVideo(modelo.Archivo{
		Ruta:                 "/tmp/video-prueba.mp4",
		Tipo:                 modelo.TipoVideo,
		FotogramasPorSegundo: 23.976,
	})

	if app.reproductorVideo.FotogramasPorSeg != 23.976 {
		t.Fatalf("se esperaba que el reproductor adoptara el fps del archivo, se obtuvo %v", app.reproductorVideo.FotogramasPorSeg)
	}
}

func TestVisorUsaReproductorParaVideoRemoto(t *testing.T) {
	if !visorUsaReproductorVideo(modelo.Archivo{
		Origen: modelo.OrigenYandex,
		Tipo:   modelo.TipoVideo,
	}) {
		t.Fatal("el visor debería mostrar el reproductor para videos remotos")
	}
	if visorUsaReproductorVideo(modelo.Archivo{
		Origen: modelo.OrigenYandex,
		Tipo:   modelo.TipoImagen,
	}) {
		t.Fatal("el visor no debería mostrar el reproductor para imágenes remotas")
	}
}

func TestAlternarReproductorVideoDejaPendienteElPlayMientrasCarganMetadatos(t *testing.T) {
	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo: modelo.Archivo{
			Origen: modelo.OrigenYandex,
			Ruta:   "disk:/media/Videos/prueba.mp4",
			Tipo:   modelo.TipoVideo,
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:              "disk:/media/Videos/prueba.mp4",
			MetadatosCargando: true,
		},
	}

	app.alternarReproductorVideo()

	if !app.reproductorVideo.Reproduciendo || !app.reproductorVideo.ReproduccionPendiente {
		t.Fatal("el primer play debería quedar pendiente mientras se cargan los metadatos")
	}
	if !app.reproductorVideo.Cargando || !app.reproductorVideo.MostrarCarga {
		t.Fatal("el primer play debería mostrar la carga mientras espera los metadatos")
	}
}

func TestCacheLotesVideoReutilizaUnBloque(t *testing.T) {
	app := &Aplicacion{}
	fotogramas := []fotogramaBufferVideo{{Instante: time.Second, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
	app.guardarLoteCacheVideo(time.Second, 960, fotogramas)

	obtenidos, existe := app.buscarLoteCacheVideo(time.Second, 640)
	if !existe || len(obtenidos) != 1 || obtenidos[0].Instante != time.Second {
		t.Fatal("la caché debería reutilizar el lote con un máximo menor")
	}
	if _, existe := app.buscarLoteCacheVideo(time.Second, 1_200); existe {
		t.Fatal("la caché no debería reutilizar un lote con resolución insuficiente")
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

func TestDebePrecargarSiguienteLoteEnLoopPideElInicio(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproducirVideoEnLoop: true,
		reproductorVideo: estadoReproductorVideo{
			Duracion:     5 * time.Second,
			Posicion:     4 * time.Second,
			InicioBuffer: 1 * time.Second,
			FinBuffer:    5*time.Second - 120*time.Millisecond,
			Fotogramas:   []fotogramaBufferVideo{{Instante: 4 * time.Second}},
		},
	}

	if !app.debePrecargarSiguienteLote() {
		t.Fatal("en loop debería precargar el inicio cuando el buffer ya cubre el final")
	}
	if inicio := app.inicioSiguienteLote(); inicio != 0 {
		t.Fatalf("en loop el siguiente lote debería comenzar en 0, se obtuvo %v", inicio)
	}

	app.reproductorVideo.FotogramasInicioLoop = []fotogramaBufferVideo{{Instante: 0}}
	if app.debePrecargarSiguienteLote() {
		t.Fatal("no debería solicitar otra precarga del inicio mientras ya existe una lista preparada")
	}
}

func TestDebeAnticiparSiguienteLoteAlRecibirElBufferInicial(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Reproduciendo:    true,
			FotogramasPorSeg: 12,
			Posicion:         0,
			InicioBuffer:     0,
			FinBuffer:        4 * time.Second,
			Fotogramas:       []fotogramaBufferVideo{{Instante: 0}},
		},
	}

	if !app.debeAnticiparSiguienteLote() {
		t.Fatal("el primer bloque debería solicitar inmediatamente el siguiente lote")
	}

	app.reproductorVideo.FinBuffer = 7 * time.Second
	if app.debeAnticiparSiguienteLote() {
		t.Fatal("no debería encadenar más de un lote adelantado")
	}
}

func TestActivarBufferInicioLoopCambiaAlPrimerFotograma(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Posicion:     5 * time.Second,
			Fotogramas:   []fotogramaBufferVideo{{Instante: 4 * time.Second}},
			InicioBuffer: 4 * time.Second,
			FinBuffer:    5 * time.Second,
			FotogramasInicioLoop: []fotogramaBufferVideo{
				{Instante: 0, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
			},
		},
	}
	app.reproductorVideo.Posicion = 0

	if !app.activarBufferInicioLoop() {
		t.Fatal("debería activar el buffer precargado del inicio")
	}
	if app.reproductorVideo.InicioBuffer != 0 || app.reproductorVideo.FinBuffer != 500*time.Millisecond {
		t.Fatalf("el buffer activo debería cubrir el inicio, se obtuvo %v-%v", app.reproductorVideo.InicioBuffer, app.reproductorVideo.FinBuffer)
	}
	if app.reproductorVideo.Fotograma == nil {
		t.Fatal("debería mostrar el primer fotograma del nuevo ciclo")
	}
}

func TestActivarBufferInicioLoopPendienteAlLlegarDespuesDelFinal(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproducirVideoEnLoop: true,
		reproductorVideo: estadoReproductorVideo{
			Reproduciendo: true,
			Posicion:      0,
			Fotogramas:    []fotogramaBufferVideo{{Instante: 4 * time.Second}},
			InicioBuffer:  4 * time.Second,
			FinBuffer:     5 * time.Second,
			FotogramasInicioLoop: []fotogramaBufferVideo{
				{Instante: 0, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
			},
		},
	}

	app.activarBufferInicioLoopPendiente()

	if app.reproductorVideo.InicioBuffer != 0 || app.reproductorVideo.FinBuffer != 500*time.Millisecond {
		t.Fatalf("el lote que llegó tarde debería activarse en el nuevo ciclo, se obtuvo %v-%v", app.reproductorVideo.InicioBuffer, app.reproductorVideo.FinBuffer)
	}
	if app.reproductorVideo.Fotograma == nil {
		t.Fatal("el nuevo ciclo debería mostrar el primer fotograma disponible")
	}
}

func TestDescartarBufferFotogramasConservaElInicioDelLoop(t *testing.T) {
	t.Parallel()

	inicio := []fotogramaBufferVideo{{Instante: 0, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Fotogramas:           []fotogramaBufferVideo{{Instante: 4 * time.Second}},
			FotogramasInicioLoop: inicio,
			InicioBuffer:         4 * time.Second,
			FinBuffer:            5 * time.Second,
		},
	}

	app.descartarBufferFotogramas()

	if len(app.reproductorVideo.Fotogramas) != 0 {
		t.Fatal("debería descartar el buffer del tramo actual")
	}
	if len(app.reproductorVideo.FotogramasInicioLoop) != 1 {
		t.Fatal("debería conservar el buffer del inicio para el siguiente ciclo")
	}
}

func TestIntervaloFotogramasBufferUsaFPSReal(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			FotogramasPorSeg: 23.976,
		},
	}

	intervalo := app.intervaloFotogramasBuffer()
	esperado := time.Duration(math.Round(float64(time.Second) / 23.976))
	if intervalo != esperado {
		t.Fatalf("intervalo inesperado: %v != %v", intervalo, esperado)
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

func TestInvalidarSolicitudesFotogramasVideoCancelaLotePendiente(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Cargando:            true,
			MostrarCarga:        true,
			TieneLotePendiente:  true,
			InicioLotePendiente: 2 * time.Second,
			MaximoLotePendiente: 640,
			InstantePendiente:   3 * time.Second,
			MaximoPendiente:     480,
			TienePendiente:      true,
			VersionSolicitud:    9,
		},
	}

	app.invalidarSolicitudesFotogramasVideo()

	if app.reproductorVideo.Cargando {
		t.Fatal("la invalidación debería cancelar la carga en curso")
	}
	if app.reproductorVideo.MostrarCarga {
		t.Fatal("la invalidación debería ocultar el indicador de carga")
	}
	if app.reproductorVideo.TienePendiente {
		t.Fatal("la invalidación debería limpiar la solicitud pendiente de fotograma")
	}
	if app.reproductorVideo.TieneLotePendiente {
		t.Fatal("la invalidación debería limpiar la solicitud pendiente de lote")
	}
	if app.reproductorVideo.VersionSolicitud != 10 {
		t.Fatalf("la versión debería incrementarse, se obtuvo %d", app.reproductorVideo.VersionSolicitud)
	}
}

func TestEncolarSolicitudLoteFotogramasVideoReemplazaPendiente(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Duracion:            5 * time.Second,
			TieneLotePendiente:  true,
			InicioLotePendiente: 4 * time.Second,
			MaximoLotePendiente: 640,
		},
	}

	app.encolarSolicitudLoteFotogramasVideo(0, 960)

	if !app.reproductorVideo.TieneLotePendiente {
		t.Fatal("debería quedar una solicitud de lote pendiente")
	}
	if app.reproductorVideo.InicioLotePendiente != 0 {
		t.Fatalf("el lote pendiente debería apuntar al inicio, se obtuvo %v", app.reproductorVideo.InicioLotePendiente)
	}
	if app.reproductorVideo.MaximoLotePendiente != 960 {
		t.Fatalf("el tamaño del lote pendiente debería actualizarse, se obtuvo %d", app.reproductorVideo.MaximoLotePendiente)
	}
}

func TestDebeMostrarIndicadorCargaVideoCuandoNoHayBuffer(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			Cargando:      true,
			MostrarCarga:  true,
			Reproduciendo: true,
		},
	}

	if !app.debeMostrarIndicadorCargaVideo() {
		t.Fatal("el indicador debería mostrarse cuando aún no hay buffer, aunque la reproducción ya haya comenzado")
	}

	app.reproductorVideo.MostrarCarga = false
	if app.debeMostrarIndicadorCargaVideo() {
		t.Fatal("el indicador no debería mostrarse cuando la UI ya dejó de esperar carga visible")
	}

	app.reproductorVideo.Reproduciendo = false
	if !app.debeMostrarIndicadorCargaVideo() {
		t.Fatal("el indicador debería mostrarse durante una carga aunque la reproducción esté pausada")
	}
}

func TestAlternarLoopReproductorVideoCambiaElEstado(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{}
	app.alternarLoopReproductorVideo()
	if !app.reproducirVideoEnLoop {
		t.Fatal("el primer cambio debería activar el loop")
	}
	app.alternarLoopReproductorVideo()
	if app.reproducirVideoEnLoop {
		t.Fatal("el segundo cambio debería desactivar el loop")
	}
}

func TestAudioVideoDisponibleRequierePistaDeclarada(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo: modelo.Archivo{
			Tipo:       modelo.TipoVideo,
			TieneAudio: true,
		},
	}

	if !app.audioVideoDisponible() {
		t.Fatal("debería considerar disponible el audio cuando el archivo de video lo declara")
	}

	app.archivoActivo.TieneAudio = false
	if app.audioVideoDisponible() {
		t.Fatal("no debería considerar disponible el audio cuando el video no tiene pista")
	}
}

func TestMaximoFotogramaBufferReduceResolucionEnAltaTasa(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{FotogramasPorSeg: 60},
	}

	altaTasa := app.maximoFotogramaBuffer(960)
	if altaTasa >= 960 || altaTasa < 480 {
		t.Fatalf("60 FPS debería reducir la resolución del buffer dentro de un rango seguro, se obtuvo %d", altaTasa)
	}

	app.reproductorVideo.FotogramasPorSeg = 30
	if normal := app.maximoFotogramaBuffer(960); normal != 960 {
		t.Fatalf("30 FPS debería conservar la resolución solicitada, se obtuvo %d", normal)
	}
}

func TestLoteAltaTasaAmpliaCoberturaYMargen(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{FotogramasPorSeg: 60},
	}

	if cantidad := app.cantidadFotogramasLoteBuffer(); cantidad != 360 {
		t.Fatalf("60 FPS debería usar un lote de seis segundos, se obtuvieron %d fotogramas", cantidad)
	}
	if margen := app.margenPrecargaBuffer(); diferenciaDuracion(margen, 4500*time.Millisecond) > app.intervaloFotogramasBuffer() {
		t.Fatalf("el lote de alta tasa debería solicitarse con 4.5 s de margen, se obtuvo %v", margen)
	}
}

func TestNoConservaCacheDuplicadoEnAltaTasa(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproducirVideoEnLoop: true,
		reproductorVideo:      estadoReproductorVideo{FotogramasPorSeg: 60},
	}
	archivo := modelo.Archivo{Origen: modelo.OrigenYandex}

	if app.debeConservarLoteCacheVideo(archivo) {
		t.Fatal("los videos de alta tasa no deberían duplicar lotes completos en cache")
	}
}

func TestIntegrarLoteFotogramasPreprendeInicioEnLoop(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproducirVideoEnLoop: true,
		reproductorVideo: estadoReproductorVideo{
			Reproduciendo: true,
			Fotogramas: []fotogramaBufferVideo{
				{Instante: 4 * time.Second, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 4500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
			},
			InicioBuffer: 4 * time.Second,
			FinBuffer:    4500 * time.Millisecond,
		},
	}

	app.integrarLoteFotogramas([]metadatos.FotogramaVideo{
		{Instante: 0, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
		{Instante: 500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
	})

	if len(app.reproductorVideo.Fotogramas) < 3 {
		t.Fatalf("se esperaban fotogramas preprendidos, se obtuvieron %d", len(app.reproductorVideo.Fotogramas))
	}
	if app.reproductorVideo.Fotogramas[0].Instante != 0 {
		t.Fatalf("el buffer debería comenzar en 0 tras precargar el inicio del loop, se obtuvo %v", app.reproductorVideo.Fotogramas[0].Instante)
	}
	if app.reproductorVideo.InicioBuffer != 0 {
		t.Fatalf("el inicio del buffer debería actualizarse a 0, se obtuvo %v", app.reproductorVideo.InicioBuffer)
	}
}

func TestBufferCubreInstanteNoSaltaHuecosEntreLotes(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		reproductorVideo: estadoReproductorVideo{
			FotogramasPorSeg: 12,
			Fotogramas: []fotogramaBufferVideo{
				{Instante: 0, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 4 * time.Second, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
				{Instante: 4500 * time.Millisecond, Imagen: image.NewRGBA(image.Rect(0, 0, 1, 1))},
			},
		},
	}

	if app.bufferCubreInstante(2 * time.Second) {
		t.Fatal("el buffer no debería considerar cubierto un hueco entre lotes lejanos")
	}
	if !app.bufferCubreInstante(4500 * time.Millisecond) {
		t.Fatal("el buffer debería considerar cubierto el instante cercano a un fotograma disponible")
	}
}

func TestActualizarPosicionVideoDesdeControlMantieneReproduccionSiEstabaActiva(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo: modelo.Archivo{
			Ruta:     "/tmp/video-prueba.mp4",
			Tipo:     modelo.TipoVideo,
			Duracion: 10 * time.Second,
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:          "/tmp/video-prueba.mp4",
			Duracion:      10 * time.Second,
			Reproduciendo: true,
			UltimoTick:    time.Now(),
		},
		controlProgresoVideo: widget.Float{Value: 0.5},
	}

	app.actualizarPosicionVideoDesdeControl(960)

	if !app.reproductorVideo.Reproduciendo {
		t.Fatal("la reproducción debería continuar después de buscar una nueva posición")
	}
	if app.reproductorVideo.Posicion != 5*time.Second {
		t.Fatalf("la posición debería actualizarse a la buscada, se obtuvo %v", app.reproductorVideo.Posicion)
	}
	if !app.reproductorVideo.UltimoTick.IsZero() {
		t.Fatal("el reloj debería reiniciarse para continuar la reproducción desde la nueva posición")
	}
}

func TestActualizarPosicionVideoDesdeControlMuestraFotogramaSiEstaPausado(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		tieneArchivoActivo: true,
		archivoActivo: modelo.Archivo{
			Ruta:     "/tmp/video-prueba.mp4",
			Tipo:     modelo.TipoVideo,
			Duracion: 10 * time.Second,
		},
		reproductorVideo: estadoReproductorVideo{
			Ruta:     "/tmp/video-prueba.mp4",
			Duracion: 10 * time.Second,
		},
		controlProgresoVideo: widget.Float{Value: 0.5},
	}

	app.actualizarPosicionVideoDesdeControl(960)

	if app.reproductorVideo.Reproduciendo {
		t.Fatal("la búsqueda no debería iniciar la reproducción si estaba pausada")
	}
	if app.reproductorVideo.Posicion != 5*time.Second {
		t.Fatalf("la posición debería actualizarse a la buscada, se obtuvo %v", app.reproductorVideo.Posicion)
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
