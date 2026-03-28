package main

import (
	"context"

	"github.com/andrewhowdencom/dux/internal/cli"
)

func main() {
	cli.Execute(context.Background())
}
