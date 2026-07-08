package metadatos

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var expresionFechaHoraFlexible = regexp.MustCompile(`(?i)(\d{4})[:/-](\d{2})[:/-](\d{2})(?:[ t_](\d{2})[:._-]?(\d{2})(?:[:._-]?(\d{2}))?)?(?:\s*(z|[+-]\d{1,2}:?\d{0,2}))?`)

func extraerFechaHoraEditable(documento map[string]any) (fecha, hora, zona string) {
	// FileModifyDate se omite a propósito porque refleja cambios del sistema
	// de archivos y no la fecha real capturada o guardada en el multimedia.
	clavesFechaHora := []string{
		"DateTimeOriginal",
		"DateTimeDigitized",
		"CreateDate",
		"MediaCreateDate",
		"TrackCreateDate",
		"CreationDate",
		"ModifyDate",
	}
	zonaSecundaria := extraerCadena(
		documento,
		"OffsetTimeOriginal",
		"OffsetTimeDigitized",
		"OffsetTime",
		"TimeZone",
		"TimeZoneOffset",
	)

	for _, clave := range clavesFechaHora {
		for _, valor := range extraerValoresPorClave(documento, clave) {
			valor = strings.TrimSpace(valor)
			if valor == "" || esFechaHoraMetadatoVacia(valor) {
				continue
			}
			fecha, hora, zona = descomponerFechaHoraMetadato(valor, zonaSecundaria)
			if fecha != "" || hora != "" {
				return fecha, hora, zona
			}
		}
	}

	return "", "", normalizarZonaHoraria(zonaSecundaria)
}

func descomponerFechaHoraMetadato(texto, zonaSecundaria string) (fecha, hora, zona string) {
	texto = strings.TrimSpace(texto)
	zonaSecundaria = normalizarZonaHoraria(zonaSecundaria)
	if texto == "" || esFechaHoraMetadatoVacia(texto) {
		return "", "", zonaSecundaria
	}

	coincidencias := expresionFechaHoraFlexible.FindStringSubmatch(texto)
	if len(coincidencias) >= 4 {
		fecha = fmt.Sprintf("%s-%s-%s", coincidencias[1], coincidencias[2], coincidencias[3])
		if esFechaHoraMetadatoVacia(fecha) {
			return "", "", zonaSecundaria
		}
		if _, err := time.Parse("2006-01-02", fecha); err != nil {
			return "", "", zonaSecundaria
		}
		if coincidencias[4] != "" && coincidencias[5] != "" {
			segundos := "00"
			if coincidencias[6] != "" {
				segundos = coincidencias[6]
			}
			hora = fmt.Sprintf("%s:%s:%s", coincidencias[4], coincidencias[5], segundos)
			if _, err := time.Parse("15:04:05", hora); err != nil {
				hora = ""
			}
		}
		if coincidencias[7] != "" {
			zona = normalizarZonaHoraria(coincidencias[7])
		} else {
			zona = zonaSecundaria
		}
		return fecha, hora, zona
	}

	disenos := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006:01:02 15:04:05-07:00",
		"2006:01:02 15:04:05Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006-01-02",
	}
	for _, diseno := range disenos {
		instante, err := time.Parse(diseno, texto)
		if err != nil {
			continue
		}
		fecha = instante.Format("2006-01-02")
		if strings.Contains(diseno, "15:04:05") {
			hora = instante.Format("15:04:05")
		}
		if strings.Contains(diseno, "Z07:00") || strings.Contains(diseno, "-07:00") {
			zona = instante.Format("-07:00")
		} else {
			zona = zonaSecundaria
		}
		return fecha, hora, zona
	}

	return "", "", zonaSecundaria
}

func esFechaHoraMetadatoVacia(texto string) bool {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return true
	}

	tieneDigitos := false
	for _, caracter := range texto {
		if caracter < '0' || caracter > '9' {
			continue
		}
		tieneDigitos = true
		if caracter != '0' {
			return false
		}
	}
	return tieneDigitos
}

func construirFechaHoraExif(fecha, hora, zona string) string {
	fecha = strings.TrimSpace(fecha)
	hora = strings.TrimSpace(hora)
	zona = normalizarZonaHoraria(zona)
	if fecha == "" {
		return ""
	}

	partesFecha := strings.Split(fecha, "-")
	if len(partesFecha) != 3 {
		return ""
	}

	fechaExif := fmt.Sprintf("%s:%s:%s", partesFecha[0], partesFecha[1], partesFecha[2])
	if hora == "" {
		return fechaExif
	}

	if zona != "" {
		return fechaExif + " " + hora + zona
	}
	return fechaExif + " " + hora
}

func normalizarZonaHoraria(zona string) string {
	zona = strings.TrimSpace(strings.ToUpper(zona))
	if zona == "" {
		return ""
	}
	if zona == "Z" {
		return "+00:00"
	}

	expresion := regexp.MustCompile(`^([+-])(\d{1,2})(?::?(\d{2}))?$`)
	coincidencias := expresion.FindStringSubmatch(zona)
	if len(coincidencias) == 0 {
		return ""
	}

	horas := coincidencias[2]
	if len(horas) == 1 {
		horas = "0" + horas
	}
	minutos := coincidencias[3]
	if minutos == "" {
		minutos = "00"
	}

	return coincidencias[1] + horas + ":" + minutos
}

func extraerTextoCombinado(documento map[string]any, claves ...string) string {
	valores := make([]string, 0, len(claves))
	for _, clave := range claves {
		valores = append(valores, extraerValoresPorClave(documento, clave)...)
	}
	valores = normalizarLista(valores)
	if len(valores) == 0 {
		return ""
	}
	return strings.Join(valores, "\n")
}
