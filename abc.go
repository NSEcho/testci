package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("goos", runtime.GOOS)
	fmt.Println("goarch", runtime.GOARCH)
}