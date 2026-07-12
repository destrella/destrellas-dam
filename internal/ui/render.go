package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"destrellas-dam/internal/component"
	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/yandex"
)

type grupoElementosUI struct {
	RutaPadre string
	Elementos []modelo.Archivo
}

type entradaListaAgrupada struct {
	Separador string
	Archivo   modelo.Archivo
}

type filaGaleriaAgrupada struct {
	Separador string
	Elementos []modelo.Archivo
}

func (a *Aplicacion) dibujar(gtx layout.Context) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			paint.Fill(gtx.Ops, a.paleta.Fondo)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBarraSuperior(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						switch a.vistaActual {
						case vistaElementoUnico:
							return a.dibujarVistaElementoUnico(gtx)
						case vistaDuplicados:
							return a.dibujarVistaDuplicados(gtx)
						case vistaUbicaciones:
							return a.dibujarVistaUbicaciones(gtx)
						case vistaConfiguracion:
							return a.dibujarVistaConfiguracion(gtx)
						default:
							return a.dibujarVistaPrincipal(gtx)
						}
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBarraEstado(gtx)
					}),
				)
			})
		},
	)
}

func (a *Aplicacion) dibujarBarraSuperior(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonNavegacion(gtx, &a.botonVistaPrincipal, "Explorador", a.vistaActual == vistaPrincipal, func() {
								a.vistaActual = vistaPrincipal
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonNavegacion(gtx, &a.botonVistaElementoUnico, "Visor", a.vistaActual == vistaElementoUnico, func() {
								if a.tieneArchivoActivo {
									a.vistaActual = vistaElementoUnico
								}
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonNavegacion(gtx, &a.botonVistaDuplicados, "Duplicados", a.vistaActual == vistaDuplicados, func() {
								a.vistaActual = vistaDuplicados
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonNavegacion(gtx, &a.botonVistaUbicaciones, "Ubicaciones", a.vistaActual == vistaUbicaciones, func() {
								a.vistaActual = vistaUbicaciones
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonNavegacion(gtx, &a.botonVistaConfiguracion, "Configuración", a.vistaActual == vistaConfiguracion, func() {
								a.vistaActual = vistaConfiguracion
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, fmt.Sprintf("Raíz: %s", a.configuracion.CarpetaInicial))
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarBarraEstado(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			mensaje := a.mensajeEstado
			if mensaje == "" {
				mensaje = "Listo."
			}
			if a.ultimoError != "" {
				mensaje += " | Error: " + a.ultimoError
			}
			estilo := material.Label(a.tema, unit.Sp(13), mensaje)
			estilo.Color = a.paleta.TextoSuave
			return estilo.Layout(gtx)
		})
	})
}

func (a *Aplicacion) dibujarVistaPrincipal(gtx layout.Context) layout.Dimensions {
	izquierda := maximo(240, gtx.Dp(unit.Dp(250)))
	derecha := maximo(320, gtx.Dp(unit.Dp(350)))

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = izquierda
			gtx.Constraints.Max.X = izquierda
			return a.dibujarColumnaLateral(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarColumnaCentral(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = derecha
			gtx.Constraints.Max.X = derecha
			return a.dibujarColumnaDetalle(gtx)
		}),
	)
}

func (a *Aplicacion) dibujarVistaElementoUnico(gtx layout.Context) layout.Dimensions {
	dim := layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if !a.tieneArchivoActivo {
						return a.dibujarTextoPrincipal(gtx, "Selecciona un elemento para abrirlo en el visor.")
					}
					return a.dibujarPreviewGrande(gtx, a.archivoActivo)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ancho := maximo(320, gtx.Dp(unit.Dp(360)))
			gtx.Constraints.Min.X = ancho
			gtx.Constraints.Max.X = ancho
			return a.dibujarColumnaDetalle(gtx)
		}),
	)
	a.procesarAtajosVisor(gtx)
	return dim
}

func (a *Aplicacion) dibujarVistaDuplicados(gtx layout.Context) layout.Dimensions {
	izquierda := maximo(250, gtx.Dp(unit.Dp(260)))
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = izquierda
			gtx.Constraints.Max.X = izquierda
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Descubrimiento")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoSecundario(gtx, fmt.Sprintf("Directorios: %d | Archivos: %d | Analizados: %d", a.progresoDuplicados.DirectoriosProcesados, a.progresoDuplicados.ArchivosEncontrados, a.progresoDuplicados.ArchivosAnalizados))
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampo(gtx, "Ruta a escanear", &a.editorRutaEscaneoDuplicados)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBarraProgreso(gtx, a.progresoDuplicados.Porcentaje)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonEscanearLocal, "Escanear locales", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
								a.iniciarDescubrimientoLocal()
							})
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonEscanearRemoto, "Escanear remotos", a.paleta.PanelElevado, a.paleta.Texto, func() {
								a.establecerEstado("La integración remota está encapsulada, pero esta primera base no la completa todavía", yandex.ErrNoImplementado)
							})
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Categorías")
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonCategoriaDuplicado(gtx, &a.botonDuplicadosLocales, "Locales", a.categoriaDuplicados == modelo.CategoriaDuplicadosLocales, modelo.CategoriaDuplicadosLocales)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonCategoriaDuplicado(gtx, &a.botonDuplicadosRemotos, "Remotos", a.categoriaDuplicados == modelo.CategoriaDuplicadosRemotos, modelo.CategoriaDuplicadosRemotos)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonCategoriaDuplicado(gtx, &a.botonDuplicadosMixtos, "Mixtos", a.categoriaDuplicados == modelo.CategoriaDuplicadosMixtos, modelo.CategoriaDuplicadosMixtos)
						}),
					)
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarControlesDuplicados(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarListaDuplicados(gtx)
						}),
					)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarVistaConfiguracion(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarListaConBarra(gtx, &a.listaConfiguracion, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTituloPanel(gtx, "Configuración")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarEditorCampo(gtx, "Carpeta inicial", &a.editorCarpetaInicial)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarEditorCampo(gtx, "Carpeta de archivado", &a.editorCarpetaArchivado)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarEditorCampo(gtx, "Clave API de Yandex.Disk", &a.editorClaveYandex)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTituloPanel(gtx, "Filtros por defecto")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarLineaChecks(gtx,
							&a.configMostrarOcultos, "Ver ocultos",
							&a.configOcultarCarpetas, "Ocultar carpetas",
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarLineaChecks(gtx,
							&a.configSoloMultimedia, "Solo multimedia",
							&a.configSoloVideos, "Solo videos",
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarLineaChecks(gtx,
							&a.configSoloImagenes, "Solo imágenes",
							&a.configSoloAudio, "Solo audio",
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(a.tema, &a.configRecursivo, "Recursivo").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, "Orden por defecto")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						controlesOrden := []layout.Widget{
							func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, &a.botonConfigOrdenAZ, "A-Z", a.criterioOrdenConfiguracion() == modelo.CriterioOrdenNombre && !a.configOrdenDescendente.Value, func() {
									a.establecerOrdenConfiguracion(modelo.CriterioOrdenNombre, false)
								})
							},
							func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, &a.botonConfigOrdenZA, "Z-A", a.criterioOrdenConfiguracion() == modelo.CriterioOrdenNombre && a.configOrdenDescendente.Value, func() {
									a.establecerOrdenConfiguracion(modelo.CriterioOrdenNombre, true)
								})
							},
							func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, &a.botonConfigOrdenAntiguos, "Más antiguos", a.criterioOrdenConfiguracion() == modelo.CriterioOrdenFechaModificacion && !a.configOrdenDescendente.Value, func() {
									a.establecerOrdenConfiguracion(modelo.CriterioOrdenFechaModificacion, false)
								})
							},
							func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, &a.botonConfigOrdenNuevos, "Más nuevos", a.criterioOrdenConfiguracion() == modelo.CriterioOrdenFechaModificacion && a.configOrdenDescendente.Value, func() {
									a.establecerOrdenConfiguracion(modelo.CriterioOrdenFechaModificacion, true)
								})
							},
						}
						return a.dibujarFlujoControles(gtx, controlesOrden)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTituloPanel(gtx, "Escaneo de metadatos")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, fmt.Sprintf("Directorios: %d | Archivos: %d | Analizados: %d", a.progresoMetadatos.DirectoriosProcesados, a.progresoMetadatos.ArchivosEncontrados, a.progresoMetadatos.ArchivosAnalizados))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if strings.TrimSpace(a.progresoMetadatos.RutaActual) == "" {
							return layout.Dimensions{}
						}
						return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoSecundario(gtx, "Actual: "+a.progresoMetadatos.RutaActual)
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarEditorCampo(gtx, "Ruta a escanear", &a.editorRutaEscaneoMetadatos)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBarraProgreso(gtx, a.progresoMetadatos.Porcentaje)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						etiquetaBoton := "Escanear metadatos locales"
						fondo := a.paleta.Acento
						colorTexto := a.paleta.TextoSobreAcento
						if a.escanandoMetadatos {
							etiquetaBoton = "Reiniciar escaneo de metadatos"
							fondo = a.paleta.PanelElevado
							colorTexto = a.paleta.Texto
						}
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonAccion(gtx, &a.botonEscanearMetadatos, etiquetaBoton, fondo, colorTexto, func() {
									a.iniciarEscaneoMetadatos()
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !a.escanandoMetadatos {
									return layout.Dimensions{}
								}
								return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return a.dibujarBotonAccion(gtx, &a.botonPausarEscaneo, "Pausar escaneo de metadatos", a.paleta.Peligro, a.paleta.Texto, func() {
										a.pausarEscaneoMetadatos()
									})
								})
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBotonAccion(gtx, &a.botonGuardarConfig, "Guardar configuración", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
							a.guardarConfiguracion()
						})
					}),
				)
			})
		})
	})
}

