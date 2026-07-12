package ui

import (
	"context"
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"destrellas-dam/internal/modelo"
)

type estadoEdicionRegiones struct {
	Ruta                string
	RegionesBase        []modelo.RegionEtiquetada
	RegionesEdicion     []modelo.RegionEtiquetada
	Etiquetando         bool
	Guardando           bool
	Arrastre            gesture.Drag
	Arrastrando         bool
	InicioArrastre      f32.Point
	FinArrastre         f32.Point
	RegionPendiente     *modelo.RegionEtiquetada
	EditorNombre        widget.Editor
	SolicitarFocoNombre bool
}

func nuevoEstadoEdicionRegiones(ruta string, regiones []modelo.RegionEtiquetada) estadoEdicionRegiones {
	estado := estadoEdicionRegiones{
		Ruta:            ruta,
		RegionesBase:    clonarRegiones(regiones),
		RegionesEdicion: clonarRegiones(regiones),
	}
	estado.EditorNombre.SingleLine = true
	estado.EditorNombre.Submit = true
	return estado
}

func clonarRegiones(regiones []modelo.RegionEtiquetada) []modelo.RegionEtiquetada {
	if len(regiones) == 0 {
		return nil
	}
	clon := make([]modelo.RegionEtiquetada, len(regiones))
	copy(clon, regiones)
	return clon
}

func regionesIguales(izquierda, derecha []modelo.RegionEtiquetada) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		actual := izquierda[indice]
		otro := derecha[indice]
		if strings.TrimSpace(actual.Nombre) != strings.TrimSpace(otro.Nombre) {
			return false
		}
		if math.Abs(actual.X-otro.X) > 0.000001 ||
			math.Abs(actual.Y-otro.Y) > 0.000001 ||
			math.Abs(actual.Ancho-otro.Ancho) > 0.000001 ||
			math.Abs(actual.Alto-otro.Alto) > 0.000001 {
			return false
		}
	}
	return true
}

func (a *Aplicacion) descartarEdicionRegiones() {
	a.edicionRegiones = estadoEdicionRegiones{}
}

func (a *Aplicacion) sincronizarEdicionRegiones(archivo modelo.Archivo) {
	if archivo.Tipo != modelo.TipoImagen || archivo.Ruta == "" {
		if a.edicionRegiones.Ruta != "" {
			a.descartarEdicionRegiones()
		}
		return
	}

	if a.edicionRegiones.Ruta != archivo.Ruta {
		a.edicionRegiones = nuevoEstadoEdicionRegiones(archivo.Ruta, archivo.Metadatos.Regiones)
		return
	}

	if a.hayCambiosPendientesRegiones() || a.edicionRegiones.RegionPendiente != nil || a.edicionRegiones.Arrastrando || a.edicionRegiones.Guardando {
		return
	}

	a.edicionRegiones.RegionesBase = clonarRegiones(archivo.Metadatos.Regiones)
	a.edicionRegiones.RegionesEdicion = clonarRegiones(archivo.Metadatos.Regiones)
}

func (a *Aplicacion) regionesEnEdicion(archivo modelo.Archivo) []modelo.RegionEtiquetada {
	if archivo.Tipo == modelo.TipoImagen && archivo.Ruta != "" && a.edicionRegiones.Ruta == archivo.Ruta {
		return a.edicionRegiones.RegionesEdicion
	}
	return archivo.Metadatos.Regiones
}

func (a *Aplicacion) hayCambiosPendientesRegiones() bool {
	if a.edicionRegiones.Ruta == "" {
		return false
	}
	return !regionesIguales(a.edicionRegiones.RegionesBase, a.edicionRegiones.RegionesEdicion)
}

func (a *Aplicacion) puedeLimpiarRegiones() bool {
	if a.edicionRegiones.Ruta == "" || a.edicionRegiones.Guardando {
		return false
	}
	return len(a.edicionRegiones.RegionesEdicion) > 0 || a.edicionRegiones.RegionPendiente != nil || a.edicionRegiones.Arrastrando
}

func (a *Aplicacion) puedeGuardarRegiones() bool {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoImagen {
		return false
	}
	if a.edicionRegiones.Ruta != a.archivoActivo.Ruta {
		return false
	}
	if a.edicionRegiones.Guardando || a.edicionRegiones.RegionPendiente != nil {
		return false
	}
	return a.hayCambiosPendientesRegiones()
}

