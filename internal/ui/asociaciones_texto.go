package ui

import (
	"context"
	"fmt"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"

	"destrellas-dam/internal/modelo"
)

func (a *Aplicacion) recargarAsociacionesTexto() {
	if a.almacen == nil {
		return
	}

	go func() {
		asociaciones, err := a.almacen.ListarAsociacionesTexto(context.Background(), 2_000)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudieron cargar las asociaciones de texto", err)
				return
			}
			a.actualizarAsociacionesTextoEnMemoria(asociaciones)
		})
	}()
}

func (a *Aplicacion) actualizarAsociacionesTextoEnMemoria(asociaciones []modelo.AsociacionTexto) {
	a.asociacionesTexto = append([]modelo.AsociacionTexto(nil), asociaciones...)
	if len(a.asociacionesTexto) == 0 {
		a.prepararNuevaAsociacionTexto()
		return
	}

	if _, ok := a.asociacionTextoActiva(); ok {
		return
	}
	a.seleccionarAsociacionTexto(a.asociacionesTexto[0].ID)
}

func (a *Aplicacion) prepararNuevaAsociacionTexto() {
	a.asociacionTextoActivaID = 0
	a.editorAsociacionOriginales.SetText("")
	a.editorAsociacionSugeridas.SetText("")
}

func (a *Aplicacion) seleccionarAsociacionTexto(id int64) {
	if id < 1 {
		a.prepararNuevaAsociacionTexto()
		return
	}

	for _, asociacion := range a.asociacionesTexto {
		if asociacion.ID != id {
			continue
		}
		a.asociacionTextoActivaID = id
		a.editorAsociacionOriginales.SetText(strings.Join(asociacion.Originales, ", "))
		a.editorAsociacionSugeridas.SetText(strings.Join(asociacion.Sugeridas, ", "))
		return
	}
}

func (a *Aplicacion) asociacionTextoActiva() (modelo.AsociacionTexto, bool) {
	if a.asociacionTextoActivaID < 1 {
		return modelo.AsociacionTexto{}, false
	}
	for _, asociacion := range a.asociacionesTexto {
		if asociacion.ID == a.asociacionTextoActivaID {
			return asociacion, true
		}
	}
	return modelo.AsociacionTexto{}, false
}

func (a *Aplicacion) guardarAsociacionTextoActiva() {
	if a.almacen == nil {
		a.establecerEstado("El catálogo local no está disponible para guardar asociaciones de texto", nil)
		return
	}

	id := a.asociacionTextoActivaID
	originales := partirListaCSV(a.editorAsociacionOriginales.Text())
	sugeridas := partirListaCSV(a.editorAsociacionSugeridas.Text())
	if len(originales) == 0 {
		a.establecerEstado("Indica al menos una cadena original para la asociación", nil)
		return
	}
	if len(sugeridas) == 0 {
		a.establecerEstado("Indica al menos una cadena sugerida para la asociación", nil)
		return
	}

	go func(id int64, originales, sugeridas []string) {
		asociacion, err := a.almacen.GuardarAsociacionTexto(context.Background(), id, originales, sugeridas)
		if err != nil {
			a.encolarActualizacion(func() {
				a.establecerEstado("No se pudo guardar la asociación de texto", err)
			})
			return
		}

		listado, errListado := a.almacen.ListarAsociacionesTexto(context.Background(), 2_000)
		a.encolarActualizacion(func() {
			if errListado != nil {
				a.establecerEstado("La asociación se guardó, pero no se pudo recargar la lista", errListado)
				a.seleccionarAsociacionTexto(asociacion.ID)
				return
			}
			a.actualizarAsociacionesTextoEnMemoria(listado)
			a.seleccionarAsociacionTexto(asociacion.ID)
			a.establecerEstado("Asociación de texto guardada correctamente", nil)
		})
	}(id, append([]string(nil), originales...), append([]string(nil), sugeridas...))
}