func (a *Aplicacion) dibujarColumnaLateral(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarPestanasLateral(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					switch a.pestanaLateral {
					case pestanaPalabras:
						return a.dibujarListaOpcionesLaterales(gtx, "Tags", &a.editorFiltroEtiquetas, a.etiquetas, origenListadoEtiqueta, func(opcion opcionFiltroLateral) {
							a.seleccionarEtiqueta(opcion.Clave)
						})
					case pestanaLugares:
						return a.dibujarListaOpcionesLaterales(gtx, "Lugares", &a.editorFiltroLugares, a.ubicacionesNombradas, origenListadoUbicacion, func(opcion opcionFiltroLateral) {
							if opcion.Clave == etiquetaUbicacionSinNombre {
								a.seleccionarUbicacionSinNombre()
								return
							}
							a.seleccionarUbicacion(opcion.Clave)
						})
					case pestanaYandex:
						return a.dibujarVistaYandex(gtx)
					default:
						return a.dibujarArbolDirectorios(gtx)
					}
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarColumnaCentral(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarControlesFiltros(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(a.seleccionLote) == 0 {
						return layout.Dimensions{}
					}
					return a.dibujarBarraLote(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(a.seleccionLote) == 0 {
						return layout.Dimensions{}
					}
					return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if a.filtros.VistaGaleria {
						return a.dibujarGaleria(gtx)
					}
					return a.dibujarListaElementos(gtx)
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarColumnaDetalle(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(18), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if !a.tieneArchivoActivo {
				return a.dibujarTextoPrincipal(gtx, "Selecciona un archivo para ver su detalle, previsualización y acciones.")
			}
			if a.vistaActual == vistaPrincipal {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBloqueResumenExplorador(gtx, a.archivoActivo)
					}),
					layout.Rigid(a.paddingSeparadorDetalle()),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return a.dibujarListaConBarra(gtx, &a.listaDetalle, 1, func(gtx layout.Context, _ int) layout.Dimensions {
							return a.dibujarBloquesDetalle(gtx)
						})
					}),
				)
			}
			return a.dibujarListaConBarra(gtx, &a.listaDetalle, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				return a.dibujarBloquesDetalle(gtx)
			})
		})
	})
}

func (a *Aplicacion) dibujarBloquesDetalle(gtx layout.Context) layout.Dimensions {
	hijos := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarAccionesArchivo(gtx)
		}),
	}

	if a.archivoActivo.EsMultimedia() {
		hijos = append(hijos,
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarFormularioMetadatos(gtx)
			}),
		)

		if analizarBloqueMetadatosIA(a.archivoActivo, a.paleta).TieneContenido() {
			hijos = append(hijos,
				layout.Rigid(a.paddingSeparadorDetalle()),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBloqueMetadatosIA(gtx, a.archivoActivo)
				}),
			)
		}

		hijos = append(hijos,
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueExiftoolCrudo(gtx)
			}),
		)
	}

	switch {
	case a.vistaActual == vistaPrincipal && a.archivoActivo.Tipo == modelo.TipoImagen:
		hijos = append(hijos,
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueAccionesImagen(gtx)
			}),
		)
	case a.vistaActual == vistaElementoUnico && a.archivoActivo.Tipo == modelo.TipoImagen:
		hijos = append(hijos,
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueEtiquetarRegiones(gtx)
			}),
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueRecorteImagen(gtx)
			}),
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueAccionesImagen(gtx)
			}),
		)
	case a.vistaActual == vistaElementoUnico && a.archivoActivo.Tipo == modelo.TipoVideo:
		hijos = append(hijos,
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueExtraerFrame(gtx)
			}),
			layout.Rigid(a.paddingSeparadorDetalle()),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueOptimizarVideo(gtx)
			}),
		)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
}

func (a *Aplicacion) dibujarPestanasLateral(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacionIcono(gtx, &a.botonPestanaDirectorios, a.pestanaLateral == pestanaDirectorios, func() {
				a.pestanaLateral = pestanaDirectorios
			}, a.dibujarIconoPestanaDirectorios)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacionIcono(gtx, &a.botonPestanaPalabras, a.pestanaLateral == pestanaPalabras, func() {
				a.pestanaLateral = pestanaPalabras
			}, a.dibujarIconoPestanaEtiqueta)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacionIcono(gtx, &a.botonPestanaLugares, a.pestanaLateral == pestanaLugares, func() {
				a.pestanaLateral = pestanaLugares
			}, a.dibujarIconoPestanaLugar)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacionIcono(gtx, &a.botonPestanaYandex, a.pestanaLateral == pestanaYandex, func() {
				a.pestanaLateral = pestanaYandex
			}, a.dibujarIconoPestanaYandex)
		}),
	)
}

func (a *Aplicacion) dibujarControlesFiltros(gtx layout.Context) layout.Dimensions {
	controlesFiltro := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.mostrarOcultos, "Ocultos").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.ocultarCarpetas, "Ocultar carpetas").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.soloMultimedia, "Solo multimedia").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.soloVideos, "Solo videos").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.soloImagenes, "Solo imágenes").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.soloAudio, "Solo audio").Layout(gtx)
		},
		func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, &a.recursivo, "Recursivo").Layout(gtx)
		},
	}

	controlesVista := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonGaleria, "Galería", a.filtros.VistaGaleria, func() {
				a.cambiarVistaCentral(true)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonLista, "Lista", !a.filtros.VistaGaleria, func() {
				a.cambiarVistaCentral(false)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonOrdenAZ, "A-Z", a.filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenNombre && !a.filtros.OrdenDescendente, func() {
				a.cambiarOrdenListado(modelo.CriterioOrdenNombre, false)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonOrdenZA, "Z-A", a.filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenNombre && a.filtros.OrdenDescendente, func() {
				a.cambiarOrdenListado(modelo.CriterioOrdenNombre, true)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonOrdenAntiguos, "Más antiguos", a.filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion && !a.filtros.OrdenDescendente, func() {
				a.cambiarOrdenListado(modelo.CriterioOrdenFechaModificacion, false)
			})
		},
		func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonNavegacion(gtx, &a.botonOrdenNuevos, "Más nuevos", a.filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion && a.filtros.OrdenDescendente, func() {
				a.cambiarOrdenListado(modelo.CriterioOrdenFechaModificacion, true)
			})
		},
	}

	dimensiones := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTituloPanel(gtx, a.tituloListadoActual())
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFlujoControles(gtx, controlesFiltro)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFlujoControles(gtx, controlesVista)
		}),
	)
	a.alternarFiltrosDesdeUI()
	return dimensiones
}

func (a *Aplicacion) dibujarFlujoControles(gtx layout.Context, controles []layout.Widget) layout.Dimensions {
	flujo := component.Flow{
		Axis:      layout.Horizontal,
		Alignment: layout.Start,
	}
	return flujo.Layout(gtx, len(controles), func(gtx layout.Context, indice int) layout.Dimensions {
		return layout.Inset{
			Right:  unit.Dp(10),
			Bottom: unit.Dp(6),
		}.Layout(gtx, controles[indice])
	})
}

func (a *Aplicacion) dibujarListaConBarra(gtx layout.Context, lista *widget.List, cantidad int, elemento layout.ListElement) layout.Dimensions {
	estilo := material.List(a.tema, lista)
	estilo.AnchorStrategy = material.Overlay
	estilo.Indicator.Color = a.paleta.Acento
	estilo.Indicator.HoverColor = a.paleta.Texto
	return estilo.Layout(gtx, cantidad, elemento)
}

