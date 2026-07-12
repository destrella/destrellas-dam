package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"destrellas-dam/internal/modelo"
	serviciometadatos "destrellas-dam/internal/servicios/metadatos"
)

type modoAjusteRecorte uint8

const (
	modoRecorteNinguno modoAjusteRecorte = iota
	modoRecorteNuevo
	modoRecorteIzquierda
	modoRecorteDerecha
	modoRecorteSuperior
	modoRecorteInferior
	modoRecorteSuperiorIzquierda
	modoRecorteSuperiorDerecha
	modoRecorteInferiorIzquierda
	modoRecorteInferiorDerecha
)

type estadoEdicionRecorte struct {
	Ruta               string
	OrientacionPreview int
	Activo             bool
	Guardando          bool
	TieneSeleccion     bool
	Seleccion          modelo.RegionEtiquetada
	Sugerida           bool
	Arrastre           gesture.Drag
	Arrastrando        bool
	Modo               modoAjusteRecorte
	InicioArrastre     f32.Point
	FinArrastre        f32.Point
	SeleccionBase      modelo.RegionEtiquetada
	TeniaSeleccionBase bool
}

func nuevoEstadoEdicionRecorte(ruta string, orientacionPreview int) estadoEdicionRecorte {
	return estadoEdicionRecorte{
		Ruta:               ruta,
		OrientacionPreview: orientacionPreview,
	}
}

func (a *Aplicacion) descartarEdicionRecorte() {
	a.edicionRecorte = estadoEdicionRecorte{}
}

func (a *Aplicacion) sincronizarEdicionRecorte(archivo modelo.Archivo) {
	if archivo.Tipo != modelo.TipoImagen || archivo.Ruta == "" {
		if a.edicionRecorte.Ruta != "" {
			a.descartarEdicionRecorte()
		}
		return
	}

	orientacionPreview := orientacionPreviewArchivo(archivo)
	if a.edicionRecorte.Ruta != archivo.Ruta || a.edicionRecorte.OrientacionPreview != orientacionPreview {
		a.edicionRecorte = nuevoEstadoEdicionRecorte(archivo.Ruta, orientacionPreview)
	}
}

func (a *Aplicacion) alternarSeleccionRecorte() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoImagen {
		return
	}
	a.sincronizarEdicionRecorte(a.archivoActivo)
	if a.edicionRecorte.Ruta != a.archivoActivo.Ruta || a.edicionRecorte.Guardando {
		return
	}
	if a.edicionRegiones.RegionPendiente != nil {
		a.establecerEstado("Confirma primero el nombre de la región pendiente antes de recortar", nil)
		return
	}

	a.edicionRecorte.Activo = !a.edicionRecorte.Activo
	a.edicionRecorte.Arrastrando = false
	a.edicionRecorte.Modo = modoRecorteNinguno

	if !a.edicionRecorte.Activo {
		a.establecerEstado("Selección de recorte desactivada", nil)
		return
	}

	a.edicionRegiones.Etiquetando = false
	a.edicionRegiones.Arrastrando = false
	if !a.edicionRecorte.TieneSeleccion && a.aplicarSugerenciaRecorte(a.archivoActivo) {
		a.establecerEstado("Sugerencia de recorte detectada. Puedes reajustarla desde los bordes o las esquinas", nil)
		return
	}
	a.establecerEstado("Modo de recorte activo. Arrastra sobre la imagen para crear o reajustar el área", nil)
}

func (a *Aplicacion) aplicarSugerenciaRecorte(archivo modelo.Archivo) bool {
	seleccion, ok := a.sugerenciaRecorte(archivo)
	if !ok {
		return false
	}
	a.edicionRecorte.Seleccion = seleccion
	a.edicionRecorte.TieneSeleccion = true
	a.edicionRecorte.Sugerida = true
	return true
}

func (a *Aplicacion) sugerenciaRecorte(archivo modelo.Archivo) (modelo.RegionEtiquetada, bool) {
	preview, existe := a.previews[archivo.Ruta]
	if !existe || preview == nil || preview.Imagen == nil {
		return modelo.RegionEtiquetada{}, false
	}

	rectangulo, ok := serviciometadatos.SugerirRecorteMate(preview.Imagen)
	if !ok {
		return modelo.RegionEtiquetada{}, false
	}
	return regionNormalizadaDesdeRectangulo(rectangulo, preview.Imagen.Bounds().Size())
}

