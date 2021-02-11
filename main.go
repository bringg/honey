package main

import (
	"github.com/bringg/honey/cmd"

	_ "github.com/bringg/honey/pkg/backend/all" // import all backends
)

func main() {
	cmd.Execute()
}