func (a *Aplicacion) dibujarArbolDirectorios(gtx layout.Context) layout.Dimensions {
	visibles := a.aplanarArbol()
	return a.dibujarListaConBarra(gtx, &a.listaLateral, len(visibles), func(gtx layout.Context, indice int) layout.Dimensions {
		nodo := visibles[indice]
		return layout.Inset{Left: unit.Dp(float32(nodo.Nivel) * 14), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonIconoCarpeta(gtx, &nodo.Nodo.Alternar, nodo.Nodo.Expandido, func() {
						if !nodo.Nodo.Cargado {
							a.asegurarHijosNodo(nodo.Nodo)
						}
						nodo.Nodo.Expandido = !nodo.Nodo.Expandido
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					activo := a.carpetaSeleccionada == nodo.Nodo.Ruta
					return a.dibujarFilaArbol(gtx, &nodo.Nodo.Seleccionar, nodo.Nodo.Nombre, activo, func() {
						a.seleccionarNodoArbol(nodo.Nodo)
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarVistaYandex(gtx layout.Context) layout.Dimensions {
	if !a.clienteYandex.Configurado() {
		return a.dibujarTextoPrincipal(gtx, "Configura una clave API para habilitar la pestaña remota.")
	}
	return a.dibujarTextoPrincipal(gtx, "La arquitectura remota ya está encapsulada. Falta completar el cliente real de Yandex.Disk sobre este contrato.")
}

func (a *Aplicacion) dibujarListaOpcionesLaterales(gtx layout.Context, titulo string, editorFiltro *widget.Editor, elementos []opcionFiltroLateral, origen tipoOrigenListado, alSeleccionar func(opcionFiltroLateral)) layout.Dimensions {
	filtrados := elementos
	if editorFiltro != nil {
		filtrados = filtrarOpcionesLaterales(elementos, editorFiltro.Text())
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTituloPanel(gtx, titulo)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if editorFiltro == nil {
				return layout.Dimensions{}
			}
			return a.dibujarEditorBusquedaLateral(gtx, editorFiltro)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(elementos) == 0 {
				return a.dibujarTextoSecundario(gtx, "Sin elementos indexados todavía.")
			}
			if len(filtrados) == 0 {
				return a.dibujarTextoSecundario(gtx, "Sin coincidencias para el filtro actual.")
			}
			return a.dibujarListaConBarra(gtx, &a.listaLateral, len(filtrados), func(gtx layout.Context, indice int) layout.Dimensions {
				opcion := filtrados[indice]
				origenOpcion := origen
				if opcion.Clave == etiquetaUbicacionSinNombre {
					origenOpcion = origenListadoUbicacionSinNombre
				}
				claveWidget := string(origenOpcion) + ":" + opcion.Clave
				return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarFilaArbol(gtx, a.asegurarWidgetLateral(claveWidget), opcion.Etiqueta, a.esFiltroLateralActivo(origenOpcion, opcion.Clave), func() {
						if alSeleccionar != nil {
							alSeleccionar(opcion)
						}
					})
				})
			})
		}),
	)
}

func filtrarOpcionesLaterales(elementos []opcionFiltroLateral, consulta string) []opcionFiltroLateral {
	consulta = strings.TrimSpace(strings.ToLower(consulta))
	if consulta == "" {
		return elementos
	}

	filtrados := make([]opcionFiltroLateral, 0, len(elementos))
	for _, elemento := range elementos {
		if strings.Contains(strings.ToLower(elemento.Etiqueta), consulta) {
			filtrados = append(filtrados, elemento)
		}
	}
	return filtrados
}

func (a *Aplicacion) dibujarBarraLote(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoPrincipal(gtx, fmt.Sprintf("%d elementos seleccionados", len(a.rutasSeleccionadas())))
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampo(gtx, "Carpeta destino", &a.editorDestinoLote)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonMoverLote, "Mover selección", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
								a.moverSeleccionLote()
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonArchivarLote, "Archivar", a.paleta.Exito, a.paleta.Texto, func() {
								a.archivarSeleccionLote()
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonPapeleraLote, "Papelera", a.paleta.Peligro, a.paleta.Texto, func() {
								a.enviarSeleccionLoteAPapelera()
							})
						}),
					)
				}),
			)
		})
	})
}

func (a *Aplicacion) mostrarAgrupacionRecursivaPorCarpeta() bool {
	return a.origenListado == origenListadoCarpeta && a.filtros.Recursivo
}

func (a *Aplicacion) agruparElementosPorCarpeta() []grupoElementosUI {
	if len(a.elementos) == 0 {
		return nil
	}

	var grupos []grupoElementosUI
	for _, elemento := range a.elementos {
		if len(grupos) == 0 || grupos[len(grupos)-1].RutaPadre != elemento.RutaPadre {
			grupos = append(grupos, grupoElementosUI{RutaPadre: elemento.RutaPadre})
		}
		grupos[len(grupos)-1].Elementos = append(grupos[len(grupos)-1].Elementos, elemento)
	}
	return grupos
}

func (a *Aplicacion) construirEntradasListaAgrupada() []entradaListaAgrupada {
	grupos := a.agruparElementosPorCarpeta()
	entradas := make([]entradaListaAgrupada, 0, len(a.elementos)+len(grupos))
	for _, grupo := range grupos {
		entradas = append(entradas, entradaListaAgrupada{
			Separador: a.etiquetaGrupoCarpeta(grupo.RutaPadre),
		})
		for _, elemento := range grupo.Elementos {
			entradas = append(entradas, entradaListaAgrupada{Archivo: elemento})
		}
	}
	return entradas
}

func (a *Aplicacion) construirFilasGaleriaAgrupada(columnas int) []filaGaleriaAgrupada {
	if columnas < 1 {
		columnas = 1
	}

	grupos := a.agruparElementosPorCarpeta()
	filas := make([]filaGaleriaAgrupada, 0, len(grupos))
	for _, grupo := range grupos {
		filas = append(filas, filaGaleriaAgrupada{
			Separador: a.etiquetaGrupoCarpeta(grupo.RutaPadre),
		})
		for inicio := 0; inicio < len(grupo.Elementos); inicio += columnas {
			fin := minimo(inicio+columnas, len(grupo.Elementos))
			filas = append(filas, filaGaleriaAgrupada{
				Elementos: grupo.Elementos[inicio:fin],
			})
		}
	}
	return filas
}

func (a *Aplicacion) etiquetaGrupoCarpeta(rutaPadre string) string {
	if rutaPadre == "" {
		return ""
	}
	if a.carpetaSeleccionada != "" {
		relativa, err := filepath.Rel(a.carpetaSeleccionada, rutaPadre)
		if err == nil {
			relativa = filepath.Clean(relativa)
			if relativa == "." {
				nombre := filepath.Base(a.carpetaSeleccionada)
				if nombre != "" && nombre != "." && nombre != string(filepath.Separator) {
					return nombre
				}
				return a.carpetaSeleccionada
			}
			return relativa
		}
	}
	return rutaPadre
}

func (a *Aplicacion) dibujarSeparadorGrupoCarpeta(gtx layout.Context, titulo string) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, titulo)
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarListaElementos(gtx layout.Context) layout.Dimensions {
	if len(a.elementos) == 0 {
		if a.cargandoElementos {
			return a.dibujarTextoPrincipal(gtx, "Cargando elementos...")
		}
		return a.dibujarTextoPrincipal(gtx, "Sin resultados para los filtros actuales.")
	}

	elementos := a.elementos

	if a.mostrarAgrupacionRecursivaPorCarpeta() {
		return a.dibujarListaElementosAgrupados(gtx)
	}

	return a.dibujarListaConBarra(gtx, &a.listaCentro, len(elementos), func(gtx layout.Context, indice int) layout.Dimensions {
		if indice >= len(elementos)-2 {
			a.cargarMasElementos()
		}
		return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFilaElemento(gtx, elementos[indice])
		})
	})
}

func (a *Aplicacion) dibujarGaleria(gtx layout.Context) layout.Dimensions {
	if len(a.elementos) == 0 {
		if a.cargandoElementos {
			return a.dibujarTextoPrincipal(gtx, "Cargando galería...")
		}
		return a.dibujarTextoPrincipal(gtx, "Sin resultados para la galería.")
	}

	elementos := a.elementos
	columnas, anchoTarjeta, separacion := a.parametrosGaleria(gtx)

	if a.mostrarAgrupacionRecursivaPorCarpeta() {
		return a.dibujarGaleriaAgrupada(gtx, columnas, anchoTarjeta, separacion)
	}

	filas := int(math.Ceil(float64(len(elementos)) / float64(columnas)))

	return a.dibujarListaConBarra(gtx, &a.listaCentro, filas, func(gtx layout.Context, fila int) layout.Dimensions {
		if fila >= filas-1 {
			a.cargarMasElementos()
		}

		inicio := fila * columnas
		fin := minimo(inicio+columnas, len(elementos))
		return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarFilaGaleriaConAncho(gtx, elementos[inicio:fin], anchoTarjeta)
		})
	})
}

func (a *Aplicacion) dibujarListaElementosAgrupados(gtx layout.Context) layout.Dimensions {
	entradas := a.construirEntradasListaAgrupada()
	return a.dibujarListaConBarra(gtx, &a.listaCentro, len(entradas), func(gtx layout.Context, indice int) layout.Dimensions {
		if indice >= len(entradas)-2 {
			a.cargarMasElementos()
		}

		entrada := entradas[indice]
		return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if entrada.Separador != "" {
				return a.dibujarSeparadorGrupoCarpeta(gtx, entrada.Separador)
			}
			return a.dibujarFilaElemento(gtx, entrada.Archivo)
		})
	})
}

func (a *Aplicacion) dibujarGaleriaAgrupada(gtx layout.Context, columnas, anchoTarjeta, separacion int) layout.Dimensions {
	filas := a.construirFilasGaleriaAgrupada(columnas)
	return a.dibujarListaConBarra(gtx, &a.listaCentro, len(filas), func(gtx layout.Context, indice int) layout.Dimensions {
		if indice >= len(filas)-2 {
			a.cargarMasElementos()
		}

		fila := filas[indice]
		return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if fila.Separador != "" {
				return a.dibujarSeparadorGrupoCarpeta(gtx, fila.Separador)
			}
			return a.dibujarFilaGaleriaConAncho(gtx, fila.Elementos, anchoTarjeta)
		})
	})
}

