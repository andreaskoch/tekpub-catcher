// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"os/user"
	"path/filepath"
)

const (
	defaultFeedUrl = "http://delivery.tekpub.com/account/itunes.xml?token=123"
)

type CatcherSettings struct {
	DownloadPath string
	FeedUrl      string
}

var settings CatcherSettings = CatcherSettings{}

func init() {

	defaultDownloadPath := filepath.Join(getHomeDir(), "Videos/TekPub")

	flag.StringVar(&settings.DownloadPath, "downloadpath", defaultDownloadPath, "The target directory for your videos")

	flag.StringVar(&settings.FeedUrl, "feedurl", defaultFeedUrl, "Your TekPub feed URL")
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		message("Cannot determine the current users home direcotry. Error: %s", err)
		os.Exit(1)
	}

	return filepath.Clean(usr.HomeDir)
}
