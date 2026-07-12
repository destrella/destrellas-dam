package metadatos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"destrellas-dam/internal/modelo"
	"destrellas-dam/internal/plataforma"
)

// Servicio encapsula herramientas externas y analisis ligeros en Go.
type Servicio struct {
	rutaExiftool string
	rutaFFmpeg   string
	rutaFFprobe  string
	rutaMagick   string
	rutaQLManage string
	formatosLote sync.Map
}

// FotogramaVideo representa un cuadro decodificado y su instante aproximado.
type FotogramaVideo struct {
	Instante time.Duration
	Imagen   image.Image
}

// ResultadoExtraccionFrame resume el archivo generado al extraer un frame.
type ResultadoExtraccionFrame struct {
	Ruta     string
	Numero   int
	Instante time.Duration
}

// NuevoServicio descubre las herramientas disponibles sin hacer obligatoria ninguna.
func NuevoServicio() *Servicio {
	return &Servicio{
		rutaExiftool: buscarComando("exiftool"),
		rutaFFmpeg:   buscarComando("ffmpeg"),
		rutaFFprobe:  buscarComando("ffprobe"),
		rutaMagick:   buscarComando("magick"),
		rutaQLManage: buscarComando("qlmanage"),
	}
}

// TieneExiftool permite degradar funcionalidad de forma elegante.
func (s *Servicio) TieneExiftool() bool {
	return s != nil && s.rutaExiftool != ""
}

// AnalizarArchivo enriquece un archivo con metadatos, dimensiones, duracion e indicadores.
func (s *Servicio) AnalizarArchivo(ctx context.Context, archivo modelo.Archivo) (modelo.Archivo, error) {
	if archivo.EsDirectorio {
		return archivo, nil
	}

	var primerError error
	if archivoSoportaExiftool(archivo) && s.TieneExiftool() {
		enriquecido, err := s.analizarConExiftool(ctx, archivo)
		if err == nil {
			archivo = enriquecido
		} else {
			primerError = err
		}
	}

	if archivo.Tipo == modelo.TipoImagen && (archivo.Ancho == 0 || archivo.Alto == 0) {
		if ancho, alto, err := leerDimensionesImagen(archivo.Ruta); err == nil {
			archivo.Ancho = ancho
			archivo.Alto = alto
		} else if primerError == nil {
			primerError = err
		}
	}

	if whereFroms, err := plataforma.LeerWhereFroms(ctx, archivo.Ruta); err == nil && len(whereFroms) > 0 {
		archivo.Metadatos.WhereFroms = whereFroms
		archivo.Indicadores.TieneWhereFrom = true
	}

	if len(archivo.Metadatos.WhereFroms) > 0 {
		archivo.Indicadores.TieneWhereFrom = true
	}
	if archivo.Metadatos.Coordenadas != nil {
		archivo.Indicadores.TieneGPS = true
	}
	if len(archivo.Metadatos.Regiones) > 0 {
		archivo.Indicadores.TieneRegiones = true
	}
	if contieneEtiquetaAdulta(archivo.Metadatos.Sujetos) || contieneEtiquetaAdulta(archivo.Metadatos.PalabrasClave) {
		archivo.Indicadores.EsAdulto = true
	}

	return archivo, primerError
}

// GenerarPreviewImagen crea una previsualizacion escalada respetando Orientation.
func (s *Servicio) GenerarPreviewImagen(ctx context.Context, ruta string, maximo int, orientacion int) (image.Image, error) {
	if maximo < 160 {
		maximo = 160
	}

	imagen, errPrincipal := decodificarImagenEscalada(ruta, maximo, orientacion)
	if errPrincipal == nil {
		return imagen, nil
	}

	if s != nil && s.rutaMagick != "" {
		imagen, errMagick := s.generarPreviewImagenMagick(ctx, ruta, maximo)
		if errMagick == nil {
			return imagen, nil
		}
	}

	if s != nil && s.rutaQLManage != "" {
		imagen, errQL := s.generarPreviewImagenQuickLook(ctx, ruta, maximo)
		if errQL == nil {
			return imagen, nil
		}
	}

	return nil, fmt.Errorf("no se pudo generar una previsualizacion de %q: %w", ruta, errPrincipal)
}

// GuardarMetadatos persiste campos editables comunes con exiftool.
func (s *Servicio) GuardarMetadatos(ctx context.Context, ruta string, metadatos modelo.MetadatosArchivo) error {
	if s.rutaExiftool == "" {
		return errors.New("exiftool no esta disponible")
	}

	palabras := normalizarLista(metadatos.PalabrasClave)
	sujetos := normalizarLista(metadatos.Sujetos)
	if len(sujetos) == 0 && len(palabras) > 0 {
		sujetos = append([]string(nil), palabras...)
	}
	if len(palabras) == 0 && len(sujetos) > 0 {
		palabras = append([]string(nil), sujetos...)
	}

	args := []string{
		// IPTC usa Latin por omisión; forzamos UTF-8 para evitar que
		// palabras clave y demás campos multilíngües terminen como "?".
		"-charset",
		"IPTC=UTF8",
		"-overwrite_original_in_place",
		"-P",
		"-m",
		"-codedcharacterset=UTF8",
		"-Keywords=",
		"-Subject=",
		"-Description=",
		"-ImageDescription=",
		"-UserComment=",
		"-XPComment=",
		"-Comment=",
		"-Location=",
		"-Copyright=",
		"-Rights=",
		"-Country-PrimaryLocationName=",
		"-Country=",
		"-Province-State=",
		"-State=",
		"-City=",
		"-Make=",
		"-Model=",
		"-Software=",
		"-CreatorTool=",
		"-DateTimeOriginal=",
		"-DateTimeDigitized=",
		"-CreateDate=",
		"-ModifyDate=",
		"-MediaCreateDate=",
		"-TrackCreateDate=",
		"-OffsetTime=",
		"-OffsetTimeOriginal=",
		"-OffsetTimeDigitized=",
		"-Rotation=",
		"-GPSLatitude=",
		"-GPSLongitude=",
	}
	for _, palabra := range palabras {
		palabra = strings.TrimSpace(palabra)
		if palabra == "" {
			continue
		}
		// En etiquetas de lista, repetir -Keywords=valor reemplaza el
		// conjunto completo tras el borrado inicial; usar += conserva los
		// valores previos del archivo y duplica contenido no deseado.
		args = append(args, "-Keywords="+palabra)
	}
	for _, sujeto := range sujetos {
		sujeto = strings.TrimSpace(sujeto)
		if sujeto == "" {
			continue
		}
		args = append(args, "-Subject="+sujeto)
	}

	descripcion := strings.TrimSpace(metadatos.Comentario)
	if descripcion != "" {
		args = append(args,
			"-Description="+descripcion,
			"-ImageDescription="+descripcion,
			"-UserComment="+descripcion,
			"-XPComment="+descripcion,
			"-Comment="+descripcion,
		)
	}

	fechaHora := construirFechaHoraExif(metadatos.Fecha, metadatos.Hora, metadatos.ZonaHoraria)
	if fechaHora != "" {
		args = append(args,
			"-DateTimeOriginal="+fechaHora,
			"-DateTimeDigitized="+fechaHora,
			"-CreateDate="+fechaHora,
			"-ModifyDate="+fechaHora,
			"-MediaCreateDate="+fechaHora,
			"-TrackCreateDate="+fechaHora,
		)
		if zona := normalizarZonaHoraria(metadatos.ZonaHoraria); zona != "" {
			args = append(args,
				"-OffsetTime="+zona,
				"-OffsetTimeOriginal="+zona,
				"-OffsetTimeDigitized="+zona,
			)
		}
	}

	if metadatos.Coordenadas != nil {
		args = append(args,
			"-GPSLatitude="+fmt.Sprintf("%.8f", metadatos.Coordenadas.Latitud),
			"-GPSLongitude="+fmt.Sprintf("%.8f", metadatos.Coordenadas.Longitud),
		)
	}

	args = append(args,
		"-Location="+metadatos.Ubicacion,
		"-Copyright="+metadatos.Copyright,
		"-Rights="+metadatos.Copyright,
		"-Country-PrimaryLocationName="+metadatos.Pais,
		"-Country="+metadatos.Pais,
		"-Province-State="+metadatos.Estado,
		"-State="+metadatos.Estado,
		"-City="+metadatos.Ciudad,
		"-Make="+metadatos.Make,
		"-Model="+metadatos.Modelo,
		"-Software="+metadatos.Software,
		"-CreatorTool="+metadatos.Software,
		ruta,
	)

	if metadatos.Orientacion > 0 {
		args = append(args[:len(args)-1], "-Orientation#="+strconv.Itoa(modelo.NormalizarOrientacionVisual(metadatos.Orientacion)), ruta)
	}
	if metadatos.Rotacion != 0 {
		args = append(args[:len(args)-1], "-Rotation="+strconv.Itoa(modelo.NormalizarRotacionCuartos(metadatos.Rotacion)), ruta)
	}

	comando := exec.CommandContext(ctx, s.rutaExiftool, args...)
	if salida, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudieron guardar los metadatos: %w: %s", err, strings.TrimSpace(string(salida)))
	}

	return nil
}