func (a *Aplicacion) permiteInteraccionRecorteEnVisor(archivo modelo.Archivo) bool {
	return a.vistaActual == vistaElementoUnico &&
		a.tieneArchivoActivo &&
		a.archivoActivo.Ruta == archivo.Ruta &&
		archivo.Tipo == modelo.TipoImagen &&
		a.edicionRecorte.Ruta == archivo.Ruta &&
		a.edicionRecorte.Activo
}

func (a *Aplicacion) actualizarInteraccionRecorte(gtx layout.Context, archivo modelo.Archivo, tamano image.Point) {
	if !a.permiteInteraccionRecorteEnVisor(archivo) || tamano.X <= 0 || tamano.Y <= 0 {
		return
	}

	tolerancia := float32(maximo(8, gtx.Dp(unit.Dp(10))))
	for {
		evento, ok := a.edicionRecorte.Arrastre.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}

		switch evento.Kind {
		case pointer.Press:
			modo := a.resolverModoAjusteRecorte(evento.Position, tamano, tolerancia)
			if modo == modoRecorteNinguno {
				continue
			}
			a.edicionRecorte.Arrastrando = true
			a.edicionRecorte.Modo = modo
			a.edicionRecorte.InicioArrastre = evento.Position
			a.edicionRecorte.FinArrastre = evento.Position
			a.edicionRecorte.SeleccionBase = a.edicionRecorte.Seleccion
			a.edicionRecorte.TeniaSeleccionBase = a.edicionRecorte.TieneSeleccion
		case pointer.Drag:
			if a.edicionRecorte.Arrastrando {
				a.edicionRecorte.FinArrastre = evento.Position
			}
		case pointer.Release:
			if !a.edicionRecorte.Arrastrando {
				continue
			}
			a.edicionRecorte.FinArrastre = evento.Position
			a.finalizarInteraccionRecorte(archivo, tamano)
		case pointer.Cancel:
			a.edicionRecorte.Arrastrando = false
			a.edicionRecorte.Modo = modoRecorteNinguno
		}
	}
}

func (a *Aplicacion) resolverModoAjusteRecorte(posicion f32.Point, tamano image.Point, tolerancia float32) modoAjusteRecorte {
	if !a.edicionRecorte.TieneSeleccion {
		return modoRecorteNuevo
	}

	inicioX, inicioY, finX, finY := ladosRectanguloRegion(a.edicionRecorte.Seleccion, tamano)
	cercaIzquierda := math.Abs(float64(posicion.X-inicioX)) <= float64(tolerancia)
	cercaDerecha := math.Abs(float64(posicion.X-finX)) <= float64(tolerancia)
	cercaSuperior := math.Abs(float64(posicion.Y-inicioY)) <= float64(tolerancia)
	cercaInferior := math.Abs(float64(posicion.Y-finY)) <= float64(tolerancia)
	entreVertical := posicion.Y >= inicioY-tolerancia && posicion.Y <= finY+tolerancia
	entreHorizontal := posicion.X >= inicioX-tolerancia && posicion.X <= finX+tolerancia

	switch {
	case cercaIzquierda && cercaSuperior:
		return modoRecorteSuperiorIzquierda
	case cercaDerecha && cercaSuperior:
		return modoRecorteSuperiorDerecha
	case cercaIzquierda && cercaInferior:
		return modoRecorteInferiorIzquierda
	case cercaDerecha && cercaInferior:
		return modoRecorteInferiorDerecha
	case cercaIzquierda && entreVertical:
		return modoRecorteIzquierda
	case cercaDerecha && entreVertical:
		return modoRecorteDerecha
	case cercaSuperior && entreHorizontal:
		return modoRecorteSuperior
	case cercaInferior && entreHorizontal:
		return modoRecorteInferior
	case posicion.X >= inicioX && posicion.X <= finX && posicion.Y >= inicioY && posicion.Y <= finY:
		return modoRecorteNinguno
	default:
		return modoRecorteNuevo
	}
}

