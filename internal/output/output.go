package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/WRAllen/kctx/internal/kube"
)

func Table(writer io.Writer, contexts []kube.Context, aliasName string, color bool) error {
	contextWidth := utf8.RuneCountInString("CONTEXT")
	filepathWidth := utf8.RuneCountInString("FILEPATH")
	aliasWidth := utf8.RuneCountInString(aliasName)

	for _, context := range contexts {
		contextWidth = max(contextWidth, utf8.RuneCountInString(context.Name))
		filepathWidth = max(filepathWidth, utf8.RuneCountInString(context.Filepath))
		aliasWidth = max(aliasWidth, utf8.RuneCountInString(context.Alias))
	}

	headerColor, resetColor := "", ""
	if color {
		headerColor = "\x1b[1;36m"
		resetColor = "\x1b[0m"
	}

	if _, err := fmt.Fprintf(writer, "%s%-*s  %-*s  %s%s\n", headerColor, contextWidth, "CONTEXT", filepathWidth, "FILEPATH", aliasName, resetColor); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s  %s  %s\n", strings.Repeat("-", contextWidth), strings.Repeat("-", filepathWidth), strings.Repeat("-", aliasWidth)); err != nil {
		return err
	}
	for _, context := range contexts {
		if _, err := fmt.Fprintf(writer, "%-*s  %-*s  %s\n", contextWidth, context.Name, filepathWidth, context.Filepath, context.Alias); err != nil {
			return err
		}
	}
	return nil
}

type jsonContext struct {
	Context  string    `json:"context"`
	Filepath string    `json:"filepath"`
	Alias    jsonAlias `json:"alias"`
}

type jsonAlias struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func JSON(writer io.Writer, contexts []kube.Context, aliasName string) error {
	values := make([]jsonContext, 0, len(contexts))
	for _, context := range contexts {
		values = append(values, jsonContext{
			Context:  context.Name,
			Filepath: context.Filepath,
			Alias:    jsonAlias{Name: aliasName, Value: context.Alias},
		})
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(values)
}
