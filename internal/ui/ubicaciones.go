package ui

import (
	"context"
	"fmt"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"destrellas-dam/internal/modelo"
	serviciometadatos "destrellas-dam/internal/servicios/metadatos"
)

func (a *Aplicacion) recargarUbicacionesGuardadas() {
	go func() {
		ubicaciones, err := a.almacen.ListarUbicacionesGuardadas(context.Background(), 1_000)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudieron cargar las ubicaciones guardadas", err)
				return
			}
			a.actualizarUbicacionesGuardadasEnMemoria(ubicaciones)
		})
	}()
}

func (a *Aplicacion) actualizarUbicacionesGuardadasEnMemoria(ubicaciones []modelo.UbicacionGuardada) {
	a.ubicacionesGuardadas = append([]modelo.UbicacionGuardada(nil), ubicaciones...)
	if len(a.ubicacionesGuardadas) == 0 {
		a.ubicacionSeleccionada = ""
		a.usosUbicacionSeleccionada = nil
		a.editorRelacionUbicacion.SetText("")
		return
	}

	if ubicacion, ok := a.ubicacionGuardadaSeleccionada(); ok {
		a.editorRelacionUbicacion.SetText(ubicacion.RelacionadaCon)
		return
	}

	a.seleccionarUbicacionGuardada(a.ubicacionesGuardadas[0].Nombre)
}

func (a *Aplicacion) seleccionarUbicacionGuardada(nombre string) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return
	}
	a.ubicacionSeleccionada = nombre
	if ubicacion, ok := a.ubicacionGuardadaSeleccionada(); ok {
		a.editorRelacionUbicacion.SetText(ubicacion.RelacionadaCon)
	} else {
		a.editorRelacionUbicacion.SetText("")
	}
	a.cargarUsosUbicacionSeleccionada()
}

func (a *Aplicacion) cargarUsosUbicacionSeleccionada() {
	nombre := strings.TrimSpace(a.ubicacionSeleccionada)
	if nombre == "" {
		a.usosUbicacionSeleccionada = nil
		a.cargandoUsosUbicacion = false
		return
	}

	a.cargandoUsosUbicacion = true
	go func(nombreSeleccionado string) {
		usos, err := a.almacen.ListarUsosUbicacionGuardada(context.Background(), nombreSeleccionado, 200)
		a.encolarActualizacion(func() {
			if !ubicacionesCoinciden(a.ubicacionSeleccionada, nombreSeleccionado) {
				return
			}
			a.cargandoUsosUbicacion = false
			if err != nil {
				a.establecerEstado("No se pudieron cargar los usos de la ubicación seleccionada", err)
				return
			}
			a.usosUbicacionSeleccionada = usos
		})
	}(nombre)
}

func (a *Aplicacion) ubicacionGuardadaSeleccionada() (modelo.UbicacionGuardada, bool) {
	for _, ubicacion := range a.ubicacionesGuardadas {
		if ubicacionesCoinciden(ubicacion.Nombre, a.ubicacionSeleccionada) {
			return ubicacion, true
		}
	}
	return modelo.UbicacionGuardada{}, false
}

func (a *Aplicacion) guardarRelacionUbicacionSeleccionada() {
	origen := strings.TrimSpace(a.ubicacionSeleccionada)
	destino := strings.TrimSpace(a.editorRelacionUbicacion.Text())
	if origen == "" {
		a.establecerEstado("Selecciona una ubicación para guardar una relación", nil)
		return
	}
	if destino == "" {
		a.establecerEstado("Indica el nombre de la ubicación destino para crear la relación", nil)
		return
	}

	go func() {
		err := a.almacen.GuardarRelacionUbicacion(context.Background(), origen, destino)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo guardar la relación de ubicación", err)
				return
			}
			a.establecerEstado("Relación de ubicación guardada correctamente", nil)
			a.recargarUbicacionesGuardadas()
		})
	}()
}

func (a *Aplicacion) quitarRelacionUbicacionSeleccionada() {
	origen := strings.TrimSpace(a.ubicacionSeleccionada)
	if origen == "" {
		return
	}

	go func() {
		err := a.almacen.EliminarRelacionUbicacion(context.Background(), origen)
		a.encolarActualizacion(func() {
			if err != nil {
				a.establecerEstado("No se pudo quitar la relación de ubicación", err)
				return
			}
			a.editorRelacionUbicacion.SetText("")
			a.establecerEstado("Relación de ubicación eliminada", nil)
			a.recargarUbicacionesGuardadas()
		})
	}()
}

