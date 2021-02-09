package printers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/rclone/rclone/fs"
	"github.com/shareed2k/honey/pkg/place"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/jsonpath"
)

type (
	PrintInput struct {
		Data   Printable
		Format string
	}

	Printable interface {
		FlattenData() (place.FlattenData, error)
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

	// set headers
	headers := i.Data.Headers()
	if IsHeaderble(format[0]) && l > 1 && format[1] != "" {
		h := fs.CommaSepList{}
		h.Set(format[1])
		if len(h) > 0 {
			headers = h
		}
	}

	data, err := i.Data.FlattenData()
	if err != nil {
		return err
	}

	cleanedData := data.Filter(headers)

	switch format[0] {
	case "json":
		out, err := json.Marshal(cleanedData)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, string(out))
	case "yaml":
		out, err := yaml.Marshal(cleanedData)
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
		if err := jp.Execute(buf, data); err != nil {
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
		table.SetHeader(headers)
		table.AppendBulk(rows)
		table.Render()
	}

	return nil
}

// IsHeaderble _
// table not supported yet
func IsHeaderble(format string) bool {
	if format == "json" || format == "yaml" {
		return true
	}

	return false
}
