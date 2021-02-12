package printers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/olekukonko/tablewriter"
	"github.com/rclone/rclone/fs"
	"github.com/tidwall/pretty"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/jsonpath"

	"github.com/bringg/honey/pkg/place"
)

type (
	PrintInput struct {
		Data    Printable
		Format  string
		NoColor bool
	}

	Printable interface {
		FlattenData() (*place.FlattenData, error)
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

	flattenData, err := i.Data.FlattenData()
	if err != nil {
		return err
	}

	cleanedData, err := flattenData.Filter(headers)
	if err != nil {
		return err
	}

	var out []byte
	switch format[0] {
	case "json":
		out, err = jsoniter.Marshal(cleanedData)
		if err != nil {
			return err
		}

		out = pretty.Pretty(out)
		if !i.NoColor {
			out = pretty.Color(out, nil)
		}
	case "yaml":
		out, err = yaml.Marshal(cleanedData)
		if err != nil {
			return err
		}
	case "jsonpath":
		if l == 1 || format[1] == "" {
			return errors.New("jsonpath expression is missing")
		}

		data, err := flattenData.ToArrayMap()
		if err != nil {
			return err
		}

		jp := jsonpath.New("honey")
		if err := jp.Parse(format[1]); err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := jp.Execute(buf, data); err != nil {
			return err
		}

		out = buf.Bytes()
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

		return nil
	}

	fmt.Fprint(os.Stdout, string(out))

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
