package metadatos

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// SugerirRecorteMate intenta detectar bandas mate o letterboxing para sugerir
// un recorte que preserve la parte útil central de la imagen orientada.
func SugerirRecorteMate(imagen image.Image) (image.Rectangle, bool) {
	if imagen == nil {
		return image.Rectangle{}, false
	}

	limites := imagen.Bounds()
	ancho := limites.Dx()
	alto := limites.Dy()
	if ancho < 48 || alto < 48 {
		return limites, false
	}

	luminancias := extraerLuminancias(imagen, limites)
	puntajesFilas := puntajesFilasMate(luminancias, ancho, alto)
	puntajesColumnas := puntajesColumnasMate(luminancias, ancho, alto)

	puntajesFilas = suavizarSerie(puntajesFilas, 2)
	puntajesColumnas = suavizarSerie(puntajesColumnas, 2)

	inicioY, finY, recorteVertical := sugerirRecorteLineal(puntajesFilas)
	inicioX, finX, recorteHorizontal := sugerirRecorteLineal(puntajesColumnas)
	if !recorteVertical && !recorteHorizontal {
		return limites, false
	}

	if !recorteHorizontal {
		inicioX = 0
		finX = ancho
	}
	if !recorteVertical {
		inicioY = 0
		finY = alto
	}

	rectangulo := image.Rect(
		limites.Min.X+inicioX,
		limites.Min.Y+inicioY,
		limites.Min.X+finX,
		limites.Min.Y+finY,
	).Intersect(limites)
	if rectangulo.Empty() {
		return limites, false
	}
	if rectangulo.Dx() >= ancho-2 && rectangulo.Dy() >= alto-2 {
		return limites, false
	}
	return rectangulo, true
}

func extraerLuminancias(imagen image.Image, limites image.Rectangle) []float64 {
	ancho := limites.Dx()
	alto := limites.Dy()
	luminancias := make([]float64, ancho*alto)
	indice := 0
	for y := limites.Min.Y; y < limites.Max.Y; y++ {
		for x := limites.Min.X; x < limites.Max.X; x++ {
			luminancias[indice] = luminanciaColor(imagen.At(x, y))
			indice++
		}
	}
	return luminancias
}

func puntajesFilasMate(luminancias []float64, ancho, alto int) []float64 {
	puntajes := make([]float64, alto)
	for y := 0; y < alto; y++ {
		offset := y * ancho
		suma := 0.0
		sumaCuadrados := 0.0
		diferencia := 0.0
		anterior := luminancias[offset]
		suma += anterior
		sumaCuadrados += anterior * anterior
		for x := 1; x < ancho; x++ {
			actual := luminancias[offset+x]
			suma += actual
			sumaCuadrados += actual * actual
			diferencia += math.Abs(actual - anterior)
			anterior = actual
		}
		puntajes[y] = puntajeDetalleSerie(suma, sumaCuadrados, diferencia, ancho)
	}
	return puntajes
}

func puntajesColumnasMate(luminancias []float64, ancho, alto int) []float64 {
	puntajes := make([]float64, ancho)
	for x := 0; x < ancho; x++ {
		suma := 0.0
		sumaCuadrados := 0.0
		diferencia := 0.0
		anterior := luminancias[x]
		suma += anterior
		sumaCuadrados += anterior * anterior
		for y := 1; y < alto; y++ {
			actual := luminancias[y*ancho+x]
			suma += actual
			sumaCuadrados += actual * actual
			diferencia += math.Abs(actual - anterior)
			anterior = actual
		}
		puntajes[x] = puntajeDetalleSerie(suma, sumaCuadrados, diferencia, alto)
	}
	return puntajes
}

func puntajeDetalleSerie(suma, sumaCuadrados, diferencia float64, longitud int) float64 {
	if longitud <= 0 {
		return 0
	}
	media := suma / float64(longitud)
	varianza := sumaCuadrados/float64(longitud) - media*media
	if varianza < 0 {
		varianza = 0
	}
	desviacion := math.Sqrt(varianza)
	if longitud == 1 {
		return desviacion * 0.35
	}
	return diferencia/float64(longitud-1) + desviacion*0.35
}

