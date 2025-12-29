package main

import (
    "flag"
    "fmt"
    "os"
    "strconv"
    "strings"

    "turboscan/scanner"
    "turboscan/utils"
)

func main() {
    // CLI flags
    url := flag.String("u", "", "Target URL (e.g. https://target.com)")
    wordlistPath := flag.String("w", "", "Wordlist path")
    threads := flag.Int("t", 50, "Number of threads")
    timeout := flag.Int("timeout", 10, "Request timeout (seconds)")
    statusCodes := flag.String("mc", "200,301,302,401,403", "Match status codes (comma-separated)")
    output := flag.String("o", "", "Output CSV file")
    jsonOutput := flag.String("json", "", "Output JSON file")
    verbose := flag.Bool("v", false, "Verbose mode")

    extensionsFlag := flag.String("e", "", "Extensions (comma-separated, e.g. php,html,txt)")
    recursive := flag.Bool("r", false, "Recursive scanning")
    depth := flag.Int("depth", 3, "Max recursion depth (used with -r)")
    rate := flag.Int("rate", 0, "Rate limit (requests per second, 0 = unlimited)")
    retries := flag.Int("retries", 0, "Max retries on temporary errors (0 = no retries)")

    flag.Parse()

    // Validation
    if *url == "" || *wordlistPath == "" {
        fmt.Println("Usage: turboscan -u <url> -w <wordlist>")
        flag.PrintDefaults()
        os.Exit(1)
    }

    printBanner()

    // Load wordlist
    words, err := utils.LoadWordlist(*wordlistPath)
    if err != nil {
        fmt.Printf("Error loading wordlist: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("[+] Loaded %d words\n", len(words))
    fmt.Printf("[+] Target: %s\n", *url)
    fmt.Printf("[+] Threads: %d\n", *threads)

    // Parsing status codes and extensions
    mc := parseStatusCodes(*statusCodes)
    exts := parseExtensions(*extensionsFlag)

    config := scanner.Config{
        BaseURL:     strings.TrimRight(*url, "/"),
        Threads:     *threads,
        Timeout:     *timeout,
        StatusCodes: mc,
        Verbose:     *verbose,
        Rate:        *rate,
        MaxRetries:  *retries,

        Recursive:  *recursive,
        MaxDepth:   *depth,
        Extensions: exts,

        WordlistPath: *wordlistPath,
    }

    s := scanner.NewScanner(config)

    var results []scanner.Result

    switch {
    case len(exts) > 0 && !*recursive:
        // bruteforce extensions
        results = s.ScanWithExtensions(words, exts)
    case *recursive:
        results = s.ScanRecursive(config.BaseURL, words, 0, config.MaxDepth)
    default:
        results = s.Scan(words)
    }

    utils.PrintResults(results)

    // CSV
    if *output != "" {
        if err := utils.SaveResults(results, *output); err != nil {
            fmt.Printf("Error saving CSV: %v\n", err)
        } else {
            fmt.Printf("[+] CSV results saved to %s\n", *output)
        }
    }

    if *jsonOutput != "" {
        if err := utils.SaveJSON(results, *jsonOutput); err != nil {
            fmt.Printf("Error saving JSON: %v\n", err)
        } else {
            fmt.Printf("[+] JSON results saved to %s\n", *jsonOutput)
        }
    }

    s.PrintStats()
}

func printBanner() {
    fmt.Print(`
████████╗██╗   ██╗██████╗ ██████╗  ██████╗ ███████╗ ██████╗ █████╗ ███╗   ██╗
╚══██╔══╝██║   ██║██╔══██╗██╔══██╗██╔═══██╗██╔════╝██╔════╝██╔══██╗████╗  ██║
   ██║   ██║   ██║██████╔╝██████╔╝██║   ██║███████╗██║     ███████║██╔██╗ ██║
   ██║   ██║   ██║██╔══██╗██╔══██╗██║   ██║╚════██║██║     ██╔══██║██║╚██╗██║
   ██║   ╚██████╔╝██║  ██║██████╔╝╚██████╔╝███████║╚██████╗██║  ██║██║ ╚████║
   ╚═╝    ╚═════╝ ╚═╝  ╚═╝╚═════╝  ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝
                    by walkinn | v1.7 | 3x faster than ffuf
`)
}

func parseStatusCodes(codes string) []int {
    var result []int
    parts := strings.Split(codes, ",")
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        v, err := strconv.Atoi(p)
        if err != nil {
            continue
        }
        result = append(result, v)
    }
    if len(result) == 0 {
        result = []int{200}
    }
    return result
}

func parseExtensions(exts string) []string {
    if exts == "" {
        return nil
    }
    parts := strings.Split(exts, ",")
    res := make([]string, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        p = strings.TrimPrefix(p, ".")
        if p == "" {
            continue
        }
        res = append(res, p)
    }
    if len(res) == 0 {
        return nil
    }
    return res
}
