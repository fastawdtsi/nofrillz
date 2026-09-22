package main

// import (
// 	"log"
// 	"os"

// 	"nofrillz/internal/app"
// 	"nofrillz/internal/config"
// )

// func main() {
// 	cfg := config.MustLoad()
// 	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

// 	a, err := app.New(cfg, logger)
// 	if err != nil {
// 		logger.Fatal(err)
// 	}
// 	defer a.Close()

// 	select {}
// }
