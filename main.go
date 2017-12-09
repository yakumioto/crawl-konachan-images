package main

import (
	"os"

	"sync"

	"os/signal"

	"log"

	"time"

	"gopkg.in/urfave/cli.v2"
)

const (
	version = "0.1.0"
)

func main() {
	app := &cli.App{
		Name:  "crawl-konachan-images",
		Usage: `crawl http://konachan.com/post images`,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "page",
				Aliases: []string{"pg"},
				Usage:   "start page",
				Value:   1,
			},
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "download path",
				Value:   "./konachan",
			},
			&cli.IntFlag{
				Name:    "num-connections",
				Aliases: []string{"n"},
				Usage:   "specify maximum number of connections",
				Value:   1,
			},
			&cli.BoolFlag{
				Name:  "r18",
				Usage: "r18 flag",
				Value: false,
			},
			&cli.BoolFlag{
				Name:    "latest",
				Usage:   "only download latest",
				Aliases: []string{"l"},
				Value:   false,
			},
		},
		Version: version,
		Action:  run,
	}

	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	page := ctx.Int("page")
	path := ctx.String("path")
	r18 := ctx.Bool("r18")
	latest := ctx.Bool("latest")
	numConnections := ctx.Int("num-connections")

	if !pathExits(path) {
		if err := os.MkdirAll(path, os.ModeDir|0755); err != nil {
			log.Fatalf("mkdir %s error: %s", path, err)
		}
	}

	// 最大的下载线程数为 10
	if numConnections > 10 {
		numConnections = 10
	}

	signalChan := make(chan os.Signal)
	doneChan := make(chan bool)
	exitGetURLHandlerChan := make(chan bool)
	imagesChan := make(chan *image, numConnections*2)

	log.Printf("[I] imagesChan max len is %d", numConnections*2)

	go getURLHandler(page, r18, imagesChan, exitGetURLHandlerChan)

	wg := new(sync.WaitGroup)
	for i := 0; i <= numConnections; i++ {
		wg.Add(1)
		go downloadHandler(path, latest, wg, imagesChan, time.Duration(numConnections/2)*time.Second)
	}

	signal.Notify(signalChan, os.Interrupt)
	go signalHandler(signalChan, exitGetURLHandlerChan, doneChan, wg)

	<-doneChan

	return nil
}
