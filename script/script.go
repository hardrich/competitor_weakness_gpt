package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	genai "cloud.google.com/go/vertexai/genai"
)

const promptTemplate = `You are an expert e-commerce product review analyst. Your task is to analyze a given product review and return a JSON object strictly adhering to the specified structure and rules.

Here are the strict rules for your analysis and JSON output:

- **sentiment**: A single enumerated value: "Positive", "Negative", or "Neutral". This should reflect the overall tone and customer satisfaction expressed in the review.
- **weaknesses**: A list containing up to 3 (three) lowercase keywords. These keywords must represent the primary issues, flaws, or negative aspects explicitly mentioned in the review. If no clear weaknesses are identified, this list must be empty ([]).
- **theme**: A single, singular keyword representing the main category of the feedback or issue. Choose from the following options. Select the most relevant one. If the review is generally positive with no specific issues, or if the primary theme doesn't fit any of these, use "General".

    * **Shipping**: Problems related to delivery, packaging, delays, or received condition (e.g., damaged box, late arrival).
    * **Material**: Issues concerning the physical composition, build quality, durability, or integrity of the product (e.g., "cheap plastic," "broke easily," "thin fabric").
    * **Functionality**: Problems with how the product operates, performs its intended purpose, or its features (e.g., "doesn't charge," "button sticks," "software glitch," "didn't work").
    * **Performance**: Related to efficiency, speed, effectiveness, or power output (e.g., "slow," "not powerful enough," "battery drains fast").
    * **Price**: Comments on the cost, value for money, or affordability of the product (e.g., "too expensive," "not worth the price," "great value").
    * **Support**: Issues with customer service, warranty, returns, or technical assistance (e.g., "bad customer service," "no reply," "difficult return").
    * **Design**: Feedback on the aesthetics, ergonomics, user-friendliness, or appearance (e.g., "ugly," "uncomfortable," "clunky design").
    * **Experience**: Pertains to the overall user interaction, ease of use, setup process, or unboxing (e.g., "hard to set up," "complicated," "smooth experience").
    * **Compatibility**: Problems with the product working with other devices, systems, or requirements (e.g., "doesn't fit," "not compatible with iOS").
    * **Accuracy**: Issues where the product description, specifications, or advertised features do not match the actual product (e.g., "wrong color," "smaller than described," "misleading image").
    * **Maintenance**: Difficulties with cleaning, upkeep, or long-term care of the product (e.g., "hard to clean," "requires constant maintenance").
    * **Assembly**: Challenges related to putting the product together (e.g., "difficult to assemble," "missing parts").
    * **General**: For overwhelmingly positive reviews without specific issues, or for issues that don't fit well into the other categories.

Ensure the output is strictly a valid JSON object and nothing else. Do not include any explanatory text or conversational elements outside the JSON.

**Review to analyze:**
%s`

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file, ensure it exists and contains GEMINI_API_KEY: %v", err)
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")

	reviewText := "The product stopped working after two days. The packaging was also damaged."
	prompt := fmt.Sprintf(promptTemplate, reviewText)

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

	var responseBuilder strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		responseBuilder.WriteString(fmt.Sprintf("%v", part))
	}

	fmt.Println("Response JSON:")
	fmt.Println(responseBuilder.String())
}
