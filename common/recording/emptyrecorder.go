package recording

import (
	"github.com/bytearena/bytearena/common/types/mapcontainer"
)

type EmptyRecorder struct{}

func MakeEmptyRecorder() EmptyRecorder {
	return EmptyRecorder{}
}

func (r EmptyRecorder) Record(UUID string, msg string) error {
	return nil
}

func (r EmptyRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	return nil
}

func (r EmptyRecorder) Close(UUID string) {}
func (r EmptyRecorder) Stop()             {}

func (r EmptyRecorder) GetDirectory() string {
	return ""
}
