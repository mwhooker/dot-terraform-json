package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/awalterschulze/gographviz"
	address "github.com/hashicorp/go-terraform-address"
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

	var walk func(string, *tfjson.StateModule) string

	walk = func(graphName string, m *tfjson.StateModule) string {
		var maddr string
		if m.Address == "" {
			maddr = graphName
		} else {
			maddr = fmt.Sprintf("%s: %s", graphName, m.Address)
		}
		// add the module
		graph.AddNode(graphName, maddr, map[string]string{"color": "blue"})
		for _, r := range m.Resources {
			a, err := address.NewAddress(r.Address)
			if err != nil {
				panic(err)
			}

			label := map[string]string{
				"label": a.ResourceSpec.String(),
			}

			rName := fmt.Sprintf("%s.%s", maddr, a.ResourceSpec.String())

			if a.Mode == address.DataResourceMode {
				label["color"] = "green"
			}

			graph.AddNode(graphName, rName, label)
			graph.AddEdge(maddr, rName, true, nil)
		}
		for _, c := range m.ChildModules {
			p := walk(graphName, c)
			graph.AddEdge(maddr, p, true, nil)
		}
		return maddr
	}

	// TODO: support state output

	graph.AddSubGraph("G", "planned", nil)
	graph.AddSubGraph("G", "prior", nil)
	walk("planned", plan.PlannedValues.RootModule)
	walk("prior", plan.PriorState.Values.RootModule)

	output := graph.String()
	fmt.Println(output)

	return nil
}
