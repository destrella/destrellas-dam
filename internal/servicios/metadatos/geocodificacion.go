package metadatos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DireccionGPS resume la ubicacion nominal devuelta por la geocodificacion inversa.
type DireccionGPS struct {
	Ciudad string
	Estado string
	Pais   string
}

// ResolverDireccionGPS consulta Nominatim para completar ciudad, estado y pais.
func (s *Servicio) ResolverDireccionGPS(ctx context.Context, latitud, longitud float64) (DireccionGPS, error) {
	consulta := url.Values{}
	consulta.Set("format", "jsonv2")
	consulta.Set("addressdetails", "1")
	consulta.Set("lat", fmt.Sprintf("%.8f", latitud))
	consulta.Set("lon", fmt.Sprintf("%.8f", longitud))

	recurso := "https://nominatim.openstreetmap.org/reverse?" + consulta.Encode()
	ctxConsulta, cancelar := context.WithTimeout(ctx, 12*time.Second)
	defer cancelar()

	solicitud, err := http.NewRequestWithContext(ctxConsulta, http.MethodGet, recurso, nil)
	if err != nil {
		return DireccionGPS{}, fmt.Errorf("no se pudo construir la solicitud a Nominatim: %w", err)
	}
	solicitud.Header.Set("User-Agent", "destrellas-dam/1.0 (+local)")
	solicitud.Header.Set("Accept", "application/json")

	respuesta, err := http.DefaultClient.Do(solicitud)
	if err != nil {
		return DireccionGPS{}, fmt.Errorf("no se pudo consultar Nominatim: %w", err)
	}
	defer respuesta.Body.Close()

	if respuesta.StatusCode < 200 || respuesta.StatusCode >= 300 {
		return DireccionGPS{}, fmt.Errorf("Nominatim respondio con estado %s", respuesta.Status)
	}

	var carga struct {
		Address map[string]any `json:"address"`
	}
	if err := json.NewDecoder(respuesta.Body).Decode(&carga); err != nil {
		return DireccionGPS{}, fmt.Errorf("no se pudo decodificar la respuesta de Nominatim: %w", err)
	}

	direccion := DireccionGPS{
		Ciudad: primerValorDireccion(carga.Address, "city", "town", "village", "municipality", "hamlet", "county"),
		Estado: primerValorDireccion(carga.Address, "state", "province", "region"),
		Pais:   primerValorDireccion(carga.Address, "country"),
	}
	return direccion, nil
}

func primerValorDireccion(direccion map[string]any, claves ...string) string {
	for _, clave := range claves {
		valor, existe := direccion[clave]
		if !existe {
			continue
		}
		texto := strings.TrimSpace(fmt.Sprint(valor))
		if texto != "" && texto != "<nil>" {
			return texto
		}
	}
	return ""
}
