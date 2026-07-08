package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"destrellas-dam/internal/modelo"
)

type bloqueMetadatosIA struct {
	Titulo          string
	Software        string
	ModeloPrincipal string
	Prompt          string
	PromptNegativo  string
	Otros           []lineaDetalleIA
}

type lineaDetalleIA struct {
	Etiqueta   string
	Valor      string
	Destacada  bool
	ColorValor color.NRGBA
}

type nodoComfyUI struct {
	ID       string
	Tipo     string
	Titulo   string
	Entradas map[string]any
}

var (
	expresionLoraPromptIA = regexp.MustCompile(`(?i)<lora:([^:>]+)(?::([^>]+))?>`)
	expresionModeloIA     = regexp.MustCompile(`(?im)^\s*(?:model|model name|sd model name|checkpoint|base model)\s*:\s*([^\n\r,]+)`)
	expresionLineaStepsIA = regexp.MustCompile(`(?i)\bsteps\s*:`)
)

func (b bloqueMetadatosIA) TieneContenido() bool {
	return strings.TrimSpace(b.Prompt) != "" ||
		strings.TrimSpace(b.PromptNegativo) != "" ||
		strings.TrimSpace(b.ModeloPrincipal) != "" ||
		len(b.Otros) > 0
}

func analizarBloqueMetadatosIA(archivo modelo.Archivo, paleta Paleta) bloqueMetadatosIA {
	if !archivo.Indicadores.TieneIA {
		return bloqueMetadatosIA{}
	}

	parametrosSD := concatenarExtraIA(archivo, "Parameters")
	if strings.TrimSpace(parametrosSD) != "" {
		return analizarParametrosStableDiffusion(parametrosSD, paleta)
	}

	promptComfy := concatenarExtraIA(archivo, "Prompt")
	if strings.TrimSpace(promptComfy) != "" {
		return analizarPromptComfy(promptComfy, paleta)
	}

	textoIA := textoIAArchivo(archivo)
	return bloqueMetadatosIA{
		Titulo:          "Metadatos IA",
		Software:        inferirSoftwareIA(textoIA),
		ModeloPrincipal: inferirModeloIA(textoIA),
	}
}

func concatenarExtraIA(archivo modelo.Archivo, clave string) string {
	if archivo.Metadatos.Extras == nil {
		return ""
	}
	valores := archivo.Metadatos.Extras[clave]
	if len(valores) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(valores, "\n\n"))
}

// analizarParametrosStableDiffusion separa el formato lineal usado por
// Automatic1111 en prompt, prompt negativo y pares clave:valor.
func analizarParametrosStableDiffusion(texto string, paleta Paleta) bloqueMetadatosIA {
	texto = normalizarSaltosIA(texto)
	prompt, promptNegativo, detallePlano := separarParametrosStableDiffusion(texto)
	campos := parsearCamposClaveValorIA(detallePlano)
	modelo := extraerModeloCamposIA(campos)

	otros := make([]lineaDetalleIA, 0, len(campos)+4)
	if modelo != "" {
		otros = append(otros, lineaDetalleIA{
			Etiqueta:   "Modelo",
			Valor:      modelo,
			Destacada:  true,
			ColorValor: paleta.Acento,
		})
	}

	for _, lora := range extraerLorasPromptIA(prompt) {
		otros = append(otros, lineaDetalleIA{
			Etiqueta:   "LoRA",
			Valor:      lora,
			ColorValor: paleta.Texto,
		})
	}

	for _, campo := range campos {
		if esCampoModeloIA(campo.Etiqueta) {
			continue
		}
		otros = append(otros, lineaDetalleIA{
			Etiqueta:   etiquetaPresentableDetalleIA(campo.Etiqueta),
			Valor:      campo.Valor,
			ColorValor: paleta.Texto,
		})
	}

	if modelo == "" {
		modelo = inferirModeloIA(texto)
	}

	return bloqueMetadatosIA{
		Titulo:          "Parámetros SD",
		Software:        "Automatic1111",
		ModeloPrincipal: modelo,
		Prompt:          prompt,
		PromptNegativo:  promptNegativo,
		Otros:           otros,
	}
}

