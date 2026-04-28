/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/orbulant/alpha-beta-sigma/internal/overview"
	"github.com/spf13/cobra"
)

// overviewCmd represents the overview command
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show an overview of the equity's performance",
	Long:  `Show an overview of the equity's performance.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the flag values
		file, _ := cmd.Flags().GetString("file")
		skipHeader, _ := cmd.Flags().GetBool("skip-header")
		delimiter, _ := cmd.Flags().GetString("delimiter")
		comment, _ := cmd.Flags().GetString("comment")

		delimiterRune := []rune(delimiter)[0]
		commentRune := []rune(comment)[0]

		overviewData := overview.Generate(file, skipHeader, delimiterRune, commentRune)

		fmt.Print("Overview of Equity Performance:\n")
		fmt.Printf("Largest Price Difference: %+v\n", overviewData.LargestPriceDifference)
		fmt.Printf("Smallest Price Difference: %+v\n", overviewData.SmallestPriceDifference)
		fmt.Printf("Average Price Difference: %.2f\n", overviewData.AveragePriceDifference)
		fmt.Printf("Average Relative Difference: %.4f\n", overviewData.AverageRelativeDifference)
	},
}

func init() {
	rootCmd.AddCommand(overviewCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// overviewCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// overviewCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	overviewCmd.Flags().StringP("file", "f", "", "Stooq CSV file to analyze")
	overviewCmd.MarkFlagRequired("file")

	overviewCmd.Flags().BoolP("skip-header", "s", true, "Skip the header row in the CSV file")

	overviewCmd.Flags().StringP("delimiter", "d", ",", "CSV delimiter character")

	overviewCmd.Flags().StringP("comment", "c", "#", "CSV comment character")
}
