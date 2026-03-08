package findings

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Section9Labs/Cartero/internal/store"
)

func Load(path, source, tool string) ([]store.Finding, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read findings file: %w", err)
	}

	if strings.TrimSpace(source) == "" {
		source = strings.TrimSpace(filepath.Base(path))
	}
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		return loadCSV(payload, source, tool)
	}

	if findings, err := loadJSON(payload, source, tool); err == nil && len(findings) > 0 {
		return findings, nil
	}
	if findings, err := loadJSONLines(payload, source, tool); err == nil && len(findings) > 0 {
		return findings, nil
	}

	return nil, errors.New("unsupported findings format; provide CSV, JSON, SARIF, or JSONL")
}

func loadCSV(payload []byte, source, tool string) ([]store.Finding, error) {
	reader := csv.NewReader(bytes.NewReader(payload))
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV findings: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("findings CSV requires a header row and at least one record")
	}

	headers := normalizeHeaders(rows[0])
	findings := make([]store.Finding, 0, len(rows)-1)
	for _, row := range rows[1:] {
		record := make(map[string]string, len(headers))
		for i, header := range headers {
			if i < len(row) {
				record[header] = strings.TrimSpace(row[i])
			}
		}

		finding := store.Finding{
			Source:   nonEmpty(record["source"], source),
			Tool:     nonEmpty(record["tool"], tool),
			Rule:     nonEmpty(record["rule"], record["id"], record["name"]),
			Severity: normalizeSeverity(record["severity"]),
			Target:   nonEmpty(record["target"], record["host"], record["url"]),
			Summary:  nonEmpty(record["summary"], record["message"], record["description"]),
			Metadata: map[string]string{},
		}
		for key, value := range record {
			if isReservedCSVKey(key) || value == "" {
				continue
			}
			finding.Metadata[key] = value
		}
		if finding.Tool == "" {
			finding.Tool = "csv-import"
		}
		findings = append(findings, finding)
	}

	return compactFindings(findings), nil
}

func loadJSON(payload []byte, source, tool string) ([]store.Finding, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil, errors.New("findings payload is empty")
	}

	if trimmed[0] == '[' {
		var objects []map[string]any
		if err := json.Unmarshal(trimmed, &objects); err != nil {
			return nil, err
		}
		findings := make([]store.Finding, 0, len(objects))
		for _, object := range objects {
			findings = append(findings, mapObject(source, tool, object))
		}
		return compactFindings(findings), nil
	}

	if sarifFindings, err := loadSARIF(trimmed, source, tool); err == nil && len(sarifFindings) > 0 {
		return compactFindings(sarifFindings), nil
	}

	var object map[string]any
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return nil, err
	}

	return compactFindings([]store.Finding{mapObject(source, tool, object)}), nil
}

func loadJSONLines(payload []byte, source, tool string) ([]store.Finding, error) {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	findings := []store.Finding{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var object map[string]any
		if err := json.Unmarshal([]byte(line), &object); err != nil {
			return nil, err
		}
		findings = append(findings, mapObject(source, tool, object))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan JSONL findings: %w", err)
	}

	return compactFindings(findings), nil
}

