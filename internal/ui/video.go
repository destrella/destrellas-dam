package ui

import (
	"context"
	"errors"
	"image"
	"math"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/metadatos"
)

type fotogramaBufferVideo struct {
	Instante time.Duration
	Imagen   image.Image
}

type loteCacheVideo struct {
	Inicio     time.Duration
	Maximo     int
	Fotogramas []fotogramaBufferVideo
}

type estadoReproductorVideo struct {
	Ruta                  string
	Duracion              time.Duration
	Rotacion              int
	Posicion              time.Duration
	Fotograma             image.Image
	InstanteFotograma     time.Duration
	MaximoFotograma       int
	Fotogramas            []fotogramaBufferVideo
	FotogramasInicioLoop  []fotogramaBufferVideo
	LotesCache            []loteCacheVideo
	FotogramasPorSeg      float64
	InicioBuffer          time.Duration
	FinBuffer             time.Duration
	Cargando              bool
	MostrarCarga          bool
	Reproduciendo         bool
	UltimoTick            time.Time
	Error                 string
	InstanteError         time.Duration
	MaximoError           int
	AudioError            string
	AudioVerificado       bool
	AudioCargando         bool
	MetadatosCargando     bool
	ReproduccionPendiente bool
	InstantePendiente     time.Duration
	MaximoPendiente       int
	TienePendiente        bool
	InicioLotePendiente   time.Duration
	MaximoLotePendiente   int
	TieneLotePendiente    bool
	VersionSolicitud      int
}

func (a *Aplicacion) limpiarReproductorVideo() {
	a.detenerAudioVideo()
	a.reproductorVideo = estadoReproductorVideo{}
	a.controlProgresoVideo = widget.Float{}
	a.controlExtraccionFrame = widget.Float{}
	a.formatoExtraccionExpandido = false
}

func (a *Aplicacion) detenerReproduccionVideo() {
	a.reproductorVideo.Reproduciendo = false
	a.reproductorVideo.ReproduccionPendiente = false
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.MostrarCarga = false
	if a.audioVideoIniciado && !a.audioVideoListo {
		a.detenerAudioVideo()
	} else {
		a.pausarAudioVideo()
	}
}

func (a *Aplicacion) sincronizarReproductorVideo(archivo modelo.Archivo) {
	if archivo.Tipo != modelo.TipoVideo || archivo.Ruta == "" {
		if a.reproductorVideo.Ruta != "" {
			a.limpiarReproductorVideo()
		}
		return
	}
	if archivo.Origen == modelo.OrigenYandex && a.servicioMetadatos != nil {
		a.servicioMetadatos.RegistrarTamanoFuenteVideo(archivo.Ruta, archivo.Tamano)
	}

	rotacion := modelo.NormalizarRotacionCuartos(archivo.Metadatos.Rotacion)
	if a.reproductorVideo.Ruta != archivo.Ruta {
		a.detenerAudioVideo()
		fotogramasPorSeg := archivo.FotogramasPorSegundo
		if fotogramasPorSeg < 1 {
			fotogramasPorSeg = 12
		}
		a.reproductorVideo = estadoReproductorVideo{
			Ruta:             archivo.Ruta,
			Duracion:         archivo.Duracion,
			Rotacion:         rotacion,
			FotogramasPorSeg: fotogramasPorSeg,
			AudioVerificado:  archivo.TieneAudio,
		}
		a.controlProgresoVideo = widget.Float{}
		a.controlExtraccionFrame = widget.Float{}
		a.formatoExtraccionExpandido = false
		a.solicitarVerificacionAudioVideo(archivo)
		return
	}

	if archivo.Duracion > 0 {
		a.reproductorVideo.Duracion = archivo.Duracion
		if a.reproductorVideo.Posicion > archivo.Duracion {
			a.reproductorVideo.Posicion = archivo.Duracion
		}
	}
	fotogramasPorSeg := archivo.FotogramasPorSegundo
	if fotogramasPorSeg < 1 {
		fotogramasPorSeg = a.reproductorVideo.FotogramasPorSeg
	}
	if fotogramasPorSeg < 1 {
		fotogramasPorSeg = 12
	}
	if a.reproductorVideo.FotogramasPorSeg != fotogramasPorSeg {
		a.reproductorVideo.FotogramasPorSeg = fotogramasPorSeg
		a.reproductorVideo.Fotograma = nil
		a.reproductorVideo.InstanteFotograma = 0
		a.reproductorVideo.MaximoFotograma = 0
		a.reproductorVideo.Fotogramas = nil
		a.reproductorVideo.FotogramasInicioLoop = nil
		a.reproductorVideo.LotesCache = nil
		a.reproductorVideo.InicioBuffer = 0
		a.reproductorVideo.FinBuffer = 0
		a.reproductorVideo.Cargando = false
		a.reproductorVideo.MostrarCarga = false
		a.reproductorVideo.Error = ""
		a.reproductorVideo.InstanteError = 0
		a.reproductorVideo.MaximoError = 0
		a.reproductorVideo.TienePendiente = false
		a.reproductorVideo.VersionSolicitud++
	}
	if a.reproductorVideo.Rotacion != rotacion {
		a.reproductorVideo.Rotacion = rotacion
		a.reproductorVideo.Fotograma = nil
		a.reproductorVideo.InstanteFotograma = 0
		a.reproductorVideo.MaximoFotograma = 0
		a.reproductorVideo.Fotogramas = nil
		a.reproductorVideo.FotogramasInicioLoop = nil
		a.reproductorVideo.LotesCache = nil
		a.reproductorVideo.InicioBuffer = 0
		a.reproductorVideo.FinBuffer = 0
		a.reproductorVideo.Cargando = false
		a.reproductorVideo.MostrarCarga = false
		a.reproductorVideo.Error = ""
		a.reproductorVideo.InstanteError = 0
		a.reproductorVideo.MaximoError = 0
		a.reproductorVideo.TienePendiente = false
		a.reproductorVideo.VersionSolicitud++
	}
	if archivo.TieneAudio {
		a.reproductorVideo.AudioVerificado = true
		a.reproductorVideo.AudioCargando = false
	}
	a.solicitarVerificacionAudioVideo(archivo)
	a.sincronizarControlesPosicionVideo()
}

