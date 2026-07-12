package duplicados

import (
	"context"
	"path/filepath"

	"destrellas-dam/internal/almacen"
	"destrellas-dam/internal/configuracion"
	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/servicios/indexador"
)

// Servicio ofrece un punto de entrada claro para la vista de duplicados.
type Servicio struct {
	repo             almacen.Repositorio
	indexador        *indexador.Servicio
	rutaEstadoRemoto string
}

// NuevoServicio crea el servicio de duplicados.
func NuevoServicio(repo almacen.Repositorio, idx *indexador.Servicio) *Servicio {
	rutaEstadoRemoto := ""
	if rutas, err := configuracion.ResolverRutas(); err == nil {
		rutaEstadoRemoto = filepath.Join(rutas.DirectorioBase, "duplicados-remotos.json")
	}
	return &Servicio{
		repo:             repo,
		indexador:        idx,
		rutaEstadoRemoto: rutaEstadoRemoto,
	}
}

// IniciarDescubrimientoLocal lanza un escaneo completo con hashes y parciales multimedia.
func (s *Servicio) IniciarDescubrimientoLocal(ctx context.Context, raiz string, rutasExcluidas []string) <-chan indexador.EventoProgreso {
	return s.indexador.Descubrir(ctx, raiz, indexador.OpcionesDescubrimiento{
		CalcularMetadatos:       true,
		CalcularHashesExactos:   true,
		CalcularHashesParciales: true,
		SoloMultimedia:          true,
		IgnorarArchivosVacios:   true,
		RutasExcluidas:          append([]string(nil), rutasExcluidas...),
	})
}

// ListarGrupos consulta la base persistente.
func (s *Servicio) ListarGrupos(ctx context.Context, tipo modelo.TipoCoincidencia, categoria modelo.CategoriaDuplicados, orden modelo.OrdenDuplicados, limite, offset int) ([]modelo.GrupoDuplicados, error) {
	return s.repo.ListarGruposDuplicados(ctx, tipo, categoria, orden, limite, offset)
}

// Estadisticas devuelve el resumen lateral.
func (s *Servicio) Estadisticas(ctx context.Context) (modelo.EstadisticasDuplicados, error) {
	return s.repo.ObtenerEstadisticasDuplicados(ctx)
}