func (a *Aplicacion) alternarEtiquetadoRegiones() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoImagen {
		return
	}
	a.sincronizarEdicionRegiones(a.archivoActivo)
	if a.edicionRegiones.Ruta != a.archivoActivo.Ruta {
		return
	}
	if a.edicionRegiones.Guardando {
		return
	}
	if a.edicionRegiones.RegionPendiente != nil {
		a.edicionRegiones.SolicitarFocoNombre = true
		return
	}
	if !a.edicionRegiones.Etiquetando {
		a.edicionRecorte.Activo = false
		a.edicionRecorte.Arrastrando = false
		a.edicionRecorte.Modo = modoRecorteNinguno
	}
	a.edicionRegiones.Etiquetando = !a.edicionRegiones.Etiquetando
	if !a.edicionRegiones.Etiquetando {
		a.edicionRegiones.Arrastrando = false
	}
}

func (a *Aplicacion) limpiarRegionesEdicion() {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoImagen {
		return
	}
	a.sincronizarEdicionRegiones(a.archivoActivo)
	if a.edicionRegiones.Guardando {
		return
	}
	a.edicionRegiones.RegionesEdicion = nil
	a.edicionRegiones.RegionPendiente = nil
	a.edicionRegiones.Arrastrando = false
	a.edicionRegiones.Etiquetando = false
	a.edicionRegiones.EditorNombre.SetText("")
}

func (a *Aplicacion) procesarEventosNombreRegion(gtx layout.Context) {
	if a.edicionRegiones.RegionPendiente == nil {
		return
	}
	for {
		evento, ok := a.edicionRegiones.EditorNombre.Update(gtx)
		if !ok {
			break
		}
		switch evento.(type) {
		case widget.SubmitEvent:
			a.confirmarRegionPendiente()
		}
	}
}

func (a *Aplicacion) confirmarRegionPendiente() {
	if a.edicionRegiones.RegionPendiente == nil {
		return
	}

	nombre := strings.TrimSpace(a.edicionRegiones.EditorNombre.Text())
	if nombre == "" {
		a.edicionRegiones.SolicitarFocoNombre = true
		a.establecerEstado("Escribe un nombre para la región antes de continuar", nil)
		return
	}

	region := *a.edicionRegiones.RegionPendiente
	region.Nombre = nombre
	a.edicionRegiones.RegionesEdicion = append(a.edicionRegiones.RegionesEdicion, region)
	a.edicionRegiones.RegionPendiente = nil
	a.edicionRegiones.EditorNombre.SetText("")
	a.edicionRegiones.SolicitarFocoNombre = false
	a.establecerEstado("Región agregada a la edición actual. Pulsa Guardar para escribirla en el archivo", nil)
}

func (a *Aplicacion) guardarRegionesArchivoActivo() {
	if !a.puedeGuardarRegiones() {
		return
	}

	archivo := a.archivoActivo
	regiones := clonarRegiones(a.edicionRegiones.RegionesEdicion)
	a.edicionRegiones.Guardando = true

	go func() {
		errExif := a.servicioMetadatos.GuardarRegiones(context.Background(), archivo, regiones)
		if errExif == nil {
			archivo.Metadatos.Regiones = clonarRegiones(regiones)
			archivo.Indicadores.TieneRegiones = len(regiones) > 0
		}
		errBD := error(nil)
		if errExif == nil {
			errBD = a.almacen.GuardarArchivo(context.Background(), archivo)
		}

		a.encolarActualizacion(func() {
			a.edicionRegiones.Guardando = false
			if errExif != nil {
				a.establecerEstado("No se pudieron guardar las regiones en la imagen", errExif)
				return
			}
			if errBD != nil {
				a.establecerEstado("Las regiones se guardaron en la imagen, pero no se pudo actualizar el catálogo", errBD)
			} else {
				a.establecerEstado("Regiones guardadas correctamente", nil)
			}
			a.archivoActivo = archivo
			a.reemplazarArchivoEnMemoria(archivo)
			a.edicionRegiones = nuevoEstadoEdicionRegiones(archivo.Ruta, archivo.Metadatos.Regiones)
			a.solicitarSalidaExiftool(archivo, true)
		})
	}()
}

