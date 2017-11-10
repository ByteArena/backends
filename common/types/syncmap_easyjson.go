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

func easyjson8de6858fDecodeGithubComBytearenaBytearenaCommonTypes(in *jlexer.Lexer, out *SyncMap) {
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
func easyjson8de6858fEncodeGithubComBytearenaBytearenaCommonTypes(out *jwriter.Writer, in SyncMap) {
	out.RawByte('{')
	first := true
	_ = first
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v SyncMap) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson8de6858fEncodeGithubComBytearenaBytearenaCommonTypes(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v SyncMap) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson8de6858fEncodeGithubComBytearenaBytearenaCommonTypes(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *SyncMap) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson8de6858fDecodeGithubComBytearenaBytearenaCommonTypes(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *SyncMap) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson8de6858fDecodeGithubComBytearenaBytearenaCommonTypes(l, v)
}
