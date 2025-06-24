package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage: %s [inputs]\n", os.Args[0])
	fmt.Fprintln(out)
	fmt.Fprintln(out, "inputs must be valid JSON files with a top-level array.")
	fmt.Fprintln(out)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var inputs []func() (io.ReadCloser, error)
	var help bool
	var skip []string
	var skipFlag string
	var commaFlag string = "\t"

	flag.Usage = usage
	flag.BoolVar(&help, "help", help, "show help")
	flag.StringVar(&skipFlag, "skip", skipFlag, "JSON key to skip (i. e. jump into before streaming)")
	flag.StringVar(&commaFlag, "separator", commaFlag, "separator to use for output")
	flag.Parse()
	args := flag.Args()

	if help {
		flag.Usage()
		return
	}

	var comma rune
	if cr := []rune(commaFlag); len(cr) != 1 {
		flag.Usage()
		log.Fatalf("invalid separator: %q", commaFlag)
	} else {
		comma = cr[0]
	}

	if skipFlag != "" {
		skip = append(skip, skipFlag)
	}

	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		inputs = append(inputs, func() (io.ReadCloser, error) { return io.NopCloser(os.Stdin), nil })
	}

	for _, arg := range args {
		inputs = append(inputs, func() (io.ReadCloser, error) {
			f, err := os.Open(arg)
			if err != nil {
				return nil, err
			}
			return f, nil
		})
	}

	records, err := process(inputs, skip)
	if err != nil {
		log.Fatal(err)
	}

	if err := tabular(records, comma, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type record map[string]string

func process(inputs []func() (io.ReadCloser, error), skip []string) ([]record, error) {
	var records []record

	for _, input := range inputs {
		r, err := input()
		if err != nil {
			return nil, err
		}

		recs, err := readAndClose(r, skip)
		if err != nil {
			return nil, err
		}

		records = append(records, recs...)
	}

	return records, nil
}

func readAndClose(r io.ReadCloser, skip []string) ([]record, error) {
	defer r.Close()

	input, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var base any
	if err := json.Unmarshal(input, &base); err != nil {
		return nil, err
	}

	prefix := ""
	for len(skip) > 0 {
		var skipID string
		skipID, skip = skip[0], skip[1:]

		switch b := base.(type) {
		case map[string]any:
			base = b[skipID]
		case []any:
			i, err := strconv.Atoi(skipID)
			if err != nil {
				return nil, err
			}
			base = b[i]
		default:
			return nil, fmt.Errorf("type of base %T isn't skippable by %s", base, skipID)
		}

		if prefix != "" {
			prefix += "."
		}
		prefix += skipID
	}

	baseArray, ok := base.([]any)
	if !ok {
		return nil, fmt.Errorf("type of baseArray %T isn't []any", baseArray)
	}

	records := make([]record, len(baseArray))
	for i, b := range baseArray {
		record := make(map[string]string)
		if err := flatten(prefix, b, record); err != nil {
			return nil, err
		}
		records[i] = record
	}

	return records, nil
}

func flatten(prefix string, jsonObject any, target map[string]string) error {
	switch t := jsonObject.(type) {
	case bool:
		target[prefix] = strconv.FormatBool(t)
	case float64:
		target[prefix] = strconv.FormatFloat(t, 'f', -1, 64)
	case string:
		target[prefix] = t
	case nil:
		target[prefix] = "null"
	case []any:
		for i, v := range t {
			if err := flatten(prefix+"["+strconv.Itoa(i)+"]", v, target); err != nil {
				return err
			}
		}
	case map[string]any:
		for k, v := range t {
			p := prefix
			if p != "" {
				p += "."
			}
			p += k
			if err := flatten(p, v, target); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cannot flatten leaf node: %v", jsonObject)
	}

	return nil
}

func tabular(records []record, comma rune, output io.Writer) error {
	columnSet := make(map[string]struct{})
	for _, record := range records {
		for k := range record {
			columnSet[k] = struct{}{}
		}
	}

	columns := make([]string, 0, len(columnSet))
	for k := range columnSet {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	w := csv.NewWriter(output)
	w.Comma = comma
	defer w.Flush()
	if err := w.Write(columns); err != nil {
		return err
	}

	for _, jsonRecord := range records {
		rec := make([]string, len(columns))
		for i, k := range columns {
			rec[i] = jsonRecord[k]
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}
