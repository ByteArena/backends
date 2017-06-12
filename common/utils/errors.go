package utils

import (
	"fmt"
	"log"

	"github.com/ttacon/chalk"
)

func Check(err error, msg string) {
	if err != nil {
		fmt.Print(chalk.Red)
		log.Print(msg, chalk.Reset)
		log.Panicln(err)
	}
}

func Assert(ok bool, msg string) {
	if !ok {
		fmt.Print(chalk.Red)
		log.Print(msg, chalk.Reset)
		log.Panic()
	}
}