// LeerSalidaExiftool ejecuta exiftool en modo legible y devuelve su salida cruda.
func (s *Servicio) LeerSalidaExiftool(ctx context.Context, ruta string) (string, error) {
	if s.rutaExiftool == "" {
		return "", errors.New("exiftool no esta disponible")
	}

	comando := exec.CommandContext(ctx, s.rutaExiftool, "-s", "-G0:1", ruta)
	salida, err := comando.CombinedOutput()
	if err != nil {
		return string(salida), fmt.Errorf("no se pudo ejecutar exiftool: %w", err)
	}
	return string(salida), nil
}

// GuardarRegiones persiste regiones MWG en la imagen usando exiftool.
func (s *Servicio) GuardarRegiones(ctx context.Context, archivo modelo.Archivo, regiones []modelo.RegionEtiquetada) error {
	if s.rutaExiftool == "" {
		return errors.New("exiftool no esta disponible")
	}
	if strings.TrimSpace(archivo.Ruta) == "" {
		return errors.New("la ruta del archivo esta vacia")
	}

	ancho := archivo.Ancho
	alto := archivo.Alto
	if ancho <= 0 || alto <= 0 {
		var err error
		ancho, alto, err = leerDimensionesImagen(archivo.Ruta)
		if err != nil {
			return fmt.Errorf("no se pudieron obtener las dimensiones de la imagen: %w", err)
		}
	}

	if len(regiones) == 0 {
		comando := exec.CommandContext(ctx, s.rutaExiftool,
			"-overwrite_original_in_place",
			"-P",
			"-use", "MWG",
			"-struct",
			"-RegionInfo=",
			archivo.Ruta,
		)
		if salida, err := comando.CombinedOutput(); err != nil {
			return fmt.Errorf("no se pudieron eliminar las regiones: %w: %s", err, strings.TrimSpace(string(salida)))
		}
		return nil
	}

	archivoTemporal, err := os.CreateTemp("", "destrellas-dam-regiones-*.json")
	if err != nil {
		return fmt.Errorf("no se pudo preparar el archivo temporal de regiones: %w", err)
	}
	defer os.Remove(archivoTemporal.Name())

	contenido := []map[string]any{
		{
			"SourceFile":            "*",
			"XMP-mwg-rs:RegionInfo": construirRegionInfoMWG(ancho, alto, regiones),
		},
	}
	codificador := json.NewEncoder(archivoTemporal)
	codificador.SetEscapeHTML(false)
	if err := codificador.Encode(contenido); err != nil {
		archivoTemporal.Close()
		return fmt.Errorf("no se pudo serializar el JSON temporal de regiones: %w", err)
	}
	if err := archivoTemporal.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el archivo temporal de regiones: %w", err)
	}

	comando := exec.CommandContext(ctx, s.rutaExiftool,
		"-overwrite_original_in_place",
		"-P",
		"-use", "MWG",
		"-struct",
		"-j="+archivoTemporal.Name(),
		archivo.Ruta,
	)
	if salida, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudieron guardar las regiones: %w: %s", err, strings.TrimSpace(string(salida)))
	}

	return nil
}

// ConvertirImagen genera un nuevo archivo en el formato solicitado.
func (s *Servicio) ConvertirImagen(ctx context.Context, origen, formato, salida string) error {
	if s.rutaMagick == "" {
		return errors.New("ImageMagick no esta disponible")
	}
	comando := exec.CommandContext(ctx, s.rutaMagick, origen, salidaEnFormato(salida, formato))
	if salidaComando, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudo convertir la imagen: %w: %s", err, strings.TrimSpace(string(salidaComando)))
	}
	return nil
}

// RecortarImagen recorta una region y devuelve la ruta final del archivo generado.
func (s *Servicio) RecortarImagen(ctx context.Context, archivo modelo.Archivo, rect image.Rectangle, salida string, reemplazarOriginal bool) (string, error) {
	if s.rutaMagick == "" {
		return "", errors.New("ImageMagick no esta disponible")
	}
	if strings.TrimSpace(archivo.Ruta) == "" {
		return "", errors.New("la ruta del archivo a recortar esta vacia")
	}
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return "", errors.New("el rectangulo de recorte no es valido")
	}

	destinoFinal := strings.TrimSpace(salida)
	if reemplazarOriginal {
		destinoFinal = archivo.Ruta
	} else {
		rutaLibre, err := rutaDisponibleLocal(destinoFinal)
		if err != nil {
			return "", err
		}
		destinoFinal = rutaLibre
	}

	destinoTemporal, limpiarTemporal, err := crearRutaTemporalCercana(destinoFinal)
	if err != nil {
		return "", err
	}
	defer limpiarTemporal()

	geometria := fmt.Sprintf("%dx%d+%d+%d", rect.Dx(), rect.Dy(), rect.Min.X, rect.Min.Y)
	comando := exec.CommandContext(ctx, s.rutaMagick, archivo.Ruta, "-auto-orient", "-crop", geometria, "+repage", destinoTemporal)
	if salidaComando, err := comando.CombinedOutput(); err != nil {
		return "", fmt.Errorf("no se pudo recortar la imagen: %w: %s", err, strings.TrimSpace(string(salidaComando)))
	}
	if err := s.copiarMetadatosImagenRecortada(ctx, archivo, rect, destinoTemporal); err != nil {
		return "", err
	}
	if err := plataforma.CopiarWhereFroms(ctx, archivo.Ruta, destinoTemporal); err != nil {
		return "", err
	}
	if err := os.Rename(destinoTemporal, destinoFinal); err != nil {
		return "", fmt.Errorf("no se pudo finalizar el archivo recortado: %w", err)
	}
	return destinoFinal, nil
}

func (s *Servicio) copiarMetadatosImagenRecortada(ctx context.Context, archivo modelo.Archivo, rect image.Rectangle, destino string) error {
	if s.rutaExiftool == "" {
		return errors.New("exiftool no esta disponible para heredar metadatos del recorte")
	}

	args := []string{
		"-charset",
		"IPTC=UTF8",
		"-overwrite_original_in_place",
		"-P",
		"-m",
		"-codedcharacterset=UTF8",
		"-TagsFromFile",
		archivo.Ruta,
		"-all:all",
		"--Orientation",
		"--Rotation",
		"--ImageWidth",
		"--ImageHeight",
		"--ExifImageWidth",
		"--ExifImageHeight",
		"--PixelXDimension",
		"--PixelYDimension",
		"--SourceImageWidth",
		"--SourceImageHeight",
		"--ThumbnailImage",
		"--PreviewImage",
		"--PreviewPICT",
		"--JpgFromRaw",
		"--RegionInfo",
		"-Orientation#=1",
		destino,
	}
	comando := exec.CommandContext(ctx, s.rutaExiftool, args...)
	if salida, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudieron heredar los metadatos del recorte: %w: %s", err, strings.TrimSpace(string(salida)))
	}

	regiones := recalcularRegionesTrasRecorte(archivo, rect)
	if len(archivo.Metadatos.Regiones) > 0 || archivo.Indicadores.TieneRegiones {
		if err := s.GuardarRegiones(ctx, modelo.Archivo{
			Ruta:  destino,
			Tipo:  modelo.TipoImagen,
			Ancho: rect.Dx(),
			Alto:  rect.Dy(),
		}, regiones); err != nil {
			return err
		}
	}
	return nil
}