func loadSARIF(payload []byte, source, tool string) ([]store.Finding, error) {
	var report struct {
		Runs []struct {
			Tool struct {
				Driver struct {
					Name string `json:"name"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID  string `json:"ruleId"`
				Level   string `json:"level"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}

	if err := json.Unmarshal(payload, &report); err != nil {
		return nil, err
	}
	if len(report.Runs) == 0 {
		return nil, errors.New("not a SARIF payload")
	}

	findings := []store.Finding{}
	for _, run := range report.Runs {
		runTool := nonEmpty(tool, run.Tool.Driver.Name, "sarif")
		for _, result := range run.Results {
			target := ""
			if len(result.Locations) > 0 {
				target = result.Locations[0].PhysicalLocation.ArtifactLocation.URI
			}
			findings = append(findings, store.Finding{
				Source:   source,
				Tool:     runTool,
				Rule:     nonEmpty(result.RuleID, result.Message.Text),
				Severity: normalizeSeverity(result.Level),
				Target:   target,
				Summary:  nonEmpty(result.Message.Text, result.RuleID),
			})
		}
	}

	return findings, nil
}

func mapObject(source, tool string, object map[string]any) store.Finding {
	if isNucleiObject(object) {
		return mapNucleiObject(source, tool, object)
	}

	finding := store.Finding{
		Source:   nonEmpty(asString(object["source"]), source),
		Tool:     nonEmpty(tool, asString(object["tool"]), "json-import"),
		Rule:     nonEmpty(asString(object["rule"]), asString(object["id"]), asString(object["name"])),
		Severity: normalizeSeverity(nonEmpty(asString(object["severity"]), asString(object["level"]))),
		Target:   nonEmpty(asString(object["target"]), asString(object["host"]), asString(object["url"]), asString(object["matched-at"])),
		Summary:  nonEmpty(asString(object["summary"]), asString(object["message"]), asString(object["description"])),
		Metadata: flattenMap("", object, map[string]struct{}{
			"source":      {},
			"tool":        {},
			"rule":        {},
			"id":          {},
			"name":        {},
			"severity":    {},
			"level":       {},
			"target":      {},
			"host":        {},
			"url":         {},
			"matched-at":  {},
			"summary":     {},
			"message":     {},
			"description": {},
		}),
	}
	if finding.Rule == "" {
		finding.Rule = nonEmpty(asString(object["template-id"]), "unnamed-finding")
	}
	if finding.Target == "" {
		finding.Target = "unknown-target"
	}
	if finding.Severity == "" {
		finding.Severity = "info"
	}
	if finding.Summary == "" {
		finding.Summary = finding.Rule
	}

	return finding
}

func isNucleiObject(object map[string]any) bool {
	if _, ok := object["template-id"]; ok {
		return true
	}
	_, infoOK := object["info"].(map[string]any)
	_, hostOK := object["host"]
	return infoOK && hostOK
}

func mapNucleiObject(source, tool string, object map[string]any) store.Finding {
	info, _ := object["info"].(map[string]any)
	metadata := flattenMap("", object, map[string]struct{}{
		"info":        {},
		"host":        {},
		"matched-at":  {},
		"template-id": {},
	})
	for key, value := range flattenMap("info", info, map[string]struct{}{}) {
		metadata[key] = value
	}

	finding := store.Finding{
		Source:   source,
		Tool:     nonEmpty(tool, "nuclei"),
		Rule:     nonEmpty(asString(object["template-id"]), asString(object["matcher-name"]), asString(info["name"])),
		Severity: normalizeSeverity(nonEmpty(asString(info["severity"]), asString(object["severity"]))),
		Target:   nonEmpty(asString(object["host"]), asString(object["matched-at"]), asString(object["ip"])),
		Summary:  nonEmpty(asString(info["name"]), asString(info["description"]), asString(object["matcher-name"])),
		Metadata: metadata,
	}
	if finding.Severity == "" {
		finding.Severity = "info"
	}
	if finding.Summary == "" {
		finding.Summary = finding.Rule
	}

	return finding
}

func compactFindings(findings []store.Finding) []store.Finding {
	compact := make([]store.Finding, 0, len(findings))
	for _, finding := range findings {
		if strings.TrimSpace(finding.Tool) == "" || strings.TrimSpace(finding.Rule) == "" || strings.TrimSpace(finding.Target) == "" {
			continue
		}
		finding.Source = strings.TrimSpace(finding.Source)
		if finding.Source == "" {
			finding.Source = "external-import"
		}
		finding.Rule = strings.TrimSpace(finding.Rule)
		finding.Target = strings.TrimSpace(finding.Target)
		finding.Summary = strings.TrimSpace(finding.Summary)
		finding.Severity = normalizeSeverity(finding.Severity)
		if finding.Severity == "" {
			finding.Severity = "info"
		}
		if finding.Summary == "" {
			finding.Summary = finding.Rule
		}
		if finding.Metadata == nil {
			finding.Metadata = map[string]string{}
		}
		compact = append(compact, finding)
	}

	return compact
}

func normalizeHeaders(headers []string) []string {
	normalized := make([]string, len(headers))
	for i, header := range headers {
		header = strings.TrimSpace(strings.ToLower(header))
		header = strings.ReplaceAll(header, " ", "_")
		normalized[i] = header
	}
	return normalized
}

func flattenMap(prefix string, value map[string]any, skip map[string]struct{}) map[string]string {
	flattened := map[string]string{}
	for key, raw := range value {
		if _, blocked := skip[key]; blocked {
			continue
		}
		flatKey := key
		if prefix != "" {
			flatKey = prefix + "." + key
		}

		switch typed := raw.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				flattened[flatKey] = typed
			}
		case bool:
			flattened[flatKey] = strconv.FormatBool(typed)
		case float64:
			flattened[flatKey] = strconv.FormatFloat(typed, 'f', -1, 64)
		case map[string]any:
			for nestedKey, nestedValue := range flattenMap(flatKey, typed, map[string]struct{}{}) {
				flattened[nestedKey] = nestedValue
			}
		}
	}

	return flattened
}

func normalizeSeverity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "error":
		return "high"
	case "warning":
		return "medium"
	case "note":
		return "low"
	default:
		return value
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

func isReservedCSVKey(key string) bool {
	switch key {
	case "source", "tool", "rule", "id", "name", "severity", "level", "target", "host", "url", "summary", "message", "description":
		return true
	default:
		return false
	}
}
