package cmd

import (
	"flag"
	"fmt"
	"time"

	"github.com/svandragt/park/internal/park"
)

// statsSources are the rows of the "How items came back" table.
var statsSources = []string{"session-hook", "file-hook", "search", "list-tag", "direct"}

func RunStats(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	days := fs.Int("days", 30, "report on usage in the last this many days")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := store.Usage(time.Now().AddDate(0, 0, -*days))
	if err != nil {
		return err
	}

	fmt.Printf("Usage in the last %d days\n\nEvents\n", *days)
	if len(st.Counts) == 0 {
		fmt.Println("  none")
	}
	for _, c := range st.Counts {
		fmt.Printf("  %-10s %-14s %d\n", c.Event, c.Source, c.N)
	}

	fmt.Printf("\nSurfaced items: %d\n", st.Surfaced)
	fmt.Printf("  later viewed:   %d (%s)\n", st.Viewed, share(st.Viewed, st.Surfaced))
	fmt.Printf("  later resolved: %d (%s)\n", st.Resolved, share(st.Resolved, st.Surfaced))

	fmt.Printf("\nHow items came back\n  %-14s %8s %7s %9s\n", "source", "surfaced", "viewed", "resolved")
	for _, src := range statsSources {
		fmt.Printf("  %-14s %8d %7d %9d\n", src,
			usageN(st.Counts, "surfaced", src), usageN(st.Attributed, "viewed", src), usageN(st.Attributed, "resolved", src))
	}
	return nil
}

// share formats n of total as a whole percentage; an empty total is 0%.
func share(n, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(n)/float64(total))
}

func usageN(counts []park.UsageCount, event, source string) int {
	for _, c := range counts {
		if c.Event == event && c.Source == source {
			return c.N
		}
	}
	return 0
}
