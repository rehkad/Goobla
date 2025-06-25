package errtypes

import (
	"encoding/json"
	"testing"
)

func TestUnknownGooblaKeyJSON(t *testing.T) {
	e := UnknownGooblaKey{Key: "bad"}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Error string `json:"error"`
		Key   string `json:"key"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	if v.Error != UnknownGooblaKeyErrMsg {
		t.Fatalf("unexpected error field: %s", v.Error)
	}
	if v.Key != "bad" {
		t.Fatalf("unexpected key: %s", v.Key)
	}
}