func (a *Aplicacion) parametrosGaleria(gtx layout.Context) (columnas, anchoTarjeta, separacion int) {
	separacion = gtx.Dp(unit.Dp(8))
	anchoMinimo := maximo(220, gtx.Dp(unit.Dp(220)))
	anchoMaximo := maximo(anchoMinimo, gtx.Dp(unit.Dp(512)))
	disponible := gtx.Constraints.Max.X
	if disponible <= 0 {
		return 1, anchoMinimo, separacion
	}

	columnas = maximo(1, (disponible+separacion)/(anchoMinimo+separacion))
	anchoTarjeta = (disponible - separacion*(columnas-1)) / columnas
	if anchoTarjeta > anchoMaximo {
		anchoTarjeta = anchoMaximo
	}
	if anchoTarjeta < anchoMinimo {
		anchoTarjeta = anchoMinimo
	}

	return columnas, anchoTarjeta, separacion
}

func (a *Aplicacion) dibujarFilaGaleriaConAncho(gtx layout.Context, elementos []modelo.Archivo, anchoTarjeta int) layout.Dimensions {
	if len(elementos) == 0 {
		return layout.Dimensions{}
	}

	var hijos []layout.FlexChild
	for indice, archivo := range elementos {
		archivo := archivo
		if indice > 0 {
			hijos = append(hijos, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
		}
		hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = anchoTarjeta
			gtx.Constraints.Max.X = anchoTarjeta
			return a.dibujarTarjetaElemento(gtx, archivo)
		}))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, hijos...)
}

func (a *Aplicacion) dibujarFilaElemento(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	widgets := a.asegurarWidgetsElemento(archivo.Ruta)
	widgets.Seleccion.Value = a.seleccionLote[archivo.Ruta]
	clicks := 0
	for {
		click, ok := widgets.Fila.Update(gtx)
		if !ok {
			break
		}
		clicks = click.NumClicks
	}

	fondo := a.paleta.PanelElevado
	if a.esElementoActivo(archivo) {
		fondo = a.paleta.AcentoSuave
	}

	dim := dibujarPanel(gtx, fondo, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
		return widgets.Fila.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.CheckBox(a.tema, &widgets.Seleccion, "").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoPrincipal(gtx, archivo.NombreVisible())
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, a.resumenArchivo(archivo))
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarIndicadores(gtx, archivo)
					}),
				)
			})
		})
	})
	if widgets.Seleccion.Value {
		a.seleccionLote[archivo.Ruta] = true
	} else {
		delete(a.seleccionLote, archivo.Ruta)
	}
	if clicks > 0 {
		a.manejarClickElementoExplorador(archivo, clicks)
	}
	return dim
}

func (a *Aplicacion) dibujarTarjetaElemento(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	widgets := a.asegurarWidgetsElemento(archivo.Ruta)
	widgets.Seleccion.Value = a.seleccionLote[archivo.Ruta]
	clicks := 0
	for {
		click, ok := widgets.Fila.Update(gtx)
		if !ok {
			break
		}
		clicks = click.NumClicks
	}

	fondo := a.paleta.PanelElevado
	if a.esElementoActivo(archivo) {
		fondo = a.paleta.AcentoSuave
	}

	dim := dibujarPanel(gtx, fondo, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
		return widgets.Fila.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.CheckBox(a.tema, &widgets.Seleccion, "").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoPrincipalTruncado(gtx, archivo.NombreVisible())
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarMiniaturaElemento(gtx, archivo)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, a.resumenArchivo(archivo))
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarIndicadores(gtx, archivo)
					}),
				)
			})
		})
	})
	if widgets.Seleccion.Value {
		a.seleccionLote[archivo.Ruta] = true
	} else {
		delete(a.seleccionLote, archivo.Ruta)
	}
	if clicks > 0 {
		a.manejarClickElementoExplorador(archivo, clicks)
	}
	return dim
}

func (a *Aplicacion) manejarClickElementoExplorador(archivo modelo.Archivo, clicks int) {
	if clicks < 1 {
		return
	}
	if archivo.EsDirectorio {
		a.manejarActivacionElemento(archivo, false)
		return
	}
	a.manejarActivacionElemento(archivo, clicks >= 2)
}

func (a *Aplicacion) dibujarMiniaturaElemento(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	const alto = 140
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(alto))
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(alto))

	if archivo.Tipo == modelo.TipoDirectorio {
		return a.dibujarPreviewCarpeta(gtx, false)
	}
	if archivo.Tipo == modelo.TipoDesconocido {
		return a.dibujarPreviewInterrogacion(gtx)
	}

	if archivo.Tipo == modelo.TipoImagen || archivo.Tipo == modelo.TipoVideo {
		a.solicitarPreview(archivo, 360)
		if preview, existe := a.previews[archivo.Ruta]; existe && preview != nil && preview.Imagen != nil {
			return a.dibujarImagenConRegiones(gtx, archivo, preview.Imagen, false)
		}
	}

	return dibujarPanel(gtx, a.paleta.AcentoSuave, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoPrincipal(gtx, strings.ToUpper(string(archivo.Tipo)))
		})
	})
}

func (a *Aplicacion) dibujarPreviewLateral(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(220))
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(220))
	return a.dibujarPreviewComun(gtx, archivo, 960, false)
}

func (a *Aplicacion) dibujarPreviewGrande(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	maximoPreview := maximo(2_048, maximo(gtx.Constraints.Max.X, gtx.Constraints.Max.Y))
	if maximoPreview > 4_096 {
		maximoPreview = 4_096
	}

	if archivo.Tipo != modelo.TipoVideo {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarContenedorNavegacionVisor(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarPreviewComun(gtx, archivo, maximoPreview, true)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBarraAccionesVisor(gtx, archivo)
			}),
		)
	}

	maximoFotograma := minimo(maximoPreview, 1_600)
	a.actualizarReproductorVideo(gtx, archivo, maximoFotograma)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarContenedorNavegacionVisor(gtx, func(gtx layout.Context) layout.Dimensions {
				if a.reproductorVideo.Fotograma != nil {
					return a.dibujarImagenConRegiones(gtx, archivo, a.reproductorVideo.Fotograma, false)
				}
				return a.dibujarPreviewComun(gtx, archivo, maximoFotograma, false)
			})
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBarraAccionesVisor(gtx, archivo)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarControlesReproductorVideo(gtx, archivo, maximoFotograma)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarPreviewComun(gtx layout.Context, archivo modelo.Archivo, maximoPreview int, interactiva bool) layout.Dimensions {
	if archivo.Tipo == modelo.TipoDirectorio {
		return a.dibujarPreviewCarpeta(gtx, false)
	}
	if archivo.Tipo == modelo.TipoDesconocido {
		return a.dibujarPreviewInterrogacion(gtx)
	}
	if archivo.Tipo == modelo.TipoImagen || archivo.Tipo == modelo.TipoVideo {
		a.solicitarPreview(archivo, maximoPreview)
		if preview, existe := a.previews[archivo.Ruta]; existe && preview != nil && preview.Imagen != nil {
			return a.dibujarImagenConRegiones(gtx, archivo, preview.Imagen, interactiva)
		}
	}
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoPrincipal(gtx, "Previsualización no disponible")
		})
	})
}

func (a *Aplicacion) dibujarPreviewCarpeta(gtx layout.Context, abierta bool) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.AcentoSuave, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarIconoCarpeta(gtx, abierta, image.Pt(96, 72))
		})
	})
}

func (a *Aplicacion) dibujarPreviewInterrogacion(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.AcentoSuave, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			estilo := material.Label(a.tema, unit.Sp(46), "?")
			estilo.Color = a.paleta.Texto
			return estilo.Layout(gtx)
		})
	})
}

func (a *Aplicacion) dibujarControlesReproductorVideo(gtx layout.Context, archivo modelo.Archivo, maximoFotograma int) layout.Dimensions {
	valorAntes := a.controlProgresoVideo.Value
	etiquetaReproduccion := "Reproducir"
	if a.reproductorVideo.Reproduciendo {
		etiquetaReproduccion = "Pausar"
	}

	dim := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonAlternarVideo, etiquetaReproduccion, a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
						a.alternarReproductorVideo()
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonReiniciarVideo, "Inicio", a.paleta.PanelElevado, a.paleta.Texto, func() {
						a.reiniciarReproductorVideo()
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							deslizador := material.Slider(a.tema, &a.controlProgresoVideo)
							deslizador.Axis = layout.Horizontal
							deslizador.Color = a.paleta.Acento
							return deslizador.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoSecundario(gtx, fmt.Sprintf("%s / %s", formatearDuracion(a.reproductorVideo.Posicion), formatearDuracion(a.reproductorVideo.Duracion)))
						}),
					)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if strings.TrimSpace(a.reproductorVideo.Error) == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarTextoSecundario(gtx, "Video: "+a.reproductorVideo.Error)
			})
		}),
	)

	if valorAntes != a.controlProgresoVideo.Value {
		a.actualizarPosicionVideoDesdeControl(maximoFotograma)
	}
	if a.reproductorVideo.Fotograma == nil && !a.reproductorVideo.Cargando {
		a.solicitarFotogramaVideo(archivo, a.reproductorVideo.Posicion, maximoFotograma)
	}
	return dim
}

