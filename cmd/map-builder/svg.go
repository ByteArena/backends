package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"regexp"
	"strconv"
	"strings"

	"fmt"

	"github.com/bytearena/bytearena/common/utils/vector"
)

func ParseSVG(source []byte) SVGNode {
	root := NewSVGRoot()
	xml.Unmarshal(source, &root)
	return root
}

type SVGNode interface {
	GetParent() SVGNode
	GetChildren() []SVGNode
	GetTransform() vector.Matrix2
	GetFullTransform() vector.Matrix2
	AddChild(child SVGNode)
	GetId() string
	GetFill() string
}

type SVGBasicNode struct {
	id        string
	fill      string
	parent    SVGNode
	children  []SVGNode
	transform vector.Matrix2
}

func NewSVGBasicNode(parent SVGNode) *SVGBasicNode {
	return &SVGBasicNode{
		parent:    parent,
		children:  make([]SVGNode, 0),
		transform: vector.IdentityMatrix2(),
	}
}

func (n *SVGBasicNode) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	for _, attr := range start.Attr {

		switch attr.Name.Local {
		case "fill":
			{
				n.fill = attr.Value
			}
		case "id":
			{
				n.id = attr.Value
			}
		case "transform":
			{
				// parse transform
				/*
					translate(<x> [<y>])
						This transform definition specifies a translation by x and y. This is equivalent to matrix(1 0 0 1 x y). If y is not provided, it is assumed to be zero.

					scale(<x> [<y>])
						This transform definition specifies a scale operation by x and y. This is equivalent to matrix(x 0 0 y 0 0). If y is not provided, it is assumed to be equal to x.

					rotate(<a> [<x> <y>])
						This transform definition specifies a rotation by a degrees about a given point. If optional parameters x and y are not supplied, the rotate is about the origin of the current user coordinate system. The operation corresponds to the matrix
						(
							cosa	-sina	0	sina	cosa	0	0	0	1
						)
						If optional parameters x and y are supplied, the rotate is about the point (x, y). The operation represents the equivalent of the following transform definitions list: translate(<x>, <y>) rotate(<a>) translate(-<x>, -<y>).

					skewX(<a>)
						This transform definition specifies a skew transformation along the x axis by a degrees. The operation corresponds to the matrix
						(
							1	tan(a)	0	0	1	0	0	0	1
						)

					skewY(<a>)
						This transform definition specifies a skew transformation along the y axis by a degrees. The operation corresponds to the matrix
						(
							1	0	0	tan(a)	1	0	0	0	1
						)
				*/

				// supported are translate, scale, rotate, space separated;
				// example: translate(610.023848, 204.635405) rotate(-41.000000) translate(-610.023848, -204.635405) translate(460.523848, 121.635405)

				var re = regexp.MustCompile(`(translate|rotate|scale|skewX|skewY)\((.*?)\)`)
				transform := n.GetTransform()

				for _, match := range re.FindAllStringSubmatch(attr.Value, -1) {
					switch match[1] {
					case "translate":
						{

							// parse translate
							coords := strings.Split(match[2], ",")
							for i, coord := range coords {
								coords[i] = strings.TrimSpace(coord)
							}

							x, err := strconv.ParseFloat(coords[0], 64)
							y, err2 := strconv.ParseFloat(coords[1], 64)
							if err != nil || err2 != nil {
								log.Panicln("Could not parse translate transform", match[2])
							}

							transform = transform.Translate(x, y)
						}
					case "rotate":
						{
							a, err := strconv.ParseFloat(match[2], 64)
							if err != nil {
								log.Panicln("Could not parse rotate transform", match[2])
							}

							transform = transform.Rotate(a)
						}
					case "scale":
						{
							coords := strings.Split(match[2], ",")
							for i, coord := range coords {
								coords[i] = strings.TrimSpace(coord)
							}

							var y float64
							x, err := strconv.ParseFloat(coords[0], 64)
							if err != nil {
								log.Panicln("Could not parse scale transform", match[2])
							}

							if len(coords) == 1 {
								y = x
							} else {
								y, err = strconv.ParseFloat(coords[1], 64)
								if err != nil {
									log.Panicln("Could not parse scale transform", match[2])
								}
							}

							transform = transform.Scale(x, y)
						}
					case "skewX":
						{
							log.Panicln("transform skewX not implemented.")
							// parse scale
						}
					case "skewY":
						{
							log.Panicln("transform skewY not implemented.")
							// parse scale
						}
					}
				}

				n.SetTransform(transform)
			}
		}
	}

