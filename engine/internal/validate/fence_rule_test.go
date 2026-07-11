package validate

import (
	"testing"

	"github.com/21StarkCom/stark-bifrost/engine/internal/model"
)

func TestFenceValidationCatchesUnterminated(t *testing.T) {
	a := &model.Artifact{
		Name: "x", Type: model.TypeCommand, Runtimes: model.AllRuntimes(),
		Body: "<!-- runtime: claude -->\noops\n",
	}
	r := &Result{}
	checkFences(r, "demo/command/x", a)
	if !r.HasErrors() {
		t.Fatal("expected unterminated-fence error")
	}
}
