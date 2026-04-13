package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("   ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "" {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", r)
				}
			}()
			tokens := tokenise(line)
			result := eval(parse(tokens))
			if result != nil {
				fmt.Println(display(result))
			}
		}()
	}
}
