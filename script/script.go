package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	genai "cloud.google.com/go/vertexai/genai"
)

type Review struct {
	ReviewerID     string  `json:"reviewerID"`
	ASIN           string  `json:"asin"`
	ReviewerName   string  `json:"reviewerName"`
	ReviewText     string  `json:"reviewText"`
	Overall        float64 `json:"overall"`
	UnixReviewTime int64   `json:"unixReviewTime"` // Using int64 for Unix timestamp
	ReviewTime     string  `json:"reviewTime"`
}

const promptHeader = `You are an expert e-commerce product review analyst. Your task is to analyze a batch of product reviews and return a JSON array. Each item in the array must strictly follow this schema:

{
  "sentiment": "Positive" | "Negative" | "Neutral",
  "weaknesses": [up to 3 lowercase keywords],
  "theme": "Shipping" | "Material" | "Functionality" | "Performance" | "Price" | "Support" | "Design" | "Experience" | "Compatibility" | "Accuracy" | "Maintenance" | "Assembly" | "General"
}

### Rules:
- Analyze **each review independently**.
- Return a **strictly valid JSON array**, where each item corresponds to the same index of the input list.
- Do not include any explanation or extra formatting.
- If no weaknesses are found, use an empty array for "weaknesses".
- Choose the most fitting theme from the list. If not sure, use "General".

### Reviews:
`

const (
	reviewsFilePath = "mapped.json"
	outputFilePath  = "analyzed_reviews_batch.json"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file, ensure it exists and contains GEMINI_API_KEY: %v", err)
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")

	fileContent, err := os.ReadFile(reviewsFilePath)
	if err != nil {
		log.Fatalf("Error reading reviews file %s, %v", reviewsFilePath, err)
	}

	var reviews []Review
	err = json.Unmarshal(fileContent, &reviews)
	if err != nil {
		log.Fatalf("Error unmarshaling reviews json: %v", err)
	}

	log.Printf("Successfully loaded %d reviews from %s\n", len(reviews), reviewsFilePath)

	// Construir el cuerpo del prompt con múltiples reseñas
	var sb strings.Builder
	sb.WriteString(promptHeader)
	for i, r := range reviews {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.ReviewText))
	}

	prompt := sb.String()

	log.Printf("Prompt %s\n", prompt)

	// Recomendación: limitar el número de reseñas si es muy largo
	if len(prompt) > 25000 {
		log.Fatalf("Prompt is too long (%d characters), reduce number of reviews", len(prompt))
	}

	ctx := context.Background()

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		log.Fatalf("Failed to create genai client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-pro")

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error generating content: %v", err)
	}

	var output strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		output.WriteString(fmt.Sprintf("%v", part))
	}

	// Limpiar Markdown como ```json ... ```
	outputStr := output.String()
	outputStr = strings.TrimSpace(outputStr)
	outputStr = strings.TrimPrefix(outputStr, "```json")
	outputStr = strings.TrimPrefix(outputStr, "```")
	outputStr = strings.TrimSuffix(outputStr, "```")
	outputStr = strings.TrimSpace(outputStr)

	// Validar que sea JSON válido
	var jsonCheck interface{}
	if err := json.Unmarshal([]byte(outputStr), &jsonCheck); err != nil {
		log.Fatalf("Invalid JSON: %v\nContent:\n%s", err, outputStr)
	}

	// Guardar en archivo
	if err := os.WriteFile(outputFilePath, []byte(outputStr), 0644); err != nil {
		log.Fatalf("Error writing output file: %v", err)
	}

	log.Printf("✅ Analyzed reviews saved: %s\n", outputFilePath)
}
