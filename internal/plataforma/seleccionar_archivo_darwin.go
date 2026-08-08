//go:build darwin

package plataforma

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// SeleccionarArchivoCSV abre el selector de archivos del sistema para elegir un CSV.
func SeleccionarArchivoCSV(ctx context.Context) (string, error) {
	const script = `set archivoElegido to choose file with prompt "Selecciona el CSV de asociaciones"
POSIX path of archivoElegido`

	salida, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("no se pudo seleccionar el archivo CSV: %w", err)
	}

	ruta := strings.TrimSpace(string(salida))
	if ruta == "" {
		return "", errors.New("no se seleccionó ningún archivo CSV")
	}
	return ruta, nil
}