func (a *Aplicacion) dibujarBarraAccionesVisor(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(14), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarCampoNombreVisor(gtx, archivo.NombreVisible())
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonReproducirVideo, "Abrir", a.paleta.Panel, a.paleta.Texto, func() {
						a.reproducirArchivoActivo()
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonAbrirCarpetaContenedora, "Abrir carpeta contenedora", a.paleta.Panel, a.paleta.Texto, func() {
						a.abrirCarpetaContenedoraArchivoActivo()
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarCampoNombreVisor(gtx layout.Context, texto string) layout.Dimensions {
	return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			altoMinimo := gtx.Dp(unit.Dp(38))
			if gtx.Constraints.Min.Y < altoMinimo {
				gtx.Constraints.Min.Y = altoMinimo
			}
			return a.dibujarListaConBarra(gtx, &a.listaNombreVisor, 1, func(gtx layout.Context, _ int) layout.Dimensions {
				gtx.Constraints.Min.X = 0
				if gtx.Constraints.Max.X < 1000000 {
					gtx.Constraints.Max.X = 1000000
				}
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoPrincipalTruncadoSinRecorte(gtx, texto)
				})
			})
		})
	})
}

func (a *Aplicacion) dibujarContenedorNavegacionVisor(gtx layout.Context, contenido layout.Widget) layout.Dimensions {
	const anchoBoton = unit.Dp(52)

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(anchoBoton)
			gtx.Constraints.Max.X = gtx.Dp(anchoBoton)
			return a.dibujarBotonFlechaVisor(gtx, &a.botonVisorAnterior, "←", a.puedeNavegarVisor(-1), func() {
				a.navegarVisor(-1)
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, contenido),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(anchoBoton)
			gtx.Constraints.Max.X = gtx.Dp(anchoBoton)
			return a.dibujarBotonFlechaVisor(gtx, &a.botonVisorSiguiente, "→", a.puedeNavegarVisor(1), func() {
				a.navegarVisor(1)
			})
		}),
	)
}

func (a *Aplicacion) dibujarBotonFlechaVisor(gtx layout.Context, clic *widget.Clickable, etiqueta string, habilitado bool, alHacer func()) layout.Dimensions {
	contexto := gtx
	fondo := a.paleta.PanelElevado
	colorTexto := a.paleta.Texto
	if !habilitado {
		contexto = contexto.Disabled()
		fondo = a.paleta.Panel
		colorTexto = a.paleta.TextoSuave
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		altoMinimo := gtx.Dp(unit.Dp(44))
		if gtx.Constraints.Min.Y < altoMinimo {
			gtx.Constraints.Min.Y = altoMinimo
		}
		return a.dibujarBotonAccion(contexto, clic, etiqueta, fondo, colorTexto, alHacer)
	})
}

func (a *Aplicacion) procesarAtajosVisor(gtx layout.Context) {
	if a.vistaActual != vistaElementoUnico || !a.tieneArchivoActivo {
		return
	}

	for {
		evento, ok := gtx.Event(
			key.Filter{Name: key.NameLeftArrow},
			key.Filter{Name: key.NameRightArrow},
		)
		if !ok {
			break
		}

		eventoTecla, ok := evento.(key.Event)
		if !ok || eventoTecla.State != key.Press {
			continue
		}
		if a.hayEditorEditableEnFoco(gtx) {
			continue
		}

		switch eventoTecla.Name {
		case key.NameLeftArrow:
			a.navegarVisor(-1)
		case key.NameRightArrow:
			a.navegarVisor(1)
		}
	}
}

func (a *Aplicacion) hayEditorEditableEnFoco(gtx layout.Context) bool {
	editores := []*widget.Editor{
		&a.editorDestinoMover,
		&a.editorFecha,
		&a.editorHora,
		&a.editorZonaHoraria,
		&a.editorPalabras,
		&a.editorUbicacion,
		&a.editorComentario,
		&a.editorCopyright,
		&a.editorGPSLatitud,
		&a.editorGPSLongitud,
		&a.editorMake,
		&a.editorModelo,
		&a.editorSoftware,
		&a.editorFormatoImagen,
		&a.edicionRegiones.EditorNombre,
	}
	for _, editor := range editores {
		if gtx.Focused(editor) {
			return true
		}
	}
	return false
}

func (a *Aplicacion) dibujarImagenConRegiones(gtx layout.Context, archivo modelo.Archivo, imagen image.Image, interactiva bool) layout.Dimensions {
	tamanoOriginal := imagen.Bounds().Size()
	if tamanoOriginal.X == 0 || tamanoOriginal.Y == 0 {
		return a.dibujarTextoPrincipal(gtx, "Imagen vacía")
	}

	contenedor := gtx.Constraints.Max
	escala := math.Min(float64(contenedor.X)/float64(tamanoOriginal.X), float64(contenedor.Y)/float64(tamanoOriginal.Y))
	if escala <= 0 {
		escala = 1
	}
	tamano := image.Pt(maximo(1, int(float64(tamanoOriginal.X)*escala)), maximo(1, int(float64(tamanoOriginal.Y)*escala)))
	radio := unit.Dp(12)
	if interactiva {
		a.actualizarInteraccionRegiones(gtx, archivo, tamano)
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(tamano)
		return dibujarPanel(gtx, a.paleta.PanelElevado, radio, func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					imagenWidget := widget.Image{
						Src:      paint.NewImageOp(imagen),
						Fit:      widget.Fill,
						Position: layout.Center,
						Scale:    1.0 / gtx.Metric.PxPerDp,
					}
					return imagenWidget.Layout(gtx)
				}),
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					a.dibujarRegiones(gtx, archivo, tamano)
					if interactiva {
						a.dibujarSuperposicionEdicionRegiones(gtx, archivo, tamano)
					}
					return layout.Dimensions{Size: tamano}
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarRegiones(gtx layout.Context, archivo modelo.Archivo, tamano image.Point) {
	for _, region := range a.regionesEnEdicion(archivo) {
		a.dibujarContornoRegion(gtx, archivo, region, tamano, a.paleta.Exito, true)
	}
}

func (a *Aplicacion) dibujarEtiquetaRegion(gtx layout.Context, posicion image.Point, nombre string) {
	if nombre == "" {
		return
	}

	defer op.Offset(posicion).Push(gtx.Ops).Pop()
	gtxEtiqueta := gtx
	gtxEtiqueta.Constraints = layout.Constraints{
		Max: image.Pt(maximo(32, gtx.Constraints.Max.X-posicion.X), maximo(24, gtx.Constraints.Max.Y-posicion.Y)),
	}
	dibujarPanel(gtxEtiqueta, a.paleta.Panel, unit.Dp(8), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(4),
			Bottom: unit.Dp(4),
			Left:   unit.Dp(6),
			Right:  unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			estilo := material.Label(a.tema, unit.Sp(11), nombre)
			estilo.Color = a.paleta.Texto
			estilo.MaxLines = 1
			estilo.Truncator = "…"
			return estilo.Layout(gtx)
		})
	})
}

func (a *Aplicacion) dibujarFichaArchivo(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoPrincipal(gtx, archivo.Ruta)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, fmt.Sprintf("Tamaño: %s", formatearTamano(archivo.Tamano)))
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			texto := fmt.Sprintf("Tipo: %s", archivo.Tipo)
			if archivo.Ancho > 0 && archivo.Alto > 0 {
				texto += fmt.Sprintf(" | %dx%d", archivo.Ancho, archivo.Alto)
			}
			if archivo.Duracion > 0 {
				texto += " | " + formatearDuracion(archivo.Duracion)
			}
			return a.dibujarTextoSecundario(gtx, texto)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			whereFroms := "-"
			if len(archivo.Metadatos.WhereFroms) > 0 {
				whereFroms = strings.Join(archivo.Metadatos.WhereFroms, " | ")
			}
			return a.dibujarTextoSecundario(gtx, "WhereFroms: "+whereFroms)
		}),
	)
}