func (a *Aplicacion) solicitarVerificacionAudioVideo(archivo modelo.Archivo) {
	if a.servicioMetadatos == nil || archivo.Tipo != modelo.TipoVideo || archivo.Ruta == "" || archivo.TieneAudio {
		return
	}
	estado := &a.reproductorVideo
	if estado.Ruta != archivo.Ruta || estado.AudioVerificado || estado.AudioCargando {
		return
	}

	estado.AudioCargando = true
	ruta := archivo.Ruta
	go func() {
		tieneAudio, err := a.servicioMetadatos.TienePistaAudioVideo(context.Background(), ruta)
		a.encolarActualizacion(func() {
			if a.reproductorVideo.Ruta != ruta {
				return
			}
			a.reproductorVideo.AudioCargando = false
			// Un fallo se reintentará al abrir nuevamente el archivo, sin lanzar
			// ffprobe en cada frame del renderizado actual.
			a.reproductorVideo.AudioVerificado = true
			if err != nil || !tieneAudio || !a.tieneArchivoActivo || a.archivoActivo.Ruta != ruta {
				return
			}
			a.archivoActivo.TieneAudio = true
			a.reemplazarArchivoEnMemoria(a.archivoActivo)
		})
	}()
}

func (a *Aplicacion) actualizarReproductorVideo(gtx layout.Context, archivo modelo.Archivo, maximoFotograma int) {
	a.sincronizarReproductorVideo(archivo)
	if a.reproductorVideo.Ruta == "" {
		return
	}

	maximoBuffer := a.maximoFotogramaBuffer(maximoFotograma)
	estado := &a.reproductorVideo
	if estado.Reproduciendo {
		if estado.MetadatosCargando {
			estado.UltimoTick = gtx.Now
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}
		if a.reproducirAudioVideo && a.audioVideoIniciado && !a.audioVideoListo {
			estado.UltimoTick = gtx.Now
			a.aplicarFotogramaDisponible(estado.Posicion)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}
		puedeIniciarAudio := !estado.Cargando || len(estado.Fotogramas) == 0 || estado.Posicion < estado.FinBuffer || a.bufferAlcanzaFinalVideo()
		if a.reproducirAudioVideo && a.audioVideoIniciado && a.audioVideoListo && a.audioVideoPausado && puedeIniciarAudio && estado.Fotograma != nil && a.bufferCubreInstante(estado.Posicion) {
			a.reanudarAudioVideo()
			estado.UltimoTick = gtx.Now
			a.aplicarFotogramaDisponible(estado.Posicion)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}
		if a.reproducirAudioVideo && a.audioVideoDisponible() && !a.audioVideoIniciado && puedeIniciarAudio && estado.Fotograma != nil && a.bufferCubreInstante(estado.Posicion) {
			a.iniciarAudioVideo(estado.Posicion)
			estado.UltimoTick = gtx.Now
			a.aplicarFotogramaDisponible(estado.Posicion)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}
		if estado.UltimoTick.IsZero() {
			estado.UltimoTick = gtx.Now
		}
		esperandoSiguienteLote := false
		if delta := gtx.Now.Sub(estado.UltimoTick); delta > 0 {
			posicionObjetivo := estado.Posicion + delta
			// Si la precarga todavía no entrega el siguiente lote, mantenemos el reloj
			// sobre el último fotograma disponible para evitar un salto brusco posterior.
			if estado.Cargando && len(estado.Fotogramas) > 0 && posicionObjetivo > estado.FinBuffer && !a.bufferAlcanzaFinalVideo() {
				posicionObjetivo = estado.FinBuffer
				esperandoSiguienteLote = true
			}
			// Cuando todavía no existe ningún fotograma cargado para el tramo actual,
			// detenemos el reloj para no "consumir" tiempo antes de poder mostrarlo.
			if estado.Cargando && !a.bufferCubreInstante(posicionObjetivo) && !a.posicionEnTramoFinalVideo(estado.Posicion) {
				posicionObjetivo = estado.Posicion
			}
			estado.Posicion = posicionObjetivo
			if estado.Duracion > 0 && estado.Posicion >= estado.Duracion {
				posicionResuelta, sigueReproduciendo := resolverPosicionFinReproductorVideo(estado.Posicion, estado.Duracion, a.reproducirVideoEnLoop)
				estado.Posicion = posicionResuelta
				estado.Reproduciendo = sigueReproduciendo
				if sigueReproduciendo {
					estado.Error = ""
					estado.InstanteError = 0
					estado.MaximoError = 0
					estado.UltimoTick = gtx.Now
					// ffmpeg repite el audio con su propia duración, que puede diferir
					// algunos milisegundos de la pista de video. Reiniciarlo desde cero
					// en cada ciclo evita que el desfase se acumule.
					a.detenerAudioVideo()
					a.prepararAudioVideo(estado.Posicion)
					if !a.activarBufferInicioLoop() {
						estado.Fotograma = nil
						estado.InstanteFotograma = 0
						a.solicitarLoteFotogramasVideo(archivo, estado.Posicion, maximoBuffer)
					}
				} else {
					a.detenerAudioVideo()
				}
			}
			a.sincronizarControlesPosicionVideo()
		}

		// Mientras el primer lote todavía no llegó, mantenemos el reloj congelado
		// en la posición ya visible. Si no hacemos esto, el panel puede quedarse
		// mostrando el frame inicial mientras la posición avanza "en silencio" y
		// luego saltar bruscamente cuando ffmpeg termina de precargar.
		if estado.Cargando && !a.bufferCubreInstante(estado.Posicion) && !a.posicionEnTramoFinalVideo(estado.Posicion) {
			if a.audioVideoIniciado {
				a.pausarAudioVideo()
			}
			estado.MostrarCarga = true
			estado.UltimoTick = gtx.Now
			a.aplicarFotogramaDisponible(estado.Posicion)
			if !a.bufferCubreInstante(estado.Posicion) {
				a.solicitarLoteFotogramasVideo(archivo, estado.Posicion, maximoBuffer)
			}
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}
		if esperandoSiguienteLote {
			if a.audioVideoIniciado {
				a.pausarAudioVideo()
			}
			estado.MostrarCarga = true
			estado.UltimoTick = gtx.Now
			a.aplicarFotogramaDisponible(estado.Posicion)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
			return
		}

		estado.UltimoTick = gtx.Now
		a.aplicarFotogramaDisponible(estado.Posicion)
		bufferDisponible := a.bufferCubreInstante(estado.Posicion) && estado.Fotograma != nil
		if bufferDisponible {
			a.iniciarAudioVideo(estado.Posicion)
		}
		if !bufferDisponible {
			if a.audioVideoIniciado {
				a.pausarAudioVideo()
			}
			estado.MostrarCarga = true
			a.solicitarLoteFotogramasVideo(archivo, estado.Posicion, maximoBuffer)
		} else if a.debePrecargarSiguienteLote() {
			a.solicitarLoteFotogramasVideo(archivo, a.inicioSiguienteLote(), maximoBuffer)
		}
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 24)})
		return
	}

	estado.UltimoTick = gtx.Now
	if a.aplicarFotogramaDisponible(estado.Posicion) {
		return
	}
	if estado.Fotograma == nil || diferenciaDuracion(estado.InstanteFotograma, estado.Posicion) >= 180*time.Millisecond {
		a.solicitarFotogramaVideo(archivo, estado.Posicion, maximoFotograma)
	}
}

