package approver

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/cache"
	"time"
)

type Approval struct {
	Spec approvals.ApprovalSpec
}

func (a *Approval) Create() error {
	cacheInstance := cache.Get()
	cacheKey := CreateApprovalCacheKey(a.Spec.Id)
	cacheData, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal approval[%s]: %s", a.Spec.Id, err)
	}
	expiryDuration := time.Duration(24 * time.Hour)
	if err := cacheInstance.Set(cacheKey, string(cacheData), expiryDuration); err != nil {
		return fmt.Errorf("failed to set cache with key[%s] for approval[%s]: %s", cacheKey, a.Spec.Id, err)
	}
	return nil
}

func (a *Approval) Exists() bool {
	cacheInstance := cache.Get()
	cacheKey := CreateApprovalCacheKey(a.Spec.Id)
	if _, err := cacheInstance.Get(cacheKey); err != nil {
		return false
	}
	return true
}

func (a *Approval) Update() error {
	if isExists := a.Exists(); !isExists {
		return fmt.Errorf("failed to find an existing approval[%s]", a.Spec.Id)
	}
	if err := a.Create(); err != nil {
		return fmt.Errorf("failed to create approval[%s]: %s", a.Spec.Id, err)
	}
	return nil
}

func (a *Approval) Load() error {
	cacheInstance := cache.Get()
	cacheKey := CreateApprovalCacheKey(a.Spec.Id)
	value, err := cacheInstance.Get(cacheKey)
	if err != nil {
		return fmt.Errorf("failed to get approval[%s] with key[%s]: %s", a.Spec.Id, cacheKey, err)
	}
	if err := json.Unmarshal([]byte(value), a); err != nil {
		return fmt.Errorf("failed to unmarshal approval[%s]: %s", a.Spec.Id, err)
	}
	return nil
}
