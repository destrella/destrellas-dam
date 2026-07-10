package indexador

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"destrellas-dam/internal/almacen"
	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/plataforma"
)

// NodoDirectorio alimenta el arbol lateral.
type NodoDirectorio struct {
	Ruta       string
	Nombre     string
	TieneHijos bool
}

// SesionListado entrega lotes de elementos sin cargar toda la carpeta a memoria.
type SesionListado struct {
	repo          almacen.Repositorio
	ruta          string
	filtros       modelo.FiltrosListado
	archivos      []modelo.Archivo
	indiceArchivo int
	finalizado    bool
	agotado       chan struct{}
	cancelar      context.CancelFunc
	canalArchivos chan modelo.Archivo
	canalErrores  chan error
	mu            sync.Mutex
}

// ListadorLocal ofrece sesiones paginadas y un arbol de directorios ligero.
type ListadorLocal struct {
	repo almacen.Repositorio
}

// NuevoListadorLocal crea un listador reutilizable.
func NuevoListadorLocal(repo almacen.Repositorio) *ListadorLocal {
	return &ListadorLocal{repo: repo}
}

// NuevaSesion crea una sesion incremental para la carpeta indicada.
func (l *ListadorLocal) NuevaSesion(ctx context.Context, ruta string, filtros modelo.FiltrosListado) (*SesionListado, error) {
	if filtros.Recursivo {
		ctxInterno, cancelar := context.WithCancel(ctx)
		sesion := &SesionListado{
			repo:          l.repo,
			ruta:          ruta,
			filtros:       filtros,
			agotado:       make(chan struct{}),
			cancelar:      cancelar,
			canalArchivos: make(chan modelo.Archivo, 256),
			canalErrores:  make(chan error, 1),
		}
		go sesion.recorrerRecursivo(ctxInterno)
		return sesion, nil
	}

	archivos, err := leerArchivosDirectorioOrdenados(ruta, filtros)
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar el listado de la carpeta %q: %w", ruta, err)
	}

	return &SesionListado{
		repo:         l.repo,
		ruta:         ruta,
		filtros:      filtros,
		archivos:     archivos,
		agotado:      make(chan struct{}),
		canalErrores: make(chan error, 1),
	}, nil
}

// ListarSubdirectorios obtiene los directorios inmediatos para el arbol lateral.
func (l *ListadorLocal) ListarSubdirectorios(_ context.Context, ruta string, mostrarOcultos bool) ([]NodoDirectorio, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la carpeta %q: %w", ruta, err)
	}
	defer archivo.Close()

	var nodos []NodoDirectorio
	for {
		entradas, err := archivo.ReadDir(256)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("no se pudieron leer subdirectorios de %q: %w", ruta, err)
		}
		for _, entrada := range entradas {
			nombre := entrada.Name()
			rutaCompleta := filepath.Join(ruta, nombre)
			if !esDirectorioRecorrible(nombre, rutaCompleta, entrada.IsDir(), mostrarOcultos) {
				continue
			}
			nodos = append(nodos, NodoDirectorio{
				Ruta:       rutaCompleta,
				Nombre:     nombre,
				TieneHijos: true,
			})
		}
		if err == io.EOF {
			break
		}
	}

	sort.SliceStable(nodos, func(i, j int) bool {
		return compararNombre(nodos[i].Nombre, nodos[j].Nombre)
	})

	return nodos, nil
}

// Siguiente devuelve el proximo lote aceptado por los filtros.
func (s *SesionListado) Siguiente(ctx context.Context, limite int) ([]modelo.Archivo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limite < 1 {
		limite = 64
	}
	if s.finalizado {
		return nil, true, nil
	}

	if s.archivos != nil {
		return s.siguienteNoRecursivo(ctx, limite)
	}
	return s.siguienteRecursivo(ctx, limite)
}

// Cerrar libera la sesion y cualquier goroutine asociada.
func (s *SesionListado) Cerrar() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelar != nil {
		s.cancelar()
		s.cancelar = nil
	}
	return nil
}

