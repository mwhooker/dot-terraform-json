package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/awalterschulze/gographviz"
	tfjson "github.com/hashicorp/terraform-json"
)

func newPlan(planInput io.Reader) (*tfjson.Plan, error) {
	parsed := &tfjson.Plan{}

	dec := json.NewDecoder(planInput)
	dec.DisallowUnknownFields()
	if err := dec.Decode(parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func usage() {
	fmt.Fprint(os.Stderr, "usage: tf2json <plan.json>")
}

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	planFile := os.Args[1]

	err := realMain(planFile)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain(planFile string) error {

	f, err := os.Open(planFile)

	plan, err := newPlan(f)
	if err != nil {
		return err
	}

	graph := gographviz.NewEscape()
	graph.SetDir(true)
	graph.SetName("G")
	graph.SetStrict(true)

	// walkState := func (graphName string, sv *tfjson.StateValues) *gographviz.SubGraph {
	// 	s := gographviz.NewSubGraph(graphName)
	// }

	// plan
	// g := walkState("plan", plan.PriorState)
	var walk func(*tfjson.StateModule) string

	walk = func(m *tfjson.StateModule) string {
		maddr := m.Address
		if maddr == "" {
			maddr = "[root]"
		}
		graph.AddNode("G", maddr, nil)
		for _, r := range m.Resources {
			// name := fmt.Sprintf("%s.%s", r.Type, r.Name)
			// if r.Index != nil {
			// 	name = fmt.Sprintf("%s[%s]", name, r.Index)
			// }
			name := r.Address

			graph.AddNode("G", name, nil)
			graph.AddEdge(maddr, name, true, nil)
		}
		for _, c := range m.ChildModules {
			p := walk(c)
			graph.AddEdge(maddr, p, true, nil)
		}
		return maddr
	}

	// walk(plan.PriorState.Values.RootModule)
	walk(plan.PlannedValues.RootModule)

	output := graph.String()
	fmt.Println(output)

	return nil
}
