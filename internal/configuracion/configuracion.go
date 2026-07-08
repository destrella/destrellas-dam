package configuracion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"destrellas-dam/internal/modelo"
)

const nombreAplicacion = "DEstrellasDAM"

// Configuracion agrupa las preferencias persistentes de la aplicacion.
type Configuracion struct {
	CarpetaInicial        string                `json:"carpeta_inicial"`
	CarpetaArchivado      string                `json:"carpeta_archivado"`
	FiltrosPorDefecto     modelo.FiltrosListado `json:"filtros_por_defecto"`
	ClaveAPIYandex        string                `json:"clave_api_yandex"`
	RutaBaseDatos         string                `json:"ruta_base_datos"`
	TamanoPaginaLocal     int                   `json:"tamano_pagina_local"`
	TamanoPaginaRemota    int                   `json:"tamano_pagina_remota"`
	ConcurrenciaIndexado  int                   `json:"concurrencia_indexado"`
	ConcurrenciaMetadatos int                   `json:"concurrencia_metadatos"`
	ConcurrenciaHashes    int                   `json:"concurrencia_hashes"`
}

// RutasAplicacion centraliza las ubicaciones persistentes de la app.
type RutasAplicacion struct {
	DirectorioBase  string
	ArchivoConfig   string
	ArchivoBD       string
	DirectorioCache string
}

// ResolverRutas determina las ubicaciones adecuadas segun el sistema.
func ResolverRutas() (RutasAplicacion, error) {
	directorioConfig, err := os.UserConfigDir()
	if err != nil {
		return RutasAplicacion{}, fmt.Errorf("no se pudo obtener el directorio de configuracion: %w", err)
	}

	directorioCache, err := os.UserCacheDir()
	if err != nil {
		return RutasAplicacion{}, fmt.Errorf("no se pudo obtener el directorio de cache: %w", err)
	}

	directorioBase := filepath.Join(directorioConfig, nombreAplicacion)
	return RutasAplicacion{
		DirectorioBase:  directorioBase,
		ArchivoConfig:   filepath.Join(directorioBase, "configuracion.json"),
		ArchivoBD:       filepath.Join(directorioBase, "catalogo.sqlite"),
		DirectorioCache: filepath.Join(directorioCache, nombreAplicacion),
	}, nil
}

// ConfiguracionPorDefecto construye una configuracion razonable para macOS y otros Unix.
func ConfiguracionPorDefecto() (Configuracion, error) {
	rutas, err := ResolverRutas()
	if err != nil {
		return Configuracion{}, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Configuracion{}, fmt.Errorf("no se pudo obtener la carpeta de usuario: %w", err)
	}

	cfg := Configuracion{
		CarpetaInicial:        home,
		CarpetaArchivado:      filepath.Join(home, "Archivado DAM"),
		FiltrosPorDefecto:     modelo.FiltrosPorDefecto(),
		ClaveAPIYandex:        "",
		RutaBaseDatos:         rutas.ArchivoBD,
		TamanoPaginaLocal:     120,
		TamanoPaginaRemota:    40,
		ConcurrenciaIndexado:  maximo(2, runtime.NumCPU()/2),
		ConcurrenciaMetadatos: maximo(1, runtime.NumCPU()/3),
		ConcurrenciaHashes:    maximo(1, runtime.NumCPU()/3),
	}

	return NormalizarConfiguracion(cfg)
}

// NormalizarConfiguracion aplica valores minimos y rutas faltantes.
func NormalizarConfiguracion(cfg Configuracion) (Configuracion, error) {
	rutas, err := ResolverRutas()
	if err != nil {
		return Configuracion{}, err
	}

	if cfg.CarpetaInicial == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Configuracion{}, err
		}
		cfg.CarpetaInicial = home
	}
	if cfg.CarpetaArchivado == "" {
		cfg.CarpetaArchivado = filepath.Join(cfg.CarpetaInicial, "Archivado DAM")
	}
	if cfg.RutaBaseDatos == "" {
		cfg.RutaBaseDatos = rutas.ArchivoBD
	}
	if cfg.TamanoPaginaLocal < 32 {
		cfg.TamanoPaginaLocal = 120
	}
	if cfg.TamanoPaginaRemota < 20 {
		cfg.TamanoPaginaRemota = 40
	}
	if cfg.ConcurrenciaIndexado < 1 {
		cfg.ConcurrenciaIndexado = maximo(1, runtime.NumCPU()/2)
	}
	if cfg.ConcurrenciaMetadatos < 1 {
		cfg.ConcurrenciaMetadatos = maximo(1, runtime.NumCPU()/3)
	}
	if cfg.ConcurrenciaHashes < 1 {
		cfg.ConcurrenciaHashes = maximo(1, runtime.NumCPU()/3)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.RutaBaseDatos), 0o755); err != nil {
		return Configuracion{}, fmt.Errorf("no se pudo crear el directorio de la base de datos: %w", err)
	}
	if err := os.MkdirAll(rutas.DirectorioCache, 0o755); err != nil {
		return Configuracion{}, fmt.Errorf("no se pudo crear el directorio de cache: %w", err)
	}

	return cfg, nil
}

func maximo(a, b int) int {
	if a > b {
		return a
	}
	return b
}