func recalcularRegionesTrasRecorte(archivo modelo.Archivo, rect image.Rectangle) []modelo.RegionEtiquetada {
	if len(archivo.Metadatos.Regiones) == 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return nil
	}

	ancho, alto := dimensionesOrientadasArchivoRecorte(archivo)
	if ancho <= 0 || alto <= 0 {
		return nil
	}

	recorte := modelo.RegionEtiquetada{
		X:     float64(rect.Min.X) / float64(ancho),
		Y:     float64(rect.Min.Y) / float64(alto),
		Ancho: float64(rect.Dx()) / float64(ancho),
		Alto:  float64(rect.Dy()) / float64(alto),
	}

	regiones := make([]modelo.RegionEtiquetada, 0, len(archivo.Metadatos.Regiones))
	for _, regionOriginal := range archivo.Metadatos.Regiones {
		regionOrientada := modelo.TransformarRegionOrientada(regionOriginal, archivo.Metadatos.Orientacion)
		interseccion, ok := interseccionRegionNormalizada(regionOrientada, recorte)
		if !ok {
			continue
		}
		regiones = append(regiones, modelo.RegionEtiquetada{
			Nombre: regionOriginal.Nombre,
			X:      limitarDecimalRegion((interseccion.X - recorte.X) / recorte.Ancho),
			Y:      limitarDecimalRegion((interseccion.Y - recorte.Y) / recorte.Alto),
			Ancho:  limitarDecimalRegion(interseccion.Ancho / recorte.Ancho),
			Alto:   limitarDecimalRegion(interseccion.Alto / recorte.Alto),
		})
	}
	return regiones
}

func interseccionRegionNormalizada(a, b modelo.RegionEtiquetada) (modelo.RegionEtiquetada, bool) {
	inicioX := math.Max(a.X, b.X)
	inicioY := math.Max(a.Y, b.Y)
	finX := math.Min(a.X+a.Ancho, b.X+b.Ancho)
	finY := math.Min(a.Y+a.Alto, b.Y+b.Alto)
	if finX <= inicioX || finY <= inicioY {
		return modelo.RegionEtiquetada{}, false
	}
	return modelo.RegionEtiquetada{
		X:     inicioX,
		Y:     inicioY,
		Ancho: finX - inicioX,
		Alto:  finY - inicioY,
	}, true
}

func dimensionesOrientadasArchivoRecorte(archivo modelo.Archivo) (int, int) {
	ancho := archivo.Ancho
	alto := archivo.Alto
	switch modelo.NormalizarOrientacionVisual(archivo.Metadatos.Orientacion) {
	case 5, 6, 7, 8:
		ancho, alto = alto, ancho
	}
	return ancho, alto
}

func crearRutaTemporalCercana(destino string) (string, func(), error) {
	directorio := filepath.Dir(destino)
	patron := strings.TrimSuffix(filepath.Base(destino), filepath.Ext(destino)) + "-tmp-*" + filepath.Ext(destino)
	archivoTemporal, err := os.CreateTemp(directorio, patron)
	if err != nil {
		return "", nil, fmt.Errorf("no se pudo preparar el archivo temporal del recorte: %w", err)
	}
	rutaTemporal := archivoTemporal.Name()
	if err := archivoTemporal.Close(); err != nil {
		os.Remove(rutaTemporal)
		return "", nil, fmt.Errorf("no se pudo cerrar el archivo temporal del recorte: %w", err)
	}
	if err := os.Remove(rutaTemporal); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", nil, fmt.Errorf("no se pudo liberar el archivo temporal del recorte: %w", err)
	}
	limpiar := func() {
		_ = os.Remove(rutaTemporal)
	}
	return rutaTemporal, limpiar, nil
}

func rutaDisponibleLocal(destino string) (string, error) {
	if strings.TrimSpace(destino) == "" {
		return "", errors.New("la ruta de salida del recorte esta vacia")
	}
	if _, err := os.Stat(destino); errors.Is(err, os.ErrNotExist) {
		return destino, nil
	}

	extension := filepath.Ext(destino)
	base := strings.TrimSuffix(destino, extension)
	for indice := 1; indice < 10_000; indice++ {
		candidato := fmt.Sprintf("%s-%d%s", base, indice, extension)
		if _, err := os.Stat(candidato); errors.Is(err, os.ErrNotExist) {
			return candidato, nil
		}
	}
	return "", fmt.Errorf("no se pudo encontrar una ruta libre para %q", destino)
}

// ExtraerFrame crea una imagen a partir de un punto del video.
func (s *Servicio) ExtraerFrame(ctx context.Context, origen, selector, formato, salida string) error {
	if s.rutaFFmpeg == "" {
		return errors.New("ffmpeg no esta disponible")
	}

	marcaTiempo, err := s.resolverSelectorFrame(ctx, origen, selector)
	if err != nil {
		return err
	}

	destino := salidaEnFormato(salida, formato)
	instante, err := strconv.ParseFloat(marcaTiempo, 64)
	if err != nil {
		return fmt.Errorf("no se pudo interpretar el instante del frame: %w", err)
	}
	return s.extraerFrameEnRutaConFormato(ctx, origen, time.Duration(instante*float64(time.Second)), destino, 0, normalizarFormatoFrameSalida(formato))
}

// ExtraerFrameEnInstante exporta un frame concreto del video, nombra el
// archivo con su número aproximado y copia los metadatos editables del original.
func (s *Servicio) ExtraerFrameEnInstante(ctx context.Context, origen string, instante time.Duration, formato string, rotacion int) (ResultadoExtraccionFrame, error) {
	if s.rutaFFmpeg == "" {
		return ResultadoExtraccionFrame{}, errors.New("ffmpeg no esta disponible")
	}

	formato = normalizarFormatoFrameSalida(formato)
	if instante < 0 {
		instante = 0
	}

	numero, errNumero := s.numeroFotogramaVideo(ctx, origen, instante)
	if errNumero != nil {
		numero = numeroFotogramaAproximado(instante, 30)
	}

	resultado := ResultadoExtraccionFrame{
		Ruta:     construirRutaSalidaFrame(origen, formato, numero),
		Numero:   numero,
		Instante: instante,
	}
	if err := s.extraerFrameEnRutaConFormato(ctx, origen, instante, resultado.Ruta, rotacion, formato); err != nil {
		return resultado, err
	}
	if err := s.copiarMetadatosArchivoAFrame(ctx, origen, resultado.Ruta); err != nil {
		return resultado, err
	}
	return resultado, nil
}

// OptimizarVideoWeb recodifica el video a H.264/AAC con faststart para web.
func (s *Servicio) OptimizarVideoWeb(ctx context.Context, origen, salida string, sobreescribir bool) error {
	if s.rutaFFmpeg == "" {
		return errors.New("ffmpeg no esta disponible")
	}

	destino := salida
	if sobreescribir {
		destino = origen + ".tmp-web.mp4"
	}

	comando := exec.CommandContext(ctx, s.rutaFFmpeg,
		"-hide_banner", "-loglevel", "error",
		"-i", origen,
		"-map", "0",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-movflags", "+faststart",
		"-c:a", "aac",
		"-b:a", "160k",
		"-y",
		destino,
	)
	if salidaComando, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudo optimizar el video: %w: %s", err, strings.TrimSpace(string(salidaComando)))
	}

	if sobreescribir {
		return os.Rename(destino, origen)
	}
	return nil
}