func (a *Aplicacion) eliminarAsociacionTextoActiva() {
	asociacion, ok := a.asociacionTextoActiva()
	if !ok {
		a.establecerEstado("Selecciona una asociación de texto para eliminarla", nil)
		return
	}
	if a.almacen == nil {
		a.establecerEstado("El catálogo local no está disponible para eliminar asociaciones de texto", nil)
		return
	}

	a.prepararNuevaAsociacionTexto()

	go func(id int64) {
		err := a.almacen.EliminarAsociacionTexto(context.Background(), id)
		if err != nil {
			a.encolarActualizacion(func() {
				a.establecerEstado("No se pudo eliminar la asociación de texto", err)
			})
			return
		}

		listado, errListado := a.almacen.ListarAsociacionesTexto(context.Background(), 2_000)
		a.encolarActualizacion(func() {
			if errListado != nil {
				a.establecerEstado("La asociación se eliminó, pero no se pudo recargar la lista", errListado)
				a.prepararNuevaAsociacionTexto()
				return
			}
			a.actualizarAsociacionesTextoEnMemoria(listado)
			a.establecerEstado("Asociación de texto eliminada", nil)
		})
	}(asociacion.ID)
}

func (a *Aplicacion) dibujarVistaAsociaciones(gtx layout.Context) layout.Dimensions {
	izquierda := maximo(300, gtx.Dp(unit.Dp(320)))
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = izquierda
			gtx.Constraints.Max.X = izquierda
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Asociaciones de texto")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorBusquedaLateral(gtx, &a.editorFiltroAsociaciones)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							controles := []layout.Widget{
								func(gtx layout.Context) layout.Dimensions {
									return a.dibujarBotonNavegacion(gtx, &a.botonFiltroAsociacionesTodas, "Todas", a.filtroAsociacionesTexto == filtroAsociacionTextoTodas, func() {
										a.filtroAsociacionesTexto = filtroAsociacionTextoTodas
									})
								},
								func(gtx layout.Context) layout.Dimensions {
									return a.dibujarBotonNavegacion(gtx, &a.botonFiltroAsociacionesOriginales, "Originales", a.filtroAsociacionesTexto == filtroAsociacionTextoOriginales, func() {
										a.filtroAsociacionesTexto = filtroAsociacionTextoOriginales
									})
								},
								func(gtx layout.Context) layout.Dimensions {
									return a.dibujarBotonNavegacion(gtx, &a.botonFiltroAsociacionesSugeridas, "Sugeridas", a.filtroAsociacionesTexto == filtroAsociacionTextoSugeridas, func() {
										a.filtroAsociacionesTexto = filtroAsociacionTextoSugeridas
									})
								},
							}
							return a.dibujarFlujoControles(gtx, controles)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonNuevaAsociacionTexto, "Nueva asociación", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
								a.prepararNuevaAsociacionTexto()
							})
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarListaAsociacionesTexto(gtx)
						}),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarDetalleAsociacionTexto(gtx)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarListaAsociacionesTexto(gtx layout.Context) layout.Dimensions {
	filtradas := filtrarAsociacionesTexto(a.asociacionesTexto, a.editorFiltroAsociaciones.Text(), a.filtroAsociacionesTexto)
	if len(filtradas) == 0 {
		if strings.TrimSpace(a.editorFiltroAsociaciones.Text()) != "" {
			return a.dibujarTextoSecundario(gtx, "Sin coincidencias para el filtro actual.")
		}
		return a.dibujarTextoSecundario(gtx, "Todavía no hay asociaciones de texto guardadas.")
	}

	return a.dibujarListaConBarra(gtx, &a.listaAsociaciones, len(filtradas), func(gtx layout.Context, indice int) layout.Dimensions {
		asociacion := filtradas[indice]
		clave := fmt.Sprintf("vista-asociacion-texto:%d", asociacion.ID)
		titulo := resumirListaAsociacionTexto(asociacion.Originales, 72)
		subtitulo := resumirListaAsociacionTexto(asociacion.Sugeridas, 96)
		if subtitulo == "" {
			subtitulo = "Sin cadenas sugeridas"
		}

		return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFilaDobleLinea(
				gtx,
				a.asegurarWidgetLateral(clave),
				titulo,
				subtitulo,
				a.asociacionTextoActivaID == asociacion.ID,
				func() {
					a.seleccionarAsociacionTexto(asociacion.ID)
				},
			)
		})
	})
}

