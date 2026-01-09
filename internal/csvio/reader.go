package csvio

import (
	"encoding/csv"
	"fmt"
	"os"
)

type CSVReader interface {
	Read(filePath string) ([][]string, error)
}

type FileCSVReader struct {
	SkipHeader bool
	Delimiter  rune
	Comment    rune
}

func NewCSVReader(skipHeader bool, delimiter rune, comment rune) *FileCSVReader {
	return &FileCSVReader{
		SkipHeader: skipHeader,
		Delimiter:  delimiter,
		Comment:    comment,
	}
}

func (f *FileCSVReader) Read(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = f.Delimiter
	reader.Comment = f.Comment

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if f.SkipHeader && len(records) > 0 {
		records = records[1:]
	}

	return records, nil
}
