//go:build !darwin

package plataforma

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// SeleccionarArchivoCSV usa el selector disponible en el entorno gráfico.
func SeleccionarArchivoCSV(ctx context.Context) (string, error) {
	if rutaZenity, err := exec.LookPath("zenity"); err == nil {
		salida, err := exec.CommandContext(ctx, rutaZenity,
			"--file-selection",
			"--title=Selecciona el CSV de asociaciones",
			"--file-filter=Archivos CSV | *.csv",
		).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("no se pudo seleccionar el archivo CSV: %w", err)
		}
		return rutaSeleccionadaCSV(string(salida))
	}

	if rutaKDialog, err := exec.LookPath("kdialog"); err == nil {
		salida, err := exec.CommandContext(ctx, rutaKDialog,
			"--getopenfilename", "", "*.csv|Archivos CSV",
			"--title", "Selecciona el CSV de asociaciones",
		).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("no se pudo seleccionar el archivo CSV: %w", err)
		}
		return rutaSeleccionadaCSV(string(salida))
	}

	return "", errors.New("no hay un selector de archivos disponible; indica la ruta manualmente")
}

func rutaSeleccionadaCSV(salida string) (string, error) {
	ruta := strings.TrimSpace(salida)
	if ruta == "" {
		return "", errors.New("no se seleccionó ningún archivo CSV")
	}
	return ruta, nil
}
