package cluster

import (
	"reflect"
	"testing"
)

func assertDeepEqual(t *testing.T, v1, v2 interface{}) {
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("Values don't match: %+v, %+v", v1, v2)
	}
}

func wrapJoin(body interface{}) interface{} {
	return map[string]interface{}{
		"Fn::Join": []interface{}{"", body},
	}
}

func TestRenderTemplate(t *testing.T) {
	assertDeepEqual(t, renderTemplate(""), wrapJoin([]interface{}{""}))
	assertDeepEqual(t, renderTemplate("test"), wrapJoin([]interface{}{"test"}))
	assertDeepEqual(t, renderTemplate(" one two "), wrapJoin([]interface{}{" one two "}))

	assertDeepEqual(t, renderTemplate("{{ test }}"), wrapJoin([]interface{}{
		"",
		map[string]interface{}{"Ref": "test"},
		"",
	}))

	assertDeepEqual(t, renderTemplate("{{test}}"), wrapJoin([]interface{}{
		"",
		map[string]interface{}{"Ref": "test"},
		"",
	}))

	assertDeepEqual(t, renderTemplate(" one{{ two }} three "), wrapJoin([]interface{}{
		" one",
		map[string]interface{}{"Ref": "two"},
		" three ",
	}))

	assertDeepEqual(t, renderTemplate(" one {{ two|base64 }}three "), wrapJoin([]interface{}{
		" one ",
		map[string]interface{}{
			"Fn::Base64": map[string]interface{}{"Ref": "two"},
		},
		"three ",
	}))
}
