/*
Copyright 2021 The tKeel Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mapper

import (
	"testing"

	"github.com/tkeel-io/core/pkg/constraint"
)

func TestMapper(t *testing.T) {
	input := map[string]constraint.Node{
		"entity1.property1":      constraint.NewNode(123),
		"entity2.property2.name": constraint.NewNode("tomas"),
		"entity2.property3":      constraint.NewNode(123),
	}

	tqlTexts := []struct {
		id       string
		tqlText  string
		input    map[string]constraint.Node
		computed bool
	}{
		{"tql1", "insert into test123 select test234.temp as temp", map[string]constraint.Node{"test234.temp": constraint.NewNode(`123`)}, true},
		{"tql2", `insert into entity3 select entity1.property1 as property1, entity2.property2.name as property2, entity1.property1 + entity2.property3 as property3`, input, true},
		{"tql3", "insert into sub123 select test123.temp", nil, false},
	}

	for _, tqlInst := range tqlTexts {
		t.Run(tqlInst.id, func(t *testing.T) {
			m, _ := NewMapper(tqlInst.id, tqlInst.tqlText)

			t.Log("parse ID: ", m.ID())

			tentacles := m.Tentacles()
			t.Logf("parse tentacles, count %d.", len(tentacles))
			for index, tentacle := range tentacles {
				t.Logf("tentacle.%d, type: %s, target: %s, items: %s.",
					index, tentacle.Type(), tentacle.TargetID(), tentacle.Items())
			}

			t.Log("parse target entity: ", m.TargetEntity())
			t.Log("parse source entities: ", m.SourceEntities())

			m.Copy()

			if tqlInst.computed {
				out, err := m.Exec(tqlInst.input)
				t.Logf("exec input: %v\n output: %v\n error: %v", tqlInst.input, out, err)
			}
		})
	}
}

func TestExec(t *testing.T) {
	tqlString := `insert into 7ffed0dc-3ed5-4137-9c16-a2c9c74e0bf6 select f8f0327b-51e4-400a-a2e1-c95e371ec99d.path  + '/' + '7ffed0dc-3ed5-4137-9c16-a2c9c74e0bf6' as path`

	mInstance, err := NewMapper("mapper123", tqlString)

	t.Log(err)

	t.Log("target: ", mInstance.TargetEntity())
	t.Log("sources: ", mInstance.SourceEntities())
	for _, tentacle := range mInstance.Tentacles() {
		t.Log("tentacle: ", tentacle)
	}

	result, err := mInstance.Exec(map[string]constraint.Node{
		"f8f0327b-51e4-400a-a2e1-c95e371ec99d.path": constraint.NewNode("test"),
		"entity.property2.name":                     constraint.NewNode("123"),
		"entity.property3":                          constraint.NewNode("g123"),
	})

	t.Log(err)
	t.Log(result)
}

func TestExec2(t *testing.T) {
	tqlString := `insert into bc90e5ba-4d15-4738-bf38-fdfe16740d9c select 0074c68f-679c-4290-a2be-3878c8fb75f6.sysField._spacePath + '/bc90e5ba-4d15-4738-bf38-fdfe16740d9c'  as sysField._spacePath`

	mInstance, err := NewMapper("mapper123", tqlString)

	t.Log(err)

	t.Log("target: ", mInstance.TargetEntity())
	t.Log("sources: ", mInstance.SourceEntities())
	for _, tentacle := range mInstance.Tentacles() {
		t.Log("tentacle: ", tentacle)
	}

	result, err := mInstance.Exec(map[string]constraint.Node{
		"0074c68f-679c-4290-a2be-3878c8fb75f6.sysField._spacePath": constraint.NewNode("test"),
		"entity.property2.name": constraint.NewNode("123"),
		"entity.property3":      constraint.NewNode("g123"),
	})

	t.Log(err)
	t.Log(result)
}

func TestExec3(t *testing.T) {
	tqlString := `insert into 43ce9690-7c5d-4e62-be07-fb16a13f67d0 select 0074c68f-679c-4290-a2be-3878c8fb75f6.sysField._spacePath + '/43ce9690-7c5d-4e62-be07-fb16a13f67d0'  as sysField._spacePath`

	mInstance, err := NewMapper("mapper123", tqlString)

	t.Log(err)

	t.Log("target: ", mInstance.TargetEntity())
	t.Log("sources: ", mInstance.SourceEntities())
	for _, tentacle := range mInstance.Tentacles() {
		t.Log("tentacle: ", tentacle)
	}

	result, err := mInstance.Exec(map[string]constraint.Node{
		"0074c68f-679c-4290-a2be-3878c8fb75f6.sysField._spacePath": constraint.NewNode("test"),
		"entity.property2.name": constraint.NewNode("123"),
		"entity.property3":      constraint.NewNode("g123"),
	})

	t.Log(err)
	t.Log(result)
}