func analizarPromptComfy(texto string, paleta Paleta) bloqueMetadatosIA {
	texto = strings.TrimSpace(normalizarSaltosIA(texto))
	if bloque, ok := analizarPromptComfyComoJSON(texto, paleta); ok {
		return bloque
	}

	return bloqueMetadatosIA{
		Titulo:          "Parámetros ComfyUI",
		Software:        "Comfy",
		ModeloPrincipal: inferirModeloIA(texto),
		Prompt:          texto,
	}
}

// analizarPromptComfyComoJSON intenta leer el grafo serializado de ComfyUI
// para resolver sus nodos principales sin depender de cadenas sueltas.
func analizarPromptComfyComoJSON(texto string, paleta Paleta) (bloqueMetadatosIA, bool) {
	documento, ok := parsearJSONIA(texto)
	if !ok {
		return bloqueMetadatosIA{}, false
	}

	if promptInterno, existe := documento["prompt"]; existe {
		if mapaPrompt, ok := promptInterno.(map[string]any); ok {
			documento = mapaPrompt
		}
	}

	nodos := construirNodosComfy(documento)
	if len(nodos) == 0 {
		return bloqueMetadatosIA{}, false
	}

	bloque := bloqueMetadatosIA{
		Titulo:   "Parámetros ComfyUI",
		Software: "Comfy",
	}

	nodoSampler, existeSampler := seleccionarNodoSamplerComfy(nodos)
	if existeSampler {
		bloque.Prompt = resolverTextoComfy(nodos, nodoSampler.Entradas["positive"], make(map[string]bool))
		bloque.PromptNegativo = resolverTextoComfy(nodos, nodoSampler.Entradas["negative"], make(map[string]bool))
		bloque.ModeloPrincipal = resolverModeloComfy(nodos, nodoSampler.Entradas["model"], make(map[string]bool))
	}

	if strings.TrimSpace(bloque.Prompt) == "" || strings.TrimSpace(bloque.PromptNegativo) == "" {
		prompt, negativo := localizarPromptsComfy(nodos)
		if strings.TrimSpace(bloque.Prompt) == "" {
			bloque.Prompt = prompt
		}
		if strings.TrimSpace(bloque.PromptNegativo) == "" {
			bloque.PromptNegativo = negativo
		}
	}

	if strings.TrimSpace(bloque.ModeloPrincipal) == "" {
		bloque.ModeloPrincipal = localizarModeloComfy(nodos)
	}

	var otros []lineaDetalleIA
	if bloque.ModeloPrincipal != "" {
		otros = append(otros, lineaDetalleIA{
			Etiqueta:   "Modelo",
			Valor:      bloque.ModeloPrincipal,
			Destacada:  true,
			ColorValor: paleta.Acento,
		})
	}

	if existeSampler {
		if tamano := resolverTamanoComfy(nodos, nodoSampler.Entradas["latent_image"], make(map[string]bool)); tamano != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Size", Valor: tamano, ColorValor: paleta.Texto})
		}
		if seed := formatearValorIA(nodoSampler.Entradas["seed"]); seed != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Seed", Valor: seed, ColorValor: paleta.Texto})
		}
		if steps := formatearValorIA(nodoSampler.Entradas["steps"]); steps != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Steps", Valor: steps, ColorValor: paleta.Texto})
		}
		if cfg := formatearValorIA(nodoSampler.Entradas["cfg"]); cfg != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "CFG scale", Valor: cfg, ColorValor: paleta.Texto})
		}
		if sampler := formatearValorIA(nodoSampler.Entradas["sampler_name"]); sampler != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Sampler", Valor: sampler, ColorValor: paleta.Texto})
		}
		if scheduler := formatearValorIA(nodoSampler.Entradas["scheduler"]); scheduler != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Scheduler", Valor: scheduler, ColorValor: paleta.Texto})
		}
		if denoise := formatearValorIA(nodoSampler.Entradas["denoise"]); denoise != "" {
			otros = append(otros, lineaDetalleIA{Etiqueta: "Denoise", Valor: denoise, ColorValor: paleta.Texto})
		}
	}

	bloque.Otros = otros
	return bloque, bloque.TieneContenido()
}

func parsearJSONIA(texto string) (map[string]any, bool) {
	decodificador := json.NewDecoder(strings.NewReader(texto))
	decodificador.UseNumber()

	var documento map[string]any
	if err := decodificador.Decode(&documento); err != nil {
		return nil, false
	}
	return documento, true
}

