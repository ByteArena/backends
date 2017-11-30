package main

import (
	"html/template"
	"os"
	"reflect"
	"regexp"

	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/game/deathmatch/mailboxmessages"
)

// COPIED

type mailboxMessagePerceptionWrapper struct {
	Subject string      `json:"subject"`
	Body    interface{} `json:"body"`
}

type agentPerceptionVisionItem struct {
	Tag      string         `json:"tag"`
	NearEdge vector.Vector2 `json:"nearedge"`
	Center   vector.Vector2 `json:"center"`
	FarEdge  vector.Vector2 `json:"faredge"`
	Velocity vector.Vector2 `json:"velocity"`
}

type agentPerception struct {
	Score int `json:"score"`

	Energy        float64                           `json:"energy"`   // niveau en millièmes; reconstitution automatique ?
	Velocity      vector.Vector2                    `json:"velocity"` // vecteur de force (direction, magnitude)
	Azimuth       float64                           `json:"azimuth"`  // azimuth en degrés par rapport au "Nord" de l'arène
	Vision        []agentPerceptionVisionItem       `json:"vision"`
	ShootEnergy   float64                           `json:"shootenergy"`
	ShootCooldown int                               `json:"shootcooldown"`
	Messages      []mailboxMessagePerceptionWrapper `json:"messages"`
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
		"unknown":                               "Object",
		"string":                                "String",
		"float64":                               "Number",
		"int":                                   "Number",
		"vector.Vector2":                        "Array of x, y",
		"array agentPerceptionVisionItem":       "Array of Object",
		"array mailboxMessagePerceptionWrapper": "Array of Object",
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

	if t == "[]main.agentPerceptionVisionItem" {

		return "array agentPerceptionVisionItem"
	} else if t == "[]main.mailboxMessagePerceptionWrapper" {

		return "array mailboxMessagePerceptionWrapper"
	} else if t == "interface {}" {

		return "unknown"
	} else {

		return t
	}
}

func main() {
	generateDocumentationFor(agentPerception{})
	generateDocumentationFor(agentPerceptionVisionItem{})
	generateDocumentationFor(mailboxMessagePerceptionWrapper{})

	generateDocumentationFor(mailboxmessages.Score{})
	generateDocumentationFor(mailboxmessages.Stats{})
	generateDocumentationFor(mailboxmessages.YouAreRespawning{})
	generateDocumentationFor(mailboxmessages.YouHaveBeenFragged{})
	generateDocumentationFor(mailboxmessages.YouHaveBeenHit{})
	generateDocumentationFor(mailboxmessages.YouHaveFragged{})
	generateDocumentationFor(mailboxmessages.YouHaveHit{})
	generateDocumentationFor(mailboxmessages.YouHaveRespawned{})
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
