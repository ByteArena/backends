// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package mailboxmessages

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonF5fe3c73DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(in *jlexer.Lexer, out *Stats) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "distanceTravelled":
			out.DistanceTravelled = float64(in.Float64())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonF5fe3c73EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(out *jwriter.Writer, in Stats) {
	out.RawByte('{')
	first := true
	_ = first
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"distanceTravelled\":")
	out.Float64(float64(in.DistanceTravelled))
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Stats) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonF5fe3c73EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Stats) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonF5fe3c73EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Stats) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonF5fe3c73DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Stats) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonF5fe3c73DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(l, v)
}