func construirNodosComfy(documento map[string]any) map[string]nodoComfyUI {
	nodos := make(map[string]nodoComfyUI)
	for id, valor := range documento {
		mapaNodo, ok := valor.(map[string]any)
		if !ok {
			continue
		}

		entradas, _ := mapaNodo["inputs"].(map[string]any)
		tipo := strings.TrimSpace(formatearValorIA(mapaNodo["class_type"]))
		titulo := ""
		if meta, ok := mapaNodo["_meta"].(map[string]any); ok {
			titulo = strings.TrimSpace(formatearValorIA(meta["title"]))
		}

		nodos[id] = nodoComfyUI{
			ID:       id,
			Tipo:     tipo,
			Titulo:   titulo,
			Entradas: entradas,
		}
	}
	return nodos
}

func seleccionarNodoSamplerComfy(nodos map[string]nodoComfyUI) (nodoComfyUI, bool) {
	for _, id := range idsNodosComfyOrdenados(nodos) {
		nodo := nodos[id]
		tipo := strings.ToLower(strings.TrimSpace(nodo.Tipo))
		if strings.Contains(tipo, "ksampler") && !strings.Contains(tipo, "select") {
			return nodo, true
		}
	}
	return nodoComfyUI{}, false
}

func localizarPromptsComfy(nodos map[string]nodoComfyUI) (prompt, negativo string) {
	for _, id := range idsNodosComfyOrdenados(nodos) {
		nodo := nodos[id]
		texto := strings.TrimSpace(formatearValorIA(nodo.Entradas["text"]))
		if texto == "" {
			continue
		}
		titulo := strings.ToLower(strings.TrimSpace(nodo.Titulo + " " + nodo.Tipo))
		if strings.Contains(titulo, "negative") {
			if negativo == "" {
				negativo = texto
			}
			continue
		}
		if prompt == "" {
			prompt = texto
		}
	}
	return prompt, negativo
}

func localizarModeloComfy(nodos map[string]nodoComfyUI) string {
	for _, id := range idsNodosComfyOrdenados(nodos) {
		nodo := nodos[id]
		if modelo := extraerModeloEntradasComfy(nodo.Entradas); modelo != "" {
			return modelo
		}
	}
	return ""
}

func resolverTextoComfy(nodos map[string]nodoComfyUI, referencia any, visitados map[string]bool) string {
	nodo, ok := resolverNodoComfy(nodos, referencia)
	if !ok {
		return strings.TrimSpace(formatearValorIA(referencia))
	}
	if visitados[nodo.ID] {
		return ""
	}
	visitados[nodo.ID] = true

	if texto := strings.TrimSpace(formatearValorIA(nodo.Entradas["text"])); texto != "" {
		return texto
	}
	for _, clave := range []string{"prompt", "string", "value"} {
		if texto := strings.TrimSpace(formatearValorIA(nodo.Entradas[clave])); texto != "" {
			return texto
		}
	}
	for _, clave := range clavesMapaComfyOrdenadas(nodo.Entradas) {
		valor := nodo.Entradas[clave]
		if texto := resolverTextoComfy(nodos, valor, visitados); texto != "" {
			return texto
		}
	}
	return ""
}

func resolverModeloComfy(nodos map[string]nodoComfyUI, referencia any, visitados map[string]bool) string {
	nodo, ok := resolverNodoComfy(nodos, referencia)
	if !ok {
		return ""
	}
	if visitados[nodo.ID] {
		return ""
	}
	visitados[nodo.ID] = true

	if modelo := extraerModeloEntradasComfy(nodo.Entradas); modelo != "" {
		return modelo
	}
	for _, clave := range clavesMapaComfyOrdenadas(nodo.Entradas) {
		valor := nodo.Entradas[clave]
		if modelo := resolverModeloComfy(nodos, valor, visitados); modelo != "" {
			return modelo
		}
	}
	return ""
}

