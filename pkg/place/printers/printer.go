package printers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/jsonpath"
)

type (
	PrintInput struct {
		Data   Printable
		Format string
	}

	Printable interface {
		MarshalJSON() ([]byte, error)
		Interface() interface{}
		Headers() []string
		Rows() [][]string
	}
)

func Print(i *PrintInput) error {
	format := strings.Split(i.Format, "=")
	l := len(format)
	if l == 0 {
		return errors.New("format is empty")
	}

	switch format[0] {
	case "json":
		out, err := i.Data.MarshalJSON()
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
		if l == 1 || format[1] == "" {
			return errors.New("jsonpath expression is missing")
		}

		jp := jsonpath.New("honey")
		if err := jp.Parse(format[1]); err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := jp.Execute(buf, i.Data.Interface()); err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, buf.String())
	case "table":
		rows := i.Data.Rows()
		if len(rows) == 0 {
			fmt.Println("no rows found")

			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(i.Data.Headers())
		table.AppendBulk(rows)
		table.Render()
	}

	return nil
}