func (a *Aplicacion) dibujarAccionesArchivo(gtx layout.Context) layout.Dimensions {
	mostrarArchivar := a.tieneArchivoActivo && archivoTieneFechaYHoraArchivables(a.archivoActivo)

	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Acciones sobre el archivo")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampo(gtx, "Mover a carpeta", &a.editorDestinoMover)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					hijos := []layout.FlexChild{
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonMoverActivo, "Mover", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
								a.moverArchivoActivo()
							})
						}),
					}
					if mostrarArchivar {
						hijos = append(hijos,
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonAccion(gtx, &a.botonArchivarActivo, "Archivar", a.paleta.Exito, a.paleta.Texto, func() {
									a.archivarArchivoActivo()
								})
							}),
						)
					}
					hijos = append(hijos,
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonPapeleraActiva, "Papelera", a.paleta.Peligro, a.paleta.Texto, func() {
								a.enviarArchivoActivoAPapelera()
							})
						}),
					)
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx, hijos...)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if mostrarArchivar {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, "Archivar aparece cuando el archivo ya tiene fecha y hora guardadas.")
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarAccionesTipo(gtx layout.Context) layout.Dimensions {
	if !a.tieneArchivoActivo {
		return layout.Dimensions{}
	}
	switch a.archivoActivo.Tipo {
	case modelo.TipoImagen:
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarTituloPanel(gtx, "Acciones de imagen")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarEditorCampo(gtx, "Formato de salida", &a.editorFormatoImagen)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBotonAccion(gtx, &a.botonRecortar, "Recorte centrado", a.paleta.PanelElevado, a.paleta.Texto, func() {
							a.recortarImagenActiva()
						})
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarBotonAccion(gtx, &a.botonConvertir, "Convertir imagen", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
							a.convertirImagenActiva()
						})
					}),
				)
			}),
		)
	case modelo.TipoVideo:
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarTituloPanel(gtx, "Acciones de video")
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBloqueExtraerFrame(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(a.tema, &a.sobreescribirVideo, "Sobrescribir al optimizar").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dibujarBotonAccion(gtx, &a.botonOptimizarVideo, "Optimizar para web", a.paleta.Exito, a.paleta.Texto, func() {
					a.optimizarVideoActivo()
				})
			}),
		)
	default:
		return layout.Dimensions{}
	}
}

func (a *Aplicacion) dibujarControlesDuplicados(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonCategoriaCoincidencia(gtx, &a.botonCoincidenciaExacta, "Exacta", a.tipoCoincidenciaActual == modelo.CoincidenciaExacta, modelo.CoincidenciaExacta)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonCategoriaCoincidencia(gtx, &a.botonCoincidenciaImagen, "dHash imagen", a.tipoCoincidenciaActual == modelo.CoincidenciaParcialImagen, modelo.CoincidenciaParcialImagen)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonCategoriaCoincidencia(gtx, &a.botonCoincidenciaVideo, "dHash video", a.tipoCoincidenciaActual == modelo.CoincidenciaParcialVideo, modelo.CoincidenciaParcialVideo)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonOrden(gtx, &a.botonOrdenGrupo, "Por grupo", a.ordenDuplicados == modelo.OrdenPorTamanoGrupo, modelo.OrdenPorTamanoGrupo)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonOrden(gtx, &a.botonOrdenEspacio, "Por espacio", a.ordenDuplicados == modelo.OrdenPorEspacioRecuperado, modelo.OrdenPorEspacioRecuperado)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonOrden(gtx, &a.botonOrdenAlfabetico, "A-Z", a.ordenDuplicados == modelo.OrdenAlfabetico, modelo.OrdenAlfabetico)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonRecargarDuplicados, "Recargar", a.paleta.PanelElevado, a.paleta.Texto, func() {
						a.recargarDuplicados()
					})
				}),
			)
		}),
	)
}

func (a *Aplicacion) dibujarListaDuplicados(gtx layout.Context) layout.Dimensions {
	if len(a.gruposDuplicados) == 0 {
		if a.cargandoDuplicados {
			return a.dibujarTextoPrincipal(gtx, "Cargando grupos...")
		}
		return a.dibujarTextoPrincipal(gtx, "Todavía no hay grupos para esta combinación de filtros.")
	}

	grupos := a.gruposDuplicados

	return a.dibujarListaConBarra(gtx, &a.listaDuplicados, len(grupos), func(gtx layout.Context, indice int) layout.Dimensions {
		grupo := grupos[indice]
		return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return a.dibujarGrupoDuplicado(gtx, grupo)
		})
	})
}

func (a *Aplicacion) dibujarGrupoDuplicado(gtx layout.Context, grupo modelo.GrupoDuplicados) layout.Dimensions {
	widgets := a.asegurarWidgetsGrupo(grupo)

	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoPrincipal(gtx, fmt.Sprintf("%s | %d elementos | recuperable %s", grupo.NombreRepresentivo, grupo.CantidadElementos, formatearTamano(grupo.TamanoRecuperable)))
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &widgets.BorrarMasAntiguo, "Borrar más antiguo", a.paleta.Peligro, a.paleta.Texto, func() {
								if len(grupo.Elementos) > 0 {
									a.borrarRutasDuplicadas([]string{grupo.Elementos[0].Ruta})
								}
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &widgets.BorrarMasNuevo, "Borrar más nuevo", a.paleta.Peligro, a.paleta.Texto, func() {
								if len(grupo.Elementos) > 0 {
									a.borrarRutasDuplicadas([]string{grupo.Elementos[len(grupo.Elementos)-1].Ruta})
								}
							})
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, a.construirFilasGrupoDuplicado(grupo, widgets)...)
				}),
			)
		})
	})
}

func (a *Aplicacion) construirFilasGrupoDuplicado(grupo modelo.GrupoDuplicados, widgets *widgetsGrupoDuplicado) []layout.FlexChild {
	var hijos []layout.FlexChild
	for _, elemento := range grupo.Elementos {
		elemento := elemento
		hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if len(grupo.Elementos) <= 2 {
									if widgets.BorrarElemento == nil {
										widgets.BorrarElemento = make(map[string]*widget.Clickable)
									}
									clic, ok := widgets.BorrarElemento[elemento.Ruta]
									if !ok {
										clic = &widget.Clickable{}
										widgets.BorrarElemento[elemento.Ruta] = clic
									}
									return a.dibujarBotonAccion(gtx, clic, "Eliminar", a.paleta.Peligro, a.paleta.Texto, func() {
										a.borrarRutasDuplicadas([]string{elemento.Ruta})
									})
								}
								if widgets.Seleccion == nil {
									widgets.Seleccion = make(map[string]*widget.Bool)
								}
								estado, ok := widgets.Seleccion[elemento.Ruta]
								if !ok {
									estado = &widget.Bool{}
									widgets.Seleccion[elemento.Ruta] = estado
								}
								return material.CheckBox(a.tema, estado, "").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.dibujarTextoPrincipal(gtx, elemento.Ruta)
									}),
									layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return a.dibujarTextoSecundario(gtx, fmt.Sprintf("%s | %s | %s", elemento.Origen, formatearTamano(elemento.Tamano), elemento.Modificado.Format("2006-01-02 15:04:05")))
									}),
								)
							}),
						)
					})
				})
			})
		}))
	}

	if len(grupo.Elementos) > 2 {
		hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			var rutas []string
			for ruta, estado := range widgets.Seleccion {
				if estado != nil && estado.Value {
					rutas = append(rutas, ruta)
				}
			}
			return a.dibujarBotonAccion(gtx, &widgets.BorrarMarcados, fmt.Sprintf("Borrar seleccionados (%d)", len(rutas)), a.paleta.Peligro, a.paleta.Texto, func() {
				if len(rutas) > 0 {
					a.borrarRutasDuplicadas(rutas)
				}
			})
		}))
	}

	return hijos
}

func (a *Aplicacion) borrarRutasDuplicadas(rutas []string) {
	if len(rutas) == 0 {
		return
	}
	go func() {
		var primerError error
		eliminados := 0
		for _, ruta := range rutas {
			if err := a.servicioArchivos.EnviarAPapelera(context.Background(), ruta); err != nil {
				if primerError == nil {
					primerError = err
				}
				continue
			}
			if err := a.almacen.EliminarArchivo(context.Background(), ruta); err != nil && primerError == nil {
				primerError = err
			}
			eliminados++
		}
		a.encolarActualizacion(func() {
			if primerError != nil {
				a.establecerEstado(fmt.Sprintf("Se eliminaron %d elementos duplicados con incidencias", eliminados), primerError)
			} else {
				a.establecerEstado(fmt.Sprintf("Se eliminaron %d elementos duplicados", eliminados), nil)
			}
			a.reiniciarListado()
			a.recargarDuplicados()
		})
	}()
}

func (a *Aplicacion) resumenArchivo(archivo modelo.Archivo) string {
	partes := []string{formatearTamano(archivo.Tamano)}
	if archivo.Ancho > 0 && archivo.Alto > 0 {
		partes = append(partes, fmt.Sprintf("%dx%d", archivo.Ancho, archivo.Alto))
	}
	if archivo.Duracion > 0 {
		partes = append(partes, formatearDuracion(archivo.Duracion))
	}
	return strings.Join(partes, " | ")
}