func resolverTamanoComfy(nodos map[string]nodoComfyUI, referencia any, visitados map[string]bool) string {
	nodo, ok := resolverNodoComfy(nodos, referencia)
	if !ok {
		return ""
	}
	if visitados[nodo.ID] {
		return ""
	}
	visitados[nodo.ID] = true

	ancho, okAncho := extraerEnteroIA(nodo.Entradas["width"])
	alto, okAlto := extraerEnteroIA(nodo.Entradas["height"])
	if okAncho && okAlto && ancho > 0 && alto > 0 {
		return fmt.Sprintf("%d x %d", ancho, alto)
	}

	for _, clave := range clavesMapaComfyOrdenadas(nodo.Entradas) {
		valor := nodo.Entradas[clave]
		if tamano := resolverTamanoComfy(nodos, valor, visitados); tamano != "" {
			return tamano
		}
	}
	return ""
}

func resolverNodoComfy(nodos map[string]nodoComfyUI, referencia any) (nodoComfyUI, bool) {
	if referencia == nil {
		return nodoComfyUI{}, false
	}

	switch convertido := referencia.(type) {
	case string:
		nodo, ok := nodos[convertido]
		return nodo, ok
	case []any:
		if len(convertido) == 0 {
			return nodoComfyUI{}, false
		}
		identificador := strings.TrimSpace(formatearValorIA(convertido[0]))
		nodo, ok := nodos[identificador]
		return nodo, ok
	}
	return nodoComfyUI{}, false
}

func extraerModeloEntradasComfy(entradas map[string]any) string {
	if entradas == nil {
		return ""
	}
	for _, clave := range []string{"ckpt_name", "base_ckpt_name", "model_name", "unet_name", "checkpoint", "base_model"} {
		valor := strings.TrimSpace(formatearValorIA(entradas[clave]))
		if valor != "" {
			return valor
		}
	}
	return ""
}

func idsNodosComfyOrdenados(nodos map[string]nodoComfyUI) []string {
	ids := make([]string, 0, len(nodos))
	for id := range nodos {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return compararClavesNodosComfy(ids[i], ids[j])
	})
	return ids
}

func clavesMapaComfyOrdenadas(entradas map[string]any) []string {
	claves := make([]string, 0, len(entradas))
	for clave := range entradas {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	return claves
}

func compararClavesNodosComfy(a, b string) bool {
	enteroA, errA := strconv.Atoi(strings.TrimSpace(a))
	enteroB, errB := strconv.Atoi(strings.TrimSpace(b))
	if errA == nil && errB == nil {
		return enteroA < enteroB
	}
	return a < b
}

func separarParametrosStableDiffusion(texto string) (prompt, promptNegativo, detallePlano string) {
	lineas := strings.Split(texto, "\n")
	promptLineas := make([]string, 0, len(lineas))
	negativoLineas := make([]string, 0, len(lineas))
	detalleLineas := make([]string, 0, 2)
	estado := "prompt"

	for _, linea := range lineas {
		lineaLimpia := strings.TrimSpace(linea)
		lineaMinuscula := strings.ToLower(lineaLimpia)

		switch {
		case strings.HasPrefix(lineaMinuscula, "negative prompt:"):
			estado = "negativo"
			resto := strings.TrimSpace(lineaLimpia[len("Negative prompt:"):])
			if resto != "" {
				negativoLineas = append(negativoLineas, resto)
			}
		case expresionLineaStepsIA.MatchString(lineaLimpia):
			estado = "detalle"
			if lineaLimpia != "" {
				detalleLineas = append(detalleLineas, lineaLimpia)
			}
		default:
			switch estado {
			case "prompt":
				if lineaLimpia != "" {
					promptLineas = append(promptLineas, lineaLimpia)
				}
			case "negativo":
				if lineaLimpia != "" {
					negativoLineas = append(negativoLineas, lineaLimpia)
				}
			case "detalle":
				if lineaLimpia != "" {
					detalleLineas = append(detalleLineas, lineaLimpia)
				}
			}
		}
	}

	return strings.TrimSpace(strings.Join(promptLineas, "\n")),
		strings.TrimSpace(strings.Join(negativoLineas, "\n")),
		strings.TrimSpace(strings.Join(detalleLineas, ", "))
}

func parsearCamposClaveValorIA(texto string) []lineaDetalleIA {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return nil
	}

	partes := segmentarCamposIA(texto)
	campos := make([]lineaDetalleIA, 0, len(partes))
	for _, parte := range partes {
		clave, valor, ok := strings.Cut(parte, ":")
		if !ok {
			continue
		}
		clave = strings.TrimSpace(clave)
		valor = strings.TrimSpace(valor)
		if clave == "" || valor == "" {
			continue
		}
		campos = append(campos, lineaDetalleIA{
			Etiqueta: clave,
			Valor:    valor,
		})
	}
	return campos
}

