package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SearchMetricsCatalog struct {
	Groups      map[string]SearchMetricsGroup   `json:"groups,omitempty"`
	ByName      map[string]SearchMetric         `json:"byName,omitempty"`
	Examples    map[string]SearchMetricsExample `json:"examples,omitempty"`
}

type SearchMetricsGroup struct {
	Label       string                        `json:"label,omitempty"`
	Description string                        `json:"description,omitempty"`
	DataSource  string                        `json:"dataSource,omitempty"`
	DocsURL     string                        `json:"docsUrl,omitempty"`
	Dimensions  []string                      `json:"dimensions,omitempty"`
	Metrics     []string                      `json:"metrics,omitempty"`
	Attributes  []SearchMetricAttribute       `json:"attributes,omitempty"`
	Enums       map[string][]SearchMetricEnum `json:"enums,omitempty"`
}

type SearchMetric struct {
	Name                string            `json:"name,omitempty"`
	Label               string            `json:"label,omitempty"`
	Instrument          string            `json:"instrument,omitempty"`
	Description         string            `json:"description,omitempty"`
	DocsURL             string            `json:"docsUrl,omitempty"`
	DataSource          string            `json:"dataSource,omitempty"`
	Unit                string            `json:"unit,omitempty"`
	Dimensions          []string          `json:"dimensions,omitempty"`
	SupportedRollups    []string          `json:"supportedRollups,omitempty"`
	QueryFieldTemplates map[string]string `json:"queryFieldTemplates,omitempty"`
	DefaultGroups       []string          `json:"defaultGroups,omitempty"`
	Groups              []string          `json:"groups,omitempty"`
	MetricType          string            `json:"metricType,omitempty"`
	ValueType           string            `json:"valueType,omitempty"`
	SupportInfo         []string          `json:"supportInfo,omitempty"`
	AvailableMetrics    map[string]string `json:"availableMetrics,omitempty"`
	Notes               []string          `json:"notes,omitempty"`
}

type SearchMetricsExample struct {
	Description string         `json:"description,omitempty"`
	MetricNames []string       `json:"metricNames,omitempty"`
	Query       map[string]any `json:"query,omitempty"`
}