// CalcularDHashImagen genera un hash perceptual simple para imagenes.
func (s *Servicio) CalcularDHashImagen(_ context.Context, ruta string) (string, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		return "", fmt.Errorf("no se pudo abrir la imagen para dHash: %w", err)
	}
	defer archivo.Close()

	imagen, _, err := image.Decode(archivo)
	if err != nil {
		return "", fmt.Errorf("no se pudo decodificar la imagen para dHash: %w", err)
	}

	return calcularDHashDesdeImagen(imagen), nil
}

// CalcularDHashVideo obtiene varios frames representativos y concatena sus hashes.
func (s *Servicio) CalcularDHashVideo(ctx context.Context, ruta string) (string, error) {
	if s.rutaFFmpeg == "" || s.rutaFFprobe == "" {
		return "", errors.New("ffmpeg/ffprobe no estan disponibles")
	}

	duracion, err := s.duracionVideo(ctx, ruta)
	if err != nil {
		return "", err
	}

	porcentajes := []float64{0.02, 0.25, 0.50, 0.75, 0.98}
	partes := make([]string, 0, len(porcentajes))
	for _, porcentaje := range porcentajes {
		instante := duracion * porcentaje
		imagen, err := s.extraerMiniaturaVideo(ctx, ruta, instante)
		if err != nil {
			return "", err
		}
		partes = append(partes, calcularDHashDesdeImagen(imagen))
	}

	return strings.Join(partes, "-"), nil
}

// GenerarPreviewVideo extrae un frame temprano del video respetando Rotation.
func (s *Servicio) GenerarPreviewVideo(ctx context.Context, ruta string, maximo int, rotacion int) (image.Image, error) {
	if s.rutaFFmpeg == "" || s.rutaFFprobe == "" {
		return nil, errors.New("ffmpeg/ffprobe no estan disponibles")
	}
	if maximo < 160 {
		maximo = 160
	}

	instante, err := s.instantePreviewVideo(ctx, ruta)
	if err != nil {
		return nil, err
	}

	return s.GenerarFotogramaVideo(ctx, ruta, time.Duration(instante*float64(time.Second)), maximo, rotacion)
}

// GenerarFotogramaVideo extrae un fotograma concreto para el visor embebido.
func (s *Servicio) GenerarFotogramaVideo(ctx context.Context, ruta string, instante time.Duration, maximo int, rotacion int) (image.Image, error) {
	if s.rutaFFmpeg == "" || s.rutaFFprobe == "" {
		return nil, errors.New("ffmpeg/ffprobe no estan disponibles")
	}
	if maximo < 160 {
		maximo = 160
	}
	if instante < 0 {
		instante = 0
	}

	filtro := construirFiltroVideoEscalado(maximo, rotacion)
	comando := exec.CommandContext(ctx, s.rutaFFmpeg,
		"-hide_banner", "-loglevel", "error",
		"-noautorotate",
		"-ss", fmt.Sprintf("%.3f", instante.Seconds()),
		"-i", ruta,
		"-frames:v", "1",
		"-vf", filtro,
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	)
	salida, err := comando.Output()
	if err != nil {
		return nil, fmt.Errorf("no se pudo extraer una previsualización del video: %w", err)
	}

	imagen, _, err := image.Decode(bytes.NewReader(salida))
	if err != nil {
		return nil, fmt.Errorf("no se pudo decodificar el fotograma del video: %w", err)
	}
	return imagen, nil
}

// GenerarLoteFotogramasVideo extrae varios fotogramas consecutivos respetando Rotation.
func (s *Servicio) GenerarLoteFotogramasVideo(ctx context.Context, ruta string, inicio time.Duration, fotogramasPorSegundo, cantidad, maximo int, rotacion int) ([]FotogramaVideo, error) {
	if s.rutaFFmpeg == "" || s.rutaFFprobe == "" {
		return nil, errors.New("ffmpeg/ffprobe no estan disponibles")
	}
	if fotogramasPorSegundo < 1 {
		fotogramasPorSegundo = 12
	}
	if cantidad < 1 {
		cantidad = 24
	}
	if maximo < 160 {
		maximo = 160
	}
	if inicio < 0 {
		inicio = 0
	}

	directorioTemporal, err := os.MkdirTemp("", "destrellas-dam-video-*")
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar el directorio temporal de video: %w", err)
	}
	defer os.RemoveAll(directorioTemporal)

	filtro := construirFiltroLoteVideo(maximo, fotogramasPorSegundo, rotacion)
	// Intentamos JPEG por rapidez; si ffmpeg rechaza ese encoder en un video concreto,
	// hacemos un respaldo automático con PNG y recordamos el formato válido por ruta.
	if s.formatoPreferidoLoteVideo(ruta) == "png" {
		if fotogramas, err := s.extraerYLeerLoteFotogramas(ctx, ruta, inicio, filtro, cantidad, fotogramasPorSegundo, directorioTemporal, "png"); err == nil {
			return fotogramas, nil
		} else if fotogramasJPEG, errJPEG := s.extraerYLeerLoteFotogramas(ctx, ruta, inicio, filtro, cantidad, fotogramasPorSegundo, directorioTemporal, "jpeg"); errJPEG == nil {
			s.guardarFormatoPreferidoLoteVideo(ruta, "jpeg")
			return fotogramasJPEG, nil
		} else {
			return nil, fmt.Errorf("no se pudo extraer un lote de fotogramas: %w | respaldo JPEG: %v", err, errJPEG)
		}
	}

	if fotogramasJPEG, err := s.extraerYLeerLoteFotogramas(ctx, ruta, inicio, filtro, cantidad, fotogramasPorSegundo, directorioTemporal, "jpeg"); err == nil {
		s.guardarFormatoPreferidoLoteVideo(ruta, "jpeg")
		return fotogramasJPEG, nil
	} else if fotogramasPNG, errPNG := s.extraerYLeerLoteFotogramas(ctx, ruta, inicio, filtro, cantidad, fotogramasPorSegundo, directorioTemporal, "png"); errPNG == nil {
		s.guardarFormatoPreferidoLoteVideo(ruta, "png")
		return fotogramasPNG, nil
	} else {
		return nil, fmt.Errorf("no se pudo extraer un lote de fotogramas: %w | respaldo PNG: %v", err, errPNG)
	}
}

func construirFiltroVideoEscalado(maximo, rotacion int) string {
	partes := make([]string, 0, 2)
	if filtroRotacion := filtroRotacionVideo(rotacion); filtroRotacion != "" {
		partes = append(partes, filtroRotacion)
	}
	partes = append(partes, fmt.Sprintf("scale='if(gte(iw,ih),min(%d,iw),-2)':'if(lt(iw,ih),min(%d,ih),-2)'", maximo, maximo))
	return strings.Join(partes, ",")
}

func construirFiltroLoteVideo(maximo, fotogramasPorSegundo, rotacion int) string {
	partes := make([]string, 0, 3)
	if filtroRotacion := filtroRotacionVideo(rotacion); filtroRotacion != "" {
		partes = append(partes, filtroRotacion)
	}
	partes = append(partes, fmt.Sprintf("fps=%d", fotogramasPorSegundo))
	partes = append(partes, fmt.Sprintf("scale='if(gte(iw,ih),min(%d,iw),-2)':'if(lt(iw,ih),min(%d,ih),-2)'", maximo, maximo))
	return strings.Join(partes, ",")
}

func filtroRotacionVideo(rotacion int) string {
	switch modelo.NormalizarRotacionCuartos(rotacion) {
	case 90:
		return "transpose=clock"
	case 180:
		return "transpose=clock,transpose=clock"
	case 270:
		return "transpose=cclock"
	default:
		return ""
	}
}

