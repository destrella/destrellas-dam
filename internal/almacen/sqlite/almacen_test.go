package sqlite

import (
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestClausulaOrdenListado(t *testing.T) {
	t.Parallel()

	pruebas := []struct {
		nombre   string
		filtros  modelo.FiltrosListado
		esperado string
	}{
		{
			nombre:   "nombre ascendente",
			filtros:  modelo.FiltrosListado{CriterioOrden: modelo.CriterioOrdenNombre},
			esperado: "ORDER BY archivos.nombre COLLATE NOCASE ASC, archivos.nombre ASC, archivos.ruta ASC",
		},
		{
			nombre: "nombre descendente",
			filtros: modelo.FiltrosListado{
				CriterioOrden:    modelo.CriterioOrdenNombre,
				OrdenDescendente: true,
			},
			esperado: "ORDER BY archivos.nombre COLLATE NOCASE DESC, archivos.nombre DESC, archivos.ruta ASC",
		},
		{
			nombre: "fecha descendente",
			filtros: modelo.FiltrosListado{
				CriterioOrden:    modelo.CriterioOrdenFechaModificacion,
				OrdenDescendente: true,
			},
			esperado: "ORDER BY archivos.modificado_unix DESC, archivos.nombre COLLATE NOCASE ASC, archivos.nombre ASC, archivos.ruta ASC",
		},
	}

	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()

			if obtenido := clausulaOrdenListado(prueba.filtros); obtenido != prueba.esperado {
				t.Fatalf("cláusula inesperada:\nesperado: %s\nobtenido: %s", prueba.esperado, obtenido)
			}
		})
	}
}
