package main

import (
	"html/template"
	"os"
	"reflect"
	"regexp"

	"github.com/bytearena/bytearena/common/types"
)

// COPIED

type Specs struct {
	MaxSpeed           float64     `json:"maxspeed"`
	MaxSteeringForce   float64     `json:"maxsteeringforce"`
	MaxAngularVelocity float64     `json:"maxangularvelocity"`
	VisionRadius       float64     `json:"visionradius"`
	VisionAngle        types.Angle `json:"visionangle"`

	BodyRadius float64 `json:"bodyradius"`

	MaxShootEnergy    float64 `json:"maxshootenergy"`
	ShootRecoveryRate float64 `json:"shootrecoveryrate"`

	Gear map[string]GearSpecs
}

type GearSpecs struct {
	Genre string
	Kind  string
	Specs interface{}
}

type GunSpecs struct {
	ShootCost        float64 `json:"shootcost"`
	ShootCooldown    int     `json:"shootcooldown"`
	ProjectileSpeed  float64 `json:"projectilespeed"`
	ProjectileDamage float64 `json:"projectiledamage"`
	ProjectileRange  float64 `json:"projectilerange"`
}

// COPIED

var (
	jsonTagRegexp = regexp.MustCompile(`json:"(.*)"`)

	docTemplate = `
<a name="{{.Title}}"></a>
### ` + "`" + `{{.Title}}` + "`" + `
{{ if .Fields }}
| Property name | Type | Representation in the JSON |
|---|---|
{{ range $value := .Fields }}| {{ $value.Name }} | ` + "`" + `{{ $value.Type }}` + "`" + ` | ` + "`" + `{{ $value.TypeInJson }}` + "`" + ` |
{{ end }}{{ end }}
`

	runtimeTypes = map[string]string{
		"unknown":   "Object",
		"Angle":     "Number (radian)",
		"GearSpecs": "Object",
		"float64":   "Number",
		"int":       "Number",
		"string":    "string",
	}
)

type DocField struct {
	Name       string
	Type       string
	TypeInJson string
}

type DocEntry struct {
	Title  string
	Fields []DocField
}

func normalizeTypeName(t string) string {

	if t == "types.Angle" {

		return "Angle"
	} else if t == "map[string]main.GearSpecs" {

		return "GearSpecs"
	} else if t == "vector.Vector2" {

		return "Vector2"
	} else if t == "interface {}" {

		return "unknown"
	} else {

		return t
	}
}

func main() {
	generateDocumentationFor(Specs{})
	generateDocumentationFor(GearSpecs{})
	generateDocumentationFor(GunSpecs{})
}

func generateDocumentationFor(_struct interface{}) {
	structType := reflect.TypeOf(_struct)

	entry := DocEntry{
		Title:  structType.Name(),
		Fields: make([]DocField, 0),
	}

	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)

		typeName := normalizeTypeName(structField.Type.String())

		docField := DocField{
			Name:       structField.Name,
			Type:       typeName,
			TypeInJson: runtimeTypes[typeName],
		}

		if docField.Type == "" {
			docField.Type = "unknown"
		}

		if structField.Tag != "" {
			matches := jsonTagRegexp.FindSubmatch([]byte(structField.Tag))
			docField.Name = string(matches[1])
		}

		entry.Fields = append(entry.Fields, docField)
	}

	tmpl, err := template.New("").Parse(docTemplate)

	err = tmpl.Execute(os.Stdout, entry)

	if err != nil {
		panic(err)
	}
}