func segmentarCamposIA(texto string) []string {
	texto = strings.ReplaceAll(texto, "\n", ", ")
	inicio := 0
	enComillas := false
	profundidad := 0
	partes := make([]string, 0, 12)

	for indice, caracter := range texto {
		switch caracter {
		case '"':
			enComillas = !enComillas
		case '[', '{', '(':
			if !enComillas {
				profundidad++
			}
		case ']', '}', ')':
			if !enComillas && profundidad > 0 {
				profundidad--
			}
		case ',':
			if enComillas || profundidad > 0 {
				continue
			}
			parte := strings.TrimSpace(texto[inicio:indice])
			if parte != "" {
				partes = append(partes, parte)
			}
			inicio = indice + 1
		}
	}

	ultimaParte := strings.TrimSpace(texto[inicio:])
	if ultimaParte != "" {
		partes = append(partes, ultimaParte)
	}
	return partes
}

func extraerModeloCamposIA(campos []lineaDetalleIA) string {
	for _, campo := range campos {
		if esCampoModeloIA(campo.Etiqueta) {
			return strings.TrimSpace(campo.Valor)
		}
	}
	return ""
}

func esCampoModeloIA(etiqueta string) bool {
	normalizada := strings.ToLower(strings.TrimSpace(etiqueta))
	switch normalizada {
	case "model", "model name", "sd model name", "checkpoint", "base model":
		return true
	default:
		return false
	}
}

func etiquetaPresentableDetalleIA(etiqueta string) string {
	switch strings.ToLower(strings.TrimSpace(etiqueta)) {
	case "cfg scale":
		return "CFG scale"
	case "clip skip":
		return "Clip skip"
	case "lora hashes":
		return "LoRA hashes"
	default:
		return strings.TrimSpace(etiqueta)
	}
}

func extraerLorasPromptIA(prompt string) []string {
	coincidencias := expresionLoraPromptIA.FindAllStringSubmatch(prompt, -1)
	if len(coincidencias) == 0 {
		return nil
	}

	salida := make([]string, 0, len(coincidencias))
	vistos := make(map[string]struct{}, len(coincidencias))
	for _, coincidencia := range coincidencias {
		if len(coincidencia) < 2 {
			continue
		}
		nombre := strings.TrimSpace(coincidencia[1])
		peso := ""
		if len(coincidencia) >= 3 {
			peso = strings.TrimSpace(coincidencia[2])
		}
		if nombre == "" {
			continue
		}

		texto := nombre
		if peso != "" {
			texto = fmt.Sprintf("%s (peso %s)", nombre, peso)
		}
		clave := strings.ToLower(texto)
		if _, existe := vistos[clave]; existe {
			continue
		}
		vistos[clave] = struct{}{}
		salida = append(salida, texto)
	}
	return salida
}

func inferirSoftwareIA(texto string) string {
	texto = strings.ToLower(texto)
	switch {
	case strings.Contains(texto, "comfyui"), strings.Contains(texto, "\"class_type\":\"ksampler\""), strings.Contains(texto, "\"class_type\": \"ksampler\""):
		return "Comfy"
	case strings.Contains(texto, "automatic1111"), strings.Contains(texto, "negative prompt:"), strings.Contains(texto, "steps:"), strings.Contains(texto, "sampler:"):
		return "Automatic1111"
	case strings.Contains(texto, "invokeai"):
		return "InvokeAI"
	case strings.Contains(texto, "midjourney"):
		return "Midjourney"
	case strings.Contains(texto, "dall-e"), strings.Contains(texto, "openai"):
		return "DALL-E"
	case strings.Contains(texto, "firefly"):
		return "Adobe Firefly"
	default:
		return ""
	}
}

func inferirModeloIA(texto string) string {
	coincidencias := expresionModeloIA.FindStringSubmatch(texto)
	if len(coincidencias) < 2 {
		return ""
	}
	return strings.TrimSpace(coincidencias[1])
}

