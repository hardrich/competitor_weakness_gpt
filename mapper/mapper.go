package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type RawReview struct {
	ReviewerID     string  `json:"id"`
	ASIN           string  `json:"product_asin"`
	ReviewerName   string  `json:"author_title"`
	ReviewText     string  `json:"body"`
	Overall        float64 `json:"rating"`
	UnixReviewTime int64   `json:"review_timestamp"` // Using int64 for Unix timestamp
	ReviewTime     string  `json:"date"`
}

type Review struct {
	ReviewerID     string  `json:"reviewerID"`
	ASIN           string  `json:"asin"`
	ReviewerName   string  `json:"reviewerName"`
	ReviewText     string  `json:"reviewText"`
	Overall        float64 `json:"overall"`
	UnixReviewTime int64   `json:"unixReviewTime"` // Using int64 for Unix timestamp
	ReviewTime     string  `json:"reviewTime"`
}

const (
	reviewsFilePath = "Outscraper-20250716164547xs98.json"
)

func isLowRated(review RawReview) bool {
	return review.Overall < 5
}

func mapper(rawReview RawReview) Review {
	return Review(rawReview)
}

func main() {
	fileContent, err := os.ReadFile(reviewsFilePath)
	if err != nil {
		log.Fatalf("Error reading file %s, %v", reviewsFilePath, err)
	}

	var rawReviews []RawReview
	err = json.Unmarshal(fileContent, &rawReviews)
	if err != nil {
		log.Fatalf("Error unmashaling reviews json: %v", err)
	}

	log.Printf("Successfully loaded %d reviews from %s\n", len(rawReviews), reviewsFilePath)

	var lowRatedReviews []Review
	for _, review := range rawReviews {
		if isLowRated(review) {
			lowRatedReviews = append(lowRatedReviews, mapper(review))
		}
	}

	outputJSON, err := json.MarshalIndent(lowRatedReviews, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling final results: %v", err)
	}

	fmt.Println("\n--- Mapper Complete ---")
	fmt.Println(string(outputJSON))

	outputFileName := "mapped.json"
	err = os.WriteFile(outputFileName, outputJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing output to file %s: %v", outputFileName, err)
	}
	log.Printf("Analysis results saved to %s\n", outputFileName)
}