Loop:
	for {
		tok, err := decoder.Token()
		if err != nil {
			return err
		}

		switch typedtoken := tok.(type) {
		case xml.StartElement:
			{
				var node SVGNode
				add := false
				tagname := typedtoken.Name.Local

				if tagname == "g" {
					node = NewSVGGroup(n)
					add = true
				} else if tagname == "path" {
					node = NewSVGPath(n)
					add = true
				} else if tagname == "circle" {
					node = NewSVGCircle(n)
					add = true
				} else if tagname == "ellipse" {
					node = NewSVGEllipse(n)
					add = true
				} else if tagname == "polygon" {
					node = NewSVGPolygon(n)
					add = true
				} else {
					node = NewSVGBasicNode(n)
				}

				err := decoder.DecodeElement(&node, &typedtoken)
				if err != nil {
					continue Loop
				}

				if add {
					n.AddChild(node)
				}

				continue Loop
			}
		case xml.EndElement:
			{
				break Loop
			}
		}
	}

	return nil
}

func (n *SVGBasicNode) GetParent() SVGNode     { return n.parent }
func (n *SVGBasicNode) GetChildren() []SVGNode { return n.children }
func (n *SVGBasicNode) AddChild(child SVGNode) { n.children = append(n.children, child) }
func (n *SVGBasicNode) GetId() string          { return n.id }
func (n *SVGBasicNode) GetFill() string {
	if n.fill != "" {
		return n.fill
	}

	if n.parent != nil {
		return n.parent.GetFill() // inherit fill from parent if not defined on current node
	}

	return ""
}

func (n *SVGBasicNode) GetTransform() vector.Matrix2          { return n.transform }
func (n *SVGBasicNode) SetTransform(transform vector.Matrix2) { n.transform = transform }
func (n *SVGBasicNode) GetFullTransform() vector.Matrix2 {
	t := vector.IdentityMatrix2()

	var node SVGNode = n
	for node != nil {
		t = t.Mul(node.GetTransform())
		node = node.GetParent()
	}

	return t
}

func GetSVGIDs(n SVGNode) SVGIDCollection {

	var node SVGNode = n

	res := make(SVGIDCollection, 0)
	for node != nil {
		if node.GetId() != "" {
			res = append(res, node.GetId())
		}

		node = node.GetParent()
	}

	return res
}

type SVGRoot struct {
	*SVGBasicNode
}

func NewSVGRoot() *SVGRoot {
	return &SVGRoot{
		SVGBasicNode: NewSVGBasicNode(nil),
	}
}

type SVGGroup struct {
	*SVGBasicNode
}

type SVGIDCollection []string

func (c SVGIDCollection) Contains(search string) int {
	for i, group := range c {
		if strings.Contains(group, search) {
			return i
		}
	}

	return -1
}

type SVGIDFunction struct {
	Function string
	Args     json.RawMessage
	Original string
}

func (c SVGIDCollection) GetFunctions() []SVGIDFunction {
	funcs := make([]SVGIDFunction, 0)
	r := regexp.MustCompile("^ba:([a-zA-Z]+)\\((.*?)\\)$")
	for _, group := range c {
		parts := strings.Split(group, "-")
		for _, part := range parts {
			if r.MatchString(part) {
				matches := r.FindStringSubmatch(part)

				funcs = append(funcs, SVGIDFunction{
					Function: matches[1],
					Args:     json.RawMessage("[" + matches[2] + "]"),
					Original: part,
				})
			}
		}
	}

	return funcs
}

func NewSVGGroup(parent SVGNode) *SVGGroup {
	return &SVGGroup{
		SVGBasicNode: NewSVGBasicNode(parent),
	}
}

type SVGPath struct {
	*SVGBasicNode
	path string
}

func (n *SVGPath) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	for _, attr := range start.Attr {
		if attr.Name.Local == "d" {
			n.path = attr.Value
		}
	}

	return n.SVGBasicNode.UnmarshalXML(decoder, start)
}

func NewSVGPath(parent SVGNode) *SVGPath {
	return &SVGPath{
		SVGBasicNode: NewSVGBasicNode(parent),
	}
}

func (n *SVGPath) GetPath() string {
	return n.path
}

type SVGCircle struct {
	*SVGBasicNode
	cx float64
	cy float64
	r  float64
}