func (a *Aplicacion) alternarReproductorVideo() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	if a.reproductorVideo.Reproduciendo {
		a.detenerReproduccionVideo()
		return
	}
	if a.reproductorVideo.Duracion <= 0 {
		if a.reproductorVideo.MetadatosCargando {
			a.reproductorVideo.ReproduccionPendiente = true
			a.reproductorVideo.Reproduciendo = true
			a.reproductorVideo.Cargando = true
			a.reproductorVideo.MostrarCarga = true
			return
		}
		a.establecerEstado("Esperando la duración real del video para reproducirlo", nil)
		return
	}

	if a.estaReproductorVideoAlFinal() {
		a.prepararInicioReproductorVideo(false)
	}

	a.comenzarReproduccionVideo()
}

func (a *Aplicacion) comenzarReproduccionVideo() {
	a.reproductorVideo.ReproduccionPendiente = false
	a.invalidarSolicitudesFotogramasVideo()
	a.reproductorVideo.Reproduciendo = true
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.MostrarCarga = true
	if !a.bufferCubreInstante(a.reproductorVideo.Posicion) {
		a.prepararAudioVideo(a.reproductorVideo.Posicion)
		a.solicitarLoteFotogramasVideo(a.archivoActivo, a.reproductorVideo.Posicion, a.maximoFotogramaReproductor())
	} else if a.reproductorVideo.Fotograma != nil {
		if a.audioVideoPausado {
			a.reanudarAudioVideo()
		} else {
			a.iniciarAudioVideo(a.reproductorVideo.Posicion)
		}
	}
}

func (a *Aplicacion) reiniciarReproductorVideo() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	a.detenerReproduccionVideo()
	a.prepararInicioReproductorVideo(true)
}

func (a *Aplicacion) alternarLoopReproductorVideo() {
	a.reproducirVideoEnLoop = !a.reproducirVideoEnLoop
	if !a.reproducirVideoEnLoop {
		a.reproductorVideo.FotogramasInicioLoop = nil
		return
	}
	if len(a.reproductorVideo.FotogramasInicioLoop) == 0 && a.reproductorVideo.InicioBuffer == 0 {
		a.reproductorVideo.FotogramasInicioLoop = copiarFotogramasVideo(a.reproductorVideo.Fotogramas)
	}
}

func (a *Aplicacion) alternarAudioReproductorVideo() {
	if !a.audioVideoDisponible() {
		return
	}
	a.reproducirAudioVideo = !a.reproducirAudioVideo
	if !a.reproducirAudioVideo {
		a.detenerAudioVideo()
		return
	}
	if a.tieneArchivoActivo && a.archivoActivo.Tipo == modelo.TipoVideo && a.reproductorVideo.Reproduciendo && a.bufferCubreInstante(a.reproductorVideo.Posicion) {
		a.iniciarAudioVideo(a.reproductorVideo.Posicion)
	}
}

func (a *Aplicacion) actualizarPosicionVideoDesdeControl(maximoFotograma int) {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	valorObjetivo := a.controlProgresoVideo.Value
	a.sincronizarReproductorVideo(a.archivoActivo)
	reproduciendoAntes := a.reproductorVideo.Reproduciendo
	if reproduciendoAntes {
		a.detenerReproduccionVideo()
		a.detenerAudioVideo()
	} else {
		a.detenerAudioVideo()
	}
	a.invalidarSolicitudesFotogramasVideo()
	a.descartarBufferFotogramas()
	a.reproductorVideo.Posicion = a.posicionDesdeProgresoVideo(valorObjetivo, a.reproductorVideo.Duracion)
	a.reproductorVideo.Fotograma = nil
	a.reproductorVideo.InstanteFotograma = 0
	a.reproductorVideo.MaximoFotograma = 0
	if !reproduciendoAntes && !a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion) {
		a.reproductorVideo.MostrarCarga = true
	}
	a.sincronizarControlesPosicionVideo()
	if reproduciendoAntes {
		a.reproductorVideo.Reproduciendo = true
		a.reproductorVideo.UltimoTick = time.Time{}
		a.reproductorVideo.MostrarCarga = true
		if !a.bufferCubreInstante(a.reproductorVideo.Posicion) {
			a.prepararAudioVideo(a.reproductorVideo.Posicion)
			a.solicitarLoteFotogramasVideo(a.archivoActivo, a.reproductorVideo.Posicion, maximoFotograma)
		} else if a.reproductorVideo.Fotograma != nil {
			a.iniciarAudioVideo(a.reproductorVideo.Posicion)
		}
		return
	}
	if !a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion) {
		a.solicitarFotogramaVideo(a.archivoActivo, a.reproductorVideo.Posicion, maximoFotograma)
	}
}

func (a *Aplicacion) actualizarPosicionVideoDesdeExtraccion(maximoFotograma int) {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	valorObjetivo := a.controlExtraccionFrame.Value
	a.sincronizarReproductorVideo(a.archivoActivo)
	a.detenerReproduccionVideo()
	a.detenerAudioVideo()
	a.invalidarSolicitudesFotogramasVideo()
	a.reproductorVideo.Posicion = a.posicionDesdeProgresoVideo(valorObjetivo, a.reproductorVideo.Duracion)
	a.descartarBufferFotogramas()
	a.sincronizarControlesPosicionVideo()
	// Para que el frame exportado coincida con el visor, pedimos un fotograma
	// puntual incluso si el buffer ya tiene uno aproximado cercano.
	a.solicitarFotogramaVideo(a.archivoActivo, a.reproductorVideo.Posicion, maximoFotograma)
}

