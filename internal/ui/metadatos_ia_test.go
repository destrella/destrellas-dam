package ui

import (
	"testing"

	"destrellas-dam/internal/modelo"
)

func TestAnalizarBloqueMetadatosIAA1111(t *testing.T) {
	t.Parallel()

	archivo := modelo.Archivo{
		Indicadores: modelo.IndicadoresArchivo{TieneIA: true},
		Metadatos: modelo.MetadatosArchivo{
			Extras: map[string][]string{
				"Parameters": {
					"Anna Yamada, dancing with her hands above her head, happy, full shot, full body <lora:AnnaYamada:1>.\nNegative prompt: low quality, blurry, text\nSteps: 40, Sampler: DPM adaptive, Schedule type: Automatic, CFG scale: 7, Seed: 524982011, Size: 704x832, Clip skip: 2, Model: darkSushiMixMix_225D, Version: v1.10.1",
				},
			},
		},
	}

	bloque := analizarBloqueMetadatosIA(archivo, nuevaPaleta())
	if bloque.Software != "Automatic1111" {
		t.Fatalf("software inesperado: %q", bloque.Software)
	}
	if bloque.ModeloPrincipal != "darkSushiMixMix_225D" {
		t.Fatalf("modelo inesperado: %q", bloque.ModeloPrincipal)
	}
	if bloque.Prompt == "" || bloque.PromptNegativo == "" {
		t.Fatalf("se esperaban prompt y prompt negativo, obtenido: %#v", bloque)
	}
	if len(bloque.Otros) == 0 {
		t.Fatal("se esperaban detalles adicionales para A1111")
	}
}

func TestAnalizarBloqueMetadatosIAComfy(t *testing.T) {
	t.Parallel()

	archivo := modelo.Archivo{
		Indicadores: modelo.IndicadoresArchivo{TieneIA: true},
		Metadatos: modelo.MetadatosArchivo{
			Extras: map[string][]string{
				"Prompt": {
					`{
						"4": {
							"inputs": {
								"ckpt_name": "cyberrealistic_v100Redux.safetensors"
							},
							"class_type": "CheckpointLoaderSimple",
							"_meta": {"title": "Load Checkpoint"}
						},
						"5": {
							"inputs": {
								"width": 1536,
								"height": 1536,
								"batch_size": 1
							},
							"class_type": "EmptyLatentImage",
							"_meta": {"title": "Empty Latent Image"}
						},
						"6": {
							"inputs": {
								"text": "young East Asian woman, photorealistic, natural skin texture",
								"clip": ["4", 1]
							},
							"class_type": "CLIPTextEncode",
							"_meta": {"title": "CLIP Text Encode (Prompt)"}
						},
						"7": {
							"inputs": {
								"text": "low quality, blurry, text, watermark",
								"clip": ["4", 1]
							},
							"class_type": "CLIPTextEncode",
							"_meta": {"title": "CLIP Text Encode (Negative Prompt)"}
						},
						"8": {
							"inputs": {
								"seed": 1745178448207010,
								"steps": 20,
								"cfg": 7,
								"sampler_name": "dpmpp_2m",
								"scheduler": "karras",
								"denoise": 0.6,
								"model": ["4", 0],
								"positive": ["6", 0],
								"negative": ["7", 0],
								"latent_image": ["5", 0]
							},
							"class_type": "KSampler",
							"_meta": {"title": "KSampler"}
						}
					}`,
				},
			},
		},
	}

	bloque := analizarBloqueMetadatosIA(archivo, nuevaPaleta())
	if bloque.Software != "Comfy" {
		t.Fatalf("software inesperado: %q", bloque.Software)
	}
	if bloque.ModeloPrincipal != "cyberrealistic_v100Redux.safetensors" {
		t.Fatalf("modelo inesperado: %q", bloque.ModeloPrincipal)
	}
	if bloque.Prompt != "young East Asian woman, photorealistic, natural skin texture" {
		t.Fatalf("prompt inesperado: %q", bloque.Prompt)
	}
	if bloque.PromptNegativo != "low quality, blurry, text, watermark" {
		t.Fatalf("prompt negativo inesperado: %q", bloque.PromptNegativo)
	}
	if len(bloque.Otros) < 4 {
		t.Fatalf("se esperaban varios detalles Comfy, obtenido: %#v", bloque.Otros)
	}
}

func TestAnalizarBloqueMetadatosIAComfyEsDeterminista(t *testing.T) {
	t.Parallel()

	archivo := modelo.Archivo{
		Indicadores: modelo.IndicadoresArchivo{TieneIA: true},
		Metadatos: modelo.MetadatosArchivo{
			Extras: map[string][]string{
				"Prompt": {
					`{
						"10": {
							"inputs": {
								"text": "prompt principal",
								"clip": ["4", 1]
							},
							"class_type": "CLIPTextEncode",
							"_meta": {"title": "CLIP Text Encode (Prompt)"}
						},
						"11": {
							"inputs": {
								"text": "prompt negativo",
								"clip": ["4", 1]
							},
							"class_type": "CLIPTextEncode",
							"_meta": {"title": "CLIP Text Encode (Negative Prompt)"}
						},
						"12": {
							"inputs": {
								"seed": 123456789,
								"steps": 28,
								"cfg": 6.5,
								"sampler_name": "euler",
								"scheduler": "normal",
								"denoise": 0.4,
								"model": ["4", 0],
								"positive": ["10", 0],
								"negative": ["11", 0],
								"latent_image": ["5", 0]
							},
							"class_type": "KSampler",
							"_meta": {"title": "KSampler"}
						},
						"4": {
							"inputs": {
								"ckpt_name": "modelo_principal.safetensors"
							},
							"class_type": "CheckpointLoaderSimple",
							"_meta": {"title": "Load Checkpoint"}
						},
						"5": {
							"inputs": {
								"width": 1024,
								"height": 1536,
								"batch_size": 1
							},
							"class_type": "EmptyLatentImage",
							"_meta": {"title": "Empty Latent Image"}
						}
					}`,
				},
			},
		},
	}

	esperado := analizarBloqueMetadatosIA(archivo, nuevaPaleta())
	for indice := 0; indice < 100; indice++ {
		actual := analizarBloqueMetadatosIA(archivo, nuevaPaleta())
		if actual.ModeloPrincipal != esperado.ModeloPrincipal ||
			actual.Prompt != esperado.Prompt ||
			actual.PromptNegativo != esperado.PromptNegativo {
			t.Fatalf("el bloque IA cambió entre iteraciones: esperado %#v, actual %#v", esperado, actual)
		}
		if len(actual.Otros) != len(esperado.Otros) {
			t.Fatalf("la cantidad de detalles cambió entre iteraciones: esperado %#v, actual %#v", esperado.Otros, actual.Otros)
		}
		for posicion := range esperado.Otros {
			if actual.Otros[posicion].Etiqueta != esperado.Otros[posicion].Etiqueta ||
				actual.Otros[posicion].Valor != esperado.Otros[posicion].Valor {
				t.Fatalf("los detalles cambiaron entre iteraciones: esperado %#v, actual %#v", esperado.Otros, actual.Otros)
			}
		}
	}
}
