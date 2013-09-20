// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/SlyMarbo/rss"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	VERSION = "0.1.0"
)

var (
	InvalidPathCharacters = regexp.MustCompile(`[^\pL\pN\p{Latin}\.-_\s#]`)
	SequenceNumberPattern = regexp.MustCompile(`https?://delivery\.tekpub\.com/.+[^/]+/(\d+)/hd/file\.mp4\?token=[\w\d]+`)
	WhitespacePattern     = regexp.MustCompile(`\s+`)
)

var usage = func() {
	message("Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {

	// print application info
	message("%s (Version: %s)\n\n", os.Args[0], VERSION)

	// parse the flags
	flag.Parse()

	// check the arguments
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}

	// clean the download path
	settings.DownloadPath = filepath.Clean(settings.DownloadPath)

	// normalize the download path
	absolutePath, err := filepath.Abs(settings.DownloadPath)
	if err != nil {
		message("Cannot determine the absolute path from %q.", settings.DownloadPath)
		os.Exit(1)
	}

	settings.DownloadPath = absolutePath

	// check the download path
	if !pathExists(settings.DownloadPath) {
		message("The directory %q does not exist.", settings.DownloadPath)
		os.Exit(1)
	}

	// check the download path
	if !isDirectory(settings.DownloadPath) {
		message("The path %q is no directory.", settings.DownloadPath)
		os.Exit(1)
	}

	// check the feed url
	if strings.TrimSpace(settings.FeedUrl) == "" || settings.FeedUrl == defaultFeedUrl {
		message("Please specify a feed url.")
		usage()
		os.Exit(1)
	}

	// stop listener
	message(`Write "stop" and press <Enter> to stop rendering.`)

	stop := false
	go func() {
		input := bufio.NewReader(os.Stdin)

		for {

			input, err := input.ReadString('\n')
			if err != nil {
				fmt.Errorf("%s\n", err)
			}

			sanatizedInput := strings.ToLower(strings.TrimSpace(input))
			if sanatizedInput == "stop" {
				message("Stopping the download process.")
				stop = true
			}
		}
	}()

	// fetch the feed
	message("Fetching feed %q", settings.FeedUrl)
	feed, err := rss.Fetch(settings.FeedUrl)
	if err != nil {
		log.Panic(err)
	}

	// download
	message("Downloading to folder %q.", settings.DownloadPath)
	for _, item := range feed.Items {

		model, err := getDownloadModel(item)
		if err != nil {
			message("Unable to parse item %q.", item.Title)
			continue
		}

		download(model)

		if stop {
			message("Download process stopped.")
			break
		}
	}
}

func download(model *Download) {
	resp, err := http.Get(model.SourceUrl)
	if err != nil {
		message("Unable to download item %q. Error: %s", model, err)
		return
	}

	// create the folder
	folder := filepath.Join(settings.DownloadPath, model.Foldername)
	if !pathExists(folder) {
		createDirectory(folder)
	}

	// assemble the target path
	downloadPath := filepath.Join(folder, model.Filename)

	// check if the file already exists
	if pathExists(downloadPath) {
		message("Skipping item %q. The file has already been downloaded before.", model)
		return
	}

	// prepare the writer
	file, err := os.OpenFile(downloadPath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		message("Unable to open file %q for writing. Error: %s", downloadPath, err)
		return
	}

	message("Downloading item %q to %q.", model, downloadPath)

	writer := bufio.NewWriter(file)

	defer func() {
		resp.Body.Close()
		writer.Flush()
		file.Close()
	}()

	reader := bufio.NewReader(resp.Body)
	readBuffer := make([]byte, 4096)

	continueReading := true
	for continueReading {

		// read
		bytesRead, readError := reader.Read(readBuffer)
		if readError != nil {
			if readError == io.EOF {
				continueReading = false
			}
		}

		// write
		_, writeError := writer.Write(readBuffer[:bytesRead])
		if writeError != nil {
			fmt.Println(writeError)
		}
	}
}

func getDownloadModel(item *rss.Item) (*Download, error) {

	title := item.Title

	// check if the title contains the expected colon (e.g. "Mastering NHibernate 2: Search")
	titleSeparator := ": "
	titleSeparatorPosition := strings.Index(title, titleSeparator)
	if titleSeparatorPosition == -1 {
		return nil, fmt.Errorf("The title format is invalid: %q", title)
	}

	// the folder name is everything until the colon
	foldername := cleanPath(title[0:titleSeparatorPosition])

	// prepare the filename
	filename := cleanPath(title[(titleSeparatorPosition+len(titleSeparator)):]) + ".mp4"

	// add the folder name and sequence number
	if sequenceNumber, found := getSequenceNumberFromLink(item.Link); found {

		// add the folder name and sequence number
		filename = fmt.Sprintf("%s %03d %s", foldername, sequenceNumber, filename)

	} else {

		// only add the folder name
		filename = fmt.Sprintf("%s %s", foldername, filename)
	}

	// remove all white space
	filename = replaceWhitespace(filename)

	// create the model
	return &Download{
		Title:       title,
		Description: item.Content,
		Foldername:  foldername,
		Filename:    filename,
		SourceUrl:   item.Link,
	}, nil
}

func getSequenceNumberFromLink(link string) (number int64, found bool) {
	matches := SequenceNumberPattern.FindStringSubmatch(link)
	if len(matches) != 2 {
		return 0, false
	}

	sequenceNumber, err := strconv.ParseInt(matches[1], 10, 0)
	if err != nil {
		return 0, false
	}

	return sequenceNumber, true
}

func cleanPath(dirtyPath string) string {
	cleanedPath := strings.TrimSpace(dirtyPath)
	cleanedPath = filepath.Clean(dirtyPath)
	cleanedPath = InvalidPathCharacters.ReplaceAllString(cleanedPath, "")
	return cleanedPath
}

func replaceWhitespace(pathWithWhitespace string) string {
	return WhitespacePattern.ReplaceAllString(pathWithWhitespace, "-")
}

type Download struct {
	Title       string
	Description string
	Foldername  string
	Filename    string

	SourceUrl string
}

func (model *Download) String() string {
	return fmt.Sprintf("%s", model.Title)
}

func message(text string, args ...interface{}) {

	// append newline character
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	fmt.Printf(text, args...)
}
