package ui

import (
	"context"
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

type estadoReproductorVideo struct {
	Ruta              string
	Duracion          time.Duration
	Rotacion          int
	Posicion          time.Duration
	Fotograma         image.Image
	InstanteFotograma time.Duration
	MaximoFotograma   int
	Fotogramas        []fotogramaBufferVideo
	FotogramasPorSeg  int
	InicioBuffer      time.Duration
	FinBuffer         time.Duration
	Cargando          bool
	Reproduciendo     bool
	UltimoTick        time.Time
	Error             string
	InstanteError     time.Duration
	MaximoError       int
	InstantePendiente time.Duration
	MaximoPendiente   int
	TienePendiente    bool
	VersionSolicitud  int
}

func (a *Aplicacion) limpiarReproductorVideo() {
	a.reproductorVideo = estadoReproductorVideo{}
	a.controlProgresoVideo = widget.Float{}
	a.controlExtraccionFrame = widget.Float{}
	a.formatoExtraccionExpandido = false
}

func (a *Aplicacion) sincronizarReproductorVideo(archivo modelo.Archivo) {
	if archivo.Tipo != modelo.TipoVideo || archivo.Ruta == "" {
		if a.reproductorVideo.Ruta != "" {
			a.limpiarReproductorVideo()
		}
		return
	}

	rotacion := modelo.NormalizarRotacionCuartos(archivo.Metadatos.Rotacion)
	if a.reproductorVideo.Ruta != archivo.Ruta {
		a.reproductorVideo = estadoReproductorVideo{
			Ruta:             archivo.Ruta,
			Duracion:         archivo.Duracion,
			Rotacion:         rotacion,
			FotogramasPorSeg: 12,
		}
		a.controlProgresoVideo = widget.Float{}
		a.controlExtraccionFrame = widget.Float{}
		a.formatoExtraccionExpandido = false
		return
	}

	if archivo.Duracion > 0 {
		a.reproductorVideo.Duracion = archivo.Duracion
		if a.reproductorVideo.Posicion > archivo.Duracion {
			a.reproductorVideo.Posicion = archivo.Duracion
		}
	}
	if a.reproductorVideo.FotogramasPorSeg < 1 {
		a.reproductorVideo.FotogramasPorSeg = 12
	}
	if a.reproductorVideo.Rotacion != rotacion {
		a.reproductorVideo.Rotacion = rotacion
		a.reproductorVideo.Fotograma = nil
		a.reproductorVideo.InstanteFotograma = 0
		a.reproductorVideo.MaximoFotograma = 0
		a.reproductorVideo.Fotogramas = nil
		a.reproductorVideo.InicioBuffer = 0
		a.reproductorVideo.FinBuffer = 0
		a.reproductorVideo.Cargando = false
		a.reproductorVideo.Error = ""
		a.reproductorVideo.InstanteError = 0
		a.reproductorVideo.MaximoError = 0
		a.reproductorVideo.TienePendiente = false
		a.reproductorVideo.VersionSolicitud++
	}
	a.sincronizarControlesPosicionVideo()
}

func (a *Aplicacion) actualizarReproductorVideo(gtx layout.Context, archivo modelo.Archivo, maximoFotograma int) {
	a.sincronizarReproductorVideo(archivo)
	if a.reproductorVideo.Ruta == "" {
		return
	}

	maximoBuffer := minimo(maximoFotograma, 960)
	estado := &a.reproductorVideo
	if estado.Reproduciendo {
		if estado.UltimoTick.IsZero() {
			estado.UltimoTick = gtx.Now
		}
		if delta := gtx.Now.Sub(estado.UltimoTick); delta > 0 {
			posicionObjetivo := estado.Posicion + delta
			// Si la precarga todavía no entrega el siguiente lote, mantenemos el reloj
			// sobre el último fotograma disponible para evitar un salto brusco posterior.
			if estado.Cargando && len(estado.Fotogramas) > 0 && posicionObjetivo > estado.FinBuffer {
				posicionObjetivo = estado.FinBuffer
			}
			estado.Posicion = posicionObjetivo
			if estado.Duracion > 0 && estado.Posicion >= estado.Duracion {
				estado.Posicion = estado.Duracion
				estado.Reproduciendo = false
			}
			a.sincronizarControlesPosicionVideo()
		}
		estado.UltimoTick = gtx.Now
		a.aplicarFotogramaDisponible(estado.Posicion)
		if !a.bufferCubreInstante(estado.Posicion) {
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
	if a.reproductorVideo.Duracion <= 0 {
		a.establecerEstado("Esperando la duración real del video para reproducirlo", nil)
		return
	}

	a.reproductorVideo.Reproduciendo = !a.reproductorVideo.Reproduciendo
	a.reproductorVideo.UltimoTick = time.Time{}
}

func (a *Aplicacion) reiniciarReproductorVideo() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	a.reproductorVideo.Reproduciendo = false
	a.reproductorVideo.Posicion = 0
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.Fotogramas = nil
	a.reproductorVideo.InicioBuffer = 0
	a.reproductorVideo.FinBuffer = 0
	a.sincronizarControlesPosicionVideo()
	a.solicitarFotogramaVideo(a.archivoActivo, 0, maximo(960, a.reproductorVideo.MaximoFotograma))
}

func (a *Aplicacion) actualizarPosicionVideoDesdeControl(maximoFotograma int) {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	a.reproductorVideo.Reproduciendo = false
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.Posicion = a.posicionDesdeProgresoVideo(a.controlProgresoVideo.Value, a.reproductorVideo.Duracion)
	a.sincronizarControlesPosicionVideo()
	if !a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion) {
		a.solicitarFotogramaVideo(a.archivoActivo, a.reproductorVideo.Posicion, maximoFotograma)
	}
}

func (a *Aplicacion) actualizarPosicionVideoDesdeExtraccion(maximoFotograma int) {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoVideo {
		return
	}

	a.sincronizarReproductorVideo(a.archivoActivo)
	a.reproductorVideo.Reproduciendo = false
	a.reproductorVideo.UltimoTick = time.Time{}
	a.reproductorVideo.Posicion = a.posicionDesdeProgresoVideo(a.controlExtraccionFrame.Value, a.reproductorVideo.Duracion)
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

	estado := &a.reproductorVideo
	if estado.Cargando {
		return
	}

	fotogramasPorSeg := estado.FotogramasPorSeg
	if fotogramasPorSeg < 1 {
		fotogramasPorSeg = 12
		estado.FotogramasPorSeg = fotogramasPorSeg
	}
	inicio = a.normalizarInicioLoteVideo(inicio, estado.Duracion)
	if a.bufferCubreInstante(inicio) && !a.debePrecargarSiguienteLote() {
		return
	}
	if estado.Error != "" && diferenciaDuracion(estado.InstanteError, inicio) < a.intervaloFotogramasBuffer()*2 && estado.MaximoError >= maximoFotograma {
		return
	}

	cantidad := a.cantidadFotogramasLoteBuffer()
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
			if err != nil {
				a.reproductorVideo.Error = err.Error()
				a.reproductorVideo.InstanteError = inicio
				a.reproductorVideo.MaximoError = maximoFotograma
				if len(a.reproductorVideo.Fotogramas) == 0 {
					a.reproductorVideo.Reproduciendo = false
				}
				return
			}

			a.integrarLoteFotogramas(lote)
			a.aplicarFotogramaDisponible(a.reproductorVideo.Posicion)
			a.reproductorVideo.Error = ""
			a.reproductorVideo.InstanteError = 0
			a.reproductorVideo.MaximoError = 0
		})
	}()
}