func (a *Aplicacion) aplicarUbicacionGuardadaAlFormulario(ubicacion modelo.UbicacionGuardada) {
	a.editorUbicacion.SetText(strings.TrimSpace(ubicacion.Nombre))
	if ubicacion.Coordenadas != nil {
		a.editorGPSLatitud.SetText(fmt.Sprintf("%.8f", ubicacion.Coordenadas.Latitud))
		a.editorGPSLongitud.SetText(fmt.Sprintf("%.8f", ubicacion.Coordenadas.Longitud))
	} else {
		a.editorGPSLatitud.SetText("")
		a.editorGPSLongitud.SetText("")
	}

	if !a.tieneArchivoActivo {
		return
	}

	a.archivoActivo.Metadatos.Ubicacion = strings.TrimSpace(ubicacion.Nombre)
	if ubicacion.Coordenadas != nil {
		coordenadas := *ubicacion.Coordenadas
		a.archivoActivo.Metadatos.Coordenadas = &coordenadas
		a.archivoActivo.Indicadores.TieneGPS = true
	} else {
		a.archivoActivo.Metadatos.Coordenadas = nil
		a.archivoActivo.Indicadores.TieneGPS = false
	}
	a.archivoActivo.Metadatos.Ciudad = strings.TrimSpace(ubicacion.Ciudad)
	a.archivoActivo.Metadatos.Estado = strings.TrimSpace(ubicacion.Estado)
	a.archivoActivo.Metadatos.Pais = strings.TrimSpace(ubicacion.Pais)
}

func (a *Aplicacion) sugerenciasUbicacionFormulario() []modelo.UbicacionGuardada {
	consulta := strings.TrimSpace(a.editorUbicacion.Text())
	if consulta == "" {
		return nil
	}
	if a.existeUbicacionGuardada(consulta) {
		return nil
	}
	return filtrarUbicacionesGuardadas(a.ubicacionesGuardadas, consulta, "", 6)
}

func (a *Aplicacion) sugerenciasRelacionUbicacion() []modelo.UbicacionGuardada {
	consulta := strings.TrimSpace(a.editorRelacionUbicacion.Text())
	if consulta == "" {
		return nil
	}
	if a.existeUbicacionGuardada(consulta) && !ubicacionesCoinciden(consulta, a.ubicacionSeleccionada) {
		return nil
	}
	return filtrarUbicacionesGuardadas(a.ubicacionesGuardadas, consulta, a.ubicacionSeleccionada, 8)
}

func (a *Aplicacion) existeUbicacionGuardada(nombre string) bool {
	for _, ubicacion := range a.ubicacionesGuardadas {
		if ubicacionesCoinciden(ubicacion.Nombre, nombre) {
			return true
		}
	}
	return false
}

func filtrarUbicacionesGuardadas(ubicaciones []modelo.UbicacionGuardada, consulta, excluir string, limite int) []modelo.UbicacionGuardada {
	consulta = strings.ToLower(strings.TrimSpace(consulta))
	excluir = strings.TrimSpace(excluir)
	if limite < 1 {
		limite = len(ubicaciones)
	}

	filtradas := make([]modelo.UbicacionGuardada, 0, minimo(limite, len(ubicaciones)))
	for _, ubicacion := range ubicaciones {
		if excluir != "" && ubicacionesCoinciden(ubicacion.Nombre, excluir) {
			continue
		}
		if consulta != "" && !strings.Contains(strings.ToLower(ubicacion.Nombre), consulta) {
			continue
		}
		filtradas = append(filtradas, ubicacion)
		if len(filtradas) >= limite {
			break
		}
	}
	return filtradas
}

func ubicacionesCoinciden(izquierda, derecha string) bool {
	return strings.EqualFold(strings.TrimSpace(izquierda), strings.TrimSpace(derecha))
}

func resumenUbicacion(ciudad, estado, pais string) string {
	partes := []string{
		strings.TrimSpace(ciudad),
		strings.TrimSpace(estado),
		strings.TrimSpace(pais),
	}
	var visibles []string
	for _, parte := range partes {
		if parte != "" {
			visibles = append(visibles, parte)
		}
	}
	return strings.Join(visibles, ", ")
}

