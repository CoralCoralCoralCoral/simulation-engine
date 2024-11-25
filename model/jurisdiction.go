package model

import "github.com/google/uuid"

type Jurisdiction struct {
	id       uuid.UUID
	name     string
	parent   *Jurisdiction
	children []*Jurisdiction
	policy   *Policy
}

func NewJurisdiction(name string, parent *Jurisdiction) *Jurisdiction {
	jur := Jurisdiction{
		id:       uuid.New(),
		name:     name,
		parent:   parent,
		children: make([]*Jurisdiction, 0),
	}

	parent.addChild(&jur)

	return &jur
}

func (jur *Jurisdiction) addChild(child *Jurisdiction) {
	for _, existing_child := range jur.children {
		if existing_child.id == child.id {
			return
		}
	}

	jur.children = append(jur.children, child)
}

func (jur *Jurisdiction) applyPolicy(policy *Policy) {
	jur.policy = policy
}

func (jur *Jurisdiction) resolvePolicy() *Policy {
	current := jur
	for {
		if current.policy != nil {
			return current.policy
		}

		if current.parent == nil {
			break
		}

		current = current.parent
	}

	return nil
}
