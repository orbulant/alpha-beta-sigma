/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/orbulant/alpha-beta-sigma/internal/csvio"
	"github.com/spf13/cobra"
)

type ClosingPriceDifference struct {
	PreviousDate         string
	CurrentDate          string
	Difference           float64
	PercentageDifference float64
}

// overviewCmd represents the overview command
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Get a brief overview of the equity data",
	Long: `This command provides a summary of key statistics
from the equity day series data, including mean, median,
standard deviation, and other relevant metrics.`,
	Run: func(cmd *cobra.Command, args []string) {
		filePath, _ := cmd.Flags().GetString("file")
		skipHeader, _ := cmd.Flags().GetBool("skip-header")
		delimiterStr, _ := cmd.Flags().GetString("delimiter")
		commentStr, _ := cmd.Flags().GetString("comment")

		delimiter := rune(delimiterStr[0])
		comment := rune(commentStr[0])

		reader := csvio.NewCSVReader(skipHeader, delimiter, comment)

		records, err := reader.Read(filePath)

		if err != nil {
			panic(err)
		}

		var currentClosingPrice float64
		var previousClosingPrice float64
		var closingPriceDifferences []ClosingPriceDifference
		var previousDate string

		for _, record := range records {
			date := record[0]
			closingPriceStr := record[4]

			closingPrice, err := strconv.ParseFloat(closingPriceStr, 64)
			if err != nil {
				panic(err)
			}

			previousClosingPrice = currentClosingPrice
			currentClosingPrice = closingPrice

			if previousClosingPrice != 0 {
				difference := currentClosingPrice - previousClosingPrice
				percentageDifference := (difference / previousClosingPrice) * 100
				closingPriceDifferences = append(closingPriceDifferences, ClosingPriceDifference{
					PreviousDate:         previousDate,
					CurrentDate:          date,
					Difference:           difference,
					PercentageDifference: percentageDifference,
				})
			}

			previousDate = date
		}

		// Calculate statistics for ALL differences before filtering
		var totalPositivePercentageDifference float64
		var countPositive int
		var totalNegativePercentageDifference float64
		var countNegative int

		for _, diff := range closingPriceDifferences {
			if diff.Difference > 0 {
				totalPositivePercentageDifference += diff.PercentageDifference
				countPositive++
			} else if diff.Difference < 0 {
				totalNegativePercentageDifference += diff.PercentageDifference
				countNegative++
			}
		}

		averagePositivePercentageDifference := totalPositivePercentageDifference / float64(countPositive)
		averageNegativePercentageDifference := totalNegativePercentageDifference / float64(countNegative)

		// Calculate additional metrics for ALL data
		winRate := (float64(countPositive) / float64(len(closingPriceDifferences))) * 100
		asymmetryRatio := averagePositivePercentageDifference / math.Abs(averageNegativePercentageDifference)

		// Sort the differences by absolute value in descending order
		sort.Slice(closingPriceDifferences, func(i, j int) bool {
			return math.Abs(closingPriceDifferences[i].Difference) > math.Abs(closingPriceDifferences[j].Difference)
		})

		// Keep top 25 for display only
		top25 := closingPriceDifferences
		if len(closingPriceDifferences) > 25 {
			top25 = closingPriceDifferences[:25]
		}

		// Calculate the average difference and percentage difference for top 25
		var totalDifference float64
		var totalPercentageDifference float64
		for _, diff := range top25 {
			totalDifference += diff.Difference
			totalPercentageDifference += diff.PercentageDifference
		}
		averageDifference := totalDifference / float64(len(top25))
		averagePercentageDifference := totalPercentageDifference / float64(len(top25))

		// Print the overview report
		fmt.Println("================================================")
		fmt.Println("                    Overview                    ")
		fmt.Println("================================================")
		fmt.Println("Top 25 Largest Day-to-Day Closing Price Differences:")
		fmt.Println("Previous Date | Current Date | Absolute Difference | Percentage Difference")
		for _, diff := range top25 {
			fmt.Printf("From %s to %s | %.2f | %.2f%%\n", diff.PreviousDate, diff.CurrentDate, diff.Difference, diff.PercentageDifference)
		}
		fmt.Println("------------------------------------------------")
		fmt.Printf("Average Difference (Top 25): %.2f\n", averageDifference)
		fmt.Printf("Average Percentage Difference (Top 25): %.2f%%\n", averagePercentageDifference)
		fmt.Println("------------------------------------------------")
		fmt.Printf("All-Time Average Positive Percentage Difference: %.2f%%\n", averagePositivePercentageDifference)
		fmt.Printf("All-Time Average Negative Percentage Difference: %.2f%%\n", averageNegativePercentageDifference)
		fmt.Printf("Count Breakdown: %d Positive | %d Negative (out of %d total days)\n", countPositive, countNegative, len(closingPriceDifferences))
		fmt.Printf("Win Rate: %.2f%%\n", winRate)
		fmt.Printf("Asymmetry Ratio: %.2f (Positive%%/|Negative%%|)\n", asymmetryRatio)

		fmt.Println("================================================")
		fmt.Printf("Records: %d\n", len(records))
	},
}

func init() {
	rootCmd.AddCommand(overviewCmd)

	overviewCmd.Flags().StringP("file", "f", "", "Stooq CSV file to analyze")
	overviewCmd.MarkFlagRequired("file")

	overviewCmd.Flags().BoolP("skip-header", "s", true, "Skip the header row in the CSV file")
	overviewCmd.Flags().StringP("delimiter", "d", ",", "CSV delimiter character")
	overviewCmd.Flags().StringP("comment", "c", "#", "CSV comment character")
}
