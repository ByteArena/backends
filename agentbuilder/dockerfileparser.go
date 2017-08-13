package agentbuilder

import (
	"io"

	dockerfileparser "github.com/docker/docker/builder/dockerfile/parser"
)

func DockerfileParserGetFroms(source io.Reader) ([]string, error) {

	result, err := dockerfileparser.Parse(source)
	if err != nil {
		return nil, err
	}

	fromValues := make([]string, 0)
	visit(result.AST, func(node *dockerfileparser.Node) {
		if node.Value == "from" {
			fromValues = append(fromValues, node.Next.Value)
		}
	})

	return fromValues, nil
}

func visit(node *dockerfileparser.Node, cbk func(n *dockerfileparser.Node)) {

	//DockerFileRow
	for _, n := range node.Children {
		cbk(n)
	}

	/*
		for n := node.Next; n != nil; n = n.Next {
			if len(n.Children) > 0 {
				//str += " " + n.Dump()
				//			visit(n, cbk)
			} else {
				//		str += " " + strconv.Quote(n.Value)
			}
		}
	*/
}