func sugerirRecorteLineal(puntajes []float64) (int, int, bool) {
	longitud := len(puntajes)
	if longitud < 32 {
		return 0, longitud, false
	}

	referencia := percentilSerie(puntajes, 0.75)
	if referencia <= 0 {
		return 0, longitud, false
	}

	umbral := math.Max(1.25, referencia*0.32)
	ventana := limitarEnteroRecorte(longitud/40, 3, 10)
	inicio := detectarInicioContenido(puntajes, umbral, ventana)
	fin := detectarFinContenido(puntajes, umbral, ventana)
	if fin <= inicio {
		return 0, longitud, false
	}

	grosorInicio := inicio
	grosorFin := longitud - fin
	minimoBorde := maximoEnteroRecorte(6, longitud/80)
	longitudCentro := fin - inicio
	if longitudCentro < int(float64(longitud)*0.45) {
		return 0, longitud, false
	}
	if grosorInicio < minimoBorde && grosorFin < minimoBorde {
		return 0, longitud, false
	}
	if grosorInicio > longitud/3 || grosorFin > longitud/3 {
		return 0, longitud, false
	}

	centro := promedioSerie(puntajes[inicio:fin])
	if centro <= 0 {
		return 0, longitud, false
	}

	sumaBordes := 0.0
	segmentos := 0
	if grosorInicio > 0 {
		sumaBordes += promedioSerie(puntajes[:inicio])
		segmentos++
	}
	if grosorFin > 0 {
		sumaBordes += promedioSerie(puntajes[fin:])
		segmentos++
	}
	if segmentos == 0 {
		return 0, longitud, false
	}
	bordes := sumaBordes / float64(segmentos)
	if bordes >= centro*0.62 {
		return 0, longitud, false
	}

	return inicio, fin, true
}

func detectarInicioContenido(puntajes []float64, umbral float64, ventana int) int {
	for indice := 0; indice <= len(puntajes)-ventana; indice++ {
		if promedioSerie(puntajes[indice:indice+ventana]) >= umbral {
			return indice
		}
	}
	return 0
}

func detectarFinContenido(puntajes []float64, umbral float64, ventana int) int {
	for indice := len(puntajes) - ventana; indice >= 0; indice-- {
		if promedioSerie(puntajes[indice:indice+ventana]) >= umbral {
			return indice + ventana
		}
	}
	return len(puntajes)
}

func suavizarSerie(datos []float64, radio int) []float64 {
	if radio <= 0 || len(datos) == 0 {
		return append([]float64(nil), datos...)
	}
	salida := make([]float64, len(datos))
	for indice := range datos {
		inicio := maximoEnteroRecorte(0, indice-radio)
		fin := minimoEnteroRecorte(len(datos), indice+radio+1)
		salida[indice] = promedioSerie(datos[inicio:fin])
	}
	return salida
}

func promedioSerie(datos []float64) float64 {
	if len(datos) == 0 {
		return 0
	}
	suma := 0.0
	for _, valor := range datos {
		suma += valor
	}
	return suma / float64(len(datos))
}

func percentilSerie(datos []float64, fraccion float64) float64 {
	if len(datos) == 0 {
		return 0
	}
	if fraccion <= 0 {
		fraccion = 0
	}
	if fraccion >= 1 {
		fraccion = 1
	}
	copia := append([]float64(nil), datos...)
	sort.Float64s(copia)
	indice := int(math.Round(fraccion * float64(len(copia)-1)))
	return copia[indice]
}

func luminanciaColor(valor color.Color) float64 {
	rojo, verde, azul, _ := valor.RGBA()
	r := float64(rojo >> 8)
	g := float64(verde >> 8)
	b := float64(azul >> 8)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func maximoEnteroRecorte(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minimoEnteroRecorte(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func limitarEnteroRecorte(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}