func (s *Servicio) extraerYLeerLoteFotogramas(ctx context.Context, ruta string, inicio time.Duration, filtro string, cantidad, fotogramasPorSegundo int, directorioBase, formato string) ([]FotogramaVideo, error) {
	usarPNG := formato == "png"
	extension := "jpg"
	if usarPNG {
		extension = "png"
	}

	directorioFormato := filepath.Join(directorioBase, formato)
	if err := os.Mkdir(directorioFormato, 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo preparar el directorio temporal %s: %w", strings.ToUpper(formato), err)
	}
	patronSalida := filepath.Join(directorioFormato, "%06d."+extension)
	if err := s.extraerLoteFotogramasConPatron(ctx, ruta, inicio, filtro, cantidad, patronSalida, usarPNG); err != nil {
		return nil, err
	}
	return s.leerFotogramasTemporales(directorioFormato, inicio, fotogramasPorSegundo)
}

func (s *Servicio) formatoPreferidoLoteVideo(ruta string) string {
	if s == nil || ruta == "" {
		return "jpeg"
	}
	if valor, existe := s.formatosLote.Load(ruta); existe {
		if formato, ok := valor.(string); ok && formato != "" {
			return formato
		}
	}
	return "jpeg"
}

func (s *Servicio) guardarFormatoPreferidoLoteVideo(ruta, formato string) {
	if s == nil || ruta == "" || formato == "" {
		return
	}
	s.formatosLote.Store(ruta, formato)
}

func (s *Servicio) extraerLoteFotogramasConPatron(ctx context.Context, ruta string, inicio time.Duration, filtro string, cantidad int, patronSalida string, usarPNG bool) error {
	argumentos := []string{
		"-hide_banner", "-loglevel", "error",
		"-noautorotate",
		"-ss", fmt.Sprintf("%.3f", inicio.Seconds()),
		"-i", ruta,
		"-vf", filtro,
		"-frames:v", strconv.Itoa(cantidad),
	}
	if usarPNG {
		argumentos = append(argumentos,
			"-f", "image2",
			"-vcodec", "png",
		)
	} else {
		argumentos = append(argumentos,
			"-q:v", "4",
			"-pix_fmt", "yuvj420p",
			"-strict", "unofficial",
		)
	}
	argumentos = append(argumentos, "-y", patronSalida)

	comando := exec.CommandContext(ctx, s.rutaFFmpeg, argumentos...)
	if salida, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(salida)))
	}
	return nil
}

func (s *Servicio) leerFotogramasTemporales(directorio string, inicio time.Duration, fotogramasPorSegundo int) ([]FotogramaVideo, error) {
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron leer los fotogramas temporales: %w", err)
	}
	sort.SliceStable(entradas, func(i, j int) bool {
		return entradas[i].Name() < entradas[j].Name()
	})

	intervalo := time.Second / time.Duration(fotogramasPorSegundo)
	fotogramas := make([]FotogramaVideo, 0, len(entradas))
	for indice, entrada := range entradas {
		if entrada.IsDir() {
			continue
		}
		rutaFotograma := filepath.Join(directorio, entrada.Name())
		archivo, err := os.Open(rutaFotograma)
		if err != nil {
			return nil, fmt.Errorf("no se pudo abrir el fotograma temporal %q: %w", rutaFotograma, err)
		}
		imagen, _, err := image.Decode(archivo)
		archivo.Close()
		if err != nil {
			return nil, fmt.Errorf("no se pudo decodificar el fotograma temporal %q: %w", rutaFotograma, err)
		}
		fotogramas = append(fotogramas, FotogramaVideo{
			Instante: inicio + time.Duration(indice)*intervalo,
			Imagen:   imagen,
		})
	}

	if len(fotogramas) == 0 {
		return nil, errors.New("ffmpeg no devolvió fotogramas para el lote solicitado")
	}
	return fotogramas, nil
}

func (s *Servicio) analizarConExiftool(ctx context.Context, archivo modelo.Archivo) (modelo.Archivo, error) {
	comando := exec.CommandContext(ctx, s.rutaExiftool, "-j", "-struct", "-n", archivo.Ruta)
	salida, err := comando.Output()
	if err != nil {
		return archivo, fmt.Errorf("no se pudo leer metadata con exiftool para %q: %w", archivo.Ruta, err)
	}

	var documentos []map[string]any
	if err := json.Unmarshal(salida, &documentos); err != nil {
		return archivo, fmt.Errorf("no se pudo interpretar el JSON de exiftool: %w", err)
	}
	if len(documentos) == 0 {
		return archivo, nil
	}

	documento := documentos[0]
	archivo.Ancho = extraerEntero(documento, "ImageWidth", "SourceImageWidth", "ExifImageWidth")
	archivo.Alto = extraerEntero(documento, "ImageHeight", "SourceImageHeight", "ExifImageHeight")
	archivo.Duracion = time.Duration(extraerFlotante(documento, "Duration", "MediaDuration") * float64(time.Second))
	archivo.Metadatos.Orientacion = extraerEntero(documento, "Orientation")
	archivo.Metadatos.Rotacion = extraerEntero(documento, "Rotation", "RotationDegrees", "TrackRotate", "Rotate")
	archivo.Metadatos.Fecha, archivo.Metadatos.Hora, archivo.Metadatos.ZonaHoraria = extraerFechaHoraEditable(documento)
	archivo.Metadatos.Ubicacion = extraerCadena(documento, "Location", "Sub-location", "SubLocation")
	archivo.Metadatos.Comentario = extraerTextoCombinado(documento, "Description", "ImageDescription", "UserComment", "XPComment", "Comment")
	archivo.Metadatos.PalabrasClave = normalizarLista(
		extraerListaCadenas(documento, "Keywords", "Keyword", "HierarchicalSubject"),
	)
	archivo.Metadatos.Sujetos = normalizarLista(
		extraerListaCadenas(documento, "Subject"),
	)
	archivo.Metadatos.Copyright = extraerCadena(documento, "Copyright", "Rights")
	archivo.Metadatos.Pais = extraerCadena(documento, "Country-PrimaryLocationName", "Country")
	archivo.Metadatos.Estado = extraerCadena(documento, "Province-State", "State")
	archivo.Metadatos.Ciudad = extraerCadena(documento, "City")
	archivo.Metadatos.Make = extraerCadena(documento, "Make")
	archivo.Metadatos.Modelo = extraerCadena(documento, "Model")
	archivo.Metadatos.Software = extraerCadena(documento, "Software", "CreatorTool")
	archivo.Metadatos.Extras = extraerExtrasMetadatos(documento, "Parameters", "Prompt")

	latitud := extraerFlotante(documento, "GPSLatitude")
	longitud := extraerFlotante(documento, "GPSLongitude")
	if latitud != 0 || longitud != 0 {
		archivo.Metadatos.Coordenadas = &modelo.Coordenadas{
			Latitud:  latitud,
			Longitud: longitud,
		}
		archivo.Indicadores.TieneGPS = true
	}

	archivo.Metadatos.Regiones = extraerRegiones(documento)
	archivo.Indicadores.TieneRegiones = len(archivo.Metadatos.Regiones) > 0

	valores := recolectarValoresPlano(documento)
	archivo.Indicadores.TieneIA = detectarIA(documento, valores)
	archivo.Indicadores.TieneSocial = detectarSocial(valores)
	archivo.Indicadores.EsAdulto = contieneEtiquetaAdulta(append(archivo.Metadatos.Sujetos, archivo.Metadatos.PalabrasClave...))

	return archivo, nil
}

func (s *Servicio) resolverSelectorFrame(ctx context.Context, ruta, selector string) (string, error) {
	selector = strings.TrimSpace(strings.ToLower(selector))
	if selector == "" || selector == "medio" || selector == "50%" {
		duracion, err := s.duracionVideo(ctx, ruta)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%.3f", duracion*0.5), nil
	}
	if strings.HasSuffix(selector, "%") {
		duracion, err := s.duracionVideo(ctx, ruta)
		if err != nil {
			return "", err
		}
		valor, err := strconv.ParseFloat(strings.TrimSuffix(selector, "%"), 64)
		if err != nil {
			return "", fmt.Errorf("porcentaje de frame invalido: %w", err)
		}
		return fmt.Sprintf("%.3f", duracion*(valor/100.0)), nil
	}
	if selector == "keyframe" || selector == "inicio" {
		return "0", nil
	}
	return selector, nil
}

