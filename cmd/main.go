package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Printf("libcognito %v, commit %v, built at %v\n", version, commit, date)
}
