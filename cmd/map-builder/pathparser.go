package main

// Regexes from https://github.com/jkroso/parse-svg-path

import (
	"log"
	"regexp"
	"strconv"
)

type PathOperation struct {
	Operation string
	Coords    []float64
}

func ParseSVGPath(path string) []PathOperation {
	ops := make([]PathOperation, 0)

	segment := regexp.MustCompile(`(?i)([astvzqmhlc])([^astvzqmhlc]*)`) //ig
	for _, match := range segment.FindAllStringSubmatch(path, -1) {
		ops = append(ops, PathOperation{
			Operation: match[1],
			Coords:    parseNumbers(match[2]),
		})

	}

	return ops
}

func parseNumbers(value string) []float64 {
	res := make([]float64, 0)
	reg := regexp.MustCompile(`(?i)-?[0-9]*\.?[0-9]+(?:e[-+]?\d+)?`) // ig
	for _, match := range reg.FindAllStringSubmatch(value, -1) {
		f, err := strconv.ParseFloat(match[0], 64)
		if err != nil {
			log.Panicln("Error: cannot parse float value", match[0])
		}

		res = append(res, f)

	}

	return res
}
