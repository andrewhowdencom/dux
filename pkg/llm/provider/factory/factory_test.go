package factory_test

import (
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/factory"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     provider.InstanceConfig
		wantErr bool
	}{
		{
			name: "static provider",
			cfg: provider.InstanceConfig{
				Type: "static",
			},
			wantErr: false,
		},
		{
			name: "ollama provider",
			cfg: provider.InstanceConfig{
				Type: "ollama",
			},
			wantErr: false,
		},
		{
			name: "openai provider",
			cfg: provider.InstanceConfig{
				Type: "openai",
			},
			wantErr: false,
		},
		{
			name: "litellm provider",
			cfg: provider.InstanceConfig{
				Type: "litellm",
			},
			wantErr: false,
		},
		{
			name: "unknown provider",
			cfg: provider.InstanceConfig{
				Type: "aws-bedrock",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
