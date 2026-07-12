package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestListarGruposDuplicadosCargaElementosEnLote(t *testing.T) {
	t.Parallel()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	defer almacen.Cerrar()

	archivos := []modelo.Archivo{
		{
			Ruta:       "/tmp/a.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "a.jpg",
			Tamano:     1200,
			Modificado: time.Unix(10, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				DHashImagen: "grupo-1",
			},
		},
		{
			Ruta:       "/tmp/b.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "b.jpg",
			Tamano:     1300,
			Modificado: time.Unix(20, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				DHashImagen: "grupo-1",
			},
		},
		{
			Ruta:       "/tmp/c.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "c.jpg",
			Tamano:     900,
			Modificado: time.Unix(30, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				DHashImagen: "grupo-2",
			},
		},
		{
			Ruta:       "/tmp/d.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "d.jpg",
			Tamano:     800,
			Modificado: time.Unix(40, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				DHashImagen: "grupo-2",
			},
		},
	}

	for _, archivo := range archivos {
		if err := almacen.GuardarArchivo(context.Background(), archivo); err != nil {
			t.Fatalf("no se pudo guardar el archivo de prueba %q: %v", archivo.Ruta, err)
		}
	}

	grupos, err := almacen.ListarGruposDuplicados(context.Background(), modelo.CoincidenciaParcialImagen, modelo.CategoriaDuplicadosLocales, modelo.OrdenAlfabetico, 10, 0)
	if err != nil {
		t.Fatalf("no se pudieron listar los grupos duplicados: %v", err)
	}
	if len(grupos) != 2 {
		t.Fatalf("se esperaban 2 grupos de duplicados, se obtuvieron %d", len(grupos))
	}
	if len(grupos[0].Elementos) != 2 || len(grupos[1].Elementos) != 2 {
		t.Fatalf("cada grupo debería contener 2 elementos, se obtuvo %+v", grupos)
	}
	if grupos[0].Elementos[0].Ruta != "/tmp/a.jpg" || grupos[0].Elementos[1].Ruta != "/tmp/b.jpg" {
		t.Fatalf("el primer grupo no respetó el orden esperado por fecha: %+v", grupos[0].Elementos)
	}
	if grupos[1].Elementos[0].Ruta != "/tmp/c.jpg" || grupos[1].Elementos[1].Ruta != "/tmp/d.jpg" {
		t.Fatalf("el segundo grupo no respetó el orden esperado por fecha: %+v", grupos[1].Elementos)
	}
}