func formatearValorIA(valor any) string {
	switch convertido := valor.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(convertido)
	case json.Number:
		return strings.TrimSpace(convertido.String())
	case float64:
		if convertido == float64(int64(convertido)) {
			return strconv.FormatInt(int64(convertido), 10)
		}
		return strconv.FormatFloat(convertido, 'f', -1, 64)
	case float32:
		if convertido == float32(int64(convertido)) {
			return strconv.FormatInt(int64(convertido), 10)
		}
		return strconv.FormatFloat(float64(convertido), 'f', -1, 32)
	case int:
		return strconv.Itoa(convertido)
	case int64:
		return strconv.FormatInt(convertido, 10)
	case int32:
		return strconv.FormatInt(int64(convertido), 10)
	case uint64:
		return strconv.FormatUint(convertido, 10)
	case uint32:
		return strconv.FormatUint(uint64(convertido), 10)
	case bool:
		if convertido {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(convertido))
	}
}

func extraerEnteroIA(valor any) (int, bool) {
	texto := formatearValorIA(valor)
	if texto == "" {
		return 0, false
	}
	entero, err := strconv.Atoi(texto)
	if err == nil {
		return entero, true
	}
	flotante, err := strconv.ParseFloat(texto, 64)
	if err != nil {
		return 0, false
	}
	return int(flotante), true
}

func normalizarSaltosIA(texto string) string {
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	texto = strings.ReplaceAll(texto, "\r", "\n")
	return strings.TrimSpace(texto)
}

func (a *Aplicacion) dibujarBloqueMetadatosIA(gtx layout.Context, archivo modelo.Archivo) layout.Dimensions {
	bloque := analizarBloqueMetadatosIA(archivo, a.paleta)
	if !bloque.TieneContenido() {
		return layout.Dimensions{}
	}

	return dibujarPanel(gtx, a.paleta.PanelElevado, unit.Dp(16), func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			hijos := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTituloPanel(gtx, bloque.Titulo)
				}),
			}

			if strings.TrimSpace(bloque.Prompt) != "" {
				hijos = append(hijos,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarSeccionMetadatosIA(gtx, "Prompt", bloque.Prompt, a.paleta.Texto)
					}),
				)
			}

			if strings.TrimSpace(bloque.PromptNegativo) != "" {
				hijos = append(hijos,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarSeccionMetadatosIA(gtx, "Prompt negativo", bloque.PromptNegativo, a.paleta.Peligro)
					}),
				)
			}

			if len(bloque.Otros) > 0 {
				hijos = append(hijos,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dibujarSeccionOtrosMetadatosIA(gtx, bloque.Otros)
					}),
				)
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
		})
	})
}

func (a *Aplicacion) dibujarSeccionMetadatosIA(gtx layout.Context, titulo, contenido string, colorTexto color.NRGBA) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoPrincipal(gtx, titulo)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.dibujarTextoDetalleIA(gtx, contenido, colorTexto)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarSeccionOtrosMetadatosIA(gtx layout.Context, lineas []lineaDetalleIA) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dibujarTextoPrincipal(gtx, "Otros")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return dibujarPanelConBorde(gtx, a.paleta.Panel, a.paleta.Borde, unit.Dp(12), unit.Dp(1), func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					hijos := make([]layout.FlexChild, 0, len(lineas)*2)
					for indice, linea := range lineas {
						linea := linea
						hijos = append(hijos, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dibujarLineaDetalleIA(gtx, linea)
						}))
						if indice < len(lineas)-1 {
							hijos = append(hijos, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
						}
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, hijos...)
				})
			})
		}),
	)
}

func (a *Aplicacion) dibujarLineaDetalleIA(gtx layout.Context, linea lineaDetalleIA) layout.Dimensions {
	texto := "• " + strings.TrimSpace(linea.Etiqueta) + ": " + strings.TrimSpace(linea.Valor)
	colorTexto := linea.ColorValor
	if colorTexto == (color.NRGBA{}) {
		colorTexto = a.paleta.Texto
	}
	return a.dibujarTextoDetalleIA(gtx, texto, colorTexto)
}

func (a *Aplicacion) dibujarTextoDetalleIA(gtx layout.Context, texto string, colorTexto color.NRGBA) layout.Dimensions {
	estilo := material.Label(a.tema, unit.Sp(13), texto)
	estilo.Color = colorTexto
	return estilo.Layout(gtx)
}
