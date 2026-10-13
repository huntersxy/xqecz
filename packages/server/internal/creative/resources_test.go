package creative

import (
	"testing"
)

func TestTemplateExpansionAndValidation(t *testing.T) {
	row := Resource{Kind: "template", Title: "Scene", Slug: "scene", Status: "published", Payload: `{"body":"保持{角色}，位于{场景}。","variables":[{"name":"角色","type":"text","required":true},{"name":"场景","type":"select","options":["学校","公园"],"default":"学校"}]}`}
	p, err := Validate(row)
	if err != nil {
		t.Fatal(err)
	}
	text, err := Expand(p, map[string]string{"角色": "小希"})
	if err != nil || text != "保持小希，位于学校。" {
		t.Fatal(text, err)
	}
	if _, err = Expand(p, map[string]string{"角色": "小希", "场景": "bad"}); err == nil {
		t.Fatal("invalid option accepted")
	}
	if _, err = Expand(p, map[string]string{}); err == nil {
		t.Fatal("required variable ignored")
	}
	text, err = Expand(p, map[string]string{"角色": "{场景}"})
	if err != nil || text != "保持{场景}，位于学校。" {
		t.Fatal("variable values must remain literal", text, err)
	}
	row.Payload = `{"body":"{unknown}"}`
	if _, err = Validate(row); err == nil {
		t.Fatal("unknown variable accepted")
	}
}
