package modelo

import "math"

// NormalizarOrientacionVisual reduce valores EXIF invalidos al caso neutro.
func NormalizarOrientacionVisual(orientacion int) int {
	switch orientacion {
	case 2, 3, 4, 5, 6, 7, 8:
		return orientacion
	default:
		return 1
	}
}

// OrientacionInversa devuelve la orientacion inversa para volver al sistema original.
func OrientacionInversa(orientacion int) int {
	switch NormalizarOrientacionVisual(orientacion) {
	case 6:
		return 8
	case 8:
		return 6
	default:
		return NormalizarOrientacionVisual(orientacion)
	}
}

// NormalizarRotacionCuartos ajusta la rotacion al cuarto de vuelta mas cercano.
func NormalizarRotacionCuartos(rotacion int) int {
	if rotacion == 0 {
		return 0
	}
	normalizada := rotacion % 360
	if normalizada < 0 {
		normalizada += 360
	}
	if normalizada%90 == 0 {
		return normalizada
	}
	normalizada = int(math.Round(float64(normalizada)/90.0)) * 90
	normalizada %= 360
	if normalizada < 0 {
		normalizada += 360
	}
	return normalizada
}

// TransformarPuntoOrientado mueve un punto normalizado desde la imagen original a la orientada.
func TransformarPuntoOrientado(x, y float64, orientacion int) (float64, float64) {
	switch NormalizarOrientacionVisual(orientacion) {
	case 2:
		return 1 - x, y
	case 3:
		return 1 - x, 1 - y
	case 4:
		return x, 1 - y
	case 5:
		return y, x
	case 6:
		return 1 - y, x
	case 7:
		return 1 - y, 1 - x
	case 8:
		return y, 1 - x
	default:
		return x, y
	}
}

// TransformarRegionOrientada calcula la caja visible de una region tras aplicar Orientation.
func TransformarRegionOrientada(region RegionEtiquetada, orientacion int) RegionEtiquetada {
	x1, y1 := TransformarPuntoOrientado(region.X, region.Y, orientacion)
	x2, y2 := TransformarPuntoOrientado(region.X+region.Ancho, region.Y, orientacion)
	x3, y3 := TransformarPuntoOrientado(region.X, region.Y+region.Alto, orientacion)
	x4, y4 := TransformarPuntoOrientado(region.X+region.Ancho, region.Y+region.Alto, orientacion)

	minimoX := math.Min(math.Min(x1, x2), math.Min(x3, x4))
	maximoX := math.Max(math.Max(x1, x2), math.Max(x3, x4))
	minimoY := math.Min(math.Min(y1, y2), math.Min(y3, y4))
	maximoY := math.Max(math.Max(y1, y2), math.Max(y3, y4))

	return RegionEtiquetada{
		Nombre: region.Nombre,
		X:      limitarFraccionUnitaria(minimoX),
		Y:      limitarFraccionUnitaria(minimoY),
		Ancho:  limitarFraccionUnitaria(maximoX - minimoX),
		Alto:   limitarFraccionUnitaria(maximoY - minimoY),
	}
}

// InvertirRegionOrientada devuelve una region orientada al sistema de coordenadas original.
func InvertirRegionOrientada(region RegionEtiquetada, orientacion int) RegionEtiquetada {
	return TransformarRegionOrientada(region, OrientacionInversa(orientacion))
}

func limitarFraccionUnitaria(valor float64) float64 {
	if valor < 0 {
		return 0
	}
	if valor > 1 {
		return 1
	}
	return valor
}