func (a *Aplicacion) permiteEdicionRegionesEnVisor(archivo modelo.Archivo) bool {
	return a.vistaActual == vistaElementoUnico &&
		a.tieneArchivoActivo &&
		a.archivoActivo.Ruta == archivo.Ruta &&
		archivo.Tipo == modelo.TipoImagen &&
		a.edicionRegiones.Ruta == archivo.Ruta
}

func (a *Aplicacion) actualizarInteraccionRegiones(gtx layout.Context, archivo modelo.Archivo, tamano image.Point) {
	if !a.permiteEdicionRegionesEnVisor(archivo) {
		return
	}
	if !a.edicionRegiones.Etiquetando || a.edicionRegiones.RegionPendiente != nil || tamano.X <= 0 || tamano.Y <= 0 {
		return
	}

	for {
		evento, ok := a.edicionRegiones.Arrastre.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}

		switch evento.Kind {
		case pointer.Press:
			a.edicionRegiones.Arrastrando = true
			a.edicionRegiones.InicioArrastre = evento.Position
			a.edicionRegiones.FinArrastre = evento.Position
		case pointer.Drag:
			if a.edicionRegiones.Arrastrando {
				a.edicionRegiones.FinArrastre = evento.Position
			}
		case pointer.Release:
			if !a.edicionRegiones.Arrastrando {
				continue
			}
			a.edicionRegiones.FinArrastre = evento.Position
			a.edicionRegiones.Arrastrando = false
			a.registrarRegionPendienteDesdeArrastre(archivo, tamano)
		case pointer.Cancel:
			a.edicionRegiones.Arrastrando = false
		}
	}
}

func (a *Aplicacion) registrarRegionPendienteDesdeArrastre(archivo modelo.Archivo, tamano image.Point) {
	inicioX := float64(a.edicionRegiones.InicioArrastre.X)
	inicioY := float64(a.edicionRegiones.InicioArrastre.Y)
	finX := float64(a.edicionRegiones.FinArrastre.X)
	finY := float64(a.edicionRegiones.FinArrastre.Y)

	if math.Abs(finX-inicioX) < 6 || math.Abs(finY-inicioY) < 6 {
		a.establecerEstado("La región es demasiado pequeña. Arrastra un rectángulo más amplio", nil)
		return
	}

	x := math.Min(inicioX, finX) / float64(tamano.X)
	y := math.Min(inicioY, finY) / float64(tamano.Y)
	ancho := math.Abs(finX-inicioX) / float64(tamano.X)
	alto := math.Abs(finY-inicioY) / float64(tamano.Y)
	if ancho <= 0 || alto <= 0 {
		return
	}

	regionDibujada := modelo.RegionEtiquetada{
		X:     limitarRegionNormalizada(x),
		Y:     limitarRegionNormalizada(y),
		Ancho: limitarRegionNormalizada(ancho),
		Alto:  limitarRegionNormalizada(alto),
	}
	regionArchivo := invertirTransformacionRegion(regionDibujada, archivo.Metadatos.Orientacion)
	a.edicionRegiones.RegionPendiente = &regionArchivo
	a.edicionRegiones.EditorNombre.SetText("")
	a.edicionRegiones.SolicitarFocoNombre = true
	a.establecerEstado("Asigna un nombre a la nueva región y pulsa Enter para añadirla a la edición", nil)
}

func limitarRegionNormalizada(valor float64) float64 {
	if valor < 0 {
		return 0
	}
	if valor > 1 {
		return 1
	}
	return valor
}

func invertirTransformacionRegion(region modelo.RegionEtiquetada, orientacion int) modelo.RegionEtiquetada {
	region = modelo.InvertirRegionOrientada(region, orientacion)
	return modelo.RegionEtiquetada{
		Nombre: region.Nombre,
		X:      limitarRegionNormalizada(region.X),
		Y:      limitarRegionNormalizada(region.Y),
		Ancho:  limitarRegionNormalizada(region.Ancho),
		Alto:   limitarRegionNormalizada(region.Alto),
	}
}

