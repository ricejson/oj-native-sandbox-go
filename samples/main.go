package main

import (
	"fmt"
	"os"
)

func main() {
	a := os.Args[1]
	b := os.Args[2]
	fmt.Println(a + b)
}