type SearchMetricAttribute struct {
	Label       string `json:"label,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	EnumName    string `json:"enumName,omitempty"`
}

type SearchMetricEnum struct {
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
}

func LoadSearchMetricsCatalogJSON(data []byte) (SearchMetricsCatalog, error) {
	var catalog SearchMetricsCatalog
	if len(data) == 0 {
		return NormalizeSearchMetricsCatalog(catalog), nil
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return SearchMetricsCatalog{}, fmt.Errorf("parse metrics catalog: %w", err)
	}
	catalog = NormalizeSearchMetricsCatalog(catalog)
	if err := ValidateSearchMetricsCatalog(catalog); err != nil {
		return SearchMetricsCatalog{}, err
	}
	return catalog, nil
}

func NormalizeSearchMetricsCatalog(catalog SearchMetricsCatalog) SearchMetricsCatalog {
	if catalog.Groups == nil {
		catalog.Groups = map[string]SearchMetricsGroup{}
	}
	if catalog.ByName == nil {
		catalog.ByName = map[string]SearchMetric{}
	}
	if catalog.Examples == nil {
		catalog.Examples = map[string]SearchMetricsExample{}
	}
	for key, group := range catalog.Groups {
		group.Label = strings.TrimSpace(group.Label)
		group.Description = strings.TrimSpace(group.Description)
		group.DataSource = strings.TrimSpace(group.DataSource)
		group.DocsURL = strings.TrimSpace(group.DocsURL)
		group.Dimensions = uniqueSortedStrings(group.Dimensions)
		group.Metrics = uniqueSortedStrings(group.Metrics)
		if group.Enums == nil {
			group.Enums = map[string][]SearchMetricEnum{}
		}
		for idx := range group.Attributes {
			group.Attributes[idx].Label = strings.TrimSpace(group.Attributes[idx].Label)
			group.Attributes[idx].Name = strings.TrimSpace(group.Attributes[idx].Name)
			group.Attributes[idx].Description = strings.TrimSpace(group.Attributes[idx].Description)
			group.Attributes[idx].EnumName = strings.TrimSpace(group.Attributes[idx].EnumName)
		}
		for enumName, values := range group.Enums {
			delete(group.Enums, enumName)
			enumName = strings.TrimSpace(enumName)
			normalized := make([]SearchMetricEnum, 0, len(values))
			for _, value := range values {
				normalized = append(normalized, SearchMetricEnum{
					Value:       strings.TrimSpace(value.Value),
					Description: strings.TrimSpace(value.Description),
				})
			}
			group.Enums[enumName] = normalized
		}
		catalog.Groups[key] = group
	}
	for key, metric := range catalog.ByName {
		metric.Name = strings.TrimSpace(metric.Name)
		metric.Label = strings.TrimSpace(metric.Label)
		metric.Instrument = strings.TrimSpace(metric.Instrument)
		metric.Description = strings.TrimSpace(metric.Description)
		metric.DocsURL = strings.TrimSpace(metric.DocsURL)
		metric.DataSource = strings.TrimSpace(metric.DataSource)
		metric.Unit = strings.TrimSpace(metric.Unit)
		metric.Dimensions = uniqueSortedStrings(metric.Dimensions)
		metric.SupportedRollups = uniqueSortedStrings(metric.SupportedRollups)
		metric.DefaultGroups = uniqueSortedStrings(metric.DefaultGroups)
		metric.Groups = uniqueSortedStrings(metric.Groups)
		metric.MetricType = strings.TrimSpace(metric.MetricType)
		metric.ValueType = strings.TrimSpace(metric.ValueType)
		metric.SupportInfo = uniqueSortedStrings(metric.SupportInfo)
		metric.Notes = uniqueSortedStrings(metric.Notes)
		if metric.QueryFieldTemplates == nil {
			metric.QueryFieldTemplates = map[string]string{}
		}
		for rollup, field := range metric.QueryFieldTemplates {
			delete(metric.QueryFieldTemplates, rollup)
			metric.QueryFieldTemplates[strings.TrimSpace(rollup)] = strings.TrimSpace(field)
		}
		if metric.AvailableMetrics == nil {
			metric.AvailableMetrics = map[string]string{}
		}
		for value, field := range metric.AvailableMetrics {
			delete(metric.AvailableMetrics, value)
			metric.AvailableMetrics[strings.TrimSpace(value)] = strings.TrimSpace(field)
		}
		if group, ok := catalog.Groups[metric.Instrument]; ok {
			if metric.DataSource == "" {
				metric.DataSource = group.DataSource
			}
			metric.Dimensions = uniqueSortedStrings(append(metric.Dimensions, group.Dimensions...))
		}
		catalog.ByName[key] = metric
	}
	for key, example := range catalog.Examples {
		example.Description = strings.TrimSpace(example.Description)
		example.MetricNames = uniqueSortedStrings(example.MetricNames)
		if example.Query == nil {
			example.Query = map[string]any{}
		}
		catalog.Examples[key] = example
	}
	return catalog
}

func ValidateSearchMetricsCatalog(catalog SearchMetricsCatalog) error {
	catalog = NormalizeSearchMetricsCatalog(catalog)

	for groupKey, group := range catalog.Groups {
		groupKey = strings.TrimSpace(groupKey)
		if groupKey == "" {
			return fmt.Errorf("metrics catalog validation failed: group key must be non-empty")
		}
		if group.Label == "" {
			return fmt.Errorf("metrics catalog validation failed: group %q label must be non-empty", groupKey)
		}
		for _, metricName := range group.Metrics {
			metric, ok := catalog.ByName[metricName]
			if !ok {
				return fmt.Errorf("metrics catalog validation failed: group %q references unknown metric %q", groupKey, metricName)
			}
			if metric.Instrument != groupKey && !containsString(metric.Groups, groupKey) {
				return fmt.Errorf("metrics catalog validation failed: group %q includes metric %q with instrument %q", groupKey, metricName, metric.Instrument)
			}
		}
	}

	for metricKey, metric := range catalog.ByName {
		metricKey = strings.TrimSpace(metricKey)
		if metricKey == "" {
			return fmt.Errorf("metrics catalog validation failed: metric key must be non-empty")
		}
		if metric.Name == "" {
			return fmt.Errorf("metrics catalog validation failed: metric %q name must be non-empty", metricKey)
		}
		if metric.Name != metricKey {
			return fmt.Errorf("metrics catalog validation failed: metric key %q must match metric name %q", metricKey, metric.Name)
		}
		if metric.Instrument == "" {
			return fmt.Errorf("metrics catalog validation failed: metric %q instrument must be non-empty", metricKey)
		}
		group, ok := catalog.Groups[metric.Instrument]
		if !ok && len(metric.Groups) == 0 {
			return fmt.Errorf("metrics catalog validation failed: metric %q references unknown group %q", metricKey, metric.Instrument)
		}
		if ok && group.DataSource != "" && metric.DataSource != "" && group.DataSource != metric.DataSource {
			return fmt.Errorf("metrics catalog validation failed: metric %q data source %q does not match group %q data source %q", metricKey, metric.DataSource, metric.Instrument, group.DataSource)
		}
		for _, groupKey := range metric.Groups {
			group, ok := catalog.Groups[groupKey]
			if !ok {
				return fmt.Errorf("metrics catalog validation failed: metric %q references unknown group %q", metricKey, groupKey)
			}
			if group.DataSource != "" && metric.DataSource != "" && group.DataSource != metric.DataSource {
				return fmt.Errorf("metrics catalog validation failed: metric %q data source %q does not match group %q data source %q", metricKey, metric.DataSource, groupKey, group.DataSource)
			}
		}
	}

	for exampleKey, example := range catalog.Examples {
		if strings.TrimSpace(exampleKey) == "" {
			return fmt.Errorf("metrics catalog validation failed: example key must be non-empty")
		}
		for _, metricName := range example.MetricNames {
			if _, ok := catalog.ByName[metricName]; !ok {
				return fmt.Errorf("metrics catalog validation failed: example %q references unknown metric %q", exampleKey, metricName)
			}
		}
	}

	return nil
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
