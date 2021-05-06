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
	graph.AddAttr("G", "rankdir", "LR")

	// walkState := func (graphName string, sv *tfjson.StateValues) *gographviz.SubGraph {
	// 	s := gographviz.NewSubGraph(graphName)
	// }

	// plan
	// g := walkState("plan", plan.PriorState)
	var walk func(string, *tfjson.StateModule) string

	walk = func(graphName string, m *tfjson.StateModule) string {
		var maddr string
		if m.Address == "" {
			maddr = graphName
		} else {
			maddr = fmt.Sprintf("%s: %s", graphName, m.Address)
		}
		foundNull := false
		for _, r := range m.Resources {
			rName := fmt.Sprintf("%s: %s.%s", graphName, r.Type, r.Name)

			if r.Type == "null_resource" {
				graph.AddNode(graphName, rName, map[string]string{"color": "blue"})
				graph.AddEdge(maddr, rName, true, nil)
				foundNull = true
			}
		}
		if maddr == graphName {
			graph.AddNode(graphName, maddr, nil)
		} else if foundNull {
			graph.AddNode(graphName, maddr, map[string]string{"color": "green"})
		} else {
			graph.AddNode(graphName, maddr, map[string]string{"color": "red"})
		}
		for _, c := range m.ChildModules {
			p := walk(graphName, c)
			graph.AddEdge(maddr, p, true, nil)
		}
		return maddr
	}

	graph.AddSubGraph("G", "planned", nil)
	graph.AddSubGraph("G", "prior", nil)
	// walk("planned", plan.PlannedValues.RootModule)
	walk("prior", plan.PriorState.Values.RootModule)

	output := graph.String()
	fmt.Println(output)

	return nil
}