func (a *Aplicacion) dibujarIndicadores(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	var insignias []layout.FlexChild
	agregarTexto := func(texto string, fondo color.NRGBA) {
		insignias = append(insignias, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarInsignia(gtx, texto, fondo)
			})
		}))
	}
	agregarIcono := func(fondo color.NRGBA, dibujar func(layout.Context, color.NRGBA, color.NRGBA) layout.Dimensions) {
		insignias = append(insignias, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarInsigniaIcono(gtx, fondo, dibujar)
			})
		}))
	}

	if archivo.Indicadores.TieneGPS {
		agregarIcono(a.paleta.Exito, a.dibujarIconoIndicadorUbicacion)
	}
	if archivo.Indicadores.TieneRegiones {
		agregarIcono(a.paleta.Exito, a.dibujarIconoIndicadorRostro)
	}
	if archivo.Indicadores.TieneWhereFrom {
		agregarIcono(a.paleta.Acento, a.dibujarIconoIndicadorInformacion)
	}
	if archivo.Indicadores.TieneIA {
		agregarIcono(a.paleta.Advertencia, a.dibujarIconoIndicadorIA)
	}
	if archivo.Indicadores.TieneSocial {
		agregarIcono(a.paleta.Peligro, a.dibujarIconoIndicadorSocial)
	}
	if archivo.Indicadores.EsAdulto {
		agregarTexto("+18", a.paleta.Peligro)
	}
	if len(insignias) == 0 {
		return a.dibujarTextoSecundario(gtx, "-")
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, insignias...)
}

func (a *Aplicacion) dibujarInsignia(gtx layout.Context, texto string, fondo color.NRGBA) layout.Dimensions {
	return dibujarPanel(gtx, fondo, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(4),
			Bottom: unit.Dp(4),
			Left:   unit.Dp(6),
			Right:  unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			estilo := material.Label(a.tema, unit.Sp(11), texto)
			estilo.Color = a.paleta.TextoSobreAcento
			return estilo.Layout(gtx)
		})
	})
}

func (a *Aplicacion) dibujarInsigniaIcono(gtx layout.Context, fondo color.NRGBA, dibujar func(layout.Context, color.NRGBA, color.NRGBA) layout.Dimensions) layout.Dimensions {
	return dibujarPanel(gtx, fondo, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return dibujar(gtx, a.paleta.TextoSobreAcento, fondo)
			})
		})
	})
}

func (a *Aplicacion) dibujarBotonAccion(gtx layout.Context, clic *widget.Clickable, texto string, fondo, colorTexto color.NRGBA, alHacer func()) layout.Dimensions {
	pulsado := false
	for clic.Clicked(gtx) {
		pulsado = true
	}

	estilo := material.Button(a.tema, clic, texto)
	estilo.Background = fondo
	estilo.Color = colorTexto
	estilo.CornerRadius = unit.Dp(10)
	dim := estilo.Layout(gtx)

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

func (a *Aplicacion) dibujarBotonNavegacion(gtx layout.Context, clic *widget.Clickable, texto string, activo bool, alHacer func()) layout.Dimensions {
	fondo := a.paleta.PanelElevado
	colorTexto := a.paleta.Texto
	if activo {
		fondo = a.paleta.Acento
		colorTexto = a.paleta.TextoSobreAcento
	}
	return a.dibujarBotonAccion(gtx, clic, texto, fondo, colorTexto, alHacer)
}

func (a *Aplicacion) dibujarBotonNavegacionIcono(gtx layout.Context, clic *widget.Clickable, activo bool, alHacer func(), dibujarIcono func(layout.Context, color.NRGBA, color.NRGBA) layout.Dimensions) layout.Dimensions {
	fondo := a.paleta.PanelElevado
	colorIcono := a.paleta.Texto
	if activo {
		fondo = a.paleta.Acento
		colorIcono = a.paleta.TextoSobreAcento
	}

	pulsado := false
	for clic.Clicked(gtx) {
		pulsado = true
	}

	dim := dibujarPanel(gtx, fondo, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		return clic.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				altoMinimo := gtx.Dp(unit.Dp(40))
				if gtx.Constraints.Min.Y < altoMinimo {
					gtx.Constraints.Min.Y = altoMinimo
				}
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return dibujarIcono(gtx, colorIcono, fondo)
				})
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

func (a *Aplicacion) dibujarFilaArbol(gtx layout.Context, clic *widget.Clickable, texto string, activo bool, alHacer func()) layout.Dimensions {
	fondo := a.paleta.PanelElevado
	colorTexto := a.paleta.Texto
	if activo {
		fondo = a.paleta.Acento
		colorTexto = a.paleta.TextoSobreAcento
	}

	pulsado := false
	for clic.Clicked(gtx) {
		pulsado = true
	}

	dim := dibujarPanel(gtx, fondo, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		return clic.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(8),
				Bottom: unit.Dp(8),
				Left:   unit.Dp(10),
				Right:  unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				estilo := material.Label(a.tema, unit.Sp(13), texto)
				estilo.Color = colorTexto
				return estilo.Layout(gtx)
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

func (a *Aplicacion) dibujarBotonIconoCarpeta(gtx layout.Context, clic *widget.Clickable, abierta bool, alHacer func()) layout.Dimensions {
	pulsado := false
	for clic.Clicked(gtx) {
		pulsado = true
	}

	dim := dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		return clic.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.dibujarIconoCarpeta(gtx, abierta, image.Pt(22, 18))
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

func (a *Aplicacion) dibujarIconoCarpeta(gtx layout.Context, abierta bool, tamano image.Point) layout.Dimensions {
	if tamano.X < 12 {
		tamano.X = 12
	}
	if tamano.Y < 10 {
		tamano.Y = 10
	}

	fondo := a.paleta.Acento
	frente := color.NRGBA{R: 236, G: 190, B: 103, A: 255}
	if abierta {
		fondo = color.NRGBA{R: 194, G: 132, B: 42, A: 255}
		frente = color.NRGBA{R: 242, G: 203, B: 128, A: 255}
	}

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(tamano.X/9, tamano.Y/5, tamano.X*6/10, tamano.Y/2), 2).Op(gtx.Ops))
			paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(tamano.X/10, tamano.Y/3, tamano.X*9/10, tamano.Y*5/6), 3).Op(gtx.Ops))
			return layout.Dimensions{Size: tamano}
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if abierta {
				var ruta clip.Path
				ruta.Begin(gtx.Ops)
				ruta.MoveTo(f32.Pt(float32(tamano.X)/7, float32(tamano.Y)/2.5))
				ruta.LineTo(f32.Pt(float32(tamano.X)/3.5, float32(tamano.Y)/3.2))
				ruta.LineTo(f32.Pt(float32(tamano.X)*0.92, float32(tamano.Y)/3.2))
				ruta.LineTo(f32.Pt(float32(tamano.X)*0.78, float32(tamano.Y)*0.88))
				ruta.LineTo(f32.Pt(float32(tamano.X)*0.15, float32(tamano.Y)*0.88))
				ruta.Close()
				paint.FillShape(gtx.Ops, frente, clip.Outline{Path: ruta.End()}.Op())
			} else {
				paint.FillShape(gtx.Ops, frente, clip.UniformRRect(image.Rect(tamano.X/9, tamano.Y/2, tamano.X*9/10, tamano.Y*5/6), 3).Op(gtx.Ops))
			}
			return layout.Dimensions{Size: tamano}
		}),
	)
}

func (a *Aplicacion) dibujarIconoPestanaDirectorios(gtx layout.Context, colorIcono, _ color.NRGBA) layout.Dimensions {
	tamano := image.Pt(24, 22)
	linea := clip.Path{}
	linea.Begin(gtx.Ops)
	linea.MoveTo(f32.Pt(6, 4))
	linea.LineTo(f32.Pt(6, 18))
	linea.LineTo(f32.Pt(11, 18))
	linea.MoveTo(f32.Pt(6, 10))
	linea.LineTo(f32.Pt(11, 10))
	paint.FillShape(gtx.Ops, colorIcono, clip.Stroke{
		Path:  linea.End(),
		Width: 2,
	}.Op())

	a.dibujarBloqueDirectorio(gtx, image.Rect(1, 1, 10, 7), colorIcono)
	a.dibujarBloqueDirectorio(gtx, image.Rect(11, 7, 23, 13), colorIcono)
	a.dibujarBloqueDirectorio(gtx, image.Rect(11, 15, 23, 21), colorIcono)
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoPestanaEtiqueta(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(24, 22)
	var ruta clip.Path
	ruta.Begin(gtx.Ops)
	ruta.MoveTo(f32.Pt(2, 9))
	ruta.LineTo(f32.Pt(9, 2))
	ruta.LineTo(f32.Pt(18, 2))
	ruta.LineTo(f32.Pt(22, 6))
	ruta.LineTo(f32.Pt(22, 13))
	ruta.LineTo(f32.Pt(13, 21))
	ruta.LineTo(f32.Pt(2, 11))
	ruta.Close()
	paint.FillShape(gtx.Ops, colorIcono, clip.Outline{Path: ruta.End()}.Op())
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(14, 4, 18, 8)).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoPestanaLugar(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(22, 24)
	paint.FillShape(gtx.Ops, colorIcono, clip.Ellipse(image.Rect(4, 1, 18, 15)).Op(gtx.Ops))

	var punta clip.Path
	punta.Begin(gtx.Ops)
	punta.MoveTo(f32.Pt(11, 23))
	punta.LineTo(f32.Pt(6, 11))
	punta.LineTo(f32.Pt(16, 11))
	punta.Close()
	paint.FillShape(gtx.Ops, colorIcono, clip.Outline{Path: punta.End()}.Op())
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(8, 5, 14, 11)).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoPestanaYandex(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(24, 20)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			a.dibujarDiscoYandex(gtx, colorIcono, fondo)
			return layout.Dimensions{Size: tamano}
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				estilo := material.Label(a.tema, unit.Sp(11), "Y")
				estilo.Color = fondo
				return estilo.Layout(gtx)
			})
		}),
	)
}

