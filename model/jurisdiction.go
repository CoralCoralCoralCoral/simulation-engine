package model

import "github.com/CoralCoralCoralCoral/simulation-engine/geo"

type Jurisdiction struct {
	id     string
	parent *Jurisdiction
	policy *Policy
}

func (jur *Jurisdiction) Parent() *Jurisdiction {
	return jur.parent
}

func newJurisdiction(id string, parent *Jurisdiction) *Jurisdiction {
	jur := Jurisdiction{
		id:     id,
		parent: parent,
	}

	return &jur
}

func jurisdictionsFromFeatures() []*Jurisdiction {
	features := geo.LoadFeatures()

	// allocate array of length feature length + 1 (to also contain the GLOBAL jurisdiction)
	jurisdictions := make([]*Jurisdiction, 0, len(features)+1)

	// create jurisdictions
	for _, feature := range features {
		jurisdictions = append(jurisdictions, newJurisdiction(feature.Code(), nil))
	}

	// assign parents
	for _, feature := range features {
		if parent_code := feature.ParentCode(); parent_code != "" {
			jur := findJurisdiction(jurisdictions, func(val *Jurisdiction) bool {
				return val.Id() == feature.Code()
			})

			// find parent_jurisdiction
			parent_jur := findJurisdiction(jurisdictions, func(val *Jurisdiction) bool {
				return val.Id() == parent_code
			})

			if jur != nil && parent_jur != nil {
				jur.parent = parent_jur
			}
		}
	}

	// assign the highest level jurisdictions (orphan jurisdictions to the GLOBAL jurisdiction)
	global_jur := newJurisdiction("GLOBAL", nil)
	for _, jur := range jurisdictions {
		if jur.parent == nil {
			jur.parent = global_jur
		}
	}

	jurisdictions = append(jurisdictions, global_jur)

	return jurisdictions
}

func sampleJurisdiction(jurisdictions []*Jurisdiction, msoa_sampler *geo.MSOASampler) *Jurisdiction {
	msoa := msoa_sampler.Sample()

	jur := findJurisdiction(jurisdictions, func(val *Jurisdiction) bool {
		return val.Id() == msoa.GISCode
	})

	return jur
}

func findJurisdiction(jurisdictions []*Jurisdiction, predicate func(value *Jurisdiction) bool) *Jurisdiction {
	for _, value := range jurisdictions {
		if predicate(value) {
			return value
		}
	}

	return nil
}

func (jur *Jurisdiction) Id() string {
	return jur.id
}

func (jur *Jurisdiction) applyPolicy(policy *Policy) {
	jur.policy = policy
}

func (jur *Jurisdiction) resolvePolicy() (policy *Policy) {
	if jur.parent != nil {
		policy = jur.parent.resolvePolicy()
	}

	if policy == nil {
		policy = jur.policy
	}

	return
}
