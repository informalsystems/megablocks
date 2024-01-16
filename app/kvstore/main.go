package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, CometBFT")

	_ = NewKVStoreApplication(nil)
}
