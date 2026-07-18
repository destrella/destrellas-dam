package almacen

import (
	"context"

	"destrellas-dam/internal/modelo"
)

// Repositorio define la persistencia minima requerida por los servicios.
type Repositorio interface {
	Cerrar() error
	GuardarArchivo(ctx context.Context, archivo modelo.Archivo) error
	EliminarArchivo(ctx context.Context, ruta string) error
	ObtenerArchivoPorRuta(ctx context.Context, ruta string) (modelo.Archivo, error)
	ListarPalabrasClave(ctx context.Context, limite int) ([]string, error)
	ListarEtiquetas(ctx context.Context, limite int) ([]string, error)
	BuscarEtiquetas(ctx context.Context, consulta string, limite int) ([]string, error)
	ListarUbicaciones(ctx context.Context, limite int) ([]string, error)
	BuscarUbicaciones(ctx context.Context, consulta string, limite int) ([]string, error)
	ListarAsociacionesTexto(ctx context.Context, limite int) ([]modelo.AsociacionTexto, error)
	GuardarAsociacionTexto(ctx context.Context, id int64, originales, sugeridas []string) (modelo.AsociacionTexto, error)
	EliminarAsociacionTexto(ctx context.Context, id int64) error
	ListarUbicacionesGuardadas(ctx context.Context, limite int) ([]modelo.UbicacionGuardada, error)
	ListarUsosUbicacionGuardada(ctx context.Context, nombre string, limite int) ([]modelo.UsoUbicacionGuardada, error)
	GuardarRelacionUbicacion(ctx context.Context, origen, destino string) error
	EliminarRelacionUbicacion(ctx context.Context, origen string) error
	TieneArchivosConUbicacionSinNombre(ctx context.Context) (bool, error)
	ListarArchivosPorEtiqueta(ctx context.Context, etiqueta string, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error)
	ListarArchivosPorUbicacion(ctx context.Context, ubicacion string, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error)
	ListarArchivosSinUbicacionNombrada(ctx context.Context, filtros modelo.FiltrosListado, limite, offset int) ([]modelo.Archivo, error)
	ListarGruposDuplicados(ctx context.Context, tipo modelo.TipoCoincidencia, categoria modelo.CategoriaDuplicados, orden modelo.OrdenDuplicados, limite, offset int) ([]modelo.GrupoDuplicados, error)
	ObtenerEstadisticasDuplicados(ctx context.Context) (modelo.EstadisticasDuplicados, error)
}
