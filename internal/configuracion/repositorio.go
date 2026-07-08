package configuracion

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Repositorio administra la configuracion en disco.
type Repositorio struct {
	ruta string
}

// NuevoRepositorio construye un repositorio de configuracion.
func NuevoRepositorio(ruta string) *Repositorio {
	return &Repositorio{ruta: ruta}
}

// Cargar obtiene la configuracion persistida o crea una por defecto.
func (r *Repositorio) Cargar() (Configuracion, error) {
	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o755); err != nil {
		return Configuracion{}, fmt.Errorf("no se pudo preparar el directorio de configuracion: %w", err)
	}

	datos, err := os.ReadFile(r.ruta)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg, err := ConfiguracionPorDefecto()
			if err != nil {
				return Configuracion{}, err
			}
			if err := r.Guardar(cfg); err != nil {
				return Configuracion{}, err
			}
			return cfg, nil
		}
		return Configuracion{}, fmt.Errorf("no se pudo leer la configuracion: %w", err)
	}

	var cfg Configuracion
	if err := json.Unmarshal(datos, &cfg); err != nil {
		return Configuracion{}, fmt.Errorf("no se pudo interpretar la configuracion: %w", err)
	}

	return NormalizarConfiguracion(cfg)
}

// Guardar persiste la configuracion con formato legible.
func (r *Repositorio) Guardar(cfg Configuracion) error {
	cfg, err := NormalizarConfiguracion(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o755); err != nil {
		return fmt.Errorf("no se pudo preparar el directorio de configuracion: %w", err)
	}

	archivo, err := os.Create(r.ruta)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo de configuracion para escritura: %w", err)
	}
	defer archivo.Close()

	codificador := json.NewEncoder(archivo)
	codificador.SetIndent("", "\t")
	if err := codificador.Encode(cfg); err != nil {
		return fmt.Errorf("no se pudo guardar la configuracion: %w", err)
	}

	return nil
}
