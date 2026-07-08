package metadatos

import (
	"fmt"
	"strings"
	"time"

	"github.com/zsefvlol/timezonemapper"
)

// InferirZonaHorariaDesdeGPS resuelve el offset UTC esperado para una fecha
// concreta a partir de coordenadas GPS.
func (s *Servicio) InferirZonaHorariaDesdeGPS(fecha, hora string, latitud, longitud float64) (string, error) {
	fecha = strings.TrimSpace(fecha)
	if fecha == "" {
		return "", fmt.Errorf("se requiere una fecha para inferir la zona horaria")
	}
	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		return "", fmt.Errorf("la fecha no es valida para inferir la zona horaria: %w", err)
	}

	hora = strings.TrimSpace(hora)
	if hora == "" {
		// Usamos mediodia para evitar ambigüedades tipicas de cambios DST de madrugada.
		hora = "12:00:00"
	} else {
		horaNormalizada, err := normalizarHoraParaZonaHoraria(hora)
		if err != nil {
			return "", err
		}
		hora = horaNormalizada
	}

	nombreZona := strings.TrimSpace(timezonemapper.LatLngToTimezoneString(latitud, longitud))
	if nombreZona == "" {
		return "", fmt.Errorf("no se pudo resolver la zona horaria de las coordenadas")
	}

	ubicacion, err := time.LoadLocation(nombreZona)
	if err != nil {
		return "", fmt.Errorf("no se pudo cargar la zona horaria %q: %w", nombreZona, err)
	}

	instante, err := time.ParseInLocation("2006-01-02 15:04:05", fecha+" "+hora, ubicacion)
	if err != nil {
		return "", fmt.Errorf("no se pudo calcular el offset de la zona horaria %q: %w", nombreZona, err)
	}
	return instante.Format("-07:00"), nil
}

func normalizarHoraParaZonaHoraria(hora string) (string, error) {
	disenos := []string{"15:04:05", "15:04"}
	for _, diseno := range disenos {
		instante, err := time.Parse(diseno, hora)
		if err == nil {
			return instante.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("la hora no es valida para inferir la zona horaria")
}
