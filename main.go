package main

import (
	"encoding/json"
	"errors"
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

func newState(planInput io.Reader) (*tfjson.State, error) {
	parsed := &tfjson.State{}

	dec := json.NewDecoder(planInput)
	dec.DisallowUnknownFields()
	if err := dec.Decode(parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func usage() {
	fmt.Fprint(os.Stderr, "usage: tf2json <plan.json|state.json>")
}

func openPlanOrState(fName string) (interface{}, error) {
	f, err := os.Open(fName)
	if err != nil {
		return nil, fmt.Errorf("could not file file: %w", err)
	}

	plan, err := newPlan(f)
	if err != nil {
		return nil, fmt.Errorf("error making plan: %w", err)
	}

	if plan.PlannedValues != nil && plan.PriorState != nil && plan.Config != nil {
		return plan, nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	state, err := newState(f)
	if err != nil {
		return nil, fmt.Errorf("error making state: %w", err)
	}

	if state.Values != nil {
		return state, nil
	}

	return nil, nil
}

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	inFile := os.Args[1]

	err := realMain(inFile)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func realMain(inFile string) error {
	i, err := openPlanOrState(inFile)
	if err != nil {
		return err
	}

	graph := gographviz.NewEscape()
	graph.SetDir(true)
	graph.SetName("G")
	graph.SetStrict(true)
	graph.AddAttr("G", "rankdir", "LR")
	graph.AddAttr("G", "newrank", "true")
	graph.AddAttr("G", "compoun", "true")

	gv := Graph{graph}

	switch ps := i.(type) {
	case *tfjson.Plan:
		if err := gv.Plan(ps); err != nil {
			return err
		}
	case *tfjson.State:
		if err := gv.State(ps); err != nil {
			return err
		}
	default:
		return errors.New("couldn't detect file type")
	}

	output := graph.String()
	fmt.Println(output)
	return nil
}

func (g *Graph) State(state *tfjson.State) error {
	g.gv.AddSubGraph("G", "state", nil)
	g.Walk("state", state.Values.RootModule)

	return nil
}

func (g *Graph) Plan(plan *tfjson.Plan) error {
	g.gv.AddSubGraph("G", "planned", nil)
	g.gv.AddSubGraph("G", "prior", nil)
	g.Walk("planned", plan.PlannedValues.RootModule)
	g.Walk("prior", plan.PriorState.Values.RootModule)

	return nil
}

type Graph struct {
	gv gographviz.Interface
}

func (g *Graph) Walk(graphName string, m *tfjson.StateModule) string {
	var maddr string
	if m.Address == "" {
		maddr = graphName
	} else {
		maddr = fmt.Sprintf("%s: %s", graphName, m.Address)
	}
	// add the module
	g.gv.AddNode(graphName, maddr, nil)
	for _, r := range m.Resources {
		a, err := address.NewAddress(r.Address)
		if err != nil {
			panic(err)
		}

		label := map[string]string{
			"label": a.ResourceSpec.String(),
			"shape": "box",
		}

		rName := fmt.Sprintf("%s.%s", maddr, a.ResourceSpec.String())

		if a.Mode == address.DataResourceMode {
			label["color"] = "green"
		} else {
			label["color"] = "blue"
		}

		g.gv.AddNode(graphName, rName, label)
		g.gv.AddEdge(maddr, rName, true, nil)
	}
	for _, c := range m.ChildModules {
		p := g.Walk(graphName, c)
		g.gv.AddEdge(maddr, p, true, nil)
	}
	return maddr
}