func coordenadasTexto(coordenadas *modelo.Coordenadas) string {
	if coordenadas == nil {
		return "Sin coordenadas asociadas"
	}
	return fmt.Sprintf("%.6f, %.6f", coordenadas.Latitud, coordenadas.Longitud)
}

func (a *Aplicacion) dibujarVistaUbicaciones(gtx layout.Context) layout.Dimensions {
	izquierda := maximo(280, gtx.Dp(unit.Dp(300)))
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = izquierda
			gtx.Constraints.Max.X = izquierda
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Ubicaciones guardadas")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorBusquedaLateral(gtx, &a.editorFiltroUbicaciones)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarListaUbicacionesGuardadas(gtx)
						}),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarDetalleUbicacionGuardada(gtx)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarListaUbicacionesGuardadas(gtx layout.Context) layout.Dimensions {
	if len(a.ubicacionesGuardadas) == 0 {
		return a.dibujarTextoSecundario(gtx, "Todavía no hay nombres de ubicación guardados.")
	}

	filtradas := filtrarUbicacionesGuardadas(a.ubicacionesGuardadas, a.editorFiltroUbicaciones.Text(), "", len(a.ubicacionesGuardadas))
	if len(filtradas) == 0 {
		return a.dibujarTextoSecundario(gtx, "Sin coincidencias para el filtro actual.")
	}

	return a.dibujarListaConBarra(gtx, &a.listaUbicaciones, len(filtradas), func(gtx layout.Context, indice int) layout.Dimensions {
		ubicacion := filtradas[indice]
		clave := "vista-ubicacion:" + strings.ToLower(strings.TrimSpace(ubicacion.Nombre))
		resumen := resumenUbicacion(ubicacion.Ciudad, ubicacion.Estado, ubicacion.Pais)
		if resumen == "" {
			resumen = coordenadasTexto(ubicacion.Coordenadas)
		}
		if ubicacion.RelacionadaCon != "" {
			resumen = "Relacionada con " + ubicacion.RelacionadaCon + " | " + resumen
		}

		return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFilaDobleLinea(
				gtx,
				a.asegurarWidgetLateral(clave),
				ubicacion.Nombre,
				fmt.Sprintf("%d usos | %s", ubicacion.CantidadUsos, resumen),
				ubicacionesCoinciden(a.ubicacionSeleccionada, ubicacion.Nombre),
				func() {
					a.seleccionarUbicacionGuardada(ubicacion.Nombre)
				},
			)
		})
	})
}

