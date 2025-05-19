package main

import (
	"fmt"
	"os"

	cowsay "github.com/blackflame007/Neo-cowsay/v2"
)

func main() {
	// Set the cow type to bender
	options := []cowsay.Option{
		cowsay.Type("archer"),
	}

	// Create a message
	message := "Hello, I'm Archer with colors!"

	// Get the cow with message
	output, err := cowsay.Say(message, options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print the output
	fmt.Println(output)
}