func (a *Aplicacion) finalizarInteraccionRecorte(archivo modelo.Archivo, tamano image.Point) {
	region, ok := a.regionRecorteTemporal(tamano)
	base := a.edicionRecorte.SeleccionBase
	teniaBase := a.edicionRecorte.TeniaSeleccionBase
	modo := a.edicionRecorte.Modo
	a.edicionRecorte.Arrastrando = false
	a.edicionRecorte.Modo = modoRecorteNinguno

	if !ok {
		if teniaBase {
			a.edicionRecorte.Seleccion = base
			a.edicionRecorte.TieneSeleccion = true
			return
		}
		a.edicionRecorte.TieneSeleccion = false
		if modo == modoRecorteNuevo {
			a.establecerEstado("El área de recorte es demasiado pequeña. Arrastra un rectángulo más amplio", nil)
		}
		return
	}

	a.edicionRecorte.Seleccion = region
	a.edicionRecorte.TieneSeleccion = true
	a.edicionRecorte.Sugerida = false
	if dimensiones := a.descripcionDimensionesRecorte(archivo, region); dimensiones != "" {
		a.establecerEstado("Área de recorte lista: "+dimensiones, nil)
	}
}

func (a *Aplicacion) regionRecorteVisible(tamano image.Point) (modelo.RegionEtiquetada, bool) {
	if a.edicionRecorte.Arrastrando {
		return a.regionRecorteTemporal(tamano)
	}
	if !a.edicionRecorte.TieneSeleccion {
		return modelo.RegionEtiquetada{}, false
	}
	return a.edicionRecorte.Seleccion, true
}

func (a *Aplicacion) regionRecorteTemporal(tamano image.Point) (modelo.RegionEtiquetada, bool) {
	switch a.edicionRecorte.Modo {
	case modoRecorteNuevo:
		return regionNormalizadaDesdePuntos(a.edicionRecorte.InicioArrastre, a.edicionRecorte.FinArrastre, tamano)
	case modoRecorteIzquierda, modoRecorteDerecha, modoRecorteSuperior, modoRecorteInferior,
		modoRecorteSuperiorIzquierda, modoRecorteSuperiorDerecha, modoRecorteInferiorIzquierda, modoRecorteInferiorDerecha:
		if !a.edicionRecorte.TeniaSeleccionBase {
			return modelo.RegionEtiquetada{}, false
		}
		return a.regionRecorteRedimensionada(tamano)
	default:
		if !a.edicionRecorte.TieneSeleccion {
			return modelo.RegionEtiquetada{}, false
		}
		return a.edicionRecorte.Seleccion, true
	}
}

func (a *Aplicacion) regionRecorteRedimensionada(tamano image.Point) (modelo.RegionEtiquetada, bool) {
	inicioX, inicioY, finX, finY := ladosRectanguloRegion(a.edicionRecorte.SeleccionBase, tamano)
	posicion := a.edicionRecorte.FinArrastre
	minimoTamano := float32(maximo(10, tamanoMinimoRecortePx(tamano)))
	limiteX := float32(tamano.X)
	limiteY := float32(tamano.Y)

	switch a.edicionRecorte.Modo {
	case modoRecorteIzquierda:
		inicioX = limitarFloat32(posicion.X, 0, finX-minimoTamano)
	case modoRecorteDerecha:
		finX = limitarFloat32(posicion.X, inicioX+minimoTamano, limiteX)
	case modoRecorteSuperior:
		inicioY = limitarFloat32(posicion.Y, 0, finY-minimoTamano)
	case modoRecorteInferior:
		finY = limitarFloat32(posicion.Y, inicioY+minimoTamano, limiteY)
	case modoRecorteSuperiorIzquierda:
		inicioX = limitarFloat32(posicion.X, 0, finX-minimoTamano)
		inicioY = limitarFloat32(posicion.Y, 0, finY-minimoTamano)
	case modoRecorteSuperiorDerecha:
		finX = limitarFloat32(posicion.X, inicioX+minimoTamano, limiteX)
		inicioY = limitarFloat32(posicion.Y, 0, finY-minimoTamano)
	case modoRecorteInferiorIzquierda:
		inicioX = limitarFloat32(posicion.X, 0, finX-minimoTamano)
		finY = limitarFloat32(posicion.Y, inicioY+minimoTamano, limiteY)
	case modoRecorteInferiorDerecha:
		finX = limitarFloat32(posicion.X, inicioX+minimoTamano, limiteX)
		finY = limitarFloat32(posicion.Y, inicioY+minimoTamano, limiteY)
	}

	return regionNormalizadaDesdeLados(inicioX, inicioY, finX, finY, tamano)
}

func (a *Aplicacion) rectanguloRecorteActivoPixeles(archivo modelo.Archivo) (image.Rectangle, bool) {
	if a.edicionRecorte.Ruta != archivo.Ruta || !a.edicionRecorte.TieneSeleccion {
		return image.Rectangle{}, false
	}
	return rectanguloRecortePixelesParaRegion(archivo, a.edicionRecorte.Seleccion)
}