func (a *Aplicacion) dibujarBloqueEtiquetarRegiones(gtx layout.Context) layout.Dimensions {
	if !a.tieneArchivoActivo || a.archivoActivo.Tipo != modelo.TipoImagen || a.vistaActual != vistaElementoUnico {
		return layout.Dimensions{}
	}

	a.procesarEventosNombreRegion(gtx)
	if a.edicionRegiones.RegionPendiente != nil && a.edicionRegiones.SolicitarFocoNombre {
		gtx.Execute(key.FocusCmd{Tag: &a.edicionRegiones.EditorNombre})
		a.edicionRegiones.SolicitarFocoNombre = false
	}

	regiones := a.regionesEnEdicion(a.archivoActivo)
	mensaje := "Pulsa Etiquetar y arrastra sobre la imagen para crear regiones."
	if a.edicionRegiones.Etiquetando && a.edicionRegiones.RegionPendiente == nil {
		mensaje = "Modo Etiquetar activo: arrastra sobre la imagen para definir una región."
	}
	if a.edicionRegiones.RegionPendiente != nil {
		mensaje = "Escribe el nombre de la nueva región y pulsa Enter para añadirla."
	}
	if a.edicionRegiones.Guardando {
		mensaje = "Guardando regiones en la imagen..."
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTituloPanel(gtx, "Etiquetar regiones")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, mensaje)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					fondo := a.paleta.PanelElevado
					colorTexto := a.paleta.Texto
					if a.edicionRegiones.Etiquetando {
						fondo = a.paleta.Acento
						colorTexto = a.paleta.TextoSobreAcento
					}
					return a.dibujarBotonAccion(gtx, &a.botonAgregarRegion, "Etiquetar", fondo, colorTexto, func() {
						a.alternarEtiquetadoRegiones()
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					habilitado := a.puedeLimpiarRegiones()
					contexto := gtx
					fondo := a.paleta.PanelElevado
					colorTexto := a.paleta.Texto
					if !habilitado {
						contexto = contexto.Disabled()
						fondo = a.paleta.Panel
						colorTexto = a.paleta.TextoSuave
					}
					return a.dibujarBotonAccion(contexto, &a.botonLimpiarRegiones, "Limpiar", fondo, colorTexto, func() {
						a.limpiarRegionesEdicion()
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					habilitado := a.puedeGuardarRegiones()
					contexto := gtx
					fondo := a.paleta.Acento
					colorTexto := a.paleta.TextoSobreAcento
					if !habilitado {
						contexto = contexto.Disabled()
						fondo = a.paleta.Panel
						colorTexto = a.paleta.TextoSuave
					}
					return a.dibujarBotonAccion(contexto, &a.botonGuardarRegiones, "Guardar", fondo, colorTexto, func() {
						a.guardarRegionesArchivoActivo()
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.edicionRegiones.RegionPendiente == nil {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarEditorCampo(gtx, "Nombre de la región", &a.edicionRegiones.EditorNombre)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, "La región se añadirá a la vista cuando confirmes con Enter.")
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.hayCambiosPendientesRegiones() {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, "Hay cambios pendientes sin guardar.")
				})
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(regiones) == 0 {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, "Sin regiones en la edición actual.")
				})
			}
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, "Regiones visibles")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarListaRegionesEdicion(gtx, regiones)
					}),
				)
			})
		}),
	)
}

func (a *Aplicacion) dibujarListaRegionesEdicion(gtx layout.Context, regiones []modelo.RegionEtiquetada) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, a.construirElementosRegiones(regiones)...)
}

func (a *Aplicacion) construirElementosRegiones(regiones []modelo.RegionEtiquetada) []layout.FlexChild {
	elementos := make([]layout.FlexChild, 0, len(regiones)*2)
	for indice, region := range regiones {
		nombre := strings.TrimSpace(region.Nombre)
		if nombre == "" {
			nombre = "Región sin nombre"
		}
		texto := nombre
		elementos = append(elementos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					etiqueta := material.Label(a.tema, unit.Sp(12), texto)
					etiqueta.Color = a.paleta.Texto
					etiqueta.MaxLines = 1
					etiqueta.Truncator = "…"
					return etiqueta.Layout(gtx)
				})
			})
		}))
		if indice < len(regiones)-1 {
			elementos = append(elementos, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
	}
	return elementos
}