func (s *SesionListado) siguienteNoRecursivo(ctx context.Context, limite int) ([]modelo.Archivo, bool, error) {
	var lote []modelo.Archivo

	for len(lote) < limite {
		select {
		case <-ctx.Done():
			return lote, false, ctx.Err()
		default:
		}

		for len(lote) < limite && s.indiceArchivo < len(s.archivos) {
			archivo := s.archivos[s.indiceArchivo]
			s.indiceArchivo++
			if !s.filtros.Acepta(archivo) {
				continue
			}
			lote = append(lote, s.enriquecerSiExiste(ctx, archivo))
		}

		if s.indiceArchivo >= len(s.archivos) {
			s.finalizado = true
			break
		}
	}

	return lote, s.finalizado, nil
}

func (s *SesionListado) siguienteRecursivo(ctx context.Context, limite int) ([]modelo.Archivo, bool, error) {
	var lote []modelo.Archivo
	for len(lote) < limite {
		select {
		case <-ctx.Done():
			return lote, false, ctx.Err()
		case err := <-s.canalErrores:
			if err != nil {
				s.finalizado = true
				return lote, true, err
			}
		case archivo, ok := <-s.canalArchivos:
			if !ok {
				s.finalizado = true
				return lote, true, nil
			}
			lote = append(lote, s.enriquecerSiExiste(ctx, archivo))
		default:
			if len(lote) > 0 {
				return lote, false, nil
			}
			select {
			case <-ctx.Done():
				return lote, false, ctx.Err()
			case err := <-s.canalErrores:
				if err != nil {
					s.finalizado = true
					return lote, true, err
				}
			case archivo, ok := <-s.canalArchivos:
				if !ok {
					s.finalizado = true
					return lote, true, nil
				}
				lote = append(lote, s.enriquecerSiExiste(ctx, archivo))
			}
		}
	}
	return lote, false, nil
}

func (s *SesionListado) recorrerRecursivo(ctx context.Context) {
	defer close(s.canalArchivos)
	defer close(s.agotado)

	pendientes := []string{s.ruta}
	for len(pendientes) > 0 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		actual := pendientes[len(pendientes)-1]
		pendientes = pendientes[:len(pendientes)-1]

		elementos, err := leerArchivosDirectorioOrdenados(actual, s.filtros)
		if err != nil {
			s.enviarError(fmt.Errorf("no se pudo recorrer %q: %w", actual, err))
			continue
		}

		var subdirectorios []string
		for _, elemento := range elementos {
			if debeOmitirDirectorioOculto(elemento.NombreVisible(), elemento.EsDirectorio, s.filtros.MostrarOcultos) {
				continue
			}
			if elemento.EsDirectorio {
				subdirectorios = append(subdirectorios, elemento.Ruta)
			}
			if s.filtros.Acepta(elemento) {
				select {
				case <-ctx.Done():
					return
				case s.canalArchivos <- elemento:
				}
			}
		}
		for indice := len(subdirectorios) - 1; indice >= 0; indice-- {
			pendientes = append(pendientes, subdirectorios[indice])
		}
	}
}

func (s *SesionListado) enriquecerSiExiste(ctx context.Context, archivo modelo.Archivo) modelo.Archivo {
	if s.repo == nil || archivo.Ruta == "" {
		return archivo
	}

	enriquecido, err := s.repo.ObtenerArchivoPorRuta(ctx, archivo.Ruta)
	if err != nil {
		return archivo
	}

	return fusionarArchivoCatalogoConSistema(archivo, enriquecido)
}

func (s *SesionListado) enviarError(err error) {
	select {
	case s.canalErrores <- err:
	default:
	}
}

func construirArchivoDesdeEntrada(rutaPadre string, entrada fs.DirEntry) (modelo.Archivo, error) {
	info, err := entrada.Info()
	if err != nil {
		return modelo.Archivo{}, err
	}

	ruta := filepath.Join(rutaPadre, entrada.Name())
	esDirectorio := entrada.IsDir() && !plataforma.EsBundle(ruta)
	return modelo.Archivo{
		Origen:       modelo.OrigenLocal,
		Ruta:         ruta,
		RutaPadre:    rutaPadre,
		Nombre:       entrada.Name(),
		Tamano:       info.Size(),
		Modificado:   info.ModTime(),
		Tipo:         modelo.TipoDesdeRuta(ruta, esDirectorio),
		EsOculto:     modelo.EsOcultoPorNombre(entrada.Name()),
		EsDirectorio: esDirectorio,
	}, nil
}

