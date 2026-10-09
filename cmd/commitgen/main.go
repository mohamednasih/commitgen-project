package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/N0ViP/commitgen-project/internal/commitgen"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		fmt.Fprintln(os.Stdout, "\n\nCtrl+C detected. Exiting gracefully.")
		commitgen.PrintFooter(os.Stdout)
		os.Exit(0)
	}()

	app := commitgen.New(os.Stdin, os.Stdout, os.Stderr)
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		commitgen.PrintFooter(os.Stderr)
		os.Exit(1)
	}
}
