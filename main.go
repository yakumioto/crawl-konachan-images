package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	wg          sync.WaitGroup
	konachanURL = "http://konachan.net/post.json?page="
)

type image struct {
	ID          int    `json:"id"`
	FileURL     string `json:"file_url"`
	HasChildren bool   `json:"has_children"`
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	err := os.Mkdir("Konachan", os.ModeDir|0755)
	if err != nil {
		if !os.IsExist(err) {
			fmt.Println("Mkdir folder error:", err)
		}
	}
}

func main() {
	imagesChan := make(chan *image, 200)

	go getImageHandler(imagesChan)

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go downloadImagesHandler(imagesChan)
	}

	wg.Wait()
}

func getImageHandler(imagesChan chan<- *image) {
	page := 1
	for {
		resp, err := http.Get(konachanURL + strconv.Itoa(page))
		if err != nil {
			fmt.Println("Get error:", err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Reading error:", err)
			return
		}

		images := make([]*image, 0)
		if err := json.Unmarshal(body, &images); err != nil {
			fmt.Println("Unmarshal error:", err)
			continue
		}

		for i := 0; i < len(images)-1; i++ {
			imagesChan <- images[i]
		}

		fmt.Println("Current Page:", page, "Current image channel len:", len(imagesChan))
		page++
	}
}

func downloadImagesHandler(imagesChan <-chan *image) {
	defer wg.Done()

	for {
		select {
		case image := <-imagesChan:
			resp, err := http.Get("http:" + image.FileURL)
			if err != nil {
				fmt.Println("Get image error:", err)
				continue
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Reading image error:", err)
				continue
			}

			if !pathExist("Konachan/" + strconv.Itoa(image.ID) + ".png") {
				if err = ioutil.WriteFile("Konachan/"+strconv.Itoa(image.ID)+".png", body, 0644); err != nil {
					fmt.Println("Writing image error:", err)
				}
			} else {
				fmt.Println("ID:", image.ID, "exist.")
			}

		case <-time.After(time.Second * 60):
			fmt.Println("Worker time out!")
			return
		}
	}
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
