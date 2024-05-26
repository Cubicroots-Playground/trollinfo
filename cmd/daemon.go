package main

import (
	"github.com/CubicrootXYZ/gologger"
	"github.com/Cubicroots-Playground/trollinfo/internal/angelapi"
	"github.com/Cubicroots-Playground/trollinfo/internal/matrixmessenger"
	"github.com/Cubicroots-Playground/trollinfo/internal/shiftnotifier"
)

func main() {
	angelConfig := angelapi.Config{}
	angelConfig.ParseFromEnvironment()

	matrixConfig := matrixmessenger.Config{}
	matrixConfig.ParseFromEnvironment()

	notifierConfig := shiftnotifier.Config{}
	notifierConfig.ParseFromEnvironment()

	angelService := angelapi.New(&angelConfig)
	messenger, err := matrixmessenger.NewMessenger(
		&matrixConfig, gologger.New(gologger.LogLevelInfo, 0),
	)
	if err != nil {
		panic(err)
	}

	shiftNotifier := shiftnotifier.New(&notifierConfig, angelService, messenger)

	err = shiftNotifier.Start()
	if err != nil {
		panic(err)
	}
}