func (a *Aplicacion) dibujarDetalleAsociacionTexto(gtx layout.Context) layout.Dimensions {
	asociacion, existe := a.asociacionTextoActiva()
	titulo := "Nueva asociación"
	resumen := "Define cadenas a buscar en el nombre de archivo y las palabras clave que deben sugerirse."
	if existe {
		titulo = "Editar asociación"
		resumen = fmt.Sprintf("%d cadenas originales | %d cadenas sugeridas", len(asociacion.Originales), len(asociacion.Sugeridas))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTituloPanel(gtx, titulo)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, resumen)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, "Usa comas para separar varias cadenas. Si una cadena original ya existe en otro grupo, sus sugerencias se fusionarán en ese mismo grupo.")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarEditorCampo(gtx, "Cadenas originales a buscar", &a.editorAsociacionOriginales)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarEditorCampo(gtx, "Cadenas sugeridas", &a.editorAsociacionSugeridas)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonGuardarAsociacionTexto, "Guardar asociación", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
						a.guardarAsociacionTextoActiva()
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !existe {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBotonAccion(gtx, &a.botonEliminarAsociacionTexto, "Eliminar asociación", a.paleta.Peligro, a.paleta.Texto, func() {
							a.eliminarAsociacionTextoActiva()
						})
					})
				}),
			)
		}),
	)
}

func filtrarAsociacionesTexto(asociaciones []modelo.AsociacionTexto, consulta string, filtro tipoFiltroAsociacionTexto) []modelo.AsociacionTexto {
	consulta = strings.ToLower(strings.TrimSpace(consulta))
	if consulta == "" {
		return append([]modelo.AsociacionTexto(nil), asociaciones...)
	}

	filtradas := make([]modelo.AsociacionTexto, 0, len(asociaciones))
	for _, asociacion := range asociaciones {
		coincideOriginales := coincideAlgunaCadenaTexto(asociacion.Originales, consulta)
		coincideSugeridas := coincideAlgunaCadenaTexto(asociacion.Sugeridas, consulta)

		coincide := coincideOriginales || coincideSugeridas
		switch filtro {
		case filtroAsociacionTextoOriginales:
			coincide = coincideOriginales
		case filtroAsociacionTextoSugeridas:
			coincide = coincideSugeridas
		}
		if coincide {
			filtradas = append(filtradas, asociacion)
		}
	}
	return filtradas
}

func coincideAlgunaCadenaTexto(valores []string, consulta string) bool {
	consulta = strings.ToLower(strings.TrimSpace(consulta))
	if consulta == "" {
		return true
	}
	for _, valor := range valores {
		if strings.Contains(strings.ToLower(strings.TrimSpace(valor)), consulta) {
			return true
		}
	}
	return false
}

func resumirListaAsociacionTexto(valores []string, maximo int) string {
	texto := strings.Join(valores, ", ")
	texto = strings.TrimSpace(texto)
	runas := []rune(texto)
	if maximo < 1 || len(runas) <= maximo {
		return texto
	}
	if maximo == 1 {
		return "…"
	}
	return strings.TrimSpace(string(runas[:maximo-1])) + "…"
}

func sugerenciasAsociacionesTexto(nombre string, existentes []string, asociaciones []modelo.AsociacionTexto) []string {
	nombreNormalizado := strings.ToLower(strings.TrimSpace(nombre))
	if nombreNormalizado == "" || len(asociaciones) == 0 {
		return nil
	}

	vistos := make(map[string]struct{}, len(existentes))
	for _, valor := range existentes {
		clave := strings.ToLower(strings.TrimSpace(valor))
		if clave == "" {
			continue
		}
		vistos[clave] = struct{}{}
	}

	var sugeridas []string
	for _, asociacion := range asociaciones {
		coincide := false
		for _, original := range asociacion.Originales {
			originalNormalizado := strings.ToLower(strings.TrimSpace(original))
			if originalNormalizado == "" {
				continue
			}
			if strings.Contains(nombreNormalizado, originalNormalizado) {
				coincide = true
				break
			}
		}
		if !coincide {
			continue
		}
		for _, sugerida := range asociacion.Sugeridas {
			sugerida = strings.TrimSpace(sugerida)
			clave := strings.ToLower(sugerida)
			if sugerida == "" {
				continue
			}
			if _, existe := vistos[clave]; existe {
				continue
			}
			vistos[clave] = struct{}{}
			sugeridas = append(sugeridas, sugerida)
		}
	}
	return sugeridas
}
