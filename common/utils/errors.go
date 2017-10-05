package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/ttacon/chalk"
)

func Check(err error, msg string) {
	if err != nil {
		RecoverableCheck(err, msg)
		os.Exit(1)
	}
}

func RecoverableCheck(err error, msg string) {
	if err != nil {
		fmt.Print(chalk.Red)
		log.Print(msg+"; "+err.Error(), chalk.Reset)
	}
}

func Assert(ok bool, msg string) {
	if !ok {
		fmt.Print(chalk.Red)
		log.Print(msg, chalk.Reset)
		os.Exit(1)
	}
}

func CheckWithFunc(err error, fn func() string) {
	if err != nil {
		msg := fn()

		Check(err, msg)
	}
}
