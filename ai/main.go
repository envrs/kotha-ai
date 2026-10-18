package main

import (
	"github.com/kothagpt/kotha/ai/cmd"
	"github.com/kothagpt/kotha/ai/internal/logging"
)

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	cmd.Execute()
}
