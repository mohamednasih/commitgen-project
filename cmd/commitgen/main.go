package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/N0ViP/commitgen-project/internal/commitgen"
)

func main() {
	var interactive bool
	flag.BoolVar(&interactive, "interactive", false, "review and edit generated commit content before committing")
	flag.BoolVar(&interactive, "i", false, "shorthand for --interactive")
	flag.Parse()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		fmt.Fprintln(os.Stdout, "\n\nCtrl+C detected. Exiting gracefully.")
		os.Exit(0)
	}()

	app := commitgen.New(os.Stdin, os.Stdout, os.Stderr)
	if err := app.Run(context.Background(), interactive); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
