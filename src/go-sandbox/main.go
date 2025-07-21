package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	var hi = "Hello, World"

	for i, v := range hi {
		if v == ' ' {
			continue
		}

		fmt.Printf("%d. %c\n", i, v)
	}
}
