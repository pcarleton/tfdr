package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

type Resource struct {
	Path []string
	Key  string
	Diff *terraform.InstanceDiff
}

type Candidates struct {
	Created   []Resource
	Destroyed []Resource
}

func (r *Resource) String() string {
	if len(r.Path) == 1 {
		return r.Key
	}
	return fmt.Sprintf("module.%s.%s", strings.Join(r.Path[1:], "."), r.Key)
}

type Pair struct {
	Old   Resource
	New   Resource
	State *terraform.ResourceState
}

func FmtError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func PickCandidates(plan *terraform.Plan) Candidates {
	candidates := Candidates{}
	// Find things being destroyed
	for _, moduleDiff := range plan.Diff.Modules {
		for key, r := range moduleDiff.Resources {
			if r.ChangeType() == terraform.DiffDestroy {
				candidates.Destroyed = append(candidates.Destroyed, Resource{moduleDiff.Path, key, r})
			}

			if r.ChangeType() == terraform.DiffCreate {
				candidates.Created = append(candidates.Created, Resource{moduleDiff.Path, key, r})
			}
		}
	}
	return candidates
}

func MatchPairs(plan *terraform.Plan, candidates Candidates) []Pair {
	var pairs []Pair
	// For the created things, find the ID of them
	for _, destroyed := range candidates.Destroyed {
		module := plan.State.ModuleByPath(destroyed.Path)
		for name, r := range module.Resources {
			if name != destroyed.Key {
				continue
			}

			// Figure out which attribute is the key
			id := r.Primary.ID
			idKey := ""
			for key, value := range r.Primary.Attributes {
				if key == "id" {
					continue
				}
				if value == id {
					idKey = key
					break
				}
			}

			if idKey == "" {
				FmtError("Couldn't find ID for %s (id value: %s)\n", destroyed.String(), id)
				continue
			}

			// Find the created resource that has the same ID
			found := false
		loop:
			for _, created := range candidates.Created {
				for attrKey, attrDiff := range created.Diff.Attributes {
					// fmt.Println(attrKey)
					if attrKey == idKey && attrDiff.New == id {
						pairs = append(pairs, Pair{Old: destroyed, New: created, State: r})
						found = true
						break loop
					}
				}
			}
			if !found {
				FmtError("Couldn't find matching resource for %s (id: %s idkey: %s)", destroyed.String(), id, idKey)
			}
		}
	}
	return pairs
}
