# dot-terraform-json

Outputs Graphviz DOT language script representing the prior and proposed state in a Terraform json plan.

Creates two disjoint graphs, a Prior and a Proposed.
Modules are outlined in blue, data resources in green, and managed resources in black.

## Example:

    # Generate pdf and open it (Mac)
    ./dot-terraform-json plan.json | dot -Tpdf > dot.pdf && open dot.pdf
