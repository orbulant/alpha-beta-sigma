package csvio

import (
	"encoding/csv"
	"fmt"
	"os"
)

func ReadCSVFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		fmt.Println("Unable to open the file.")
		return nil, err
	}

	defer file.Close()

	// Read the file CSV content
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	if err != nil {
		fmt.Println("Unable to read the file.")
		return nil, err
	}

	return records, nil
}
