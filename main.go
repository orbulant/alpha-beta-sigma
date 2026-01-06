package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

func main() {
	// Open the CSV file
	file, err := os.Open("csv/^spx_d.csv")

	if err != nil {
		fmt.Println("Unable to open the file.")
		return
	}

	defer file.Close()

	fmt.Println("File opened successfully.")

	// Read the file CSV content
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	if err != nil {
		fmt.Println("Unable to read the file.")
		return
	}

	// Print the CSV content
	for _, record := range records {
		fmt.Println(record)
	}

}
