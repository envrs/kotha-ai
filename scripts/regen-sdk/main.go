package main

import (
	"fmt"
	"os"

	"github.com/kothagpt/kotha/sdk"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "regen-sdk:", err)
		os.Exit(1)
	}
}

func run() error {
	sc, err := sdk.NewSpecCheck(sdk.SpecPath)
	if err != nil {
		return err
	}
	fmt.Printf("spec: %d operations\n", len(sc.Methods()))
	if err := sc.Validate(); err != nil {
		return err
	}
	fmt.Println("regen-sdk: OK — spec matches SDK surface")
	return nil
}
