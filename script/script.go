package main

import (
	"context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "sync"

    "github.com/joho/godotenv" // For loading .env file
    "google.golang.org/api/option"

    // The correct import for the Generative Language API *client*
    genai "cloud.google.com/go/generativelanguage/apiv1beta"
    // The correct import for the Generative Language API *protobuf types*
    genai_pb "cloud.google.com/genproto/googleapis/cloud/generativelanguage/v1beta/generativelanguagepb"
)

// Review represents the structure of a single review in your JSON file.
type Review struct {
	ReviewerID     string  `json:"reviewerID"`
	ASIN           string  `json:"asin"`
	ReviewerName   string  `json:"reviewerName"`
	ReviewText     string  `json:"reviewText"`
	Overall        float64 `json:"overall"`
	UnixReviewTime int64   `json:"unixReviewTime"` // Using int64 for Unix timestamp
	ReviewTime     string  `json:"reviewTime"`
}

// AnalysisResult represents the structure of the JSON output from the AI model.
type AnalysisResult struct {
	Sentiment  string   `json:"sentiment"`
	Weaknesses []string `json:"weaknesses"`
	Theme      string   `json:"theme"`
}

// ReviewWithAnalysis combines the original review with its analysis result.
type ReviewWithAnalysis struct {
	Review
	AnalysisResult *AnalysisResult // Pointer to allow nil if analysis fails
	Error          string          `json:"error,omitempty"` // To store any error during analysis
}

const (
	reviewsFilePath       = "mapped.json"    // Path to your input JSON file
	batchSize             = 5                // Number of reviews to process in each batch
	maxParallelGoRoutines = 10               // Max concurrent API requests
	geminiModelID         = "gemini-2.5-pro" // Or "gemini-1.0-pro-001"
	// Adjust based on your region; 'us-central1' is common.
	apiEndpoint = "us-central1-aiplatform.googleapis.com:443"
)

// generatePrompt creates the specific prompt string for the AI model.
func generatePrompt(reviewText string) string {
	return fmt.Sprintf(`You are an expert e-commerce product review analyst. Your task is to analyze a given product review and return a JSON object strictly adhering to the specified structure and rules.

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

Ensure the output is **strictly a valid JSON object** and nothing else. Do not include any explanatory text or conversational elements outside the JSON.

**Review to analyze:**
%s`, reviewText) // %s will be replaced by reviewText
}