func (a *Aplicacion) dibujarSuperposicionEdicionRegiones(gtx layout.Context, archivo modelo.Archivo, tamano image.Point) {
	if !a.permiteEdicionRegionesEnVisor(archivo) {
		return
	}

	defer clip.Rect(image.Rectangle{Max: tamano}).Push(gtx.Ops).Pop()
	if a.edicionRegiones.Etiquetando && a.edicionRegiones.RegionPendiente == nil {
		a.edicionRegiones.Arrastre.Add(gtx.Ops)
		if a.edicionRegiones.Arrastrando {
			pointer.CursorGrabbing.Add(gtx.Ops)
		} else {
			pointer.CursorCrosshair.Add(gtx.Ops)
		}
	}

	if a.edicionRegiones.RegionPendiente != nil {
		a.dibujarContornoRegion(gtx, archivo, *a.edicionRegiones.RegionPendiente, tamano, a.paleta.Acento, false)
	}
	if a.edicionRegiones.Arrastrando {
		regionDibujada := modelo.RegionEtiquetada{
			X:     limitarRegionNormalizada(float64(minimoFlotante(a.edicionRegiones.InicioArrastre.X, a.edicionRegiones.FinArrastre.X)) / float64(tamano.X)),
			Y:     limitarRegionNormalizada(float64(minimoFlotante(a.edicionRegiones.InicioArrastre.Y, a.edicionRegiones.FinArrastre.Y)) / float64(tamano.Y)),
			Ancho: limitarRegionNormalizada(float64(math.Abs(float64(a.edicionRegiones.FinArrastre.X-a.edicionRegiones.InicioArrastre.X))) / float64(tamano.X)),
			Alto:  limitarRegionNormalizada(float64(math.Abs(float64(a.edicionRegiones.FinArrastre.Y-a.edicionRegiones.InicioArrastre.Y))) / float64(tamano.Y)),
		}
		a.dibujarContornoRegionDirecto(gtx, regionDibujada, tamano, a.paleta.Acento)
	}
}

func (a *Aplicacion) dibujarContornoRegionDirecto(gtx layout.Context, region modelo.RegionEtiquetada, tamano image.Point, contorno color.NRGBA) {
	inicioX := float32(region.X * float64(tamano.X))
	inicioY := float32(region.Y * float64(tamano.Y))
	finX := float32((region.X + region.Ancho) * float64(tamano.X))
	finY := float32((region.Y + region.Alto) * float64(tamano.Y))
	a.dibujarContornoRectangulo(gtx, inicioX, inicioY, finX, finY, contorno)
}

func (a *Aplicacion) dibujarContornoRegion(gtx layout.Context, archivo modelo.Archivo, region modelo.RegionEtiquetada, tamano image.Point, contorno color.NRGBA, dibujarEtiqueta bool) {
	x, y, ancho, alto := transformarRegion(region, archivo.Metadatos.Orientacion)
	inicioX := float32(x * float64(tamano.X))
	inicioY := float32(y * float64(tamano.Y))
	finX := float32((x + ancho) * float64(tamano.X))
	finY := float32((y + alto) * float64(tamano.Y))

	a.dibujarContornoRectangulo(gtx, inicioX, inicioY, finX, finY, contorno)
	if !dibujarEtiqueta {
		return
	}
	nombre := strings.TrimSpace(region.Nombre)
	if nombre == "" {
		return
	}
	a.dibujarEtiquetaRegion(gtx, image.Pt(int(inicioX)+6, int(inicioY)+6), nombre)
}

func (a *Aplicacion) dibujarContornoRectangulo(gtx layout.Context, inicioX, inicioY, finX, finY float32, contorno color.NRGBA) {
	var camino clip.Path
	camino.Begin(gtx.Ops)
	camino.MoveTo(f32.Pt(inicioX, inicioY))
	camino.LineTo(f32.Pt(finX, inicioY))
	camino.LineTo(f32.Pt(finX, finY))
	camino.LineTo(f32.Pt(inicioX, finY))
	camino.Close()
	paint.FillShape(gtx.Ops, color.NRGBA{R: contorno.R, G: contorno.G, B: contorno.B, A: 40}, clip.Outline{Path: camino.End()}.Op())

	var borde clip.Path
	borde.Begin(gtx.Ops)
	borde.MoveTo(f32.Pt(inicioX, inicioY))
	borde.LineTo(f32.Pt(finX, inicioY))
	borde.LineTo(f32.Pt(finX, finY))
	borde.LineTo(f32.Pt(inicioX, finY))
	borde.Close()
	paint.FillShape(gtx.Ops, contorno, clip.Stroke{
		Path:  borde.End(),
		Width: float32(maximo(1, gtx.Dp(unit.Dp(2)))),
	}.Op())
}

func minimoFlotante(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
