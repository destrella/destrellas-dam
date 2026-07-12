package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"destrellas-dam/internal/modelo"
	serviciometadatos "destrellas-dam/internal/servicios/metadatos"
)

type estadoFormularioMetadatos struct {
	Ruta                      string
	FechaSugerida             string
	FechaSugeridaActiva       bool
	HoraSugerida              string
	HoraSugeridaActiva        bool
	ZonaHorariaSugerida       string
	ZonaHorariaSugeridaActiva bool
	PalabrasSugeridas         []string
	CopyrightSugerido         string
	CopyrightSugeridoActivo   bool
	MakeSugerido              string
	MakeSugeridoActivo        bool
	ModeloSugerido            string
	ModeloSugeridoActivo      bool
	SoftwareSugerido          string
	SoftwareSugeridoActivo    bool
	OrientacionExpandida      bool
	CalendarioExpandido       bool
	SeleccionOrientacion      string
	opcionesOrientacion       map[string]*widget.Clickable
	opcionesCalendario        map[string]*widget.Clickable
	MesCalendario             time.Time
	listaAtributoExtendido    widget.List
	listaUbicacionesSugeridas widget.List
	botonSelectorOrientacion  widget.Clickable
	botonSelectorFecha        widget.Clickable
	botonCalendarioAnterior   widget.Clickable
	botonCalendarioSiguiente  widget.Clickable
	SalidaExiftoolRuta        string
	SalidaExiftoolTexto       string
	SalidaExiftoolCargando    bool
	SalidaExiftoolExpandida   bool
	listaSalidaExiftool       widget.List
	botonAlternarExiftool     widget.Clickable
}

type opcionOrientacionUI struct {
	Clave       string
	Etiqueta    string
	Orientacion int
	Rotacion    int
}

type diaCalendarioUI struct {
	Fecha       time.Time
	EnMesActivo bool
}

