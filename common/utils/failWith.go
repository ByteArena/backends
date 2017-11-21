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
			New(command).
			SetContext("version", GetVersion()).
			With(err)

		msg := bettererrorstree.PrintChain(berror)

		urlOptions := url.Values{}
		urlOptions.Set("body", wrapInMarkdownCode(msg))

		fmt.Println("")
		fmt.Println("❌  An error occurred.")
		fmt.Println("")

		fmt.Print(msg)

		fmt.Println("")

		fmt.Println("Please report this error here: https://github.com/ByteArena/cli/issues/new?" + urlOptions.Encode())

		os.Exit(1)
	} else {
		panic(err)
	}
}

func wrapInMarkdownCode(str string) string {
	return fmt.Sprintf("```sh\n%s\n```", str)
}

func WarnWith(err error) {
	if bettererrors.IsBetterError(err) {
		msg := bettererrorstree.PrintChain(err.(*bettererrors.Chain))

		fmt.Println("")
		fmt.Println("⚠️  Warning")
		fmt.Println("")

		fmt.Print(msg)

		fmt.Println("")
	} else {
		fmt.Println(err.Error())
	}
}
