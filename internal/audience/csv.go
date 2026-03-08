package audience

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/Section9Labs/Cartero/internal/store"
)

func LoadCSV(path string) ([]store.AudienceMember, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open audience csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read audience csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("read audience csv: expected a header and at least one row")
	}

	header := make(map[string]int, len(rows[0]))
	for i, column := range rows[0] {
		header[normalize(column)] = i
	}
	if _, ok := header["email"]; !ok {
		return nil, fmt.Errorf("read audience csv: missing required email column")
	}

	members := make([]store.AudienceMember, 0, len(rows)-1)
	for rowIndex, row := range rows[1:] {
		email := strings.TrimSpace(columnValue(row, header, "email"))
		if email == "" {
			return nil, fmt.Errorf("read audience csv: row %d is missing email", rowIndex+2)
		}

		displayName := strings.TrimSpace(columnValue(row, header, "displayname"))
		if displayName == "" {
			displayName = strings.TrimSpace(columnValue(row, header, "name"))
		}
		if displayName == "" {
			displayName = email
		}

		members = append(members, store.AudienceMember{
			Email:       email,
			DisplayName: displayName,
			Department:  strings.TrimSpace(columnValue(row, header, "department")),
			Title:       strings.TrimSpace(columnValue(row, header, "title")),
		})
	}

	return members, nil
}

func columnValue(row []string, header map[string]int, key string) string {
	index, ok := header[key]
	if !ok || index >= len(row) {
		return ""
	}

	return row[index]
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}
