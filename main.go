package main

import (
	"github.com/shareed2k/honey/cmd"

	_ "github.com/shareed2k/honey/pkg/backend/all" // import all backends
)

func main() {
	cmd.Execute()
}
