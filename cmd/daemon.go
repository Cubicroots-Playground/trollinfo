package main

import (
	"fmt"

	"github.com/Cubicroots-Playground/trollinfo/internal/angelapi"
)

func main() {
	angelConfig := angelapi.Config{}
	angelConfig.ParseFromEnvironment()

	angelService := angelapi.New(&angelConfig)

	locs, err := angelService.ListShiftsInLocation(1, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(locs)
}