func leerArchivosDirectorioOrdenados(ruta string, filtros modelo.FiltrosListado) ([]modelo.Archivo, error) {
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return nil, err
	}

	archivos := make([]modelo.Archivo, 0, len(entradas))
	for _, entrada := range entradas {
		archivo, err := construirArchivoDesdeEntrada(ruta, entrada)
		if err != nil {
			continue
		}
		archivos = append(archivos, archivo)
	}

	ordenarArchivosSegunFiltros(archivos, filtros)
	return archivos, nil
}

func debeOmitirDirectorioOculto(nombre string, esDirectorio bool, mostrarOcultos bool) bool {
	return esDirectorio && !mostrarOcultos && modelo.EsOcultoPorNombre(nombre)
}

func esDirectorioRecorrible(nombre, ruta string, esDirectorio bool, mostrarOcultos bool) bool {
	if !esDirectorio || plataforma.EsBundle(ruta) {
		return false
	}
	if !mostrarOcultos && modelo.EsOcultoPorNombre(nombre) {
		return false
	}
	return true
}

func maximo(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func compararNombre(izquierda, derecha string) bool {
	return compararTextoNormalizado(izquierda, derecha) < 0
}

func compararNombreSegunOrden(izquierda, derecha string, descendente bool) bool {
	comparacion := compararTextoNormalizado(izquierda, derecha)
	if descendente {
		comparacion = -comparacion
	}
	return comparacion < 0
}

func ordenarArchivosSegunFiltros(archivos []modelo.Archivo, filtros modelo.FiltrosListado) {
	sort.SliceStable(archivos, func(i, j int) bool {
		return compararArchivosSegunFiltros(archivos[i], archivos[j], filtros) < 0
	})
}

func compararArchivosSegunFiltros(izquierda, derecha modelo.Archivo, filtros modelo.FiltrosListado) int {
	if filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenFechaModificacion {
		comparacion := compararInstantes(izquierda.Modificado, derecha.Modificado)
		if filtros.OrdenDescendente {
			comparacion = -comparacion
		}
		if comparacion != 0 {
			return comparacion
		}
	}

	comparacionNombre := compararTextoNormalizado(izquierda.NombreVisible(), derecha.NombreVisible())
	if filtros.CriterioOrdenNormalizado() == modelo.CriterioOrdenNombre && filtros.OrdenDescendente {
		comparacionNombre = -comparacionNombre
	}
	if comparacionNombre != 0 {
		return comparacionNombre
	}

	return compararTextoNormalizado(izquierda.Ruta, derecha.Ruta)
}

func compararTextoNormalizado(izquierda, derecha string) int {
	izquierdaNormalizada := strings.ToLower(strings.TrimSpace(izquierda))
	derechaNormalizada := strings.ToLower(strings.TrimSpace(derecha))
	switch {
	case izquierdaNormalizada < derechaNormalizada:
		return -1
	case izquierdaNormalizada > derechaNormalizada:
		return 1
	}

	izquierdaAjustada := strings.TrimSpace(izquierda)
	derechaAjustada := strings.TrimSpace(derecha)
	switch {
	case izquierdaAjustada < derechaAjustada:
		return -1
	case izquierdaAjustada > derechaAjustada:
		return 1
	default:
		return 0
	}
}

func compararInstantes(izquierda, derecha time.Time) int {
	switch {
	case izquierda.Before(derecha):
		return -1
	case izquierda.After(derecha):
		return 1
	default:
		return 0
	}
}

func fusionarArchivoCatalogoConSistema(archivoSistema, archivoCatalogo modelo.Archivo) modelo.Archivo {
	if archivoCatalogo.Ruta == "" {
		return archivoSistema
	}

	archivoCatalogo.Origen = archivoSistema.Origen
	archivoCatalogo.Ruta = archivoSistema.Ruta
	archivoCatalogo.RutaPadre = archivoSistema.RutaPadre
	archivoCatalogo.Nombre = archivoSistema.Nombre
	archivoCatalogo.Tamano = archivoSistema.Tamano
	archivoCatalogo.Modificado = archivoSistema.Modificado
	archivoCatalogo.Tipo = archivoSistema.Tipo
	archivoCatalogo.EsOculto = archivoSistema.EsOculto
	archivoCatalogo.EsDirectorio = archivoSistema.EsDirectorio

	return archivoCatalogo
}
