package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/WRAllen/kctx/internal/kube"
)

func TestTable(t *testing.T) {
	var output bytes.Buffer
	contexts := []kube.Context{{Name: "a800", Filepath: "/tmp/a800.yaml", Alias: "sco-dev-sh01-bmsk8s"}}
	if err := Table(&output, contexts, "AMS_ENV", false); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"CONTEXT", "FILEPATH", "AMS_ENV", "a800", "/tmp/a800.yaml", "sco-dev-sh01-bmsk8s"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("table output does not contain %q:\n%s", expected, output.String())
		}
	}
}
