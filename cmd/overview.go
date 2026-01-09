/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// overviewCmd represents the overview command
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Get a brief overview of the equity data",
	Long: `This command provides a summary of key statistics
from the equity day series data, including mean, median,
standard deviation, and other relevant metrics.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("overview called")
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
}
