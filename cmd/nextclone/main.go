package main

import (
	"context"
	"flag"
	"log"

	"github.com/marvinscham/nextclone/internal/gui"
	"github.com/marvinscham/nextclone/internal/scheduler"
)

func main() {
	background := flag.Bool("background", false, "run scheduled backups in the background")
	runDue := flag.Bool("run-due", false, "run due scheduled backups once without opening the app window")
	flag.Parse()

	if *background {
		if err := scheduler.Loop(context.Background()); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		return
	}
	if *runDue {
		if _, err := scheduler.RunDue(context.Background()); err != nil {
			log.Fatal(err)
		}
		return
	}

	gui.Run()
}
