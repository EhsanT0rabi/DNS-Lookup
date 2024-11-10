package main

import (
	"github.com/urfave/cli"
)

func NewCli() *cli.App {
	app := cli.App{Flags: []cli.Flag{
		cli.StringFlag{Name: "dns-server", Value: "1.1.1.1", Usage: "--dns-server \"your dns server\""},
		cli.StringFlag{Name: "domains", Value: "", Usage: "--domains domain1,domain2,..."},
	},

		Name:   "dns-lookup",
		Author: "Ehasn Torabi",
		Email:  "torabi782@gmail.com",
	}
	return &app
}
