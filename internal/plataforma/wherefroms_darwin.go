//go:build darwin

package plataforma

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// LeerWhereFroms usa Spotlight para recuperar el atributo extendido donde aplique.
func LeerWhereFroms(ctx context.Context, ruta string) ([]string, error) {
	salida, err := exec.CommandContext(ctx, "mdls", "-raw", "-name", "kMDItemWhereFroms", ruta).CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, ctx.Err()
		}
		if _, esErrorSalida := err.(*exec.ExitError); esErrorSalida {
			return nil, nil
		}
		return nil, fmt.Errorf("no se pudo consultar kMDItemWhereFroms: %w", err)
	}

	texto := strings.TrimSpace(string(salida))
	if texto == "" || texto == "(null)" {
		return nil, nil
	}

	texto = strings.TrimPrefix(texto, "(")
	texto = strings.TrimSuffix(texto, ")")
	lineas := strings.Split(texto, "\n")
	var valores []string
	for _, linea := range lineas {
		linea = strings.TrimSpace(strings.TrimSuffix(linea, ","))
		linea = strings.Trim(linea, `"`)
		if linea == "" {
			continue
		}
		valores = append(valores, linea)
	}

	return valores, nil
}

// EnviarAPapelera usa Finder para respetar la papelera del sistema.
func EnviarAPapelera(ctx context.Context, ruta string) error {
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file %q`, ruta)
	return exec.CommandContext(ctx, "osascript", "-e", script).Run()
}