func (s *Servicio) extraerFrameEnRutaConFormato(ctx context.Context, origen string, instante time.Duration, destino string, rotacion int, formato string) error {
	formato = normalizarFormatoFrameSalida(formato)
	destino = salidaEnFormato(destino, formato)

	if formato == "webp" && s.rutaMagick != "" {
		return s.extraerFrameWebPConRespaldo(ctx, origen, instante, destino, rotacion)
	}

	if err := s.extraerFrameEnRuta(ctx, origen, instante, destino, rotacion); err != nil {
		if formato == "webp" && s.rutaMagick == "" {
			return fmt.Errorf("%w. Este ffmpeg no parece incluir encoder WebP y tampoco hay ImageMagick disponible para usar un respaldo", err)
		}
		return err
	}
	return nil
}

func (s *Servicio) extraerFrameWebPConRespaldo(ctx context.Context, origen string, instante time.Duration, destino string, rotacion int) error {
	directorioTemporal, err := os.MkdirTemp("", "destrellas-dam-frame-webp-*")
	if err != nil {
		return fmt.Errorf("no se pudo preparar el directorio temporal para WebP: %w", err)
	}
	defer os.RemoveAll(directorioTemporal)

	intermedioPNG := filepath.Join(directorioTemporal, "frame.png")
	if err := s.extraerFrameEnRuta(ctx, origen, instante, intermedioPNG, rotacion); err != nil {
		return err
	}
	if err := s.ConvertirImagen(ctx, intermedioPNG, "webp", destino); err != nil {
		return fmt.Errorf("no se pudo convertir el frame extraído a WebP: %w", err)
	}
	return nil
}

func (s *Servicio) extraerFrameEnRuta(ctx context.Context, origen string, instante time.Duration, destino string, rotacion int) error {
	argumentos := []string{
		"-hide_banner", "-loglevel", "error",
		"-noautorotate",
		"-ss", fmt.Sprintf("%.3f", instante.Seconds()),
		"-i", origen,
		"-frames:v", "1",
	}
	if filtroRotacion := filtroRotacionVideo(rotacion); filtroRotacion != "" {
		argumentos = append(argumentos, "-vf", filtroRotacion)
	}
	argumentos = append(argumentos, "-y", destino)

	comando := exec.CommandContext(ctx, s.rutaFFmpeg, argumentos...)
	if salidaComando, err := comando.CombinedOutput(); err != nil {
		return fmt.Errorf("no se pudo extraer el frame: %w: %s", err, strings.TrimSpace(string(salidaComando)))
	}
	return nil
}

func (s *Servicio) copiarMetadatosArchivoAFrame(ctx context.Context, origen, destino string) error {
	if s.rutaExiftool == "" {
		return errors.New("exiftool no esta disponible para copiar metadatos al frame")
	}

	archivo, err := s.analizarConExiftool(ctx, modelo.Archivo{
		Ruta: origen,
		Tipo: modelo.TipoVideo,
	})
	if err != nil {
		return fmt.Errorf("no se pudieron leer los metadatos del archivo original: %w", err)
	}

	metadatos := archivo.Metadatos
	// El frame exportado ya queda con la orientación visual correcta, así que
	// evitamos propagar rotaciones u orientaciones del video original.
	metadatos.Orientacion = 0
	metadatos.Rotacion = 0
	metadatos.Regiones = nil
	return s.GuardarMetadatos(ctx, destino, metadatos)
}

func (s *Servicio) numeroFotogramaVideo(ctx context.Context, ruta string, instante time.Duration) (int, error) {
	fotogramasPorSegundo, err := s.fotogramasPorSegundoVideo(ctx, ruta)
	if err != nil {
		return 0, err
	}
	return numeroFotogramaAproximado(instante, fotogramasPorSegundo), nil
}

func (s *Servicio) fotogramasPorSegundoVideo(ctx context.Context, ruta string) (float64, error) {
	if s.rutaFFprobe == "" {
		return 0, errors.New("ffprobe no esta disponible")
	}

	comando := exec.CommandContext(ctx, s.rutaFFprobe,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=avg_frame_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		ruta,
	)
	salida, err := comando.Output()
	if err != nil {
		return 0, fmt.Errorf("no se pudo obtener la tasa de fotogramas del video: %w", err)
	}

	fotogramasPorSegundo, err := parsearTasaFotogramas(strings.TrimSpace(string(salida)))
	if err != nil {
		return 0, err
	}
	if fotogramasPorSegundo <= 0 {
		return 0, errors.New("la tasa de fotogramas del video no es válida")
	}
	return fotogramasPorSegundo, nil
}

func parsearTasaFotogramas(texto string) (float64, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return 0, errors.New("la tasa de fotogramas está vacía")
	}

	if strings.Contains(texto, "/") {
		partes := strings.SplitN(texto, "/", 2)
		if len(partes) != 2 {
			return 0, fmt.Errorf("tasa de fotogramas inválida: %q", texto)
		}
		numerador, err := strconv.ParseFloat(strings.TrimSpace(partes[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("numerador de fps inválido: %w", err)
		}
		denominador, err := strconv.ParseFloat(strings.TrimSpace(partes[1]), 64)
		if err != nil {
			return 0, fmt.Errorf("denominador de fps inválido: %w", err)
		}
		if denominador == 0 {
			return 0, errors.New("el denominador de fps no puede ser cero")
		}
		return numerador / denominador, nil
	}

	valor, err := strconv.ParseFloat(texto, 64)
	if err != nil {
		return 0, fmt.Errorf("tasa de fotogramas inválida: %w", err)
	}
	return valor, nil
}

func numeroFotogramaAproximado(instante time.Duration, fotogramasPorSegundo float64) int {
	if instante < 0 {
		instante = 0
	}
	if fotogramasPorSegundo <= 0 {
		fotogramasPorSegundo = 30
	}
	numero := int(math.Round(instante.Seconds() * fotogramasPorSegundo))
	if numero < 0 {
		return 0
	}
	return numero
}

func construirRutaSalidaFrame(origen, formato string, numero int) string {
	formato = normalizarFormatoFrameSalida(formato)
	base := strings.TrimSuffix(origen, filepath.Ext(origen))
	if numero < 0 {
		numero = 0
	}
	return fmt.Sprintf("%s-frame-%06d.%s", base, numero, formato)
}

func normalizarFormatoFrameSalida(formato string) string {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(formato), ".")) {
	case "png":
		return "png"
	case "jpg", "jpeg":
		return "jpg"
	default:
		return "webp"
	}
}

func (s *Servicio) duracionVideo(ctx context.Context, ruta string) (float64, error) {
	if s.rutaFFprobe == "" {
		return 0, errors.New("ffprobe no esta disponible")
	}
	comando := exec.CommandContext(ctx, s.rutaFFprobe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		ruta,
	)
	salida, err := comando.Output()
	if err != nil {
		return 0, fmt.Errorf("no se pudo obtener la duracion del video: %w", err)
	}
	valor, err := strconv.ParseFloat(strings.TrimSpace(string(salida)), 64)
	if err != nil {
		return 0, fmt.Errorf("duracion de video invalida: %w", err)
	}
	return valor, nil
}

func (s *Servicio) extraerMiniaturaVideo(ctx context.Context, ruta string, segundo float64) (image.Image, error) {
	if s.rutaFFmpeg == "" {
		return nil, errors.New("ffmpeg no esta disponible")
	}

	comando := exec.CommandContext(ctx, s.rutaFFmpeg,
		"-hide_banner", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.3f", segundo),
		"-i", ruta,
		"-frames:v", "1",
		"-vf", "scale=9:8,format=gray",
		"-f", "image2pipe",
		"-vcodec", "png",
		"-",
	)
	salida, err := comando.Output()
	if err != nil {
		return nil, fmt.Errorf("no se pudo extraer una miniatura del video: %w", err)
	}

	imagen, _, err := image.Decode(bytes.NewReader(salida))
	if err != nil {
		return nil, fmt.Errorf("no se pudo decodificar el frame del video: %w", err)
	}

	return imagen, nil
}

