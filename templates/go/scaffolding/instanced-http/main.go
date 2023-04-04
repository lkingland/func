package main

import (
	"fmt"
	"os"

	"github.com/lkingland/func-runtimes/go/http"

	f "f"
)

func main() {
	if err := http.Start(f.New()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
