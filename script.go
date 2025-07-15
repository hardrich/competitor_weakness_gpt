package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file, ensure it exists and contains GEMINI_API_KEY: %v", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not found in environment variables. Please set it.")
	}

	ctx := context.Background()

	// Crea un cliente para la API de Gemini.
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close() // Asegúrate de cerrar el cliente cuando termines.

	// models := client.ListModels(ctx)

	// for {
	// 	model, err := models.Next()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	// Imprime el nombre del modelo y los métodos que soporta
	// 	fmt.Printf("  - Nombre: %s, Métodos soportados: %v\n", model.Name, model.SupportedGenerationMethods)
	// }

	// Selecciona el modelo Gemini Pro.
	model := client.GenerativeModel("gemini-2.5-pro")

	// Envía una solicitud de generación de texto.
	resp, err := model.GenerateContent(ctx, genai.Text("¿Cuál es el significado de la vida?"))
	if err != nil {
		log.Fatal(err)
	}

	// Imprime la respuesta.
	for _, part := range resp.Candidates[0].Content.Parts {
		fmt.Print(part)
	}
	fmt.Println()
}