// analyzeReview calls the Gemini API for a single review.
func analyzeReview(ctx context.Context, client *genai.PredictionClient, projectID, locationID, reviewText string) (*AnalysisResult, error) {
	prompt := generatePrompt(reviewText)

	// Construct the request for the Gemini model
	// The Gemini API typically expects the input within a 'content' field,
	// which then contains 'parts' (e.g., text, images).
	// We'll create the instance as a structpb.Struct.
	instance, err := structpb.NewStruct(map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []interface{}{
				map[string]interface{}{
					"text": prompt,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance struct: %w", err)
	}

	req := &genaipb.PredictRequest{
		Endpoint: fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, locationID, geminiModelID),
		Instances: []*structpb.Value{ // Explicitly use []*structpb.Value here
			structpb.NewStructValue(instance),
		},
	}

	resp, err := client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Predict API: %w", err)
	}

	// ... rest of your response handling code ...
	if len(resp.Predictions) == 0 || resp.Predictions[0].GetStructValue() == nil {
		return nil, fmt.Errorf("no predictions returned or unexpected format")
	}

	// Extract the generated text (the JSON string)
	// The response structure might also be a structpb.StructValue
	generatedContent := resp.Predictions[0].GetStructValue().Fields["content"].GetStringValue()
	if generatedContent == "" {
		// Fallback for models that might return directly as 'text' or other fields
		if textValue, ok := resp.Predictions[0].GetStructValue().Fields["text"]; ok {
			generatedContent = textValue.GetStringValue()
		} else {
			return nil, fmt.Errorf("generated content is empty and no 'text' field found")
		}
	}

	var analysisResult AnalysisResult
	err = json.Unmarshal([]byte(generatedContent), &analysisResult)
	if err != nil {
		log.Printf("Warning: Could not unmarshal generated content into JSON. Raw content: %s, Error: %v\n", generatedContent, err)
		return nil, fmt.Errorf("failed to unmarshal analysis result: %w", err)
	}

	return &analysisResult, nil
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file, ensure it exists and contains GEMINI_API_KEY: %v", err)
	}

	// apiKey := os.Getenv("GEMINI_API_KEY")
	// if apiKey == "" {
	// 	log.Fatal("GEMINI_API_KEY not found in environment variables. Please set it.")
	// }

	// Read reviews from JSON file
	fileContent, err := os.ReadFile(reviewsFilePath)
	if err != nil {
		log.Fatalf("Error reading reviews file %s: %v", reviewsFilePath, err)
	}

	var reviews []Review
	err = json.Unmarshal(fileContent, &reviews)
	if err != nil {
		log.Fatalf("Error unmarshaling reviews JSON: %v", err)
	}

	log.Printf("Successfully loaded %d reviews from %s\n", len(reviews), reviewsFilePath)

	// Initialize Gemini client
	ctx := context.Background()
	client, err := genai.NewPredictionClient(ctx, option.WithEndpoint(apiEndpoint))
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	projectID := os.Getenv("GCP_PROJECT_ID") // Optional: set if needed, usually managed by ADC
	if projectID == "" {
		log.Println("GCP_PROJECT_ID not set. API calls might default to a project configured via gcloud CLI.")
	}
	locationID := "us-central1" // Ensure this matches your API endpoint region

	var allAnalyzedReviews []ReviewWithAnalysis
	var wg sync.WaitGroup // WaitGroup to wait for all goroutines to finish

	// Semaphore to limit concurrent goroutines (API requests)
	sem := make(chan struct{}, maxParallelGoRoutines)

	for i := 0; i < len(reviews); i += batchSize {
		end := i + batchSize
		if end > len(reviews) {
			end = len(reviews)
		}
		batch := reviews[i:end]

		log.Printf("Processing batch %d to %d\n", i, end-1)

		for _, review := range batch {
			wg.Add(1)
			sem <- struct{}{} // Acquire a slot in the semaphore
			go func(r Review) {
				defer wg.Done()
				defer func() { <-sem }() // Release the slot when goroutine finishes

				log.Printf("  Analyzing review: %s (ID: %s)\n", r.ReviewText, r.ReviewerID)
				analysis, err := analyzeReview(ctx, client, projectID, locationID, r.ReviewText)

				analyzedReview := ReviewWithAnalysis{Review: r}
				if err != nil {
					log.Printf("    Error analyzing review %s: %v\n", r.ReviewText, err)
					analyzedReview.Error = err.Error()
				} else {
					analyzedReview.AnalysisResult = analysis
				}

				// Use a mutex if writing to a shared slice directly,
				// but for simplicity, we'll append after the loop finishes
				// or to a channel if you need real-time aggregation.
				// For now, collecting after all are done.
				allAnalyzedReviews = append(allAnalyzedReviews, analyzedReview)

			}(review)
		}
		// Wait for the current batch to complete before potentially starting a new one
		// or just let them run if there are no rate limits issues.
		// For smaller batch sizes and potential rate limits, you might want to uncomment:
		// wg.Wait() // Wait for current batch to finish before proceeding to next batch (if any)
		// time.Sleep(1 * time.Second) // Small delay to prevent hitting rate limits
	}

	wg.Wait() // Wait for all goroutines from all batches to finish

	// Output results
	outputJSON, err := json.MarshalIndent(allAnalyzedReviews, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling final results: %v", err)
	}

	fmt.Println("\n--- Analysis Complete ---")
	fmt.Println(string(outputJSON))

	// Optionally save to a file
	outputFileName := "analyzed_reviews.json"
	err = os.WriteFile(outputFileName, outputJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing output to file %s: %v", outputFileName, err)
	}
	log.Printf("Analysis results saved to %s\n", outputFileName)
}
