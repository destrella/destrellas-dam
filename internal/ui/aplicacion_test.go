package ui

import "testing"

func TestDrenarActualizacionesLimitaElTrabajoPorFrame(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		actualizaciones: make(chan func(), maxActualizacionesPorFrame+12),
	}

	procesadas := 0
	for indice := 0; indice < maxActualizacionesPorFrame+12; indice++ {
		app.actualizaciones <- func() {
			procesadas++
		}
	}

	app.drenarActualizaciones()

	if procesadas != maxActualizacionesPorFrame {
		t.Fatalf("se esperaban %d actualizaciones procesadas en un frame, se obtuvieron %d", maxActualizacionesPorFrame, procesadas)
	}
	if restantes := len(app.actualizaciones); restantes != 12 {
		t.Fatalf("deberían quedar 12 actualizaciones pendientes, se obtuvieron %d", restantes)
	}
}
