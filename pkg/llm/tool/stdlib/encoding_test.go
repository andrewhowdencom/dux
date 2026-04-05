package stdlib

import (
	"context"
	"testing"
)

func TestEncodingTools(t *testing.T) {
	ctx := context.Background()

	t.Run("Base64", func(t *testing.T) {
		encTool := NewBase64Encode()
		decTool := NewBase64Decode()

		encRes, err := encTool.Execute(ctx, map[string]interface{}{"text": "hello world"})
		if err != nil {
			t.Fatal(err)
		}
		encoded := encRes.(map[string]string)["encoded"]

		decRes, err := decTool.Execute(ctx, map[string]interface{}{"encoded": encoded})
		if err != nil {
			t.Fatal(err)
		}
		if decRes.(map[string]string)["text"] != "hello world" {
			t.Fatal("decode mismatch")
		}
	})

	t.Run("URL", func(t *testing.T) {
		encTool := NewURLEncode()
		decTool := NewURLDecode()

		encRes, err := encTool.Execute(ctx, map[string]interface{}{"text": "hello/world?id=1"})
		if err != nil {
			t.Fatal(err)
		}
		encoded := encRes.(map[string]string)["encoded"]

		decRes, err := decTool.Execute(ctx, map[string]interface{}{"encoded": encoded})
		if err != nil {
			t.Fatal(err)
		}
		if decRes.(map[string]string)["text"] != "hello/world?id=1" {
			t.Fatal("decode mismatch")
		}
	})
}
