package main

import (
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultSources = "https://standards-oui.ieee.org/oui/oui.csv,https://standards-oui.ieee.org/oui28/mam.csv,https://standards-oui.ieee.org/oui36/oui36.csv"

func main() {
	sources := flag.String("sources", defaultSources, "comma-separated IEEE CSV URLs or local files")
	output := flag.String("output", "internal/discovery/data/oui.tsv.gz", "compressed registry output")
	flag.Parse()

	entries := make(map[string]string)
	for _, source := range strings.Split(*sources, ",") {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		input, err := openSource(source)
		if err != nil {
			fatal(err)
		}
		if err := readRegistry(input, entries); err != nil {
			input.Close()
			fatal(fmt.Errorf("%s: %w", source, err))
		}
		input.Close()
	}

	if err := writeRegistry(*output, entries); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d OUI assignments to %s\n", len(entries), *output)
}

func readRegistry(input io.Reader, entries map[string]string) error {
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read CSV header: %w", err)
	}
	assignmentColumn := columnIndex(header, "Assignment")
	organizationColumn := columnIndex(header, "Organization Name")
	if assignmentColumn < 0 || organizationColumn < 0 {
		return fmt.Errorf("CSV is missing Assignment or Organization Name columns")
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read CSV record: %w", err)
		}
		if assignmentColumn >= len(record) || organizationColumn >= len(record) {
			continue
		}
		prefix := normalizePrefix(record[assignmentColumn])
		vendor := strings.TrimSpace(record[organizationColumn])
		if (len(prefix) == 6 || len(prefix) == 7 || len(prefix) == 9) && vendor != "" {
			entries[prefix] = vendor
		}
	}
}

func openSource(source string) (io.ReadCloser, error) {
	if !strings.Contains(source, "://") {
		return os.Open(source)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", source, err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("download %s: %s", source, response.Status)
	}
	return response.Body, nil
}

func columnIndex(header []string, name string) int {
	for index, value := range header {
		if strings.EqualFold(strings.TrimSpace(value), name) {
			return index
		}
	}
	return -1
}

func normalizePrefix(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "", ":", "", ".", "").Replace(value)
	return value
}

func writeRegistry(path string, entries map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", key, entries[key]); err != nil {
			writer.Close()
			return err
		}
	}
	return writer.Close()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