func (a *Aplicacion) dibujarIconoIndicadorInformacion(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(14, 14)
	paint.FillShape(gtx.Ops, colorIcono, clip.Ellipse(image.Rect(0, 0, tamano.X, tamano.Y)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(2, 2, tamano.X-2, tamano.Y-2)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.Ellipse(image.Rect(5, 3, 9, 7)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(6, 6, 8, 12), 1).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoIndicadorSocial(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(16, 14)
	var triangulo clip.Path
	triangulo.Begin(gtx.Ops)
	triangulo.MoveTo(f32.Pt(8, 0))
	triangulo.LineTo(f32.Pt(16, 14))
	triangulo.LineTo(f32.Pt(0, 14))
	triangulo.Close()
	paint.FillShape(gtx.Ops, colorIcono, clip.Outline{Path: triangulo.End()}.Op())
	paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(7, 4, 9, 10), 1).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(6, 11, 10, 14)).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoIndicadorIA(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(16, 14)
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(2, 3, 14, 13), 3).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(7, 0, 9, 4), 1).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(5, 6, 7, 8)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(9, 6, 11, 8)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(5, 9, 11, 10), 1).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(4, 13, 6, 15), 1).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(10, 13, 12, 15), 1).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoIndicadorRostro(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(16, 14)
	paint.FillShape(gtx.Ops, colorIcono, clip.Ellipse(image.Rect(4, 0, 12, 8)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(6, 2, 8, 4)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(8, 2, 10, 4)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(6, 5, 10, 6), 1).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(2, 8, 14, 14), 4).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(image.Rect(5, 9, 11, 14), 2).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarIconoIndicadorUbicacion(gtx layout.Context, colorIcono, fondo color.NRGBA) layout.Dimensions {
	tamano := image.Pt(14, 16)
	paint.FillShape(gtx.Ops, colorIcono, clip.Ellipse(image.Rect(2, 0, 12, 10)).Op(gtx.Ops))
	var punta clip.Path
	punta.Begin(gtx.Ops)
	punta.MoveTo(f32.Pt(7, 16))
	punta.LineTo(f32.Pt(3, 7))
	punta.LineTo(f32.Pt(11, 7))
	punta.Close()
	paint.FillShape(gtx.Ops, colorIcono, clip.Outline{Path: punta.End()}.Op())
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(5, 3, 9, 7)).Op(gtx.Ops))
	return layout.Dimensions{Size: tamano}
}

func (a *Aplicacion) dibujarBloqueDirectorio(gtx layout.Context, rect image.Rectangle, colorIcono color.NRGBA) {
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(rect, 2).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(image.Rect(rect.Min.X+1, rect.Min.Y-1, rect.Min.X+5, rect.Min.Y+1), 1).Op(gtx.Ops))
}

func (a *Aplicacion) dibujarDiscoYandex(gtx layout.Context, colorIcono, fondo color.NRGBA) {
	cuerpo := image.Rect(1, 3, 23, 19)
	paint.FillShape(gtx.Ops, colorIcono, clip.UniformRRect(cuerpo, 4).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Rect(image.Rect(4, 8, 20, 9)).Op())
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(17, 12, 19, 14)).Op(gtx.Ops))
	paint.FillShape(gtx.Ops, fondo, clip.Ellipse(image.Rect(20, 12, 22, 14)).Op(gtx.Ops))
}

func (a *Aplicacion) dibujarBotonCategoriaDuplicado(gtx layout.Context, clic *widget.Clickable, texto string, activo bool, categoria modelo.CategoriaDuplicados) layout.Dimensions {
	return a.dibujarBotonNavegacion(gtx, clic, texto, activo, func() {
		if a.categoriaDuplicados != categoria {
			a.categoriaDuplicados = categoria
			a.recargarDuplicados()
		}
	})
}

func (a *Aplicacion) dibujarBotonCategoriaCoincidencia(gtx layout.Context, clic *widget.Clickable, texto string, activo bool, tipo modelo.TipoCoincidencia) layout.Dimensions {
	return a.dibujarBotonNavegacion(gtx, clic, texto, activo, func() {
		if a.tipoCoincidenciaActual != tipo {
			a.tipoCoincidenciaActual = tipo
			a.recargarDuplicados()
		}
	})
}

func (a *Aplicacion) dibujarBotonOrden(gtx layout.Context, clic *widget.Clickable, texto string, activo bool, orden modelo.OrdenDuplicados) layout.Dimensions {
	return a.dibujarBotonNavegacion(gtx, clic, texto, activo, func() {
		if a.ordenDuplicados != orden {
			a.ordenDuplicados = orden
			a.recargarDuplicados()
		}
	})
}

func (a *Aplicacion) dibujarTituloPanel(gtx layout.Context, texto string) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(17), texto)
	estilo.Color = a.paleta.Texto
	return estilo.Layout(gtx)
}

func (a *Aplicacion) dibujarTextoPrincipal(gtx layout.Context, texto string) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(14), texto)
	estilo.Color = a.paleta.Texto
	return estilo.Layout(gtx)
}

func (a *Aplicacion) dibujarTextoPrincipalTruncado(gtx layout.Context, texto string) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(14), texto)
	estilo.Color = a.paleta.Texto
	estilo.MaxLines = 1
	estilo.Truncator = "…"
	return estilo.Layout(gtx)
}

func (a *Aplicacion) dibujarTextoPrincipalTruncadoSinRecorte(gtx layout.Context, texto string) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(14), texto)
	estilo.Color = a.paleta.Texto
	estilo.MaxLines = 1
	return estilo.Layout(gtx)
}

func (a *Aplicacion) dibujarTextoSecundario(gtx layout.Context, texto string) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(12), texto)
	estilo.Color = a.paleta.TextoSuave
	return estilo.Layout(gtx)
}

func (a *Aplicacion) dibujarEditorCampo(gtx layout.Context, etiqueta string, editor *widget.Editor) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, etiqueta)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					editorEstilo := material.Editor(a.tema, editor, "")
					editorEstilo.Color = a.paleta.Texto
					editorEstilo.HintColor = a.paleta.TextoSuave
					return editorEstilo.Layout(gtx)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarEditorBusquedaLateral(gtx layout.Context, editor *widget.Editor) layout.Dimensions {
	return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			editorEstilo := material.Editor(a.tema, editor, "Filtrar...")
			editorEstilo.Color = a.paleta.Texto
			editorEstilo.HintColor = a.paleta.TextoSuave
			return editorEstilo.Layout(gtx)
		})
	})
}

func (a *Aplicacion) dibujarLineaChecks(gtx layout.Context, primero *widget.Bool, textoPrimero string, segundo *widget.Bool, textoSegundo string) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, primero, textoPrimero).Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.CheckBox(a.tema, segundo, textoSegundo).Layout(gtx)
		}),
	)
}

func (a *Aplicacion) dibujarBarraProgreso(gtx layout.Context, porcentaje float64) layout.Dimensions {
	porcentaje = math.Max(0, math.Min(100, porcentaje))
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(10), func(gtx layout.Context) layout.Dimensions {
		ancho := gtx.Constraints.Max.X
		alto := maximo(16, gtx.Dp(unit.Dp(16)))
		relleno := int(float64(ancho) * (porcentaje / 100))

		return layout.Stack{}.Layout(gtx,
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, a.paleta.PanelElevado, clip.Rect(image.Rect(0, 0, ancho, alto)).Op())
				return layout.Dimensions{Size: image.Pt(ancho, alto)}
			}),
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, a.paleta.Acento, clip.Rect(image.Rect(0, 0, relleno, alto)).Op())
				return layout.Dimensions{Size: image.Pt(ancho, alto)}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, fmt.Sprintf("%.1f%%", porcentaje))
				})
			}),
		)
	})
}

func (a *Aplicacion) asegurarWidgetsElemento(ruta string) *widgetsElemento {
	if widgets, existe := a.elementoWidgets[ruta]; existe {
		return widgets
	}
	widgets := &widgetsElemento{}
	a.elementoWidgets[ruta] = widgets
	return widgets
}

func (a *Aplicacion) asegurarWidgetsGrupo(grupo modelo.GrupoDuplicados) *widgetsGrupoDuplicado {
	clave := string(grupo.Tipo) + "|" + grupo.Clave
	if widgets, existe := a.grupoWidgets[clave]; existe {
		return widgets
	}
	widgets := &widgetsGrupoDuplicado{
		Seleccion:      make(map[string]*widget.Bool),
		BorrarElemento: make(map[string]*widget.Clickable),
	}
	a.grupoWidgets[clave] = widgets
	return widgets
}

func transformarRegion(region modelo.RegionEtiquetada, orientacion int) (x, y, ancho, alto float64) {
	regionTransformada := modelo.TransformarRegionOrientada(region, orientacion)
	return regionTransformada.X, regionTransformada.Y, regionTransformada.Ancho, regionTransformada.Alto
}