func (s *Servicio) instantePreviewVideo(ctx context.Context, ruta string) (float64, error) {
	duracion, err := s.duracionVideo(ctx, ruta)
	if err != nil {
		return 0, err
	}

	if duracion <= 0 {
		return 0.8, nil
	}

	instante := duracion * 0.06
	if instante < 0.8 {
		instante = 0.8
	}

	limiteTemprano := duracion * 0.25
	if limiteTemprano < 0.2 {
		limiteTemprano = duracion * 0.5
	}
	if limiteTemprano > 3.0 {
		limiteTemprano = 3.0
	}
	if instante > limiteTemprano {
		instante = limiteTemprano
	}
	if instante >= duracion {
		instante = duracion / 2
	}
	if instante < 0 {
		instante = 0
	}

	return instante, nil
}

func leerDimensionesImagen(ruta string) (int, int, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		return 0, 0, fmt.Errorf("no se pudo abrir la imagen: %w", err)
	}
	defer archivo.Close()

	configuracion, _, err := image.DecodeConfig(archivo)
	if err != nil {
		return 0, 0, fmt.Errorf("no se pudo leer la configuracion de la imagen: %w", err)
	}

	return configuracion.Width, configuracion.Height, nil
}

func decodificarImagenEscalada(ruta string, maximo int, orientacion int) (image.Image, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer archivo.Close()

	imagen, _, err := image.Decode(archivo)
	if err != nil {
		return nil, err
	}

	imagen = escalarImagenSiHaceFalta(imagen, maximo)
	return orientarImagenPreview(imagen, orientacion), nil
}

func escalarImagenSiHaceFalta(imagen image.Image, maximo int) image.Image {
	if imagen == nil {
		return nil
	}
	if maximo < 64 {
		maximo = 256
	}

	original := imagen.Bounds().Size()
	if original.X <= maximo && original.Y <= maximo {
		return imagen
	}

	var escala float64
	if original.X >= original.Y {
		escala = float64(maximo) / float64(original.X)
	} else {
		escala = float64(maximo) / float64(original.Y)
	}

	destino := image.NewRGBA(image.Rect(0, 0, maximoEntero(1, int(float64(original.X)*escala)), maximoEntero(1, int(float64(original.Y)*escala))))
	draw.ApproxBiLinear.Scale(destino, destino.Bounds(), imagen, imagen.Bounds(), draw.Over, nil)
	return destino
}

func orientarImagenPreview(imagen image.Image, orientacion int) image.Image {
	if imagen == nil {
		return nil
	}
	orientacion = modelo.NormalizarOrientacionVisual(orientacion)
	if orientacion == 1 {
		return imagen
	}

	origen := imagen.Bounds()
	ancho := origen.Dx()
	alto := origen.Dy()
	if ancho <= 0 || alto <= 0 {
		return imagen
	}

	anchoDestino := ancho
	altoDestino := alto
	if orientacion == 5 || orientacion == 6 || orientacion == 7 || orientacion == 8 {
		anchoDestino = alto
		altoDestino = ancho
	}

	destino := image.NewRGBA(image.Rect(0, 0, anchoDestino, altoDestino))
	for yDestino := 0; yDestino < altoDestino; yDestino++ {
		for xDestino := 0; xDestino < anchoDestino; xDestino++ {
			xOrigen, yOrigen := coordenadasOrigenOrientadas(xDestino, yDestino, ancho, alto, orientacion)
			destino.Set(xDestino, yDestino, imagen.At(origen.Min.X+xOrigen, origen.Min.Y+yOrigen))
		}
	}
	return destino
}

func coordenadasOrigenOrientadas(xDestino, yDestino, ancho, alto, orientacion int) (int, int) {
	switch orientacion {
	case 2:
		return ancho - 1 - xDestino, yDestino
	case 3:
		return ancho - 1 - xDestino, alto - 1 - yDestino
	case 4:
		return xDestino, alto - 1 - yDestino
	case 5:
		return yDestino, xDestino
	case 6:
		return yDestino, alto - 1 - xDestino
	case 7:
		return ancho - 1 - yDestino, alto - 1 - xDestino
	case 8:
		return ancho - 1 - yDestino, xDestino
	default:
		return xDestino, yDestino
	}
}

func (s *Servicio) generarPreviewImagenMagick(ctx context.Context, ruta string, maximo int) (image.Image, error) {
	comando := exec.CommandContext(ctx, s.rutaMagick,
		ruta,
		"-auto-orient",
		"-thumbnail", fmt.Sprintf("%dx%d>", maximo, maximo),
		"png:-",
	)
	salida, err := comando.Output()
	if err != nil {
		return nil, err
	}

	imagen, _, err := image.Decode(bytes.NewReader(salida))
	if err != nil {
		return nil, err
	}
	return imagen, nil
}

func (s *Servicio) generarPreviewImagenQuickLook(ctx context.Context, ruta string, maximo int) (image.Image, error) {
	directorioTemporal, err := os.MkdirTemp("", "dam-qlpreview-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directorioTemporal)

	comando := exec.CommandContext(ctx, s.rutaQLManage,
		"-t",
		"-s", strconv.Itoa(maximo),
		"-o", directorioTemporal,
		ruta,
	)
	if _, err := comando.CombinedOutput(); err != nil {
		return nil, err
	}

	entradas, err := os.ReadDir(directorioTemporal)
	if err != nil {
		return nil, err
	}
	for _, entrada := range entradas {
		if entrada.IsDir() {
			continue
		}
		return decodificarImagenEscalada(filepath.Join(directorioTemporal, entrada.Name()), maximo, 1)
	}

	return nil, fmt.Errorf("Quick Look no genero archivos para %q", ruta)
}

func calcularDHashDesdeImagen(imagen image.Image) string {
	reducida := image.NewGray(image.Rect(0, 0, 9, 8))
	draw.ApproxBiLinear.Scale(reducida, reducida.Bounds(), imagen, imagen.Bounds(), draw.Over, nil)

	var bits uint64
	var indice uint
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			actual := color.GrayModel.Convert(reducida.At(x, y)).(color.Gray).Y
			siguiente := color.GrayModel.Convert(reducida.At(x+1, y)).(color.Gray).Y
			if actual > siguiente {
				bits |= 1 << indice
			}
			indice++
		}
	}

	return fmt.Sprintf("%016x", bits)
}

func extraerCadena(documento map[string]any, claves ...string) string {
	for _, clave := range claves {
		if valor, existe := documento[clave]; existe {
			switch convertido := valor.(type) {
			case string:
				return strings.TrimSpace(convertido)
			case fmt.Stringer:
				return strings.TrimSpace(convertido.String())
			}
		}
	}
	return ""
}

func extraerEntero(documento map[string]any, claves ...string) int {
	for _, clave := range claves {
		if valor, existe := documento[clave]; existe {
			switch convertido := valor.(type) {
			case float64:
				return int(math.Round(convertido))
			case int:
				return convertido
			case string:
				if numero, err := strconv.Atoi(strings.TrimSpace(convertido)); err == nil {
					return numero
				}
				if numero, err := strconv.ParseFloat(strings.TrimSpace(convertido), 64); err == nil {
					return int(math.Round(numero))
				}
			}
		}
	}
	return 0
}

func extraerFlotante(documento map[string]any, claves ...string) float64 {
	for _, clave := range claves {
		if valor, existe := documento[clave]; existe {
			switch convertido := valor.(type) {
			case float64:
				return convertido
			case int:
				return float64(convertido)
			case string:
				if numero, err := strconv.ParseFloat(strings.TrimSpace(convertido), 64); err == nil {
					return numero
				}
			}
		}
	}
	return 0
}

func extraerListaCadenas(documento map[string]any, claves ...string) []string {
	for _, clave := range claves {
		if valor, existe := documento[clave]; existe {
			switch convertido := valor.(type) {
			case []any:
				var lista []string
				for _, item := range convertido {
					lista = append(lista, fmt.Sprint(item))
				}
				return lista
			case []string:
				return convertido
			case string:
				if strings.TrimSpace(convertido) == "" {
					return nil
				}
				partes := strings.Split(convertido, ",")
				return normalizarLista(partes)
			}
		}
	}
	return nil
}

