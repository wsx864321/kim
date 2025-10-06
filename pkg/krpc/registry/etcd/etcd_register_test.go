package etcd

import (
	"context"
	"github.com/wsx864321/kim/pkg/krpc/registry"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewETCDRegister(t *testing.T) {
	_, err := NewETCDRegister()

	assert.Nil(t, err)
}

func TestRegister_Register(t *testing.T) {
	register, _ := NewETCDRegister()

	service := &registry.Service{
		Name: "test",
		Endpoints: []*registry.Endpoint{
			{
				ServerName: "test",
				IP:         "127.0.0.1",
				Port:       9557,
				Weight:     100,
				Enable:     true,
			},
		},
	}
	register.Register(context.TODO(), service)
	time.Sleep(2 * time.Second)
	registerService := register.GetService(context.TODO(), "test")

	assert.Equal(t, *service.Endpoints[0], *registerService.Endpoints[0])
}