func (a *Aplicacion) solicitarFotogramaVideo(archivo modelo.Archivo, instante time.Duration, maximoFotograma int) {
	if a.servicioMetadatos == nil || archivo.Ruta == "" || archivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(archivo)
	if a.reproductorVideo.Ruta != archivo.Ruta {
		return
	}

	estado := &a.reproductorVideo
	instante = a.normalizarInstanteFotogramaVideo(instante, duracionMayor(archivo.Duracion, estado.Duracion))
	if estado.Cargando {
		estado.InstantePendiente = instante
		estado.MaximoPendiente = maximoFotograma
		estado.TienePendiente = true
		return
	}
	if estado.Error != "" && estado.InstanteError == instante && estado.MaximoError >= maximoFotograma {
		return
	}
	if estado.Fotograma != nil && estado.InstanteFotograma == instante && estado.MaximoFotograma >= maximoFotograma {
		return
	}

	estado.Cargando = true
	estado.Error = ""
	estado.VersionSolicitud++
	versionSolicitud := estado.VersionSolicitud
	ruta := archivo.Ruta
	rotacion := estado.Rotacion

	go func() {
		imagen, err := a.servicioMetadatos.GenerarFotogramaVideo(context.Background(), ruta, instante, maximoFotograma, rotacion)
		a.encolarActualizacion(func() {
			if a.reproductorVideo.Ruta != ruta || a.reproductorVideo.VersionSolicitud != versionSolicitud {
				return
			}

			a.reproductorVideo.Cargando = false
			a.reproductorVideo.MostrarCarga = false
			if err != nil {
				a.reproductorVideo.Error = err.Error()
				a.reproductorVideo.InstanteError = instante
				a.reproductorVideo.MaximoError = maximoFotograma
				a.reproductorVideo.Reproduciendo = false
			} else {
				a.reproductorVideo.Fotograma = imagen
				a.reproductorVideo.InstanteFotograma = instante
				a.reproductorVideo.MaximoFotograma = maximoFotograma
				a.reproductorVideo.Error = ""
				a.reproductorVideo.InstanteError = 0
				a.reproductorVideo.MaximoError = 0
			}

			if a.reproductorVideo.TienePendiente {
				pendiente := a.reproductorVideo.InstantePendiente
				maximoPendiente := a.reproductorVideo.MaximoPendiente
				a.reproductorVideo.TienePendiente = false
				a.solicitarFotogramaVideo(archivo, pendiente, maximoPendiente)
			}
		})
	}()
}

func (a *Aplicacion) solicitarLoteFotogramasVideo(archivo modelo.Archivo, inicio time.Duration, maximoFotograma int) {
	if a.servicioMetadatos == nil || archivo.Ruta == "" || archivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(archivo)
	if a.reproductorVideo.Ruta != archivo.Ruta {
		return
	}
	maximoFotograma = a.maximoFotogramaBuffer(maximoFotograma)

	estado := &a.reproductorVideo
	if estado.Cargando {
		a.encolarSolicitudLoteFotogramasVideo(inicio, maximoFotograma)
		return
	}

	fotogramasPorSeg := estado.FotogramasPorSeg
	if fotogramasPorSeg < 1 {
		fotogramasPorSeg = 12
		estado.FotogramasPorSeg = fotogramasPorSeg
	}
	inicio = a.normalizarInicioLoteVideo(inicio, estado.Duracion)
	if a.bufferCubreInstante(inicio) && !a.debePrecargarSiguienteLote() && !a.debeAnticiparSiguienteLote() {
		return
	}
	if estado.Error != "" && diferenciaDuracion(estado.InstanteError, inicio) < a.intervaloFotogramasBuffer()*2 && estado.MaximoError >= maximoFotograma {
		return
	}

	cantidad := a.cantidadFotogramasLoteBuffer()
	esPrecargaInicioLoop := a.esPrecargaInicioLoop(inicio)
	if lote, existe := a.buscarLoteCacheVideo(inicio, maximoFotograma); existe {
		a.aplicarLoteCacheVideo(lote, inicio, maximoFotograma, esPrecargaInicioLoop)
		return
	}
	estado.Cargando = true
	estado.Error = ""
	estado.VersionSolicitud++
	versionSolicitud := estado.VersionSolicitud
	ruta := archivo.Ruta
	rotacion := estado.Rotacion

	go func() {
		lote, err := a.servicioMetadatos.GenerarLoteFotogramasVideo(context.Background(), ruta, inicio, fotogramasPorSeg, cantidad, maximoFotograma, rotacion)
		a.encolarActualizacion(func() {
			if a.reproductorVideo.Ruta != ruta || a.reproductorVideo.VersionSolicitud != versionSolicitud {
				return
			}

			a.reproductorVideo.Cargando = false
			a.reproductorVideo.MostrarCarga = false
			if err != nil {
				a.reproductorVideo.Error = err.Error()
				a.reproductorVideo.InstanteError = inicio
				a.reproductorVideo.MaximoError = maximoFotograma
				if len(a.reproductorVideo.Fotogramas) == 0 {
					a.reproductorVideo.Reproduciendo = false
					a.detenerAudioVideo()
				}
				return
			}

			convertidos := convertirFotogramasVideo(lote)
			if a.debeConservarLoteCacheVideo(archivo) {
				a.guardarLoteCacheVideo(inicio, maximoFotograma, convertidos)
			}
			if esPrecargaInicioLoop {
				a.reproductorVideo.FotogramasInicioLoop = convertidos
				a.activarBufferInicioLoopPendiente()
			} else {
				a.integrarLoteFotogramas(lote)
				if a.reproducirVideoEnLoop && inicio == 0 && len(a.reproductorVideo.FotogramasInicioLoop) == 0 {
					a.reproductorVideo.FotogramasInicioLoop = copiarFotogramasVideo(convertidos)
				}
				a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion)
			}
			a.reproductorVideo.Error = ""
			a.reproductorVideo.InstanteError = 0
			a.reproductorVideo.MaximoError = 0
			a.iniciarSolicitudLoteFotogramasVideoPendiente(archivo)
			a.anticiparSiguienteLoteFotogramasVideo(archivo, maximoFotograma)
		})
	}()
}

func (a *Aplicacion) encolarSolicitudLoteFotogramasVideo(inicio time.Duration, maximoFotograma int) {
	estado := &a.reproductorVideo
	inicio = a.normalizarInicioLoteVideo(inicio, estado.Duracion)
	if estado.TieneLotePendiente && estado.InicioLotePendiente == inicio && estado.MaximoLotePendiente >= maximoFotograma {
		return
	}
	estado.InicioLotePendiente = inicio
	estado.MaximoLotePendiente = maximoFotograma
	estado.TieneLotePendiente = true
}

func (a *Aplicacion) iniciarSolicitudLoteFotogramasVideoPendiente(archivo modelo.Archivo) {
	estado := &a.reproductorVideo
	if !estado.TieneLotePendiente {
		return
	}
	inicio := estado.InicioLotePendiente
	maximoFotograma := estado.MaximoLotePendiente
	estado.TieneLotePendiente = false
	estado.InicioLotePendiente = 0
	estado.MaximoLotePendiente = 0
	a.solicitarLoteFotogramasVideo(archivo, inicio, maximoFotograma)
}