func NewSVGCircle(parent SVGNode) *SVGCircle {
	return &SVGCircle{
		SVGBasicNode: NewSVGBasicNode(parent),
	}
}

func (n *SVGCircle) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	for _, attr := range start.Attr {
		name := attr.Name.Local
		if name == "cx" {
			n.cx, _ = strconv.ParseFloat(attr.Value, 64)
		} else if name == "cy" {
			n.cy, _ = strconv.ParseFloat(attr.Value, 64)
		} else if name == "r" {
			n.r, _ = strconv.ParseFloat(attr.Value, 64)
		}
	}

	return n.SVGBasicNode.UnmarshalXML(decoder, start)
}

func (n *SVGCircle) GetCenter() (float64, float64) {
	return n.cx, n.cy
}

func (n *SVGCircle) GetRadius() float64 {
	return n.r
}

type SVGEllipse struct {
	*SVGBasicNode
	cx float64
	cy float64
	rx float64
	ry float64
}

func (n *SVGEllipse) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	for _, attr := range start.Attr {
		name := attr.Name.Local
		if name == "cx" {
			n.cx, _ = strconv.ParseFloat(attr.Value, 64)
		} else if name == "cy" {
			n.cy, _ = strconv.ParseFloat(attr.Value, 64)
		} else if name == "rx" {
			n.rx, _ = strconv.ParseFloat(attr.Value, 64)
		} else if name == "ry" {
			n.ry, _ = strconv.ParseFloat(attr.Value, 64)
		}
	}

	return n.SVGBasicNode.UnmarshalXML(decoder, start)
}

func NewSVGEllipse(parent SVGNode) *SVGEllipse {
	return &SVGEllipse{
		SVGBasicNode: NewSVGBasicNode(parent),
	}
}

func (n *SVGEllipse) GetCenter() (float64, float64) {
	return n.cx, n.cy
}

func (n *SVGEllipse) GetRadius() (float64, float64) {
	return n.rx, n.ry
}

type SVGPolygon struct {
	*SVGBasicNode
	points string
}

func NewSVGPolygon(parent SVGNode) *SVGPolygon {
	return &SVGPolygon{
		SVGBasicNode: NewSVGBasicNode(parent),
	}
}

func (n *SVGPolygon) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	for _, attr := range start.Attr {
		if attr.Name.Local == "points" {
			n.points = attr.Value
		}
	}

	return n.SVGBasicNode.UnmarshalXML(decoder, start)
}

func SVGDebug(node SVGNode, depth int) string {
	var res []string
	var nodename string

	prefix := strings.Repeat("\t", depth)
	additionalAttrs := ""

	switch typednode := node.(type) {
	case *SVGRoot:
		nodename = "SVGRoot"
	case *SVGGroup:
		nodename = "SVGGroup"
	case *SVGPath:
		nodename = "SVGPath"
	case *SVGCircle:
		nodename = "SVGCircle"
		cx, cy := typednode.GetCenter()
		additionalAttrs = fmt.Sprintf("cx=\"%f\" cy=\"%f\" r=\"%f\"", cx, cy, typednode.GetRadius())
	case *SVGEllipse:
		nodename = "SVGEllipse"
		cx, cy := typednode.GetCenter()
		rx, ry := typednode.GetRadius()
		additionalAttrs = fmt.Sprintf("cx=\"%f\" cy=\"%f\" rx=\"%f\" ry=\"%f\"", cx, cy, rx, ry)
	case *SVGPolygon:
		nodename = "SVGPolygon"
	default:
		nodename = "UnknownNode"
	}

	signature := nodename
	if node.GetId() != "" {
		signature += " id=\"" + node.GetId() + "\""
	}

	if node.GetFill() != "" {
		signature += " fill=\"" + node.GetFill() + "\""
	}

	if !node.GetTransform().IsIdentity() {
		signature += " transform=\"" + node.GetTransform().String() + "\""
		signature += " fulltransform=\"" + node.GetFullTransform().String() + "\""
	}

	if additionalAttrs != "" {
		signature += " " + additionalAttrs
	}

	children := node.GetChildren()
	if len(children) == 0 {
		return prefix + "<" + signature + " />"
	}

	res = append(res, prefix+"<"+signature+">")
	for _, child := range children {
		res = append(res, SVGDebug(child, depth+1))
	}
	res = append(res, prefix+"</"+nodename+">")
	return strings.Join(res, "\n")

}
