package model

type Policy struct {
	IsMaskMandate bool `json:"is_mask_mandate"`
	IsLockDown    bool `json:"is_lockdown"`
}