func (a *Aplicacion) integrarLoteFotogramas(nuevos []metadatos.FotogramaVideo) {
	convertidos := convertirFotogramasVideo(nuevos)
	a.integrarFotogramasBuffer(convertidos)
}

func (a *Aplicacion) integrarFotogramasBuffer(convertidos []fotogramaBufferVideo) {
	if len(convertidos) == 0 {
		return
	}

	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		estado.Fotogramas = convertidos
	} else {
		estado.Fotogramas = fusionarFotogramasVideo(estado.Fotogramas, convertidos)
	}

	a.recortarBufferFotogramas()
	if len(estado.Fotogramas) > 0 {
		estado.InicioBuffer = estado.Fotogramas[0].Instante
		estado.FinBuffer = estado.Fotogramas[len(estado.Fotogramas)-1].Instante
	}
}

func (a *Aplicacion) buscarLoteCacheVideo(inicio time.Duration, maximoFotograma int) ([]fotogramaBufferVideo, bool) {
	for indice := len(a.reproductorVideo.LotesCache) - 1; indice >= 0; indice-- {
		lote := a.reproductorVideo.LotesCache[indice]
		if lote.Inicio == inicio && lote.Maximo >= maximoFotograma && len(lote.Fotogramas) > 0 {
			return copiarFotogramasVideo(lote.Fotogramas), true
		}
	}
	return nil, false
}

func (a *Aplicacion) guardarLoteCacheVideo(inicio time.Duration, maximoFotograma int, fotogramas []fotogramaBufferVideo) {
	if len(fotogramas) == 0 {
		return
	}
	cache := a.reproductorVideo.LotesCache
	for indice := range cache {
		if cache[indice].Inicio != inicio {
			continue
		}
		if cache[indice].Maximo >= maximoFotograma {
			return
		}
		cache[indice] = loteCacheVideo{
			Inicio:     inicio,
			Maximo:     maximoFotograma,
			Fotogramas: copiarFotogramasVideo(fotogramas),
		}
		a.reproductorVideo.LotesCache = cache
		return
	}

	cache = append(cache, loteCacheVideo{
		Inicio:     inicio,
		Maximo:     maximoFotograma,
		Fotogramas: copiarFotogramasVideo(fotogramas),
	})
	const maximoLotesCache = 2
	if len(cache) > maximoLotesCache {
		cache = cache[len(cache)-maximoLotesCache:]
	}
	a.reproductorVideo.LotesCache = cache
}

func (a *Aplicacion) aplicarLoteCacheVideo(fotogramas []fotogramaBufferVideo, inicio time.Duration, maximoFotograma int, esPrecargaInicioLoop bool) {
	a.reproductorVideo.Cargando = false
	a.reproductorVideo.MostrarCarga = false
	if esPrecargaInicioLoop {
		a.reproductorVideo.FotogramasInicioLoop = copiarFotogramasVideo(fotogramas)
		a.activarBufferInicioLoopPendiente()
	} else {
		a.integrarFotogramasBuffer(fotogramas)
		if a.reproducirVideoEnLoop && inicio == 0 && len(a.reproductorVideo.FotogramasInicioLoop) == 0 {
			a.reproductorVideo.FotogramasInicioLoop = copiarFotogramasVideo(fotogramas)
		}
		a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion)
	}
	a.reproductorVideo.Error = ""
	a.reproductorVideo.InstanteError = 0
	a.reproductorVideo.MaximoError = 0
	a.iniciarSolicitudLoteFotogramasVideoPendiente(a.archivoActivo)
	a.anticiparSiguienteLoteFotogramasVideo(a.archivoActivo, maximoFotograma)
}

func (a *Aplicacion) anticiparSiguienteLoteFotogramasVideo(archivo modelo.Archivo, maximoFotograma int) {
	estado := &a.reproductorVideo
	if estado.Cargando || estado.TieneLotePendiente || !estado.Reproduciendo || !a.debeAnticiparSiguienteLote() {
		return
	}
	a.solicitarLoteFotogramasVideo(archivo, a.inicioSiguienteLote(), maximoFotograma)
}

func convertirFotogramasVideo(nuevos []metadatos.FotogramaVideo) []fotogramaBufferVideo {
	convertidos := make([]fotogramaBufferVideo, 0, len(nuevos))
	for _, fotograma := range nuevos {
		if fotograma.Imagen == nil {
			continue
		}
		convertidos = append(convertidos, fotogramaBufferVideo{
			Instante: fotograma.Instante,
			Imagen:   fotograma.Imagen,
		})
	}
	return convertidos
}

func copiarFotogramasVideo(fotogramas []fotogramaBufferVideo) []fotogramaBufferVideo {
	if len(fotogramas) == 0 {
		return nil
	}
	copia := make([]fotogramaBufferVideo, len(fotogramas))
	copy(copia, fotogramas)
	return copia
}

func fusionarFotogramasVideo(actuales, nuevos []fotogramaBufferVideo) []fotogramaBufferVideo {
	fusionados := make([]fotogramaBufferVideo, 0, len(actuales)+len(nuevos))
	i, j := 0, 0
	for i < len(actuales) && j < len(nuevos) {
		instanteActual := actuales[i].Instante
		instanteNuevo := nuevos[j].Instante
		switch {
		case instanteActual < instanteNuevo:
			fusionados = append(fusionados, actuales[i])
			i++
		case instanteNuevo < instanteActual:
			fusionados = append(fusionados, nuevos[j])
			j++
		default:
			fusionados = append(fusionados, nuevos[j])
			i++
			j++
		}
	}
	fusionados = append(fusionados, actuales[i:]...)
	fusionados = append(fusionados, nuevos[j:]...)
	return fusionados
}

func (a *Aplicacion) recortarBufferFotogramas() {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return
	}

	limiteInferior := estado.Posicion - 2*time.Second
	if limiteInferior < 0 {
		limiteInferior = 0
	}
	indiceInicio := 0
	for indiceInicio < len(estado.Fotogramas) && estado.Fotogramas[indiceInicio].Instante < limiteInferior {
		indiceInicio++
	}
	if indiceInicio > 0 && indiceInicio < len(estado.Fotogramas) {
		estado.Fotogramas = append([]fotogramaBufferVideo(nil), estado.Fotogramas[indiceInicio:]...)
	}
	limiteFotogramas := maximoEntero(96, a.cantidadFotogramasLoteBuffer()*2)
	if len(estado.Fotogramas) > limiteFotogramas {
		estado.Fotogramas = append([]fotogramaBufferVideo(nil), estado.Fotogramas[len(estado.Fotogramas)-limiteFotogramas:]...)
	}
}

