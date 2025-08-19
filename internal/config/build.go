package config

import (
	"fmt"
	"time"
)

var (
	DistributionBranch  = "dev"
	DistributionEnv     = "local"
	DistributionVersion = "0.0.0"
	DistribubtionDate   = fmt.Sprintf("%s", time.Now().Format("2006.01.02"))
)

func GetVersion() string {
	return fmt.Sprintf("%s-%s-%s", DistributionVersion, DistributionBranch, DistribubtionDate)
}
