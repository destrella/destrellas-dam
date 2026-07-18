package sqlite

import (
	"context"
	"fmt"
	"os"
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

func TestListarGruposDuplicadosOmiteArchivosVacios(t *testing.T) {
	t.Parallel()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	defer almacen.Cerrar()

	archivos := []modelo.Archivo{
		{
			Ruta:       "/tmp/vacio-a.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "vacio-a.jpg",
			Tamano:     0,
			Modificado: time.Unix(10, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				SHA256: "grupo-vacio",
				MD5:    "grupo-vacio",
			},
		},
		{
			Ruta:       "/tmp/vacio-b.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "vacio-b.jpg",
			Tamano:     0,
			Modificado: time.Unix(20, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				SHA256: "grupo-vacio",
				MD5:    "grupo-vacio",
			},
		},
		{
			Ruta:       "/tmp/lleno-a.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "lleno-a.jpg",
			Tamano:     1200,
			Modificado: time.Unix(30, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				SHA256: "grupo-lleno",
				MD5:    "grupo-lleno",
			},
		},
		{
			Ruta:       "/tmp/lleno-b.jpg",
			RutaPadre:  "/tmp",
			Nombre:     "lleno-b.jpg",
			Tamano:     1400,
			Modificado: time.Unix(40, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				SHA256: "grupo-lleno",
				MD5:    "grupo-lleno",
			},
		},
	}

	for _, archivo := range archivos {
		if err := almacen.GuardarArchivo(context.Background(), archivo); err != nil {
			t.Fatalf("no se pudo guardar el archivo de prueba %q: %v", archivo.Ruta, err)
		}
	}

	grupos, err := almacen.ListarGruposDuplicados(context.Background(), modelo.CoincidenciaExacta, modelo.CategoriaDuplicadosLocales, modelo.OrdenAlfabetico, 10, 0)
	if err != nil {
		t.Fatalf("no se pudieron listar los grupos duplicados exactos: %v", err)
	}
	if len(grupos) != 1 {
		t.Fatalf("se esperaba un único grupo válido sin archivos vacíos, se obtuvieron %d", len(grupos))
	}
	if grupos[0].Clave != "grupo-lleno|grupo-lleno" {
		t.Fatalf("se obtuvo un grupo inesperado: %q", grupos[0].Clave)
	}
	if len(grupos[0].Elementos) != 2 {
		t.Fatalf("el grupo válido debería contener 2 elementos, se obtuvo %+v", grupos[0].Elementos)
	}
}

func TestLimpiarRegistrosLocalesAusentesEliminaEntradasSinArchivo(t *testing.T) {
	t.Parallel()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	defer almacen.Cerrar()

	directorio := t.TempDir()
	rutaExistente := filepath.Join(directorio, "foto-existente.jpg")
	rutaAusente := filepath.Join(directorio, "foto-ausente.jpg")
	if err := os.WriteFile(rutaExistente, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no se pudo crear el archivo existente de prueba: %v", err)
	}

	archivos := []modelo.Archivo{
		{
			Ruta:       rutaExistente,
			RutaPadre:  directorio,
			Nombre:     filepath.Base(rutaExistente),
			Tamano:     1200,
			Modificado: time.Unix(10, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				MD5: "grupo-prueba",
			},
		},
		{
			Ruta:       rutaAusente,
			RutaPadre:  directorio,
			Nombre:     filepath.Base(rutaAusente),
			Tamano:     1200,
			Modificado: time.Unix(20, 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Hashes: modelo.HashesArchivo{
				MD5: "grupo-prueba",
			},
		},
	}
	for _, archivo := range archivos {
		if err := almacen.GuardarArchivo(context.Background(), archivo); err != nil {
			t.Fatalf("no se pudo guardar el archivo de prueba %q: %v", archivo.Ruta, err)
		}
	}

	eliminados, err := almacen.LimpiarRegistrosLocalesAusentes(context.Background())
	if err != nil {
		t.Fatalf("no se pudieron depurar los registros locales ausentes: %v", err)
	}
	if eliminados != 1 {
		t.Fatalf("se esperaba depurar 1 ruta ausente, se obtuvieron %d", eliminados)
	}

	if _, err := almacen.ObtenerArchivoPorRuta(context.Background(), rutaAusente); err == nil {
		t.Fatal("la ruta ausente debería haberse eliminado del catálogo")
	}
	if _, err := almacen.ObtenerArchivoPorRuta(context.Background(), rutaExistente); err != nil {
		t.Fatalf("la ruta existente debería permanecer en el catálogo: %v", err)
	}

	grupos, err := almacen.ListarGruposDuplicados(context.Background(), modelo.CoincidenciaExacta, modelo.CategoriaDuplicadosLocales, modelo.OrdenAlfabetico, 10, 0)
	if err != nil {
		t.Fatalf("no se pudieron listar los grupos duplicados tras depurar: %v", err)
	}
	if len(grupos) != 0 {
		t.Fatalf("no deberían quedar grupos locales tras eliminar la ruta ausente, se obtuvo %+v", grupos)
	}
}

func TestBuscarEtiquetasEncuentraCoincidenciasFueraDelLimiteVisible(t *testing.T) {
	t.Parallel()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	defer almacen.Cerrar()

	var palabras []string
	for indice := 1; indice <= 205; indice++ {
		palabras = append(palabras, fmt.Sprintf("etiqueta-%03d", indice))
	}
	palabras = append(palabras, "zzz-objetivo")

	if err := almacen.GuardarArchivo(context.Background(), modelo.Archivo{
		Ruta:       "/tmp/etiquetas.jpg",
		RutaPadre:  "/tmp",
		Nombre:     "etiquetas.jpg",
		Tamano:     1024,
		Modificado: time.Unix(10, 0),
		Origen:     modelo.OrigenLocal,
		Tipo:       modelo.TipoImagen,
		Metadatos: modelo.MetadatosArchivo{
			PalabrasClave: append([]string(nil), palabras...),
			Sujetos:       append([]string(nil), palabras...),
		},
	}); err != nil {
		t.Fatalf("no se pudo guardar el archivo de etiquetas: %v", err)
	}

	listadoVisible, err := almacen.ListarEtiquetas(context.Background(), 200)
	if err != nil {
		t.Fatalf("no se pudieron listar las etiquetas: %v", err)
	}
	for _, etiqueta := range listadoVisible {
		if etiqueta == "zzz-objetivo" {
			t.Fatal("la etiqueta objetivo no debería entrar en el listado limitado de 200 elementos")
		}
	}

	resultados, err := almacen.BuscarEtiquetas(context.Background(), "objetivo", 20)
	if err != nil {
		t.Fatalf("no se pudieron buscar las etiquetas: %v", err)
	}
	if len(resultados) != 1 || resultados[0] != "zzz-objetivo" {
		t.Fatalf("resultado de búsqueda inesperado: %+v", resultados)
	}
}

func TestBuscarUbicacionesEncuentraCoincidenciasFueraDelLimiteVisible(t *testing.T) {
	t.Parallel()

	rutaBase := filepath.Join(t.TempDir(), "catalogo.sqlite")
	almacen, err := Nuevo(rutaBase)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén sqlite de prueba: %v", err)
	}
	defer almacen.Cerrar()

	for indice := 1; indice <= 205; indice++ {
		nombre := fmt.Sprintf("Lugar %03d", indice)
		if err := almacen.GuardarArchivo(context.Background(), modelo.Archivo{
			Ruta:       fmt.Sprintf("/tmp/lugar-%03d.jpg", indice),
			RutaPadre:  "/tmp",
			Nombre:     fmt.Sprintf("lugar-%03d.jpg", indice),
			Tamano:     1024,
			Modificado: time.Unix(int64(indice), 0),
			Origen:     modelo.OrigenLocal,
			Tipo:       modelo.TipoImagen,
			Metadatos: modelo.MetadatosArchivo{
				Ubicacion: nombre,
			},
		}); err != nil {
			t.Fatalf("no se pudo guardar la ubicación %q: %v", nombre, err)
		}
	}

	if err := almacen.GuardarArchivo(context.Background(), modelo.Archivo{
		Ruta:       "/tmp/objetivo.jpg",
		RutaPadre:  "/tmp",
		Nombre:     "objetivo.jpg",
		Tamano:     1024,
		Modificado: time.Unix(500, 0),
		Origen:     modelo.OrigenLocal,
		Tipo:       modelo.TipoImagen,
		Metadatos: modelo.MetadatosArchivo{
			Ubicacion: "ZZZ Palacio nuevo",
		},
	}); err != nil {
		t.Fatalf("no se pudo guardar la ubicación objetivo: %v", err)
	}

	listadoVisible, err := almacen.ListarUbicaciones(context.Background(), 200)
	if err != nil {
		t.Fatalf("no se pudieron listar las ubicaciones: %v", err)
	}
	for _, ubicacion := range listadoVisible {
		if ubicacion == "ZZZ Palacio nuevo" {
			t.Fatal("la ubicación objetivo no debería entrar en el listado limitado de 200 elementos")
		}
	}

	resultados, err := almacen.BuscarUbicaciones(context.Background(), "Palacio", 20)
	if err != nil {
		t.Fatalf("no se pudieron buscar las ubicaciones: %v", err)
	}
	if len(resultados) != 1 || resultados[0] != "ZZZ Palacio nuevo" {
		t.Fatalf("resultado de búsqueda inesperado: %+v", resultados)
	}
}
