package utils

import (
	"fmt"
	"net/url"
	"os"

	bettererrors "github.com/xtuc/better-errors"
	bettererrorstree "github.com/xtuc/better-errors/printer/tree"
)

func WarnWith(err error) {
	if bettererrors.IsBetterError(err) {
		msg := bettererrorstree.PrintChain(err.(*bettererrors.Chain))

		fmt.Println("")
		fmt.Println("=== ❌ warning")
		fmt.Println("")

		fmt.Print(msg)

	} else {
		fmt.Println(err.Error())
	}
}

func FailWith(err error) {
	if bettererrors.IsBetterError(err) {

		msg := bettererrorstree.PrintChain(err.(*bettererrors.Chain))

		urlOptions := url.Values{}
		urlOptions.Set("body", msg)

		fmt.Println("")
		fmt.Println("=== ")
		fmt.Println("=== ❌ an error occurred.")
		fmt.Println("===")
		fmt.Println("=== Please report this error here: https://github.com/ByteArena/cli/issues/new?" + urlOptions.Encode())
		fmt.Println("=== We will fix it as soon as possible.")
		fmt.Println("===")
		fmt.Println("")

		fmt.Print(msg)

		os.Exit(1)
	} else {
		panic(err)
	}
}
