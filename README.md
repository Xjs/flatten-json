# flatten-json

Takes a number of JSON files containing an array and flattens the output into a CSV file containing the combined set
of columns of all input files.

Columns will be sorted alphabetically. Files will be processed in input order. Give no argument or "-" for stdin.

Output will always be stdout.

## Usage

```
flatten-json [input files] > output.csv
```

Call `-help` for arguments.

## Skipping keys

Optionally pass a `-skip` flag to skip a key in files containing a JSON object as root elements, using only the named
element as base array.