func rectanguloRecortePixelesParaRegion(archivo modelo.Archivo, region modelo.RegionEtiquetada) (image.Rectangle, bool) {
	ancho, alto := dimensionesOrientadasArchivo(archivo)
	if ancho <= 0 || alto <= 0 {
		return image.Rectangle{}, false
	}

	minX := int(math.Round(region.X * float64(ancho)))
	minY := int(math.Round(region.Y * float64(alto)))
	maxX := int(math.Round((region.X + region.Ancho) * float64(ancho)))
	maxY := int(math.Round((region.Y + region.Alto) * float64(alto)))

	minX = limitarEntero(minX, 0, ancho-1)
	minY = limitarEntero(minY, 0, alto-1)
	maxX = limitarEntero(maxX, minX+1, ancho)
	maxY = limitarEntero(maxY, minY+1, alto)
	return image.Rect(minX, minY, maxX, maxY), true
}

func dimensionesOrientadasArchivo(archivo modelo.Archivo) (int, int) {
	ancho := archivo.Ancho
	alto := archivo.Alto
	switch modelo.NormalizarOrientacionVisual(archivo.Metadatos.Orientacion) {
	case 5, 6, 7, 8:
		ancho, alto = alto, ancho
	}
	return ancho, alto
}

func (a *Aplicacion) descripcionDimensionesRecorte(archivo modelo.Archivo, region modelo.RegionEtiquetada) string {
	rectangulo, ok := rectanguloRecortePixelesParaRegion(archivo, region)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d x %d px", rectangulo.Dx(), rectangulo.Dy())
}

func (a *Aplicacion) dibujarSuperposicionRecorte(gtx layout.Context, archivo modelo.Archivo, tamano image.Point) {
	if a.edicionRecorte.Ruta != archivo.Ruta {
		return
	}

	region, visible := a.regionRecorteVisible(tamano)
	if !visible && !a.permiteInteraccionRecorteEnVisor(archivo) {
		return
	}

	defer clip.Rect(image.Rectangle{Max: tamano}).Push(gtx.Ops).Pop()
	if a.permiteInteraccionRecorteEnVisor(archivo) {
		a.edicionRecorte.Arrastre.Add(gtx.Ops)
		cursor := pointer.CursorCrosshair
		if a.edicionRecorte.Arrastrando {
			cursor = cursorRecorteSegunModo(a.edicionRecorte.Modo)
		}
		cursor.Add(gtx.Ops)
	}

	if !visible {
		return
	}

	contorno := color.NRGBA{R: 255, G: 184, B: 61, A: 255}
	a.dibujarMascaraExteriorRecorte(gtx, region, tamano)
	a.dibujarContornoRegionDirecto(gtx, region, tamano, contorno)
	a.dibujarAsasRecorte(gtx, region, tamano, contorno)

	dimensiones := a.descripcionDimensionesRecorte(archivo, region)
	if dimensiones == "" {
		return
	}
	inicioX, inicioY, _, _ := ladosRectanguloRegion(region, tamano)
	a.dibujarEtiquetaRegion(gtx, image.Pt(int(inicioX)+6, int(inicioY)+6), dimensiones)
}

func (a *Aplicacion) dibujarMascaraExteriorRecorte(gtx layout.Context, region modelo.RegionEtiquetada, tamano image.Point) {
	inicioX, inicioY, finX, finY := ladosRectanguloRegion(region, tamano)
	mascara := color.NRGBA{A: 84}
	zonas := []image.Rectangle{
		image.Rect(0, 0, tamano.X, int(inicioY)),
		image.Rect(0, int(inicioY), int(inicioX), int(finY)),
		image.Rect(int(finX), int(inicioY), tamano.X, int(finY)),
		image.Rect(0, int(finY), tamano.X, tamano.Y),
	}
	for _, zona := range zonas {
		if zona.Dx() <= 0 || zona.Dy() <= 0 {
			continue
		}
		paint.FillShape(gtx.Ops, mascara, clip.Rect(zona).Op())
	}
}

