package recording

type EmptyRecorder struct{}

func MakeEmptyRecorder() Recorder {
	return EmptyRecorder{}
}

func (r EmptyRecorder) Record(arenaId string, msg string) error {
	return nil
}

func (r EmptyRecorder) Close() {
}

func (r EmptyRecorder) GetDirectory() string {
	return ""
}
