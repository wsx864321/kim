package logic

import (
	sessionpb "github.com/wsx864321/kim/idl/session"
	"github.com/wsx864321/kim/internal/session/pkg/config"
	"github.com/wsx864321/kim/pkg/xjson"
	"testing"
	"time"
)

func TestParseJWT(t *testing.T) {
	config.Init("../../../config/session.yaml")

	jwt, err := GenerateJWT("111111", time.Now().Add(1*time.Hour).Unix())
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	t.Logf("Generated JWT: %s", jwt)
	claim, err := ParseJWT(jwt)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}
	if claim.UserID != "111111" {
		t.Fatalf("ParseJWT UserID mismatch: got %v, want %v", claim.UserID, "111111")
	}

	info := &sessionpb.AuthInfo{
		Token:      jwt,
		DeviceId:   "device123",
		DeviceType: sessionpb.DeviceType_DEVICE_TYPE_MOBILE,
		AppVersion: "1.0.0",
		Meta:       nil,
	}

	t.Log(xjson.MarshalString(info))
}