func extraerRegiones(documento map[string]any) []modelo.RegionEtiquetada {
	valor, existe := documento["RegionInfo"]
	if !existe {
		return nil
	}

	mapaRegion, ok := valor.(map[string]any)
	if !ok {
		return nil
	}
	listaBruta, ok := mapaRegion["RegionList"].([]any)
	if !ok {
		return nil
	}

	var regiones []modelo.RegionEtiquetada
	for _, item := range listaBruta {
		regionMapa, ok := item.(map[string]any)
		if !ok {
			continue
		}

		nombre := extraerCadena(regionMapa, "Name", "PersonDisplayName")
		areaMapa, _ := regionMapa["Area"].(map[string]any)
		xCentro := extraerFlotante(areaMapa, "X")
		yCentro := extraerFlotante(areaMapa, "Y")
		ancho := extraerFlotante(areaMapa, "W")
		alto := extraerFlotante(areaMapa, "H")
		if ancho <= 0 || alto <= 0 {
			continue
		}

		regiones = append(regiones, modelo.RegionEtiquetada{
			Nombre: nombre,
			X:      xCentro - (ancho / 2),
			Y:      yCentro - (alto / 2),
			Ancho:  ancho,
			Alto:   alto,
		})
	}

	return regiones
}

func recolectarValoresPlano(documento map[string]any) []string {
	var valores []string
	var recorrer func(any)
	recorrer = func(valor any) {
		switch convertido := valor.(type) {
		case map[string]any:
			for _, item := range convertido {
				recorrer(item)
			}
		case []any:
			for _, item := range convertido {
				recorrer(item)
			}
		case string:
			valores = append(valores, convertido)
		}
	}
	recorrer(documento)
	return valores
}

func detectarIA(documento map[string]any, valores []string) bool {
	if contieneClaveMetadato(documento, "Parameters", "Prompt") {
		return true
	}

	patrones := []string{
		"generative ai",
		"generated by ai",
		"trainedalgorithmicmedia",
		"midjourney",
		"stable diffusion",
		"dall-e",
		"firefly",
	}
	return contieneAlgunPatron(valores, patrones)
}

func detectarSocial(valores []string) bool {
	patrones := []string{
		"fbmd",
		"instagram",
		"tiktok",
		"snapchat",
		"whatsapp",
	}
	return contieneAlgunPatron(valores, patrones)
}

func contieneEtiquetaAdulta(etiquetas []string) bool {
	for _, etiqueta := range etiquetas {
		if strings.TrimSpace(strings.ToLower(etiqueta)) == "+18" {
			return true
		}
	}
	return false
}

func contieneAlgunPatron(valores, patrones []string) bool {
	for _, valor := range valores {
		valor = strings.ToLower(valor)
		for _, patron := range patrones {
			if strings.Contains(valor, patron) {
				return true
			}
		}
	}
	return false
}

func contieneClaveMetadato(documento map[string]any, claves ...string) bool {
	objetivos := make(map[string]struct{}, len(claves))
	for _, clave := range claves {
		objetivos[normalizarClaveMetadato(clave)] = struct{}{}
	}

	var recorrer func(any) bool
	recorrer = func(valor any) bool {
		switch convertido := valor.(type) {
		case map[string]any:
			for clave, interno := range convertido {
				if _, existe := objetivos[normalizarClaveMetadato(clave)]; existe {
					return true
				}
				if recorrer(interno) {
					return true
				}
			}
		case []any:
			for _, interno := range convertido {
				if recorrer(interno) {
					return true
				}
			}
		}
		return false
	}

	return recorrer(documento)
}

func extraerExtrasMetadatos(documento map[string]any, claves ...string) map[string][]string {
	extras := make(map[string][]string)
	for _, clave := range claves {
		valores := normalizarLista(extraerValoresPorClave(documento, clave))
		if len(valores) == 0 {
			continue
		}
		extras[clave] = valores
	}
	if len(extras) == 0 {
		return nil
	}
	return extras
}

func extraerValoresPorClave(documento map[string]any, claveBuscada string) []string {
	var valores []string
	claveBuscada = normalizarClaveMetadato(claveBuscada)

	var recorrer func(any)
	recorrer = func(valor any) {
		switch convertido := valor.(type) {
		case map[string]any:
			for clave, interno := range convertido {
				if normalizarClaveMetadato(clave) == claveBuscada {
					agregarValoresTexto(interno, &valores)
				}
				recorrer(interno)
			}
		case []any:
			for _, interno := range convertido {
				recorrer(interno)
			}
		}
	}
	recorrer(documento)
	return valores
}

func agregarValoresTexto(valor any, valores *[]string) {
	switch convertido := valor.(type) {
	case string:
		texto := strings.TrimSpace(convertido)
		if texto != "" {
			*valores = append(*valores, texto)
		}
	case []string:
		for _, item := range convertido {
			agregarValoresTexto(item, valores)
		}
	case []any:
		for _, item := range convertido {
			agregarValoresTexto(item, valores)
		}
	default:
		texto := strings.TrimSpace(fmt.Sprint(convertido))
		if texto != "" && texto != "<nil>" {
			*valores = append(*valores, texto)
		}
	}
}

func normalizarClaveMetadato(clave string) string {
	reemplazador := strings.NewReplacer("_", "", "-", "", " ", "")
	return strings.ToLower(reemplazador.Replace(strings.TrimSpace(clave)))
}

func archivoSoportaExiftool(archivo modelo.Archivo) bool {
	switch archivo.Tipo {
	case modelo.TipoImagen, modelo.TipoVideo:
		return true
	default:
		return false
	}
}

func normalizarLista(entrada []string) []string {
	vistos := make(map[string]struct{}, len(entrada))
	var salida []string
	for _, item := range entrada {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		clave := strings.ToLower(item)
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		salida = append(salida, item)
	}
	return salida
}

func salidaEnFormato(salida, formato string) string {
	formato = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(formato)), ".")
	if formato == "" {
		return salida
	}
	extension := filepath.Ext(salida)
	if strings.ToLower(strings.TrimPrefix(extension, ".")) == formato {
		return salida
	}
	base := strings.TrimSuffix(salida, extension)
	return base + "." + formato
}

func construirRegionInfoMWG(ancho, alto int, regiones []modelo.RegionEtiquetada) map[string]any {
	lista := make([]map[string]any, 0, len(regiones))
	for _, region := range regiones {
		x := limitarDecimalRegion(region.X)
		y := limitarDecimalRegion(region.Y)
		anchoRegion := limitarDecimalRegion(region.Ancho)
		altoRegion := limitarDecimalRegion(region.Alto)
		lista = append(lista, map[string]any{
			"Area": map[string]any{
				"X":    redondearDecimalRegion(x + (anchoRegion / 2)),
				"Y":    redondearDecimalRegion(y + (altoRegion / 2)),
				"W":    redondearDecimalRegion(anchoRegion),
				"H":    redondearDecimalRegion(altoRegion),
				"Unit": "normalized",
			},
			"Name": strings.TrimSpace(region.Nombre),
			"Type": "Face",
		})
	}
	return map[string]any{
		"AppliedToDimensions": map[string]any{
			"W":    ancho,
			"H":    alto,
			"Unit": "pixel",
		},
		"RegionList": lista,
	}
}

func limitarDecimalRegion(valor float64) float64 {
	if valor < 0 {
		return 0
	}
	if valor > 1 {
		return 1
	}
	return redondearDecimalRegion(valor)
}

func redondearDecimalRegion(valor float64) float64 {
	return math.Round(valor*1_000_000) / 1_000_000
}

func maximoEntero(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buscarComando(nombre string) string {
	ruta, err := exec.LookPath(nombre)
	if err != nil {
		return ""
	}
	return ruta
}

var _ io.Reader
