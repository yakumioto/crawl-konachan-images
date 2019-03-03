package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type image struct {
	ID       int    `json:"id"`
	FileURL  string `json:"file_url"`
	Rating   string `json:"rating"`
	retryNum int    `json:"-"`
}

func getURLHandler(page int, r18 bool, size int, imagesChan chan<- *image, exitChan <-chan bool) {
	downSizeNum := 1

	if size == -1 {
		downSizeNum = -1
	}

	for {
		select {
		case <-exitChan:
			fmt.Println("[I] exit get url handler")
			return
		default:
			resp, err := client.Get("http://konachan.net/post.json?page=" + strconv.Itoa(page))
			if err != nil {
				log.Printf("[E] get post.json?page=%d error: %s\n", page, err)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("[E] read post.json?page=%d error: %s\n", page, err)
				continue
			}

			images := [21]*image{}
			if err := json.Unmarshal(body, &images); err != nil {
				log.Printf("[E] unmarshal json post.json?page=%d error: %s\n", page, err)
				continue
			}

			for _, image := range images {
				image.retryNum = 3
				if image == nil {
					continue
				}

				if downSizeNum > 0 {
					downSizeNum++
					if downSizeNum == size {
						return
					}
				}

				if !r18 {
					if image.Rating == "s" {

						imagesChan <- image
						continue
					}
				}
				imagesChan <- image
			}
			_ = resp.Body.Close()
		}
		log.Printf("[I] current page: %d, current image chan len: %d\n", page, len(imagesChan))
		page++
	}
}

func downloadHandler(path string, latest bool, wg *sync.WaitGroup, imagesChan chan *image, timeout time.Duration) {
	defer wg.Done()

	for {
		select {
		case image := <-imagesChan:
			if image.retryNum <= 0 {
				continue
			}

			image.retryNum--

			if latest {
				log.Printf("[I] download over, quit now!")
				os.Exit(0)
			}

			if pathExits(path + "/" + strconv.Itoa(image.ID) + ".png") {
				log.Printf("[W] %d exist\n", image.ID)
				continue
			}

			url := ""
			if strings.HasPrefix("//", image.FileURL) {
				url = "https:" + image.FileURL
			} else {
				url = image.FileURL
			}

			resp, err := client.Get(url)
			if err != nil {
				log.Printf("[E] get %d image error: %s\n", image.ID, err)
				imagesChan <- image
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("[E] read %d image error: %s\n", image.ID, err)
				imagesChan <- image
				continue
			}

			if err := ioutil.WriteFile(path+"/"+strconv.Itoa(image.ID)+".png", body, 0664); err != nil {
				log.Printf("[E] write %d image error: %s\n", image.ID, err)
				imagesChan <- image
				continue
			}
		case <-time.After(timeout):
			log.Printf("[I] exit download worker\n")
			return
		}
	}
}

func signalHandler(signalChan <-chan os.Signal, exitGetURLHandlerChan chan<- bool) {
	<-signalChan
	log.Printf("[I] cleaning work is under way...\n")
	exitGetURLHandlerChan <- true
}

func pathExits(path string) bool {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func showInfo(ticker *time.Ticker, imagesChan chan *image) {
	for range ticker.C {
		log.Printf("[I] current image chan len: %d\n", len(imagesChan))
	}
}
