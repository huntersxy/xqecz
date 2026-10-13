package contenttags

import (
	"reflect"
	"testing"
)

func TestManualMarkerOwnership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   []string
		checked bool
		want    []string
	}{
		{"append", []string{"校园", "小泉"}, true, []string{"校园", "小泉", "AIGC"}},
		{"deduplicate", []string{"校园", "aigc", "#AIGC", "ＡＩＧＣ"}, true, []string{"校园", "AIGC"}},
		{"manual survives uncheck", []string{"校园", "aIgC"}, false, []string{"校园", "AIGC"}},
		{"no declaration", []string{"校园"}, false, []string{"校园"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]string{}, tc.input...)
			got := WithAIGC(tc.input, tc.checked)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(tc.input, before) {
				t.Fatalf("%v input=%v", got, tc.input)
			}
		})
	}
}

func TestModelCannotAddOrRemoveUserDeclaration(t *testing.T) {
	if got := MergeGenerated([]string{"校园", "aigc", "#AIGC"}, []string{"待生成标签"}); !reflect.DeepEqual(got, []string{"校园"}) {
		t.Fatal(got)
	}
	if got := MergeGenerated([]string{"校园"}, []string{"待生成标签", "AIGC"}); !reflect.DeepEqual(got, []string{"校园", "AIGC"}) {
		t.Fatal(got)
	}
}
