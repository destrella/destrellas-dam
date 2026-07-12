package ui

import (
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestSeleccionElementoConPosibleRangoSeleccionaIntervaloVisible(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
			{Ruta: "/tmp/d.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarElementoConPosibleRango("/tmp/a.jpg")
	if app.anclaSeleccionLote != "/tmp/a.jpg" {
		t.Fatalf("se esperaba que la ancla inicial fuera /tmp/a.jpg, se obtuvo %q", app.anclaSeleccionLote)
	}

	app.seleccionarElementoConPosibleRango("/tmp/d.jpg")

	esperadas := []string{"/tmp/a.jpg", "/tmp/b.jpg", "/tmp/c.jpg", "/tmp/d.jpg"}
	if len(app.seleccionLote) != len(esperadas) {
		t.Fatalf("cantidad inesperada de seleccionados: %d", len(app.seleccionLote))
	}
	for _, ruta := range esperadas {
		if !app.seleccionLote[ruta] {
			t.Fatalf("la ruta %q debería formar parte del rango seleccionado", ruta)
		}
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("la ancla debería liberarse tras construir el rango, se obtuvo %q", app.anclaSeleccionLote)
	}
}

func TestSeleccionElementoConPosibleRangoMantieneSeleccionLibreTrasRango(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
			{Ruta: "/tmp/d.jpg"},
			{Ruta: "/tmp/e.jpg"},
			{Ruta: "/tmp/f.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarElementoConPosibleRango("/tmp/b.jpg")
	app.seleccionarElementoConPosibleRango("/tmp/d.jpg")
	app.seleccionarElementoConPosibleRango("/tmp/f.jpg")

	esperadas := []string{"/tmp/b.jpg", "/tmp/c.jpg", "/tmp/d.jpg", "/tmp/f.jpg"}
	if len(app.seleccionLote) != len(esperadas) {
		t.Fatalf("cantidad inesperada de seleccionados: %d", len(app.seleccionLote))
	}
	for _, ruta := range esperadas {
		if !app.seleccionLote[ruta] {
			t.Fatalf("la ruta %q debería seguir seleccionada", ruta)
		}
	}
	if app.seleccionLote["/tmp/e.jpg"] {
		t.Fatal("la ruta /tmp/e.jpg no debería añadirse automáticamente tras completar el rango anterior")
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("no debería quedar ancla activa con varias selecciones independientes, se obtuvo %q", app.anclaSeleccionLote)
	}
}

func TestSeleccionarTodoYDeseleccionarTodoActualizaElEstado(t *testing.T) {
	t.Parallel()

	app := &Aplicacion{
		elementos: []modelo.Archivo{
			{Ruta: "/tmp/a.jpg"},
			{Ruta: "/tmp/b.jpg"},
			{Ruta: "/tmp/c.jpg"},
		},
		seleccionLote: make(map[string]bool),
	}

	app.seleccionarTodosElementosCargados()
	if len(app.seleccionLote) != len(app.elementos) {
		t.Fatalf("se esperaban %d elementos seleccionados, se obtuvieron %d", len(app.elementos), len(app.seleccionLote))
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("no debería quedar ancla activa al seleccionar todo, se obtuvo %q", app.anclaSeleccionLote)
	}

	app.deseleccionarTodosElementos()
	if len(app.seleccionLote) != 0 {
		t.Fatalf("la selección debería quedar vacía, se obtuvieron %d elementos", len(app.seleccionLote))
	}
	if app.anclaSeleccionLote != "" {
		t.Fatalf("la ancla debería limpiarse al deseleccionar todo, se obtuvo %q", app.anclaSeleccionLote)
	}
}