func (a *Aplicacion) dibujarDetalleUbicacionGuardada(gtx layout.Context) layout.Dimensions {
	ubicacion, ok := a.ubicacionGuardadaSeleccionada()
	if !ok {
		return a.dibujarTextoPrincipal(gtx, "Selecciona un nombre de ubicación para ver sus usos, dirección y relaciones.")
	}

	direccion := serviciometadatos.DireccionGPS{
		Ciudad: ubicacion.Ciudad,
		Estado: ubicacion.Estado,
		Pais:   ubicacion.Pais,
	}
	sugerenciasRelacion := a.sugerenciasRelacionUbicacion()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTituloPanel(gtx, ubicacion.Nombre)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, fmt.Sprintf("Usos detectados: %d", ubicacion.CantidadUsos))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if strings.TrimSpace(ubicacion.RelacionadaCon) == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarTextoSecundario(gtx, "Relacionada con: "+ubicacion.RelacionadaCon)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoSecundario(gtx, "Coordenadas efectivas")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoPrincipal(gtx, coordenadasTexto(ubicacion.Coordenadas))
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarDireccionGPS(gtx, direccion)
						}),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Relación")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampo(gtx, "Relacionar con", &a.editorRelacionUbicacion)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if len(sugerenciasRelacion) == 0 {
								return layout.Dimensions{}
							}
							return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarListaSugerenciasUbicacion(gtx, "relacion-ubicacion", &a.listaRelacionUbicacion, sugerenciasRelacion, func(ubicacion modelo.UbicacionGuardada) {
									a.editorRelacionUbicacion.SetText(ubicacion.Nombre)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return a.dibujarBotonAccion(gtx, &a.botonGuardarRelacionUbicacion, "Guardar relación", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
										a.guardarRelacionUbicacionSeleccionada()
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if strings.TrimSpace(ubicacion.RelacionadaCon) == "" {
										return layout.Dimensions{}
									}
									return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.dibujarBotonAccion(gtx, &a.botonQuitarRelacionUbicacion, "Quitar relación", a.paleta.Panel, a.paleta.Texto, func() {
											a.quitarRelacionUbicacionSeleccionada()
										})
									})
								}),
							)
						}),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Usos detectados")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if a.cargandoUsosUbicacion {
								return a.dibujarTextoSecundario(gtx, "Cargando usos de la ubicación seleccionada...")
							}
							if len(a.usosUbicacionSeleccionada) == 0 {
								return a.dibujarTextoSecundario(gtx, "Este nombre todavía no tiene usos directos con coordenadas o dirección guardadas.")
							}
							return a.dibujarListaConBarra(gtx, &a.listaUsosUbicacion, len(a.usosUbicacionSeleccionada), func(gtx layout.Context, indice int) layout.Dimensions {
								uso := a.usosUbicacionSeleccionada[indice]
								return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return a.dibujarTextoPrincipal(gtx, fmt.Sprintf("%d archivos | %s", uso.CantidadUsos, coordenadasTexto(uso.Coordenadas)))
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													resumen := resumenUbicacion(uso.Ciudad, uso.Estado, uso.Pais)
													if resumen == "" {
														resumen = "Sin dirección nominal asociada"
													}
													return a.dibujarTextoSecundario(gtx, resumen)
												}),
											)
										})
									})
								})
							})
						}),
					)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarCampoUbicacionConSugerencias(gtx layout.Context) layout.Dimensions {
	sugerencias := a.sugerenciasUbicacionFormulario()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarEditorCampoDecorado(gtx, "Nombre de ubicación", &a.editorUbicacion, false, "", "", false)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(sugerencias) == 0 {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarListaSugerenciasUbicacion(gtx, "formulario-ubicacion", &a.formularioMetadatos.listaUbicacionesSugeridas, sugerencias, func(ubicacion modelo.UbicacionGuardada) {
					a.aplicarUbicacionGuardadaAlFormulario(ubicacion)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarListaSugerenciasUbicacion(gtx layout.Context, prefijo string, lista *widget.List, sugerencias []modelo.UbicacionGuardada, alSeleccionar func(modelo.UbicacionGuardada)) layout.Dimensions {
	if len(sugerencias) == 0 {
		return layout.Dimensions{}
	}

	return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarListaConBarra(gtx, lista, len(sugerencias), func(gtx layout.Context, indice int) layout.Dimensions {
				ubicacion := sugerencias[indice]
				clave := prefijo + ":" + strings.ToLower(strings.TrimSpace(ubicacion.Nombre))
				resumen := resumenUbicacion(ubicacion.Ciudad, ubicacion.Estado, ubicacion.Pais)
				if resumen == "" {
					resumen = coordenadasTexto(ubicacion.Coordenadas)
				}
				return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarFilaDobleLinea(gtx, a.asegurarWidgetLateral(clave), ubicacion.Nombre, resumen, false, func() {
						if alSeleccionar != nil {
							alSeleccionar(ubicacion)
						}
					})
				})
			})
		})
	})
}

func (a *Aplicacion) dibujarFilaDobleLinea(gtx layout.Context, clic *widget.Clickable, titulo, subtitulo string, activo bool, alHacer func()) layout.Dimensions {
	fondo := a.paleta.PanelElevado
	colorTitulo := a.paleta.Texto
	colorSubtitulo := a.paleta.TextoSuave
	if activo {
		fondo = a.paleta.Acento
		colorTitulo = a.paleta.TextoSobreAcento
		colorSubtitulo = a.paleta.TextoSobreAcento
	}

	pulsado := false
	for clic.Clicked(gtx) {
		pulsado = true
	}

	dim := dibujarPanel(gtx, fondo, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return clic.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						estilo := material.Label(a.tema, unit.Sp(13), titulo)
						estilo.Color = colorTitulo
						return estilo.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						estilo := material.Label(a.tema, unit.Sp(11), subtitulo)
						estilo.Color = colorSubtitulo
						return estilo.Layout(gtx)
					}),
				)
			})
		})
	})

	if pulsado {
		if alHacer != nil {
			alHacer()
		}
		if a.ventana != nil {
			a.ventana.Invalidate()
		}
	}
	return dim
}
