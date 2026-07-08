package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Paleta concentra la identidad visual de la aplicacion.
type Paleta struct {
	Fondo            color.NRGBA
	Panel            color.NRGBA
	PanelElevado     color.NRGBA
	Borde            color.NRGBA
	Acento           color.NRGBA
	AcentoSuave      color.NRGBA
	Texto            color.NRGBA
	TextoSuave       color.NRGBA
	Exito            color.NRGBA
	Advertencia      color.NRGBA
	Peligro          color.NRGBA
	TextoSobreAcento color.NRGBA
}

func nuevaPaleta() Paleta {
	return Paleta{
		Fondo:            color.NRGBA{R: 18, G: 24, B: 31, A: 255},
		Panel:            color.NRGBA{R: 29, G: 36, B: 45, A: 255},
		PanelElevado:     color.NRGBA{R: 37, G: 46, B: 57, A: 255},
		Borde:            color.NRGBA{R: 63, G: 75, B: 89, A: 255},
		Acento:           color.NRGBA{R: 212, G: 145, B: 52, A: 255},
		AcentoSuave:      color.NRGBA{R: 87, G: 62, B: 31, A: 255},
		Texto:            color.NRGBA{R: 236, G: 241, B: 245, A: 255},
		TextoSuave:       color.NRGBA{R: 162, G: 174, B: 186, A: 255},
		Exito:            color.NRGBA{R: 65, G: 166, B: 120, A: 255},
		Advertencia:      color.NRGBA{R: 214, G: 168, B: 67, A: 255},
		Peligro:          color.NRGBA{R: 196, G: 85, B: 78, A: 255},
		TextoSobreAcento: color.NRGBA{R: 16, G: 18, B: 20, A: 255},
	}
}

func nuevaTema(paleta Paleta) *material.Theme {
	tema := material.NewTheme()
	tema.Palette = material.Palette{
		Bg:         paleta.Fondo,
		Fg:         paleta.Texto,
		ContrastBg: paleta.Acento,
		ContrastFg: paleta.TextoSobreAcento,
	}
	return tema
}

func dibujarPanel(gtx layout.Context, fondo color.NRGBA, radio unit.Dp, contenido layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			// El fondo del panel debe ajustarse al tamano real del contenido.
			// Si usamos Max aqui, el panel consume toda el area disponible y
			// rompe el layout de Flex, ocultando el resto de la interfaz.
			rect := image.Rectangle{Max: gtx.Constraints.Min}
			paint.FillShape(gtx.Ops, fondo, clip.UniformRRect(rect, gtx.Dp(radio)).Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		contenido,
	)
}

func dibujarPanelConBorde(gtx layout.Context, fondo, borde color.NRGBA, radio, anchoBorde unit.Dp, contenido layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			rect := image.Rectangle{Max: gtx.Constraints.Min}
			paint.FillShape(gtx.Ops, borde, clip.UniformRRect(rect, gtx.Dp(radio)).Op(gtx.Ops))
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			bordePx := gtx.Dp(anchoBorde)
			if bordePx < 1 {
				bordePx = 1
			}
			radioInterno := radio - anchoBorde
			if radioInterno < 0 {
				radioInterno = 0
			}
			return layout.Inset{
				Top:    unit.Dp(float32(bordePx)),
				Right:  unit.Dp(float32(bordePx)),
				Bottom: unit.Dp(float32(bordePx)),
				Left:   unit.Dp(float32(bordePx)),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return dibujarPanel(gtx, fondo, radioInterno, contenido)
			})
		},
	)
}
