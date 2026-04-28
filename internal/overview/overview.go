package overview

import (
	"fmt"
	"math"

	"github.com/orbulant/alpha-beta-sigma/internal/csvio"
)

type ClosingPriceDifference struct {
	PreviousDate         string
	CurrentDate          string
	Difference           float64
	PercentageDifference float64
	RelativeDifference   float64
}

type ClosingPriceDifferences []ClosingPriceDifference

type Overview struct {
	LargestPriceDifference     float64
	SmallestPriceDifference    float64
	AveragePriceDifference     float64
	LargestRelativeDifference  float64
	SmallestRelativeDifference float64
	AverageRelativeDifference  float64
}

func parsePrice(priceStr string) (float64, error) {
	var price float64

	_, err := fmt.Sscanf(priceStr, "%f", &price)

	if err != nil {
		return 0, fmt.Errorf("failed to parse price '%s': %w", priceStr, err)
	}

	return price, nil
}

func GeneratePriceDifferences(csv [][]string) ClosingPriceDifferences {
	var previousRecord []string
	var closingPriceDifferences ClosingPriceDifferences

	for index, record := range csv {
		if index == 0 {
			previousRecord = record
			continue
		}

		previousClosingPrice := previousRecord[4]
		currentClosingPrice := record[4]

		previousClosingPriceFloat, _ := parsePrice(previousClosingPrice)
		currentClosingPriceFloat, _ := parsePrice(currentClosingPrice)

		closingPriceDifference := currentClosingPriceFloat - previousClosingPriceFloat
		percentageDifference := (closingPriceDifference / previousClosingPriceFloat) * 100
		relativeDifference := closingPriceDifference / previousClosingPriceFloat

		closingPriceDifferences = append(closingPriceDifferences, ClosingPriceDifference{
			PreviousDate:         previousRecord[0],
			CurrentDate:          record[0],
			Difference:           closingPriceDifference,
			PercentageDifference: percentageDifference,
			RelativeDifference:   relativeDifference,
		})

		previousRecord = record
	}
	return closingPriceDifferences
}

func Generate(file string, skipHeader bool, delimiter rune, comment rune) Overview {
	// For each line in the CSV file, fmt.Printf("Processing line: %s\n", line)
	csvFile := csvio.NewCSVReader(skipHeader, delimiter, comment)
	csv, err := csvFile.Read(file)

	if err != nil {
		fmt.Printf("Error reading CSV file: %v\n", err)
		return Overview{}
	}

	closingPriceDifferences := GeneratePriceDifferences(csv)
	var largestPriceDifference float64
	var smallestPriceDifference float64
	var averagePriceDifference float64
	var averageRelativeDifference float64

	for _, diff := range closingPriceDifferences {
		difference := diff.Difference
		averageRelativeDiffernece := diff.RelativeDifference

		// Convert difference to absolute value for comparison
		if difference < 0 {
			difference = -difference
		}

		if difference > largestPriceDifference {
			largestPriceDifference = difference
		}

		if difference < smallestPriceDifference {
			smallestPriceDifference = difference
		}

		averagePriceDifference += math.Abs(difference)
		averageRelativeDiffernece += math.Abs(averageRelativeDifference)
	}

	averagePriceDifference /= float64(len(closingPriceDifferences))
	averageRelativeDifference /= float64(len(closingPriceDifferences))

	return Overview{
		LargestPriceDifference:  largestPriceDifference,
		SmallestPriceDifference: smallestPriceDifference,
		AveragePriceDifference:  averagePriceDifference,
	}
}
