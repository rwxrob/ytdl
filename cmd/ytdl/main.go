package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rwxrob/ytdl"
)

func main() {
	path, err := ytdl.Download(context.Background(), ytdl.DownloadOptions{
		URL:    "https://www.youtube.com/watch?v=BaW_jenozKc",
		OutDir: "downloads",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(path)
}