func (a *Aplicacion) aplicarFotogramaDisponible(instante time.Duration) bool {
	if a.aplicarFotogramaDesdeBuffer(instante) {
		return true
	}
	estado := &a.reproductorVideo
	if estado.Fotograma == nil {
		return false
	}
	margen := maximoDuracion(220*time.Millisecond, a.intervaloFotogramasBuffer()*3)
	return diferenciaDuracion(estado.InstanteFotograma, instante) < margen
}

func (a *Aplicacion) aplicarFotogramaDesdeBuffer(instante time.Duration) bool {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return false
	}
	intervalo := a.intervaloFotogramasBuffer()
	if instante < estado.InicioBuffer-intervalo || instante > estado.FinBuffer+intervalo {
		return false
	}

	indice := 0
	mejorDiferencia := time.Duration(1<<63 - 1)
	for actual, fotograma := range estado.Fotogramas {
		diferencia := diferenciaDuracion(fotograma.Instante, instante)
		if diferencia < mejorDiferencia {
			mejorDiferencia = diferencia
			indice = actual
		}
		if fotograma.Instante > instante && diferencia > mejorDiferencia {
			break
		}
	}
	if mejorDiferencia > intervalo*2 {
		return false
	}

	estado.Fotograma = estado.Fotogramas[indice].Imagen
	estado.InstanteFotograma = estado.Fotogramas[indice].Instante
	return true
}

func (a *Aplicacion) bufferCubreInstante(instante time.Duration) bool {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return false
	}
	intervalo := a.intervaloFotogramasBuffer()
	limite := intervalo * 2
	for _, fotograma := range estado.Fotogramas {
		if diferenciaDuracion(fotograma.Instante, instante) <= limite {
			return true
		}
		if fotograma.Instante > instante && diferenciaDuracion(fotograma.Instante, instante) > limite {
			break
		}
	}
	return false
}

func (a *Aplicacion) debePrecargarSiguienteLote() bool {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return true
	}
	if estado.Duracion > 0 && estado.FinBuffer >= estado.Duracion-a.margenFinalLoteVideo() {
		return a.reproducirVideoEnLoop && estado.InicioBuffer > 0 && len(estado.FotogramasInicioLoop) == 0
	}
	return estado.FinBuffer-estado.Posicion <= a.margenPrecargaBuffer()
}

// debeAnticiparSiguienteLote evita la primera pausa del reproductor: al
// recibir el lote visible pedimos inmediatamente un único lote consecutivo.
func (a *Aplicacion) debeAnticiparSiguienteLote() bool {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return false
	}
	if estado.Duracion > 0 && estado.FinBuffer >= estado.Duracion-a.margenFinalLoteVideo() {
		return a.reproducirVideoEnLoop && estado.InicioBuffer > 0 && len(estado.FotogramasInicioLoop) == 0
	}
	return estado.FinBuffer-estado.Posicion <= a.duracionLoteBuffer()+a.intervaloFotogramasBuffer()
}

func (a *Aplicacion) inicioSiguienteLote() time.Duration {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return a.normalizarInicioLoteVideo(estado.Posicion, estado.Duracion)
	}
	if a.reproducirVideoEnLoop && estado.Duracion > 0 && estado.FinBuffer >= estado.Duracion-a.margenFinalLoteVideo() {
		return 0
	}
	inicio := estado.FinBuffer - a.solapeLoteBuffer()
	if inicio < estado.Posicion {
		inicio = estado.Posicion
	}
	if inicio < 0 {
		return 0
	}
	return a.normalizarInicioLoteVideo(inicio, estado.Duracion)
}

func (a *Aplicacion) intervaloFotogramasBuffer() time.Duration {
	fps := a.reproductorVideo.FotogramasPorSeg
	if fps < 1 {
		fps = 12
	}
	return time.Duration(float64(time.Second) / fps)
}

// cantidadFotogramasLoteBuffer define un bloque algo más largo para esconder mejor la latencia de ffmpeg.
func (a *Aplicacion) cantidadFotogramasLoteBuffer() int {
	fps := a.reproductorVideo.FotogramasPorSeg
	if fps < 1 {
		fps = 12
	}
	duracion := 4.0
	if fps > 30 {
		// Un bloque más largo absorbe la latencia de extracción de videos de
		// 60 FPS sin tener que detener el reloj cada pocos segundos.
		duracion = 6
	}
	return maximoEntero(48, int(math.Ceil(fps*duracion)))
}

func (a *Aplicacion) duracionLoteBuffer() time.Duration {
	return time.Duration(a.cantidadFotogramasLoteBuffer()) * a.intervaloFotogramasBuffer()
}

func (a *Aplicacion) margenPrecargaBuffer() time.Duration {
	margen := a.duracionLoteBuffer() * 3 / 4
	if margen < 2*time.Second {
		margen = 2 * time.Second
	}
	if margen > 4500*time.Millisecond {
		margen = 4500 * time.Millisecond
	}
	return margen
}

func (a *Aplicacion) solapeLoteBuffer() time.Duration {
	solape := a.margenPrecargaBuffer() / 2
	solapeMinimo := a.intervaloFotogramasBuffer() * 6
	if solape < solapeMinimo {
		solape = solapeMinimo
	}
	if solape > 1200*time.Millisecond {
		solape = 1200 * time.Millisecond
	}
	return solape
}

func (a *Aplicacion) sincronizarControlesPosicionVideo() {
	valor := a.valorProgresoVideo(a.reproductorVideo.Posicion, a.reproductorVideo.Duracion)
	if !a.controlProgresoVideo.Dragging() {
		a.controlProgresoVideo.Value = valor
	}
	if !a.controlExtraccionFrame.Dragging() {
		a.controlExtraccionFrame.Value = valor
	}
}

func (a *Aplicacion) normalizarInicioLoteVideo(inicio, duracion time.Duration) time.Duration {
	paso := a.intervaloFotogramasBuffer()
	inicio = normalizarInstanteVideoConPaso(inicio, duracion, paso)
	if duracion <= 0 {
		return inicio
	}

	ultimoInicio := duracion - a.margenFinalLoteVideo()
	if ultimoInicio < 0 {
		ultimoInicio = 0
	}
	if inicio > ultimoInicio {
		inicio = ultimoInicio
	}
	return normalizarInstanteVideoConPaso(inicio, duracion, paso)
}