func (a *Aplicacion) integrarLoteFotogramas(nuevos []metadatos.FotogramaVideo) {
	if len(nuevos) == 0 {
		return
	}

	estado := &a.reproductorVideo
	intervalo := a.intervaloFotogramasBuffer()
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
	if len(convertidos) == 0 {
		return
	}

	if len(estado.Fotogramas) == 0 {
		estado.Fotogramas = convertidos
	} else {
		ultimoExistente := estado.Fotogramas[len(estado.Fotogramas)-1].Instante
		primerNuevo := convertidos[0].Instante
		if primerNuevo <= ultimoExistente+intervalo {
			for _, fotograma := range convertidos {
				if fotograma.Instante <= ultimoExistente {
					continue
				}
				estado.Fotogramas = append(estado.Fotogramas, fotograma)
			}
		} else {
			estado.Fotogramas = convertidos
		}
	}

	a.recortarBufferFotogramas()
	if len(estado.Fotogramas) > 0 {
		estado.InicioBuffer = estado.Fotogramas[0].Instante
		estado.FinBuffer = estado.Fotogramas[len(estado.Fotogramas)-1].Instante
	}
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
	return instante >= estado.InicioBuffer-intervalo && instante <= estado.FinBuffer+intervalo
}

func (a *Aplicacion) debePrecargarSiguienteLote() bool {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return true
	}
	if estado.Duracion > 0 && estado.FinBuffer >= estado.Duracion-a.margenFinalLoteVideo() {
		return false
	}
	return estado.FinBuffer-estado.Posicion <= a.margenPrecargaBuffer()
}

func (a *Aplicacion) inicioSiguienteLote() time.Duration {
	estado := &a.reproductorVideo
	if len(estado.Fotogramas) == 0 {
		return a.normalizarInicioLoteVideo(estado.Posicion, estado.Duracion)
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
	return time.Second / time.Duration(fps)
}

// cantidadFotogramasLoteBuffer define un bloque algo más largo para esconder mejor la latencia de ffmpeg.
func (a *Aplicacion) cantidadFotogramasLoteBuffer() int {
	fps := a.reproductorVideo.FotogramasPorSeg
	if fps < 1 {
		fps = 12
	}
	return maximoEntero(48, fps*4)
}

func (a *Aplicacion) duracionLoteBuffer() time.Duration {
	return time.Duration(a.cantidadFotogramasLoteBuffer()) * a.intervaloFotogramasBuffer()
}

func (a *Aplicacion) margenPrecargaBuffer() time.Duration {
	margen := a.duracionLoteBuffer() / 2
	if margen < 1500*time.Millisecond {
		margen = 1500 * time.Millisecond
	}
	if margen > 3*time.Second {
		margen = 3 * time.Second
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
	a.controlProgresoVideo.Value = valor
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

func (a *Aplicacion) descartarBufferFotogramas() {
	a.reproductorVideo.Fotogramas = nil
	a.reproductorVideo.InicioBuffer = 0
	a.reproductorVideo.FinBuffer = 0
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
