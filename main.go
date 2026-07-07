package main

import "ubustt-proxy/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