func (a *Aplicacion) normalizarInstanteFotogramaVideo(instante, duracion time.Duration) time.Duration {
	instante = normalizarInstanteVideo(instante, duracion)
	if duracion <= 0 {
		return instante
	}

	margen := maximoDuracion(160*time.Millisecond, a.intervaloFotogramasBuffer()*2)
	if margen > duracion {
		margen = duracion
	}
	ultimoInstante := duracion - margen
	if ultimoInstante < 0 {
		ultimoInstante = 0
	}
	if instante > ultimoInstante {
		instante = ultimoInstante
	}
	return normalizarInstanteVideoConPaso(instante, duracion, a.intervaloFotogramasBuffer())
}

func (a *Aplicacion) margenFinalLoteVideo() time.Duration {
	margen := maximoDuracion(400*time.Millisecond, a.intervaloFotogramasBuffer()*4)
	if a.reproductorVideo.Duracion > 0 && margen > a.reproductorVideo.Duracion {
		return a.reproductorVideo.Duracion
	}
	return margen
}

func (a *Aplicacion) estaReproductorVideoAlFinal() bool {
	if a.reproductorVideo.Duracion <= 0 {
		return false
	}
	return a.reproductorVideo.Posicion >= a.reproductorVideo.Duracion
}

func (a *Aplicacion) esPrecargaInicioLoop(inicio time.Duration) bool {
	estado := &a.reproductorVideo
	return a.reproducirVideoEnLoop && inicio == 0 && len(estado.Fotogramas) > 0 && estado.InicioBuffer > 0
}

func (a *Aplicacion) activarBufferInicioLoop() bool {
	estado := &a.reproductorVideo
	if len(estado.FotogramasInicioLoop) == 0 {
		return false
	}
	estado.Fotogramas = estado.FotogramasInicioLoop
	estado.FotogramasInicioLoop = copiarFotogramasVideo(estado.FotogramasInicioLoop)
	estado.InicioBuffer = estado.Fotogramas[0].Instante
	estado.FinBuffer = estado.Fotogramas[len(estado.Fotogramas)-1].Instante
	estado.Fotograma = nil
	estado.InstanteFotograma = 0
	estado.MaximoFotograma = 0
	a.aplicarFotogramaDisponible(estado.Posicion)
	return true
}

// activarBufferInicioLoopPendiente cubre el caso en que el video ya llegó al
// final antes de que termine la precarga del inicio del siguiente ciclo.
func (a *Aplicacion) activarBufferInicioLoopPendiente() {
	estado := &a.reproductorVideo
	if !a.reproducirVideoEnLoop || !estado.Reproduciendo || a.bufferCubreInstante(estado.Posicion) {
		return
	}
	_ = a.activarBufferInicioLoop()
}

func (a *Aplicacion) posicionEnTramoFinalVideo(posicion time.Duration) bool {
	return a.reproductorVideo.Duracion > 0 && posicion >= a.reproductorVideo.Duracion-a.margenFinalLoteVideo()
}

func (a *Aplicacion) bufferAlcanzaFinalVideo() bool {
	estado := &a.reproductorVideo
	return estado.Duracion > 0 && estado.FinBuffer >= estado.Duracion-a.margenFinalLoteVideo()
}

func (a *Aplicacion) prepararInicioReproductorVideo(cargarFotograma bool) {
	a.detenerAudioVideo()
	a.reproductorVideo.Posicion = 0
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.Error = ""
	a.reproductorVideo.InstanteError = 0
	a.reproductorVideo.MaximoError = 0
	a.reproductorVideo.Fotograma = nil
	a.reproductorVideo.InstanteFotograma = 0
	a.invalidarSolicitudesFotogramasVideo()
	a.descartarBufferFotogramas()
	a.sincronizarControlesPosicionVideo()
	if cargarFotograma && a.tieneArchivoActivo && a.archivoActivo.Tipo == modelo.TipoVideo {
		a.solicitarFotogramaVideo(a.archivoActivo, 0, a.maximoFotogramaReproductor())
	}
}

func resolverPosicionFinReproductorVideo(posicionObjetivo, duracion time.Duration, enLoop bool) (time.Duration, bool) {
	if duracion <= 0 || posicionObjetivo < duracion {
		return posicionObjetivo, true
	}
	if enLoop {
		// Reiniciar exactamente desde el inicio evita saltarnos los primeros
		// fotogramas visibles del siguiente ciclo.
		return 0, true
	}
	return duracion, false
}

func (a *Aplicacion) descartarBufferFotogramas() {
	a.reproductorVideo.Fotogramas = nil
	a.reproductorVideo.InicioBuffer = 0
	a.reproductorVideo.FinBuffer = 0
}

func (a *Aplicacion) invalidarSolicitudesFotogramasVideo() {
	a.reproductorVideo.Cargando = false
	a.reproductorVideo.MostrarCarga = false
	a.reproductorVideo.TienePendiente = false
	a.reproductorVideo.InstantePendiente = 0
	a.reproductorVideo.MaximoPendiente = 0
	a.reproductorVideo.TieneLotePendiente = false
	a.reproductorVideo.InicioLotePendiente = 0
	a.reproductorVideo.MaximoLotePendiente = 0
	a.reproductorVideo.VersionSolicitud++
}

func (a *Aplicacion) maximoFotogramaReproductor() int {
	return a.maximoFotogramaBuffer(maximo(960, a.reproductorVideo.MaximoFotograma))
}

// maximoFotogramaBuffer limita la memoria de buffers de alta tasa sin reducir
// los FPS. Con 60 FPS, conservar dos lotes a 960 px puede requerir cientos de
// MB y provocar pausas por recolección de memoria.
func (a *Aplicacion) maximoFotogramaBuffer(maximoFotograma int) int {
	maximoFotograma = minimo(maximoFotograma, 960)
	fotogramasPorSeg := a.reproductorVideo.FotogramasPorSeg
	if fotogramasPorSeg <= 30 {
		return maximoFotograma
	}
	factorDuracion := float64(a.cantidadFotogramasLoteBuffer()) / (fotogramasPorSeg * 4)
	ajuste := math.Sqrt(30 / (fotogramasPorSeg * factorDuracion))
	ajustado := int(float64(maximoFotograma) * ajuste)
	if ajustado < 480 && maximoFotograma >= 480 {
		ajustado = 480
	}
	if ajustado > maximoFotograma {
		return maximoFotograma
	}
	return ajustado
}