var (
	expresionFechaHoraNombre = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\d{4})(\d{2})(\d{2})[ _-]?(\d{2})(\d{2})(\d{2})(z|[+-]\d{2}:?\d{2})?`),
		regexp.MustCompile(`(?i)(\d{4})[-_](\d{2})[-_](\d{2})[ t_-](\d{2})[._-](\d{2})[._-](\d{2})(z|[+-]\d{2}:?\d{2})?`),
		regexp.MustCompile(`(?i)screenshot[ _-]?(\d{4})[-_](\d{2})[-_](\d{2})[ _-]at[ _-](\d{2})[._-](\d{2})[._-](\d{2})`),
	}
	expresionZonaHorariaEditable = regexp.MustCompile(`^(?i:z|[+-]\d{1,2}:?\d{2})$`)
	expresionPartesZonaHoraria   = regexp.MustCompile(`^([+-])(\d{1,2}):?(\d{2})$`)
	mesesCalendarioEspanol       = []string{
		"Enero",
		"Febrero",
		"Marzo",
		"Abril",
		"Mayo",
		"Junio",
		"Julio",
		"Agosto",
		"Septiembre",
		"Octubre",
		"Noviembre",
		"Diciembre",
	}
	diasSemanaCalendario = []string{"L", "M", "X", "J", "V", "S", "D"}
)

func (a *Aplicacion) sincronizarEditoresMetadatos(archivo modelo.Archivo) {
	rutaAnterior := a.formularioMetadatos.Ruta
	a.formularioMetadatos.Ruta = archivo.Ruta
	a.formularioMetadatos.OrientacionExpandida = false
	a.formularioMetadatos.PalabrasSugeridas = nil
	a.formularioMetadatos.FechaSugerida = ""
	a.formularioMetadatos.FechaSugeridaActiva = false
	a.formularioMetadatos.HoraSugerida = ""
	a.formularioMetadatos.HoraSugeridaActiva = false
	a.formularioMetadatos.ZonaHorariaSugerida = ""
	a.formularioMetadatos.ZonaHorariaSugeridaActiva = false
	a.formularioMetadatos.CopyrightSugerido = ""
	a.formularioMetadatos.CopyrightSugeridoActivo = false
	a.formularioMetadatos.MakeSugerido = ""
	a.formularioMetadatos.MakeSugeridoActivo = false
	a.formularioMetadatos.ModeloSugerido = ""
	a.formularioMetadatos.ModeloSugeridoActivo = false
	a.formularioMetadatos.SoftwareSugerido = ""
	a.formularioMetadatos.SoftwareSugeridoActivo = false
	a.formularioMetadatos.SeleccionOrientacion = claveOrientacionArchivo(archivo)
	if rutaAnterior != archivo.Ruta {
		a.formularioMetadatos.SalidaExiftoolExpandida = false
		a.formularioMetadatos.CalendarioExpandido = false
	}

	fecha := strings.TrimSpace(archivo.Metadatos.Fecha)
	hora := strings.TrimSpace(archivo.Metadatos.Hora)
	zona := strings.TrimSpace(archivo.Metadatos.ZonaHoraria)
	if fecha == "" || hora == "" || zona == "" {
		if fechaInferida, horaInferida, zonaInferida, ok := inferirFechaHoraDesdeNombre(archivo.NombreVisible()); ok {
			if fecha == "" && fechaInferida != "" {
				fecha = fechaInferida
				a.formularioMetadatos.FechaSugerida = fechaInferida
				a.formularioMetadatos.FechaSugeridaActiva = true
			}
			if hora == "" && horaInferida != "" {
				hora = horaInferida
				a.formularioMetadatos.HoraSugerida = horaInferida
				a.formularioMetadatos.HoraSugeridaActiva = true
			}
			if zona == "" && zonaInferida != "" {
				zona = zonaInferida
				a.formularioMetadatos.ZonaHorariaSugerida = zonaInferida
				a.formularioMetadatos.ZonaHorariaSugeridaActiva = true
			}
		}
	}

	if zona == "" && archivo.Metadatos.Coordenadas != nil && a.servicioMetadatos != nil {
		zonaInferidaGPS, err := a.servicioMetadatos.InferirZonaHorariaDesdeGPS(fecha, hora, archivo.Metadatos.Coordenadas.Latitud, archivo.Metadatos.Coordenadas.Longitud)
		if err == nil && zonaInferidaGPS != "" {
			zona = zonaInferidaGPS
			a.formularioMetadatos.ZonaHorariaSugerida = zonaInferidaGPS
			a.formularioMetadatos.ZonaHorariaSugeridaActiva = true
		}
	}

	a.editorFecha.SetText(fecha)
	a.editorHora.SetText(hora)
	a.editorZonaHoraria.SetText(zona)
	a.sincronizarCalendarioFecha(fecha)
	a.editorComentario.SetText(strings.TrimSpace(archivo.Metadatos.Comentario))

	palabrasBase := normalizarListaCSV(append(
		partirListaCSV(strings.Join(archivo.Metadatos.Sujetos, ", ")),
		partirListaCSV(strings.Join(archivo.Metadatos.PalabrasClave, ", "))...,
	))
	sugeridas := []string(nil)
	if len(palabrasBase) == 0 {
		sugeridas = sugerenciasPalabrasArchivo(archivo, palabrasBase)
	}
	a.formularioMetadatos.PalabrasSugeridas = sugeridas
	palabrasVisibles := append([]string(nil), palabrasBase...)
	palabrasVisibles = append(palabrasVisibles, sugeridas...)
	a.editorPalabras.SetText(strings.Join(normalizarListaCSV(palabrasVisibles), ", "))

	a.editorUbicacion.SetText(strings.TrimSpace(archivo.Metadatos.Ubicacion))

	copyright := strings.TrimSpace(archivo.Metadatos.Copyright)
	if copyright == "" {
		copyright = inferirCopyrightSugerido(archivo, fecha)
		a.formularioMetadatos.CopyrightSugerido = copyright
		a.formularioMetadatos.CopyrightSugeridoActivo = copyright != ""
	}
	a.editorCopyright.SetText(copyright)

	if archivo.Metadatos.Coordenadas != nil {
		a.editorGPSLatitud.SetText(fmt.Sprintf("%.8f", archivo.Metadatos.Coordenadas.Latitud))
		a.editorGPSLongitud.SetText(fmt.Sprintf("%.8f", archivo.Metadatos.Coordenadas.Longitud))
	} else {
		a.editorGPSLatitud.SetText("")
		a.editorGPSLongitud.SetText("")
	}

	makeTexto := strings.TrimSpace(archivo.Metadatos.Make)
	modeloTexto := strings.TrimSpace(archivo.Metadatos.Modelo)
	softwareTexto := strings.TrimSpace(archivo.Metadatos.Software)
	makeInferido, modeloInferido, softwareInferido := inferirProcedenciaArchivo(archivo)
	if makeTexto == "" && makeInferido != "" {
		makeTexto = makeInferido
		a.formularioMetadatos.MakeSugerido = makeInferido
		a.formularioMetadatos.MakeSugeridoActivo = true
	}
	if modeloTexto == "" && modeloInferido != "" {
		modeloTexto = modeloInferido
		a.formularioMetadatos.ModeloSugerido = modeloInferido
		a.formularioMetadatos.ModeloSugeridoActivo = true
	}
	if softwareTexto == "" && softwareInferido != "" {
		softwareTexto = softwareInferido
		a.formularioMetadatos.SoftwareSugerido = softwareInferido
		a.formularioMetadatos.SoftwareSugeridoActivo = true
	}
	a.editorMake.SetText(makeTexto)
	a.editorModelo.SetText(modeloTexto)
	a.editorSoftware.SetText(softwareTexto)
}

func (a *Aplicacion) guardarMetadatosArchivoActivo() {
	if !a.tieneArchivoActivo {
		return
	}

	archivo := a.archivoActivo
	fecha, hora, zona, err := normalizarFechaHoraEditada(a.editorFecha.Text(), a.editorHora.Text(), a.editorZonaHoraria.Text())
	if err != nil {
		a.establecerEstado("La fecha, hora o zona horaria no tienen un formato válido", err)
		return
	}

	coordenadas, err := parsearCoordenadasEditadas(a.editorGPSLatitud.Text(), a.editorGPSLongitud.Text())
	if err != nil {
		a.establecerEstado("Las coordenadas GPS no son válidas", err)
		return
	}

	palabras := partirListaCSV(a.editorPalabras.Text())
	archivo.Metadatos.Fecha = fecha
	archivo.Metadatos.Hora = hora
	archivo.Metadatos.ZonaHoraria = zona
	archivo.Metadatos.PalabrasClave = append([]string(nil), palabras...)
	archivo.Metadatos.Sujetos = append([]string(nil), palabras...)
	archivo.Metadatos.Ubicacion = strings.TrimSpace(a.editorUbicacion.Text())
	archivo.Metadatos.Comentario = strings.TrimSpace(a.editorComentario.Text())
	archivo.Metadatos.Copyright = strings.TrimSpace(a.editorCopyright.Text())
	archivo.Metadatos.Coordenadas = coordenadas
	archivo.Metadatos.Make = strings.TrimSpace(a.editorMake.Text())
	archivo.Metadatos.Modelo = strings.TrimSpace(a.editorModelo.Text())
	archivo.Metadatos.Software = strings.TrimSpace(a.editorSoftware.Text())
	aplicarOrientacionSeleccionada(&archivo, a.formularioMetadatos.SeleccionOrientacion)
	archivo.Indicadores.TieneGPS = coordenadas != nil
	archivo.Indicadores.EsAdulto = tieneEtiquetaAdulta(append(append([]string(nil), archivo.Metadatos.Sujetos...), archivo.Metadatos.PalabrasClave...))

	go func() {
		var incidenciaDireccion error
		if archivo.Metadatos.Coordenadas != nil &&
			(strings.TrimSpace(archivo.Metadatos.Pais) == "" || strings.TrimSpace(archivo.Metadatos.Estado) == "" || strings.TrimSpace(archivo.Metadatos.Ciudad) == "") {
			direccion, errDireccion := a.servicioMetadatos.ResolverDireccionGPS(context.Background(), archivo.Metadatos.Coordenadas.Latitud, archivo.Metadatos.Coordenadas.Longitud)
			if errDireccion != nil {
				incidenciaDireccion = errDireccion
			} else {
				if strings.TrimSpace(archivo.Metadatos.Ciudad) == "" {
					archivo.Metadatos.Ciudad = strings.TrimSpace(direccion.Ciudad)
				}
				if strings.TrimSpace(archivo.Metadatos.Estado) == "" {
					archivo.Metadatos.Estado = strings.TrimSpace(direccion.Estado)
				}
				if strings.TrimSpace(archivo.Metadatos.Pais) == "" {
					archivo.Metadatos.Pais = strings.TrimSpace(direccion.Pais)
				}
			}
		}

		errExif := a.servicioMetadatos.GuardarMetadatos(context.Background(), archivo.Ruta, archivo.Metadatos)
		errBD := a.almacen.GuardarArchivo(context.Background(), archivo)
		a.encolarActualizacion(func() {
			if errExif != nil && !errors.Is(errExif, os.ErrNotExist) {
				a.establecerEstado("Los metadatos se guardaron en el catálogo local, pero exiftool no pudo escribir el archivo", errExif)
			} else if errBD != nil {
				a.establecerEstado("No se pudieron guardar los metadatos en el catálogo", errBD)
				return
			} else if incidenciaDireccion != nil {
				a.establecerEstado("Metadatos guardados con incidencia al resolver la dirección GPS", incidenciaDireccion)
			} else {
				a.establecerEstado("Metadatos guardados correctamente", nil)
			}
			a.archivoActivo = archivo
			a.sincronizarEdicionRegiones(archivo)
			a.sincronizarEdicionRecorte(archivo)
			a.sincronizarEditoresMetadatos(archivo)
			a.solicitarSalidaExiftool(archivo, true)
			a.reemplazarArchivoEnMemoria(archivo)
			a.recargarColeccionesLaterales()
		})
	}()
}

func (a *Aplicacion) solicitarSalidaExiftool(archivo modelo.Archivo, forzar bool) {
	if !archivo.EsMultimedia() || strings.TrimSpace(archivo.Ruta) == "" || a.servicioMetadatos == nil {
		a.formularioMetadatos.SalidaExiftoolRuta = ""
		a.formularioMetadatos.SalidaExiftoolTexto = ""
		a.formularioMetadatos.SalidaExiftoolCargando = false
		a.formularioMetadatos.SalidaExiftoolExpandida = false
		return
	}
	if !forzar && a.formularioMetadatos.SalidaExiftoolRuta == archivo.Ruta &&
		(a.formularioMetadatos.SalidaExiftoolCargando || a.formularioMetadatos.SalidaExiftoolTexto != "") {
		return
	}

	a.formularioMetadatos.SalidaExiftoolRuta = archivo.Ruta
	a.formularioMetadatos.SalidaExiftoolCargando = true
	if forzar {
		a.formularioMetadatos.SalidaExiftoolTexto = ""
	}

	go func() {
		texto, err := a.servicioMetadatos.LeerSalidaExiftool(context.Background(), archivo.Ruta)
		a.encolarActualizacion(func() {
			if a.formularioMetadatos.SalidaExiftoolRuta != archivo.Ruta {
				return
			}

			a.formularioMetadatos.SalidaExiftoolCargando = false
			texto = strings.TrimRight(texto, "\n")
			if err != nil {
				if strings.TrimSpace(texto) == "" {
					texto = "No se pudo obtener la salida de exiftool."
				}
				a.formularioMetadatos.SalidaExiftoolTexto = texto
				a.establecerEstado("No se pudo leer la salida cruda de exiftool", err)
				return
			}
			a.formularioMetadatos.SalidaExiftoolTexto = texto
		})
	}()
}

func claveOrientacionArchivo(archivo modelo.Archivo) string {
	if archivo.Tipo == modelo.TipoVideo {
		return fmt.Sprintf("video:%d", modelo.NormalizarRotacionCuartos(archivo.Metadatos.Rotacion))
	}
	return fmt.Sprintf("imagen:%d", modelo.NormalizarOrientacionVisual(archivo.Metadatos.Orientacion))
}

func aplicarOrientacionSeleccionada(archivo *modelo.Archivo, clave string) {
	for _, opcion := range opcionesOrientacionArchivo(*archivo) {
		if opcion.Clave != clave {
			continue
		}
		if archivo.Tipo == modelo.TipoVideo {
			archivo.Metadatos.Rotacion = opcion.Rotacion
			return
		}
		archivo.Metadatos.Orientacion = opcion.Orientacion
		return
	}
}

func (e *estadoFormularioMetadatos) asegurarOpcionOrientacion(clave string) *widget.Clickable {
	if e.opcionesOrientacion == nil {
		e.opcionesOrientacion = make(map[string]*widget.Clickable)
	}
	if clic, existe := e.opcionesOrientacion[clave]; existe {
		return clic
	}
	clic := &widget.Clickable{}
	e.opcionesOrientacion[clave] = clic
	return clic
}

func (e *estadoFormularioMetadatos) asegurarOpcionCalendario(clave string) *widget.Clickable {
	if e.opcionesCalendario == nil {
		e.opcionesCalendario = make(map[string]*widget.Clickable)
	}
	if clic, existe := e.opcionesCalendario[clave]; existe {
		return clic
	}
	clic := &widget.Clickable{}
	e.opcionesCalendario[clave] = clic
	return clic
}

func (a *Aplicacion) sincronizarCalendarioFecha(fecha string) {
	a.formularioMetadatos.MesCalendario = primerDiaMesCalendario(fecha)
}

func opcionesOrientacionArchivo(archivo modelo.Archivo) []opcionOrientacionUI {
	if archivo.Tipo == modelo.TipoVideo {
		return []opcionOrientacionUI{
			{Clave: "video:0", Etiqueta: "0°", Rotacion: 0},
			{Clave: "video:90", Etiqueta: "90° horario", Rotacion: 90},
			{Clave: "video:180", Etiqueta: "180°", Rotacion: 180},
			{Clave: "video:270", Etiqueta: "90° antihorario", Rotacion: 270},
		}
	}
	return []opcionOrientacionUI{
		{Clave: "imagen:1", Etiqueta: "Normal", Orientacion: 1},
		{Clave: "imagen:2", Etiqueta: "Espejo horizontal", Orientacion: 2},
		{Clave: "imagen:3", Etiqueta: "180°", Orientacion: 3},
		{Clave: "imagen:4", Etiqueta: "Espejo vertical", Orientacion: 4},
		{Clave: "imagen:5", Etiqueta: "Transpuesta", Orientacion: 5},
		{Clave: "imagen:6", Etiqueta: "90° horario", Orientacion: 6},
		{Clave: "imagen:7", Etiqueta: "Transversal", Orientacion: 7},
		{Clave: "imagen:8", Etiqueta: "90° antihorario", Orientacion: 8},
	}
}

func inferirFechaHoraDesdeNombre(nombre string) (fecha, hora, zona string, ok bool) {
	base := strings.TrimSpace(strings.TrimSuffix(nombre, filepath.Ext(nombre)))
	for _, expresion := range expresionFechaHoraNombre {
		coincidencias := expresion.FindStringSubmatch(base)
		if len(coincidencias) < 7 {
			continue
		}
		fecha = fmt.Sprintf("%s-%s-%s", coincidencias[1], coincidencias[2], coincidencias[3])
		hora = fmt.Sprintf("%s:%s:%s", coincidencias[4], coincidencias[5], coincidencias[6])
		if len(coincidencias) >= 8 {
			zona, _ = validarYNormalizarZonaHorariaEditable(coincidencias[7])
		}
		if _, err := time.Parse("2006-01-02 15:04:05", fecha+" "+hora); err == nil {
			return fecha, hora, zona, true
		}
	}
	return "", "", "", false
}

func sugerenciasPalabrasArchivo(archivo modelo.Archivo, existentes []string) []string {
	vistos := make(map[string]struct{}, len(existentes))
	for _, valor := range existentes {
		valor = strings.TrimSpace(strings.ToLower(valor))
		if valor == "" {
			continue
		}
		vistos[valor] = struct{}{}
	}

	agregar := func(valor string, salida *[]string) {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			return
		}
		clave := strings.ToLower(valor)
		if _, existe := vistos[clave]; existe {
			return
		}
		vistos[clave] = struct{}{}
		*salida = append(*salida, valor)
	}

	var sugeridas []string
	if archivo.Indicadores.TieneIA {
		agregar("IA", &sugeridas)
	}
	agregar(archivo.Metadatos.Ubicacion, &sugeridas)
	for _, region := range archivo.Metadatos.Regiones {
		agregar(region.Nombre, &sugeridas)
	}
	return sugeridas
}

func inferirCopyrightSugerido(archivo modelo.Archivo, fecha string) string {
	fecha = strings.TrimSpace(fecha)
	if fecha == "" {
		return ""
	}
	partes := strings.Split(fecha, "-")
	if len(partes) == 0 || strings.TrimSpace(partes[0]) == "" {
		return ""
	}
	sugerido := "© " + partes[0]
	for _, region := range archivo.Metadatos.Regiones {
		nombre := strings.TrimSpace(region.Nombre)
		if nombre == "" {
			continue
		}
		return sugerido + " " + nombre
	}
	return sugerido
}

func inferirProcedenciaArchivo(archivo modelo.Archivo) (makeInferido, modeloInferido, softwareInferido string) {
	if archivo.Indicadores.TieneIA {
		bloqueIA := analizarBloqueMetadatosIA(archivo, Paleta{})
		softwareInferido = strings.TrimSpace(bloqueIA.Software)
		modeloInferido = strings.TrimSpace(bloqueIA.ModeloPrincipal)
		if softwareInferido == "" {
			softwareInferido = inferirSoftwareIA(textoIAArchivo(archivo))
		}
		if modeloInferido == "" {
			modeloInferido = inferirModeloIA(textoIAArchivo(archivo))
		}
	}

	redSocial := inferirRedSocial(archivo.Metadatos.WhereFroms)
	if redSocial != "" {
		makeInferido = "Red social"
		if modeloInferido == "" {
			modeloInferido = redSocial
		}
	}
	return strings.TrimSpace(makeInferido), strings.TrimSpace(modeloInferido), strings.TrimSpace(softwareInferido)
}

func textoIAArchivo(archivo modelo.Archivo) string {
	var partes []string
	for _, clave := range []string{"Parameters", "Prompt"} {
		valores := archivo.Metadatos.Extras[clave]
		if len(valores) == 0 {
			continue
		}
		partes = append(partes, clave)
		partes = append(partes, valores...)
	}
	for clave, valores := range archivo.Metadatos.Extras {
		if clave == "Parameters" || clave == "Prompt" {
			continue
		}
		partes = append(partes, clave)
		partes = append(partes, valores...)
	}
	partes = append(partes, archivo.Metadatos.Comentario)
	return strings.Join(partes, "\n")
}

func inferirRedSocial(whereFroms []string) string {
	patrones := map[string]string{
		"instagram.com":  "Instagram",
		"onlyfans.com":   "OnlyFans",
		"tiktok.com":     "TikTok",
		"facebook.com":   "Facebook",
		".x.com":         "X",
		"telegram.org":   "Telegram",
		"twitter.com":    "Twitter",
		"reddit.com":     "Reddit",
		"redd.it":        "Reddit",
		"pinterest.com":  "Pinterest",
		"tumblr.com":     "Tumblr",
		"flickr.com":     "Flickr",
		"vk.com":         "VK",
		"weibo.com":      "Weibo",
		"bilibili.com":   "Bilibili",
		"twitch.tv":      "Twitch",
		"youtube.com":    "YouTube",
		"vimeo.com":      "Vimeo",
		"deviantart.com": "DeviantArt",
		"behance.net":    "Behance",
		"dribbble.com":   "Dribbble",
		"medium.com":     "Medium",
		"patreon.com":    "Patreon",
		"hidden.com":     "Hidden",
	}
	for _, valor := range whereFroms {
		valor = strings.ToLower(valor)
		for patron, etiqueta := range patrones {
			if strings.Contains(valor, patron) {
				return etiqueta
			}
		}
	}
	return ""
}

func parsearCoordenadasEditadas(latitudTexto, longitudTexto string) (*modelo.Coordenadas, error) {
	latitudTexto = strings.TrimSpace(latitudTexto)
	longitudTexto = strings.TrimSpace(longitudTexto)
	if latitudTexto == "" && longitudTexto == "" {
		return nil, nil
	}
	if latitudTexto == "" || longitudTexto == "" {
		return nil, fmt.Errorf("se requieren latitud y longitud")
	}
	latitud, err := strconv.ParseFloat(latitudTexto, 64)
	if err != nil {
		return nil, err
	}
	longitud, err := strconv.ParseFloat(longitudTexto, 64)
	if err != nil {
		return nil, err
	}
	return &modelo.Coordenadas{Latitud: latitud, Longitud: longitud}, nil
}

func normalizarFechaHoraEditada(fecha, hora, zona string) (string, string, string, error) {
	fecha = strings.TrimSpace(fecha)
	hora = strings.TrimSpace(hora)
	zonaNormalizada, err := validarYNormalizarZonaHorariaEditable(zona)
	if err != nil {
		return "", "", "", err
	}
	if fecha == "" {
		if hora != "" || zonaNormalizada != "" {
			return "", "", "", fmt.Errorf("la fecha es obligatoria cuando se indica hora o zona")
		}
		return "", "", "", nil
	}
	if err := validarFechaEditable(fecha); err != nil {
		return "", "", "", err
	}
	if hora != "" {
		horaNormalizada, err := normalizarHoraEditable(hora)
		if err != nil {
			return "", "", "", err
		}
		hora = horaNormalizada
	}
	return fecha, hora, zonaNormalizada, nil
}

func validarFechaEditable(fecha string) error {
	fecha = strings.TrimSpace(fecha)
	if fecha == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		return fmt.Errorf("fecha inválida")
	}
	return nil
}

func normalizarHoraEditable(hora string) (string, error) {
	hora = strings.TrimSpace(hora)
	if hora == "" {
		return "", nil
	}
	disenos := []string{"15:04:05", "15:04"}
	for _, diseno := range disenos {
		instante, err := time.Parse(diseno, hora)
		if err == nil {
			return instante.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("hora inválida")
}

func normalizarZonaHorariaEditable(zona string) string {
	zonaNormalizada, err := validarYNormalizarZonaHorariaEditable(zona)
	if err != nil {
		return strings.TrimSpace(strings.ToUpper(zona))
	}
	return zonaNormalizada
}

func validarYNormalizarZonaHorariaEditable(zona string) (string, error) {
	zona = strings.TrimSpace(strings.ToUpper(zona))
	if zona == "" {
		return "", nil
	}
	if zona == "Z" {
		return "+00:00", nil
	}
	if !expresionZonaHorariaEditable.MatchString(zona) {
		return "", fmt.Errorf("zona horaria inválida")
	}
	coincidencias := expresionPartesZonaHoraria.FindStringSubmatch(zona)
	if len(coincidencias) != 4 {
		return "", fmt.Errorf("zona horaria inválida")
	}
	horas, err := strconv.Atoi(coincidencias[2])
	if err != nil {
		return "", fmt.Errorf("zona horaria inválida")
	}
	minutos, err := strconv.Atoi(coincidencias[3])
	if err != nil {
		return "", fmt.Errorf("zona horaria inválida")
	}
	if horas < 0 || horas > 14 || minutos < 0 || minutos > 59 {
		return "", fmt.Errorf("zona horaria inválida")
	}
	if horas == 14 && minutos != 0 {
		return "", fmt.Errorf("zona horaria inválida")
	}
	return fmt.Sprintf("%s%02d:%02d", coincidencias[1], horas, minutos), nil
}

func mensajeErrorFechaEditable(fecha string) string {
	if strings.TrimSpace(fecha) == "" {
		return ""
	}
	if err := validarFechaEditable(fecha); err != nil {
		return "Usa el formato AAAA-MM-DD."
	}
	return ""
}

func mensajeErrorHoraEditable(hora string) string {
	if strings.TrimSpace(hora) == "" {
		return ""
	}
	if _, err := normalizarHoraEditable(hora); err != nil {
		return "Usa HH:MM o HH:MM:SS."
	}
	return ""
}

func mensajeErrorZonaHorariaEditable(zona string) string {
	if strings.TrimSpace(zona) == "" {
		return ""
	}
	if _, err := validarYNormalizarZonaHorariaEditable(zona); err != nil {
		return "Usa Z o un offset como +05:30."
	}
	return ""
}

func archivoTieneFechaYHoraArchivables(archivo modelo.Archivo) bool {
	fecha := strings.TrimSpace(archivo.Metadatos.Fecha)
	hora := strings.TrimSpace(archivo.Metadatos.Hora)
	if fecha == "" || hora == "" {
		return false
	}
	if err := validarFechaEditable(fecha); err != nil {
		return false
	}
	if _, err := normalizarHoraEditable(hora); err != nil {
		return false
	}
	return true
}

func normalizarListaCSV(valores []string) []string {
	vistos := make(map[string]struct{}, len(valores))
	var salida []string
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor == "" {
			continue
		}
		clave := strings.ToLower(valor)
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		salida = append(salida, valor)
	}
	return salida
}

func (a *Aplicacion) direccionMetadatosActual() serviciometadatos.DireccionGPS {
	return serviciometadatos.DireccionGPS{
		Ciudad: strings.TrimSpace(a.archivoActivo.Metadatos.Ciudad),
		Estado: strings.TrimSpace(a.archivoActivo.Metadatos.Estado),
		Pais:   strings.TrimSpace(a.archivoActivo.Metadatos.Pais),
	}
}

func (a *Aplicacion) palabrasSugeridasVisibles() []string {
	actuales := partirListaCSV(a.editorPalabras.Text())
	conjunto := make(map[string]struct{}, len(actuales))
	for _, palabra := range actuales {
		conjunto[strings.ToLower(strings.TrimSpace(palabra))] = struct{}{}
	}

	var visibles []string
	for _, palabra := range a.formularioMetadatos.PalabrasSugeridas {
		if _, existe := conjunto[strings.ToLower(strings.TrimSpace(palabra))]; existe {
			visibles = append(visibles, palabra)
		}
	}
	return visibles
}

func (a *Aplicacion) campoCoincideSugerencia(valorActual, sugerido string, sugerenciaActiva bool) bool {
	if !sugerenciaActiva {
		return false
	}
	valorActual = strings.TrimSpace(valorActual)
	sugerido = strings.TrimSpace(sugerido)
	return valorActual != "" && sugerido != "" && strings.EqualFold(valorActual, sugerido)
}

func (a *Aplicacion) tieneSugerenciasPalabrasActivas() bool {
	return len(a.palabrasSugeridasVisibles()) > 0
}

func (a *Aplicacion) paddingSeparadorDetalle() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
		}.Layout(gtx, a.dibujarSeparadorHorizontalDetalle)
	}
}

func (a *Aplicacion) dibujarBloqueResumenExplorador(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarPreviewLateral(gtx, archivo)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoPrincipal(gtx, archivo.Ruta)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					texto := fmt.Sprintf("Tipo: %s", archivo.Tipo)
					if archivo.Ancho > 0 && archivo.Alto > 0 {
						texto += fmt.Sprintf(" | %dx%d", archivo.Ancho, archivo.Alto)
					}
					if archivo.Duracion > 0 {
						texto += " | " + formatearDuracion(archivo.Duracion)
					}
					return a.dibujarTextoSecundario(gtx, texto)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, "Tamaño en disco: "+formatearTamano(archivo.Tamano))
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarSeparadorHorizontalDetalle(gtx layout.Context) layout.Dimensions {
	alto := maximo(1, gtx.Dp(unit.Dp(1)))
	ancho := gtx.Constraints.Max.X
	if ancho <= 0 {
		ancho = gtx.Constraints.Min.X
	}
	paint.FillShape(gtx.Ops, a.paleta.Borde, clip.Rect(image.Rect(0, 0, ancho, alto)).Op())
	return layout.Dimensions{Size: image.Pt(ancho, alto)}
}

func (a *Aplicacion) dibujarBloqueAccionesImagen(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Acciones de imagen")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampo(gtx, "Formato de salida", &a.editorFormatoImagen)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonConvertir, "Convertir", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
						a.convertirImagenActiva()
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarBloqueRecorteImagen(gtx layout.Context) layout.Dimensions {
	a.sincronizarEdicionRecorte(a.archivoActivo)

	mensaje := "Activa la selección para dibujar un área de recorte o aceptar una sugerencia automática cuando detectemos bandas mate."
	if a.edicionRecorte.Ruta == a.archivoActivo.Ruta {
		switch {
		case a.edicionRecorte.Guardando:
			mensaje = "Recortando imagen..."
		case a.edicionRecorte.Activo && a.edicionRecorte.TieneSeleccion && a.edicionRecorte.Sugerida:
			mensaje = "Sugerencia automática aplicada. Puedes reajustarla arrastrando los bordes o las esquinas."
		case a.edicionRecorte.Activo && a.edicionRecorte.TieneSeleccion:
			mensaje = "Arrastra los bordes o las esquinas para refinar el área seleccionada."
		case a.edicionRecorte.Activo:
			mensaje = "Arrastra sobre la imagen para definir el área de recorte."
		}
	}

	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Recorte")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, mensaje)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					contexto := gtx
					if a.edicionRecorte.Guardando {
						contexto = contexto.Disabled()
					}
					return material.CheckBox(a.tema, &a.reemplazarOriginalRecorte, "Reemplazar archivo original").Layout(contexto)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					contexto := gtx
					activo := a.edicionRecorte.Activo && a.edicionRecorte.Ruta == a.archivoActivo.Ruta
					if a.edicionRecorte.Guardando {
						contexto = contexto.Disabled()
					}
					fondoSeleccion := a.paleta.Panel
					colorSeleccion := a.paleta.Texto
					if activo {
						fondoSeleccion = a.paleta.Acento
						colorSeleccion = a.paleta.TextoSobreAcento
					}
					fondoRecorte := a.paleta.Panel
					colorRecorte := a.paleta.Texto
					if !a.edicionRecorte.Guardando && a.edicionRecorte.TieneSeleccion && a.edicionRecorte.Ruta == a.archivoActivo.Ruta {
						fondoRecorte = a.paleta.Acento
						colorRecorte = a.paleta.TextoSobreAcento
					}

					return layout.Flex{Alignment: layout.Middle}.Layout(contexto,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonSeleccionarRecorte, "Seleccionar región", fondoSeleccion, colorSeleccion, func() {
								a.alternarSeleccionRecorte()
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.botonRecortar, "Recortar", fondoRecorte, colorRecorte, func() {
								a.recortarImagenActiva()
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.edicionRecorte.Ruta != a.archivoActivo.Ruta || !a.edicionRecorte.TieneSeleccion {
						return layout.Dimensions{}
					}
					dimensiones := a.descripcionDimensionesRecorte(a.archivoActivo, a.edicionRecorte.Seleccion)
					if dimensiones == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.dibujarTextoSecundario(gtx, "Área actual: "+dimensiones)
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarBloqueExtraerFrame(gtx layout.Context) layout.Dimensions {
	valorAntes := a.controlExtraccionFrame.Value
	maximoFotograma := maximo(960, a.reproductorVideo.MaximoFotograma)

	dim := dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Extraer frame")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, "El deslizador sincroniza la vista previa con el frame que se exportará.")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					deslizador := material.Slider(a.tema, &a.controlExtraccionFrame)
					deslizador.Axis = layout.Horizontal
					deslizador.Color = a.paleta.Acento
					return deslizador.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoSecundario(gtx, a.descripcionExtraccionFrame())
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarSelectorFormatoExtraccion(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonExtraerFrame, "Extraer frame", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
						a.extraerFrameActivo()
					})
				}),
			)
		})
	})

	if valorAntes != a.controlExtraccionFrame.Value {
		a.actualizarPosicionVideoDesdeExtraccion(maximoFotograma)
	}
	return dim
}

func (a *Aplicacion) descripcionExtraccionFrame() string {
	porcentaje := float64(a.controlExtraccionFrame.Value) * 100
	if a.reproductorVideo.Duracion <= 0 {
		return fmt.Sprintf("%.0f%%", porcentaje)
	}
	return fmt.Sprintf("%.0f%% | %s de %s", porcentaje, formatearDuracion(a.reproductorVideo.Posicion), formatearDuracion(a.reproductorVideo.Duracion))
}

func (a *Aplicacion) dibujarSelectorFormatoExtraccion(gtx layout.Context) layout.Dimensions {
	formato := a.formatoExtraccionFrameNormalizado()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, "Formato del frame")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonAccion(gtx, &a.botonSelectorFormatoExtraccion, etiquetaFormatoExtraccion(formato), a.paleta.Panel, a.paleta.Texto, func() {
				a.formatoExtraccionExpandido = !a.formatoExtraccionExpandido
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.formatoExtraccionExpandido {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						opciones := []string{"webp", "png", "jpg"}
						hijos := make([]layout.FlexChild, 0, len(opciones)*2)
						for indice, opcion := range opciones {
							opcion := opcion
							hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, a.asegurarOpcionFormatoExtraccion(opcion), etiquetaFormatoExtraccion(opcion), opcion == formato, func() {
									a.formatoExtraccionFrame = opcion
									a.formatoExtraccionExpandido = false
								})
							}))
							if indice < len(opciones)-1 {
								hijos = append(hijos, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
							}
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
					})
				})
			})
		}),
	)
}

func (a *Aplicacion) asegurarOpcionFormatoExtraccion(clave string) *widget.Clickable {
	if a.opcionesFormatoExtraccion == nil {
		a.opcionesFormatoExtraccion = make(map[string]*widget.Clickable)
	}
	if clic, existe := a.opcionesFormatoExtraccion[clave]; existe {
		return clic
	}
	clic := &widget.Clickable{}
	a.opcionesFormatoExtraccion[clave] = clic
	return clic
}

func (a *Aplicacion) formatoExtraccionFrameNormalizado() string {
	switch strings.ToLower(strings.TrimSpace(a.formatoExtraccionFrame)) {
	case "png":
		return "png"
	case "jpg", "jpeg":
		return "jpg"
	default:
		return "webp"
	}
}

func etiquetaFormatoExtraccion(formato string) string {
	return strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(formato), "."))
}

func (a *Aplicacion) dibujarBloqueOptimizarVideo(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Optimizar para web")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.CheckBox(a.tema, &a.sobreescribirVideo, "Sobrescribir al optimizar").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonOptimizarVideo, "Optimizar para web", a.paleta.Exito, a.paleta.Texto, func() {
						a.optimizarVideoActivo()
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarBloqueExiftoolCrudo(gtx layout.Context) layout.Dimensions {
	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			etiquetaBoton := "Mostrar"
			if a.formularioMetadatos.SalidaExiftoolExpandida {
				etiquetaBoton = "Ocultar"
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTituloPanel(gtx, "Salida cruda de ExifTool")
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.formularioMetadatos.botonAlternarExiftool, etiquetaBoton, a.paleta.Panel, a.paleta.Texto, func() {
								a.formularioMetadatos.SalidaExiftoolExpandida = !a.formularioMetadatos.SalidaExiftoolExpandida
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !a.formularioMetadatos.SalidaExiftoolExpandida {
						return layout.Dimensions{}
					}

					return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if a.formularioMetadatos.SalidaExiftoolCargando {
							return a.dibujarTextoSecundario(gtx, "Cargando salida de exiftool...")
						}

						lineas := a.lineasSalidaExiftool()
						alturaMaxima := gtx.Dp(unit.Dp(220))
						gtx.Constraints.Max.Y = alturaMaxima
						if gtx.Constraints.Min.Y > alturaMaxima {
							gtx.Constraints.Min.Y = alturaMaxima
						}
						return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarListaConBarra(gtx, &a.formularioMetadatos.listaSalidaExiftool, len(lineas), func(gtx layout.Context, indice int) layout.Dimensions {
									return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return a.dibujarTextoSecundario(gtx, lineas[indice])
									})
								})
							})
						})
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) lineasSalidaExiftool() []string {
	texto := strings.TrimRight(a.formularioMetadatos.SalidaExiftoolTexto, "\n")
	if strings.TrimSpace(texto) == "" {
		return []string{"Sin salida de exiftool disponible."}
	}
	return strings.Split(texto, "\n")
}

func (a *Aplicacion) dibujarFormularioMetadatos(gtx layout.Context) layout.Dimensions {
	if !a.tieneArchivoActivo || !a.archivoActivo.EsMultimedia() {
		return layout.Dimensions{}
	}

	notePalabras := ""
	if sugeridas := a.palabrasSugeridasVisibles(); len(sugeridas) > 0 {
		notePalabras = "Sugeridas: " + strings.Join(sugeridas, ", ")
	}

	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, "Edición de metadatos")
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarCampoFechaConCalendario(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "Hora", &a.editorHora, a.campoCoincideSugerencia(a.editorHora.Text(), a.formularioMetadatos.HoraSugerida, a.formularioMetadatos.HoraSugeridaActiva), "", mensajeErrorHoraEditable(a.editorHora.Text()), false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "Zona horaria", &a.editorZonaHoraria, a.campoCoincideSugerencia(a.editorZonaHoraria.Text(), a.formularioMetadatos.ZonaHorariaSugerida, a.formularioMetadatos.ZonaHorariaSugeridaActiva), "", mensajeErrorZonaHorariaEditable(a.editorZonaHoraria.Text()), false)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampoDecorado(gtx, "Descripción", &a.editorComentario, false, "", "", false)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampoDecorado(gtx, "Palabras clave", &a.editorPalabras, a.tieneSugerenciasPalabrasActivas(), notePalabras, "", false)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampoDecorado(gtx, "Copyright", &a.editorCopyright, a.campoCoincideSugerencia(a.editorCopyright.Text(), a.formularioMetadatos.CopyrightSugerido, a.formularioMetadatos.CopyrightSugeridoActivo), "", "", false)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarCampoUbicacionConSugerencias(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "GPS latitud", &a.editorGPSLatitud, false, "", "", false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "GPS longitud", &a.editorGPSLongitud, false, "", "", false)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarDireccionGPS(gtx, a.direccionMetadatosActual())
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.archivoActivo.Tipo != modelo.TipoImagen && a.archivoActivo.Tipo != modelo.TipoVideo {
						return layout.Dimensions{}
					}
					return a.dibujarSelectorOrientacion(gtx, a.archivoActivo)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "Make", &a.editorMake, a.campoCoincideSugerencia(a.editorMake.Text(), a.formularioMetadatos.MakeSugerido, a.formularioMetadatos.MakeSugeridoActivo), "", "", false)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarEditorCampoDecorado(gtx, "Model", &a.editorModelo, a.campoCoincideSugerencia(a.editorModelo.Text(), a.formularioMetadatos.ModeloSugerido, a.formularioMetadatos.ModeloSugeridoActivo), "", "", false)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarEditorCampoDecorado(gtx, "Software", &a.editorSoftware, a.campoCoincideSugerencia(a.editorSoftware.Text(), a.formularioMetadatos.SoftwareSugerido, a.formularioMetadatos.SoftwareSugeridoActivo), "", "", false)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBloqueAtributoExtendido(gtx, a.archivoActivo.Metadatos.WhereFroms)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.botonGuardarMetadatos, "Guardar metadatos", a.paleta.Acento, a.paleta.TextoSobreAcento, func() {
						a.guardarMetadatosArchivoActivo()
					})
				}),
			)
		})
	})
}

func (a *Aplicacion) dibujarCampoFechaConCalendario(gtx layout.Context) layout.Dimensions {
	sugerido := a.campoCoincideSugerencia(a.editorFecha.Text(), a.formularioMetadatos.FechaSugerida, a.formularioMetadatos.FechaSugeridaActiva)
	errorValidacion := mensajeErrorFechaEditable(a.editorFecha.Text())
	colorBorde := a.paleta.Borde
	colorNota := a.paleta.Exito
	nota := ""
	if errorValidacion != "" {
		colorBorde = a.paleta.Peligro
		colorNota = a.paleta.Peligro
		nota = errorValidacion
	} else if sugerido {
		colorBorde = a.paleta.Exito
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, "Fecha")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return dibujarPanelConBorde(gtx, a.paleta.Panel, colorBorde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							editorEstilo := material.Editor(a.tema, &a.editorFecha, "")
							editorEstilo.Color = a.paleta.Texto
							editorEstilo.HintColor = a.paleta.TextoSuave
							return editorEstilo.Layout(gtx)
						})
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarBotonAccion(gtx, &a.formularioMetadatos.botonSelectorFecha, "Calendario", a.paleta.Panel, a.paleta.Texto, func() {
						if !a.formularioMetadatos.CalendarioExpandido {
							a.sincronizarCalendarioFecha(a.editorFecha.Text())
						}
						a.formularioMetadatos.CalendarioExpandido = !a.formularioMetadatos.CalendarioExpandido
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if strings.TrimSpace(nota) == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				estilo := material.Label(a.tema, unit.Sp(11), nota)
				estilo.Color = colorNota
				return estilo.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.formularioMetadatos.CalendarioExpandido {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, a.dibujarCalendarioFecha)
		}),
	)
}

func (a *Aplicacion) dibujarEditorCampoDecorado(gtx layout.Context, etiqueta string, editor *widget.Editor, sugerido bool, nota string, errorValidacion string, deshabilitado bool) layout.Dimensions {
	colorBorde := a.paleta.Borde
	colorNota := a.paleta.Exito
	if strings.TrimSpace(errorValidacion) != "" {
		colorBorde = a.paleta.Peligro
		colorNota = a.paleta.Peligro
		nota = errorValidacion
	} else if sugerido {
		colorBorde = a.paleta.Exito
	}
	contexto := gtx
	if deshabilitado {
		contexto = contexto.Disabled()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, etiqueta)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanelConBorde(contexto, a.paleta.Panel, colorBorde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					editorEstilo := material.Editor(a.tema, editor, "")
					editorEstilo.Color = a.paleta.Texto
					editorEstilo.HintColor = a.paleta.TextoSuave
					return editorEstilo.Layout(gtx)
				})
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if strings.TrimSpace(nota) == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				estilo := material.Label(a.tema, unit.Sp(11), nota)
				estilo.Color = colorNota
				return estilo.Layout(gtx)
			})
		}),
	)
}

func (a *Aplicacion) dibujarSelectorOrientacion(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	seleccion := a.formularioMetadatos.SeleccionOrientacion
	etiqueta := "Seleccionar"
	opciones := opcionesOrientacionArchivo(archivo)
	for _, opcion := range opciones {
		if opcion.Clave == seleccion {
			etiqueta = opcion.Etiqueta
			break
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, "Orientación")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarBotonAccion(gtx, &a.formularioMetadatos.botonSelectorOrientacion, etiqueta, a.paleta.Panel, a.paleta.Texto, func() {
				a.formularioMetadatos.OrientacionExpandida = !a.formularioMetadatos.OrientacionExpandida
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !a.formularioMetadatos.OrientacionExpandida {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						hijos := make([]layout.FlexChild, 0, len(opciones)*2)
						for indice, opcion := range opciones {
							opcion := opcion
							hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.dibujarBotonNavegacion(gtx, a.formularioMetadatos.asegurarOpcionOrientacion(opcion.Clave), opcion.Etiqueta, opcion.Clave == a.formularioMetadatos.SeleccionOrientacion, func() {
									a.formularioMetadatos.SeleccionOrientacion = opcion.Clave
									a.formularioMetadatos.OrientacionExpandida = false
								})
							}))
							if indice < len(opciones)-1 {
								hijos = append(hijos, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
							}
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
					})
				})
			})
		}),
	)
}

func primerDiaMesCalendario(fecha string) time.Time {
	fecha = strings.TrimSpace(fecha)
	if fecha != "" {
		if instante, err := time.Parse("2006-01-02", fecha); err == nil {
			return time.Date(instante.Year(), instante.Month(), 1, 0, 0, 0, 0, instante.Location())
		}
	}

	ahora := time.Now()
	return time.Date(ahora.Year(), ahora.Month(), 1, 0, 0, 0, 0, ahora.Location())
}

func construirDiasCalendario(mes time.Time) []diaCalendarioUI {
	mes = time.Date(mes.Year(), mes.Month(), 1, 0, 0, 0, 0, mes.Location())
	desplazamientoInicio := (int(mes.Weekday()) + 6) % 7
	inicio := mes.AddDate(0, 0, -desplazamientoInicio)

	dias := make([]diaCalendarioUI, 0, 42)
	for indice := 0; indice < 42; indice++ {
		fecha := inicio.AddDate(0, 0, indice)
		dias = append(dias, diaCalendarioUI{
			Fecha:       fecha,
			EnMesActivo: fecha.Month() == mes.Month(),
		})
	}
	return dias
}

func nombreMesCalendario(fecha time.Time) string {
	indiceMes := int(fecha.Month()) - 1
	if indiceMes < 0 || indiceMes >= len(mesesCalendarioEspanol) {
		return fecha.Month().String()
	}
	return mesesCalendarioEspanol[indiceMes]
}

func (a *Aplicacion) dibujarCalendarioFecha(gtx layout.Context) layout.Dimensions {
	mes := a.formularioMetadatos.MesCalendario
	if mes.IsZero() {
		mes = primerDiaMesCalendario(a.editorFecha.Text())
		a.formularioMetadatos.MesCalendario = mes
	}

	dias := construirDiasCalendario(mes)
	seleccionActual := strings.TrimSpace(a.editorFecha.Text())

	return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			hijos := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.formularioMetadatos.botonCalendarioAnterior, "<", a.paleta.PanelElevado, a.paleta.Texto, func() {
								a.formularioMetadatos.MesCalendario = a.formularioMetadatos.MesCalendario.AddDate(0, -1, 0)
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoPrincipal(gtx, fmt.Sprintf("%s %d", nombreMesCalendario(mes), mes.Year()))
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonAccion(gtx, &a.formularioMetadatos.botonCalendarioSiguiente, ">", a.paleta.PanelElevado, a.paleta.Texto, func() {
								a.formularioMetadatos.MesCalendario = a.formularioMetadatos.MesCalendario.AddDate(0, 1, 0)
							})
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[0])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[1])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[2])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[3])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[4])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[5])
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, diasSemanaCalendario[6])
							})
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			}

			for fila := 0; fila < 6; fila++ {
				inicio := fila * 7
				semana := dias[inicio : inicio+7]
				hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[0], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[1], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[2], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[3], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[4], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[5], seleccionActual)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarBotonDiaCalendario(gtx, semana[6], seleccionActual)
						}),
					)
				}))
				if fila < 5 {
					hijos = append(hijos, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
				}
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
		})
	})
}

func (a *Aplicacion) dibujarBotonDiaCalendario(gtx layout.Context, dia diaCalendarioUI, seleccionActual string) layout.Dimensions {
	claveFecha := dia.Fecha.Format("2006-01-02")
	activo := claveFecha == seleccionActual
	fondo := a.paleta.PanelElevado
	colorTexto := a.paleta.Texto
	if !dia.EnMesActivo {
		colorTexto = a.paleta.TextoSuave
	}
	if activo {
		fondo = a.paleta.Acento
		colorTexto = a.paleta.TextoSobreAcento
	}

	return a.dibujarBotonAccion(gtx, a.formularioMetadatos.asegurarOpcionCalendario(claveFecha), fmt.Sprintf("%d", dia.Fecha.Day()), fondo, colorTexto, func() {
		a.editorFecha.SetText(claveFecha)
		a.formularioMetadatos.MesCalendario = time.Date(dia.Fecha.Year(), dia.Fecha.Month(), 1, 0, 0, 0, 0, dia.Fecha.Location())
		a.formularioMetadatos.CalendarioExpandido = false
	})
}

func (a *Aplicacion) dibujarDireccionGPS(gtx layout.Context, direccion serviciometadatos.DireccionGPS) layout.Dimensions {
	return a.dibujarDireccionGPSConCajas(gtx, direccion, image.Pt(20, 20), image.Pt(18, 18))
}

func (a *Aplicacion) dibujarDireccionGPSConCajas(gtx layout.Context, direccion serviciometadatos.DireccionGPS, cajaUbicacion, cajaBandera image.Point) layout.Dimensions {
	if strings.TrimSpace(direccion.Ciudad) == "" && strings.TrimSpace(direccion.Estado) == "" && strings.TrimSpace(direccion.Pais) == "" {
		return layout.Dimensions{}
	}
	if cajaUbicacion.X < 1 {
		cajaUbicacion.X = 20
	}
	if cajaUbicacion.Y < 1 {
		cajaUbicacion.Y = 20
	}
	if cajaBandera.X < 1 {
		cajaBandera.X = 18
	}
	if cajaBandera.Y < 1 {
		cajaBandera.Y = 18
	}

	primeraLinea := strings.Trim(strings.Join([]string{direccion.Ciudad, direccion.Estado}, ", "), ", ")
	return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return dibujarPanel(gtx, a.paleta.Panel, unit.Dp(12), func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return dibujarIconoEnCaja(gtx, cajaUbicacion, a.dibujarIconoIndicadorUbicacion, a.paleta.Exito, a.paleta.Panel)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, primeraLinea)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.dibujarIconoBanderaDetalleEnCaja(gtx, cajaBandera)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return a.dibujarTextoSecundario(gtx, direccion.Pais)
							}),
						)
					}),
				)
			})
		})
	})
}

func (a *Aplicacion) dibujarBloqueAtributoExtendido(gtx layout.Context, valores []string) layout.Dimensions {
	texto := "-"
	if len(valores) > 0 {
		texto = strings.Join(valores, "\n")
	}
	lineas := strings.Split(texto, "\n")
	alturaMaxima := gtx.Dp(unit.Dp(84))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoSecundario(gtx, "Atributo extendido")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.Y = alturaMaxima
			if gtx.Constraints.Min.Y > alturaMaxima {
				gtx.Constraints.Min.Y = alturaMaxima
			}
			return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarListaConBarra(gtx, &a.formularioMetadatos.listaAtributoExtendido, len(lineas), func(gtx layout.Context, indice int) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return a.dibujarTextoSecundario(gtx, lineas[indice])
						})
					})
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarIconoBanderaDetalle(gtx layout.Context) layout.Dimensions {
	return a.dibujarIconoBanderaDetalleEnCaja(gtx, image.Pt(18, 18))
}

func (a *Aplicacion) dibujarIconoBanderaDetalleEnCaja(gtx layout.Context, caja image.Point) layout.Dimensions {
	if caja.X < 1 {
		caja.X = 18
	}
	if caja.Y < 1 {
		caja.Y = 18
	}

	contexto := gtx
	contexto.Constraints = layout.Exact(caja)
	return a.dibujarIconoBanderaDetalleEscalado(contexto)
}

func (a *Aplicacion) dibujarIconoBanderaDetalleEscalado(gtx layout.Context) layout.Dimensions {
	base := image.Pt(14, 14)
	gtx, restaurar, objetivo := prepararIconoEscalado(gtx, base)
	defer restaurar()

	paint.FillShape(gtx.Ops, a.paleta.Acento, clip.Rect(image.Rect(2, 2, 4, 12)).Op())
	paint.FillShape(gtx.Ops, a.paleta.Texto, clip.Rect(image.Rect(4, 2, 12, 7)).Op())
	return layout.Dimensions{Size: objetivo}
}