func (a *Aplicacion) dibujarAsasRecorte(gtx layout.Context, region modelo.RegionEtiquetada, tamano image.Point, colorAsa color.NRGBA) {
	inicioX, inicioY, finX, finY := ladosRectanguloRegion(region, tamano)
	centroX := (inicioX + finX) / 2
	centroY := (inicioY + finY) / 2
	lado := float32(maximo(8, gtx.Dp(unit.Dp(8))))

	puntos := []f32.Point{
		f32.Pt(inicioX, inicioY),
		f32.Pt(centroX, inicioY),
		f32.Pt(finX, inicioY),
		f32.Pt(inicioX, centroY),
		f32.Pt(finX, centroY),
		f32.Pt(inicioX, finY),
		f32.Pt(centroX, finY),
		f32.Pt(finX, finY),
	}
	for _, punto := range puntos {
		rectangulo := image.Rect(
			int(punto.X-lado/2),
			int(punto.Y-lado/2),
			int(punto.X+lado/2),
			int(punto.Y+lado/2),
		)
		paint.FillShape(gtx.Ops, colorAsa, clip.Rect(rectangulo).Op())
	}
}

func regionNormalizadaDesdeRectangulo(rectangulo image.Rectangle, tamano image.Point) (modelo.RegionEtiquetada, bool) {
	return regionNormalizadaDesdeLados(
		float32(rectangulo.Min.X),
		float32(rectangulo.Min.Y),
		float32(rectangulo.Max.X),
		float32(rectangulo.Max.Y),
		tamano,
	)
}

func regionNormalizadaDesdePuntos(inicio, fin f32.Point, tamano image.Point) (modelo.RegionEtiquetada, bool) {
	return regionNormalizadaDesdeLados(
		minimoFlotante(inicio.X, fin.X),
		minimoFlotante(inicio.Y, fin.Y),
		maximoFlotante(inicio.X, fin.X),
		maximoFlotante(inicio.Y, fin.Y),
		tamano,
	)
}

func regionNormalizadaDesdeLados(inicioX, inicioY, finX, finY float32, tamano image.Point) (modelo.RegionEtiquetada, bool) {
	if tamano.X <= 0 || tamano.Y <= 0 {
		return modelo.RegionEtiquetada{}, false
	}

	inicioX = limitarFloat32(inicioX, 0, float32(tamano.X))
	inicioY = limitarFloat32(inicioY, 0, float32(tamano.Y))
	finX = limitarFloat32(finX, 0, float32(tamano.X))
	finY = limitarFloat32(finY, 0, float32(tamano.Y))

	if finX-inicioX < float32(tamanoMinimoRecortePx(tamano)) || finY-inicioY < float32(tamanoMinimoRecortePx(tamano)) {
		return modelo.RegionEtiquetada{}, false
	}

	return modelo.RegionEtiquetada{
		X:     limitarRegionNormalizada(float64(inicioX) / float64(tamano.X)),
		Y:     limitarRegionNormalizada(float64(inicioY) / float64(tamano.Y)),
		Ancho: limitarRegionNormalizada(float64(finX-inicioX) / float64(tamano.X)),
		Alto:  limitarRegionNormalizada(float64(finY-inicioY) / float64(tamano.Y)),
	}, true
}

func ladosRectanguloRegion(region modelo.RegionEtiquetada, tamano image.Point) (inicioX, inicioY, finX, finY float32) {
	inicioX = float32(region.X * float64(tamano.X))
	inicioY = float32(region.Y * float64(tamano.Y))
	finX = float32((region.X + region.Ancho) * float64(tamano.X))
	finY = float32((region.Y + region.Alto) * float64(tamano.Y))
	return inicioX, inicioY, finX, finY
}

func cursorRecorteSegunModo(modo modoAjusteRecorte) pointer.Cursor {
	switch modo {
	case modoRecorteIzquierda:
		return pointer.CursorWestResize
	case modoRecorteDerecha:
		return pointer.CursorEastResize
	case modoRecorteSuperior:
		return pointer.CursorNorthResize
	case modoRecorteInferior:
		return pointer.CursorSouthResize
	case modoRecorteSuperiorIzquierda, modoRecorteInferiorDerecha:
		return pointer.CursorNorthWestSouthEastResize
	case modoRecorteSuperiorDerecha, modoRecorteInferiorIzquierda:
		return pointer.CursorNorthEastSouthWestResize
	default:
		return pointer.CursorCrosshair
	}
}

func tamanoMinimoRecortePx(tamano image.Point) int {
	return maximo(12, minimo(tamano.X, tamano.Y)/30)
}

func limitarFloat32(valor, minimo, maximo float32) float32 {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}

func limitarEntero(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}

func maximoFlotante(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
