package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestResolvePlaygroundUsingGroupFallsBackToUserGroup(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	relayInfo := &relaycommon.RelayInfo{}
	userCache := &model.UserBase{Group: "TunnelForY"}

	usingGroup := resolvePlaygroundUsingGroup(ctx, relayInfo, userCache)
	if usingGroup != "TunnelForY" {
		t.Fatalf("expected using group to fall back to user group, got %q", usingGroup)
	}

	if got := common.GetContextKeyString(ctx, constant.ContextKeyUsingGroup); got != "TunnelForY" {
		t.Fatalf("expected context using group to be written, got %q", got)
	}
}

func TestResolvePlaygroundUsingGroupKeepsExplicitGroup(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	relayInfo := &relaycommon.RelayInfo{UsingGroup: "auto"}
	userCache := &model.UserBase{Group: "TunnelForY"}

	usingGroup := resolvePlaygroundUsingGroup(ctx, relayInfo, userCache)
	if usingGroup != "auto" {
		t.Fatalf("expected explicit using group to be kept, got %q", usingGroup)
	}

	if got := common.GetContextKeyString(ctx, constant.ContextKeyUsingGroup); got != "auto" {
		t.Fatalf("expected context using group to keep explicit value, got %q", got)
	}
}
