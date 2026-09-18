package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir, _ := os.MkdirTemp("", "pdfdbg");
	_ = dir
	p := filepath.Join(os.TempDir(), "dbg.pdf")
	_ = p
	fmt.Println("stub")
}
