package indexador

import (
	"testing"
	"time"

	"destrellas-dam/internal/modelo"
)

func TestOrdenarArchivosSegunFiltros(t *testing.T) {
	t.Parallel()

	base := []modelo.Archivo{
		{
			Ruta:       "/tmp/c.jpg",
			Nombre:     "c.jpg",
			Modificado: time.Unix(100, 0),
		},
		{
			Ruta:       "/tmp/b.jpg",
			Nombre:     "b.jpg",
			Modificado: time.Unix(200, 0),
		},
		{
			Ruta:       "/tmp/a.jpg",
			Nombre:     "a.jpg",
			Modificado: time.Unix(200, 0),
		},
	}

	pruebas := []struct {
		nombre   string
		filtros  modelo.FiltrosListado
		esperado []string
	}{
		{
			nombre: "mas nuevos primero",
			filtros: modelo.FiltrosListado{
				CriterioOrden:    modelo.CriterioOrdenFechaModificacion,
				OrdenDescendente: true,
			},
			esperado: []string{"a.jpg", "b.jpg", "c.jpg"},
		},
		{
			nombre: "mas antiguos primero",
			filtros: modelo.FiltrosListado{
				CriterioOrden:    modelo.CriterioOrdenFechaModificacion,
				OrdenDescendente: false,
			},
			esperado: []string{"c.jpg", "a.jpg", "b.jpg"},
		},
		{
			nombre: "alfabetico descendente",
			filtros: modelo.FiltrosListado{
				CriterioOrden:    modelo.CriterioOrdenNombre,
				OrdenDescendente: true,
			},
			esperado: []string{"c.jpg", "b.jpg", "a.jpg"},
		},
	}

	for _, prueba := range pruebas {
		prueba := prueba
		t.Run(prueba.nombre, func(t *testing.T) {
			t.Parallel()

			archivos := append([]modelo.Archivo(nil), base...)
			ordenarArchivosSegunFiltros(archivos, prueba.filtros)

			if len(archivos) != len(prueba.esperado) {
				t.Fatalf("cantidad inesperada: %d", len(archivos))
			}
			for indice, esperado := range prueba.esperado {
				if archivos[indice].Nombre != esperado {
					t.Fatalf("posición %d: esperado %q, obtenido %q", indice, esperado, archivos[indice].Nombre)
				}
			}
		})
	}
}
