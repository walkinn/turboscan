package utils

import (
    "encoding/csv"
    "encoding/json"
    "fmt"
    "os"
    "text/tabwriter"
    "time"

    "turboscan/scanner"
)

func PrintResults(results []scanner.Result) {
    if len(results) == 0 {
        fmt.Println("[*] No results found")
        return
    }

    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, "STATUS\tSIZE\tTIME(ms)\tURL")

    for _, r := range results {
        ms := r.Time / time.Millisecond
        fmt.Fprintf(w, "%d\t%d\t%d\t%s\n",
            r.StatusCode,
            r.Size,
            ms,
            r.URL,
        )
    }

    w.Flush()
}

func SaveResults(results []scanner.Result, filename string) error {
    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    writer := csv.NewWriter(f)
    defer writer.Flush()

    writer.Write([]string{"status", "size", "time_ms", "url"})

    for _, r := range results {
        writer.Write([]string{
            fmt.Sprintf("%d", r.StatusCode),
            fmt.Sprintf("%d", r.Size),
            fmt.Sprintf("%d", r.Time/time.Millisecond),
            r.URL,
        })
    }

    return writer.Error()
}

func SaveJSON(results []scanner.Result, filename string) error {
    data, err := json.MarshalIndent(results, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0o644)
}
