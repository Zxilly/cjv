package main

import (
	"fmt"
	"io"
	"os"
)

var marker = "unset"

func main() {
	fmt.Println(marker)
	if len(os.Args) > 1 && os.Args[1] == "hold" {
		_, _ = io.Copy(io.Discard, os.Stdin)
	}
}
