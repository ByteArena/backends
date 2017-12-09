// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package types

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

func easyjson8c019eaDecodeGithubComBytearenaBytearenaCommonTypes(in *jlexer.Lexer, out *GameDescriptionGQL) {
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
func easyjson8c019eaEncodeGithubComBytearenaBytearenaCommonTypes(out *jwriter.Writer, in GameDescriptionGQL) {
	out.RawByte('{')
	first := true
	_ = first
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v GameDescriptionGQL) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson8c019eaEncodeGithubComBytearenaBytearenaCommonTypes(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v GameDescriptionGQL) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson8c019eaEncodeGithubComBytearenaBytearenaCommonTypes(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *GameDescriptionGQL) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson8c019eaDecodeGithubComBytearenaBytearenaCommonTypes(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *GameDescriptionGQL) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson8c019eaDecodeGithubComBytearenaBytearenaCommonTypes(l, v)
}
