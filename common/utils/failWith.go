package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	bettererrors "github.com/xtuc/better-errors"
	bettererrorstree "github.com/xtuc/better-errors/printer/tree"
)

func FailWith(err error) {
	if bettererrors.IsBetterError(err) {

		command := strings.Join(os.Args, " ")

		berror := bettererrors.
			NewFromString(command).
			With(err)

		msg := bettererrorstree.PrintChain(berror)

		urlOptions := url.Values{}
		urlOptions.Set("body", wrapInMarkdownCode(msg))

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

func wrapInMarkdownCode(str string) string {
	return fmt.Sprintf("```sh\n%s\n```", str)
}