func (a *Aplicacion) debeConservarLoteCacheVideo(archivo modelo.Archivo) bool {
	if a.reproductorVideo.FotogramasPorSeg > 30 {
		// FotogramasInicioLoop conserva el arranque del siguiente ciclo; para
		// videos de alta tasa no duplicamos además los lotes completos.
		return false
	}
	return archivo.Origen == modelo.OrigenYandex || a.reproducirVideoEnLoop
}

func (a *Aplicacion) audioVideoDisponible() bool {
	return a.tieneArchivoActivo && a.archivoActivo.Tipo == modelo.TipoVideo && a.archivoActivo.TieneAudio
}

func (a *Aplicacion) detenerAudioVideo() {
	a.audioVideoVersion++
	a.audioVideoIniciado = false
	a.audioVideoListo = false
	a.audioVideoPausado = false
	if a.servicioMetadatos != nil {
		a.servicioMetadatos.DetenerAudioVideo()
	}
	if a.audioVideoCancel != nil {
		a.audioVideoCancel()
		a.audioVideoCancel = nil
	}
	a.reproductorVideo.AudioError = ""
}

func (a *Aplicacion) pausarAudioVideo() {
	if !a.audioVideoIniciado {
		return
	}
	if !a.audioVideoListo {
		a.detenerAudioVideo()
		return
	}
	if a.audioVideoPausado {
		return
	}
	if a.servicioMetadatos != nil {
		a.servicioMetadatos.PausarAudioVideo()
	}
	a.audioVideoPausado = true
}

func (a *Aplicacion) reanudarAudioVideo() {
	if !a.audioVideoIniciado || !a.audioVideoListo || !a.audioVideoPausado {
		return
	}
	if a.servicioMetadatos != nil {
		a.servicioMetadatos.ReanudarAudioVideo()
	}
	a.audioVideoPausado = false
}

func (a *Aplicacion) iniciarAudioVideo(instante time.Duration) {
	a.iniciarAudioVideoConPausa(instante, false)
}

func (a *Aplicacion) prepararAudioVideo(instante time.Duration) {
	a.iniciarAudioVideoConPausa(instante, true)
}

func (a *Aplicacion) iniciarAudioVideoConPausa(instante time.Duration, iniciarPausado bool) {
	if a.audioVideoIniciado || !a.reproducirAudioVideo || a.servicioMetadatos == nil || !a.audioVideoDisponible() {
		return
	}
	if a.reproductorVideo.Ruta == "" {
		return
	}

	a.detenerAudioVideo()
	ctx, cancelar := context.WithCancel(context.Background())
	a.audioVideoCancel = cancelar
	a.audioVideoIniciado = true
	a.audioVideoListo = false
	a.audioVideoPausado = iniciarPausado
	version := a.audioVideoVersion
	ruta := a.reproductorVideo.Ruta
	repetir := a.reproducirVideoEnLoop

	go func() {
		var err error
		if iniciarPausado {
			err = a.servicioMetadatos.PrepararAudioVideoConAviso(ctx, ruta, instante, repetir, func() {
				a.encolarActualizacion(func() {
					if a.reproductorVideo.Ruta == ruta && a.audioVideoVersion == version {
						a.audioVideoListo = true
					}
				})
			})
		} else {
			err = a.servicioMetadatos.ReproducirAudioVideoConAviso(ctx, ruta, instante, repetir, func() {
				a.encolarActualizacion(func() {
					if a.reproductorVideo.Ruta == ruta && a.audioVideoVersion == version {
						a.audioVideoListo = true
					}
				})
			})
		}
		if errors.Is(err, context.Canceled) {
			return
		}

		a.encolarActualizacion(func() {
			if a.reproductorVideo.Ruta != ruta || a.audioVideoVersion != version {
				return
			}
			if err != nil {
				a.audioVideoIniciado = false
				a.audioVideoListo = true
				a.reproductorVideo.AudioError = err.Error()
			} else {
				a.reproductorVideo.AudioError = ""
			}
		})
	}()
}

func controlVideoFueManipuladoPorUsuario(valorAnterior, valorActual float32, arrastrando bool) bool {
	return valorAnterior != valorActual && arrastrando
}

func (a *Aplicacion) valorProgresoVideo(posicion, duracion time.Duration) float32 {
	if duracion <= 0 {
		return 0
	}
	valor := float32(float64(posicion) / float64(duracion))
	if valor < 0 {
		return 0
	}
	if valor > 1 {
		return 1
	}
	return valor
}

func (a *Aplicacion) posicionDesdeProgresoVideo(valor float32, duracion time.Duration) time.Duration {
	if duracion <= 0 {
		return 0
	}
	if valor < 0 {
		valor = 0
	}
	if valor > 1 {
		valor = 1
	}
	return time.Duration(float64(duracion) * float64(valor))
}

func normalizarInstanteVideo(instante, duracion time.Duration) time.Duration {
	return normalizarInstanteVideoConPaso(instante, duracion, 200*time.Millisecond)
}

func normalizarInstanteVideoConPaso(instante, duracion, paso time.Duration) time.Duration {
	if instante < 0 {
		instante = 0
	}
	if duracion > 0 && instante > duracion {
		instante = duracion
	}
	if instante == 0 {
		return 0
	}

	if duracion > 0 && duracion < paso {
		paso = duracion
	}
	if paso <= 0 {
		return instante
	}

	normalizado := time.Duration(math.Round(float64(instante)/float64(paso))) * paso
	if duracion > 0 && normalizado > duracion {
		normalizado = duracion
	}
	if normalizado < 0 {
		return 0
	}
	return normalizado
}

func diferenciaDuracion(izquierda, derecha time.Duration) time.Duration {
	diferencia := izquierda - derecha
	if diferencia < 0 {
		return -diferencia
	}
	return diferencia
}

func (a *Aplicacion) instanteExtraerFrameActivo() time.Duration {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return 0
	}
	if a.reproductorVideo.Ruta != a.archivoActivo.Ruta {
		return 0
	}
	if a.reproductorVideo.Fotograma != nil && diferenciaDuracion(a.reproductorVideo.InstanteFotograma, a.reproductorVideo.Posicion) <= 200*time.Millisecond {
		return a.reproductorVideo.InstanteFotograma
	}
	return a.reproductorVideo.Posicion
}

func duracionMayor(izquierda, derecha time.Duration) time.Duration {
	if izquierda > derecha {
		return izquierda
	}
	return derecha
}

func maximoDuracion(izquierda, derecha time.Duration) time.Duration {
	if izquierda > derecha {
		return izquierda
	}
	return derecha
}

func maximoEntero(a, b int) int {
	if a > b {
		return a
	}
	return b
}
