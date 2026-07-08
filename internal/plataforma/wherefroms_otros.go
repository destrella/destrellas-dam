//go:build !darwin

package plataforma

import "context"

// LeerWhereFroms no esta soportado fuera de macOS en esta primera version.
func LeerWhereFroms(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// EnviarAPapelera aplica una señal simple de no soporte fuera de macOS.
func EnviarAPapelera(_ context.Context, _ string) error {
	return nil
}
