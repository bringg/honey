package printers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ohler55/ojg/jp"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

type (
	PrintInput struct {
		Data   Printable
		Format string
	}

	Printable interface {
		Interface() interface{}
		Headers() []string
		Rows() [][]string
	}
)

func Print(i *PrintInput) error {
	formatName := strings.Split(i.Format, "=")
	l := len(formatName)
	if l == 0 {
		return errors.New("format is empty")
	}

	switch formatName[0] {
	case "json":
		out, err := json.Marshal(i.Data.Interface())
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, string(out))
	case "yaml":
		out, err := yaml.Marshal(i.Data.Interface())
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, string(out))
	case "jsonpath":
		if l == 1 || formatName[1] == "" {
			return errors.New("jsonpath format is missing expression")
		}

		expr, err := jp.ParseString(formatName[1])
		if err != nil {
			return err
		}

		out, err := json.Marshal(expr.Get(i.Data.Interface()))
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, string(out))
	case "table":
		if len(i.Data.Rows()) == 0 {
			fmt.Println("no instances found")

			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(i.Data.Headers())
		table.AppendBulk(i.Data.Rows())
		table.Render()
	}

	return nil
}
