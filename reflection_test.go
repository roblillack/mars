package mars

import (
	"reflect"
	"testing"
)

type T struct{}

func (t *T) Hello() {}

func TestFindMethod(t *testing.T) {
	for name, tv := range map[string]struct {
		reflect.Type
		reflect.Value
	}{
		"Hello":  {reflect.TypeOf(&T{}), reflect.ValueOf((*T).Hello)},
		"Helper": {reflect.TypeOf(t), reflect.ValueOf((*testing.T).Helper)},
		"":       {reflect.TypeOf(t), reflect.ValueOf((reflect.Type).Comparable)},
	} {
		m := findMethod(tv.Type, tv.Value)
		if name == "" {
			if m != nil {
				t.Errorf("method found that shouldn't be here: %v", m)
			}
			continue
		}
		if m == nil {
			t.Errorf("No method found when looking for %s", name)
			continue
		}
		if m.Name != name {
			t.Errorf("Expected method %s, got %s: %v", name, m.Name, m)
		}
	}
}
