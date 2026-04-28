package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	gptprotoimage "github.com/QuantumNous/new-api/relay/channel/task/gptproto_image"
)

func TestGetTaskAdaptorReturnsGPTProtoImageAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatformGPTProtoImage)
	if adaptor == nil {
		t.Fatalf("GetTaskAdaptor returned nil")
	}
	if _, ok := adaptor.(*gptprotoimage.TaskAdaptor); !ok {
		t.Fatalf("GetTaskAdaptor returned %T, want *gptproto_image.TaskAdaptor", adaptor)
	}
}
