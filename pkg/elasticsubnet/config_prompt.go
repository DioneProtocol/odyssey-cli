// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package elasticsubnet

import (
	"fmt"
	"math"
	"time"

	"github.com/DioneProtocol/odyssey-cli/pkg/application"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/prompts"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/vms/omegavm/reward"
)

// default elastic config parameter values are from
const (
	defaultInitialSupply                        = 240_000_000
	defaultMaximumSupply                        = 720_000_000
	defaultMinConsumptionRate                   = 0.1
	defaultMaxConsumptionRate                   = 0.12
	defaultMinValidatorStake                    = 2_000
	defaultMaxValidatorStake                    = 3_000_000
	defaultMinValidatorStakeDurationHours       = 24
	defaultMinValidatorStakeDurationHoursString = "24"
	defaultMaxValidatorStakeDurationHours       = 365 * 24
	defaultMaxValidatorStakeDurationHoursString = "365 x 24"
	defaultMinDelegatorStakeDurationHours       = 24
	defaultMinDelegatorStakeDurationHoursString = "24"
	defaultMaxDelegatorStakeDurationHours       = 365 * 24
	defaultMaxDelegatorStakeDurationHoursString = "365 x 24"
	defaultMinValidatorStakeDuration            = defaultMinValidatorStakeDurationHours * time.Hour
	defaultMaxValidatorStakeDuration            = defaultMaxValidatorStakeDurationHours * time.Hour
	defaultMinDelegatorStakeDuration            = defaultMinDelegatorStakeDurationHours * time.Hour
	defaultMaxDelegatorStakeDuration            = defaultMaxDelegatorStakeDurationHours * time.Hour
	defaultMinDelegationFee                     = 20_000
	defaultMinDelegatorStake                    = 25
	defaultMaxValidatorWeightFactor             = 5
	defaultUptimeRequirement                    = 0.8
)

func GetElasticSubnetConfig(app *application.Odyssey, tokenSymbol string, useDefaultConfig bool) (models.ElasticSubnetConfig, error) {
	const (
		defaultConfig   = "Use default elastic subnet config"
		customizeConfig = "Customize elastic subnet config"
	)
	elasticSubnetConfig := models.ElasticSubnetConfig{
		InitialSupply:             defaultInitialSupply,
		MaxSupply:                 defaultMaximumSupply,
		MinConsumptionRate:        defaultMinConsumptionRate * reward.PercentDenominator,
		MaxConsumptionRate:        defaultMaxConsumptionRate * reward.PercentDenominator,
		MinValidatorStake:         defaultMinValidatorStake,
		MaxValidatorStake:         defaultMaxValidatorStake,
		MinValidatorStakeDuration: defaultMinValidatorStakeDuration,
		MaxValidatorStakeDuration: defaultMaxValidatorStakeDuration,
		MinDelegatorStakeDuration: defaultMinDelegatorStakeDuration,
		MaxDelegatorStakeDuration: defaultMaxDelegatorStakeDuration,
		MinDelegationFee:          defaultMinDelegationFee,
		MinDelegatorStake:         defaultMinDelegatorStake,
		MaxValidatorWeightFactor:  defaultMaxValidatorWeightFactor,
		UptimeRequirement:         defaultUptimeRequirement * reward.PercentDenominator,
	}
	if useDefaultConfig {
		return elasticSubnetConfig, nil
	}
	elasticSubnetConfigOptions := []string{defaultConfig, customizeConfig}
	chosenConfig, err := app.Prompt.CaptureList(
		"How would you like to set fees",
		elasticSubnetConfigOptions,
	)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}

	if chosenConfig == defaultConfig {
		return elasticSubnetConfig, nil
	}
	customElasticSubnetConfig, err := getCustomElasticSubnetConfig(app, tokenSymbol)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	return customElasticSubnetConfig, nil
}

func getCustomElasticSubnetConfig(app *application.Odyssey, tokenSymbol string) (models.ElasticSubnetConfig, error) {
	initialSupply, err := getInitialSupply(app, tokenSymbol)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	maxSupply, err := getMaximumSupply(app, tokenSymbol, initialSupply)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minConsumptionRate, maxConsumptionRate, err := getConsumptionRate(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minValidatorStake, maxValidatorStake, err := getValidatorStake(app, initialSupply, maxSupply)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minValidatorStakeDuration, maxValidatorStakeDuration, err := getValidatorStakeDuration(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minDelegatorStakeDuration, maxDelegatorStakeDuration, err := getDelegatorStakeDuration(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minDelegationFee, err := getMinDelegationFee(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	minDelegatorStake, err := getMinDelegatorStake(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	maxValidatorWeightFactor, err := getMaxValidatorWeightFactor(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}
	uptimeReq, err := getUptimeRequirement(app)
	if err != nil {
		return models.ElasticSubnetConfig{}, err
	}

	elasticSubnetConfig := models.ElasticSubnetConfig{
		InitialSupply:             initialSupply,
		MaxSupply:                 maxSupply,
		MinConsumptionRate:        minConsumptionRate,
		MaxConsumptionRate:        maxConsumptionRate,
		MinValidatorStake:         minValidatorStake,
		MaxValidatorStake:         maxValidatorStake,
		MinValidatorStakeDuration: minValidatorStakeDuration,
		MaxValidatorStakeDuration: maxValidatorStakeDuration,
		MinDelegatorStakeDuration: minDelegatorStakeDuration,
		MaxDelegatorStakeDuration: maxDelegatorStakeDuration,
		MinDelegationFee:          minDelegationFee,
		MinDelegatorStake:         minDelegatorStake,
		MaxValidatorWeightFactor:  maxValidatorWeightFactor,
		UptimeRequirement:         uptimeReq,
	}
	return elasticSubnetConfig, err
}

func getInitialSupply(app *application.Odyssey, tokenName string) (uint64, error) {
	ux.Logger.PrintToUser(fmt.Sprintf("Select the Initial Supply of %s. \"_\" can be used as thousand separator", tokenName))
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Initial Supply is %s", ux.ConvertToStringWithThousandSeparator(defaultInitialSupply)))
	initialSupply, err := app.Prompt.CaptureUint64("Initial Supply amount")
	if err != nil {
		return 0, err
	}
	return initialSupply, nil
}

func getMaximumSupply(app *application.Odyssey, tokenName string, initialSupply uint64) (uint64, error) {
	ux.Logger.PrintToUser(fmt.Sprintf("Select the Maximum Supply of %s. \"_\" can be used as thousand separator", tokenName))
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Supply is %s", ux.ConvertToStringWithThousandSeparator(defaultMaximumSupply)))
	maxSupply, err := app.Prompt.CaptureUint64Compare(
		"Maximum Supply amount",
		[]prompts.Comparator{
			{
				Label: "Initial Supply",
				Type:  prompts.MoreThanEq,
				Value: initialSupply,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	return maxSupply, nil
}

func getConsumptionRate(app *application.Odyssey) (uint64, uint64, error) {
	ux.Logger.PrintToUser("Select the Minimum Consumption Rate. Please denominate your percentage in PercentDenominator")
	ux.Logger.PrintToUser("To denominate your percentage in PercentDenominator just multiply it by 10_000. For example, 1 percent corresponds to 10_000")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Consumption Rate is %s", ux.ConvertToStringWithThousandSeparator(uint64(defaultMinConsumptionRate*reward.PercentDenominator))))
	minConsumptionRate, err := app.Prompt.CaptureUint64Compare(
		"Minimum Consumption Rate",
		[]prompts.Comparator{
			{
				Label: "Percent Denominator(1_0000_0000)",
				Type:  prompts.LessThanEq,
				Value: reward.PercentDenominator,
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	ux.Logger.PrintToUser("Select the Maximum Consumption Rate. Please denominate your percentage in PercentDenominator")
	ux.Logger.PrintToUser("To denominate your percentage in PercentDenominator just multiply it by 10_000. For example, 1 percent corresponds to 10_000")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Consumption Rate is %s", ux.ConvertToStringWithThousandSeparator(uint64(defaultMaxConsumptionRate*reward.PercentDenominator))))
	maxConsumptionRate, err := app.Prompt.CaptureUint64Compare(
		"Maximum Consumption Rate",
		[]prompts.Comparator{
			{
				Label: "Percent Denominator(1_0000_0000)",
				Type:  prompts.LessThanEq,
				Value: reward.PercentDenominator,
			},
			{
				Label: "Minimum Consumption Rate",
				Type:  prompts.MoreThanEq,
				Value: minConsumptionRate,
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}
	return minConsumptionRate, maxConsumptionRate, nil
}

func getValidatorStake(app *application.Odyssey, initialSupply uint64, maximumSupply uint64) (uint64, uint64, error) {
	ux.Logger.PrintToUser("Select the Minimum Validator Stake. \"_\" can be used as thousand separator")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Validator Stake is %s", ux.ConvertToStringWithThousandSeparator(defaultMinValidatorStake)))
	minValidatorStake, err := app.Prompt.CaptureUint64Compare(
		"Minimum Validator Stake",
		[]prompts.Comparator{
			{
				Label: "Initial Supply",
				Type:  prompts.LessThanEq,
				Value: initialSupply,
			},
			{
				Label: "0",
				Type:  prompts.MoreThan,
				Value: 0,
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	ux.Logger.PrintToUser("Select the Maximum Validator Stake. \"_\" can be used as thousand separator")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Validator Stake is %s", ux.ConvertToStringWithThousandSeparator(defaultMaxValidatorStake)))
	maxValidatorStake, err := app.Prompt.CaptureUint64Compare(
		"Maximum Validator Stake",
		[]prompts.Comparator{
			{
				Label: "Maximum Supply",
				Type:  prompts.LessThanEq,
				Value: maximumSupply,
			},
			{
				Label: "Minimum Validator Stake",
				Type:  prompts.MoreThan,
				Value: minValidatorStake,
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}
	return minValidatorStake, maxValidatorStake, nil
}

func getValidatorStakeDuration(app *application.Odyssey) (time.Duration, time.Duration, error) {
	ux.Logger.PrintToUser("Select the Minimum Stake Duration. Please enter in units of hours")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Stake Duration is %d (%s)", defaultMinValidatorStakeDurationHours, defaultMinValidatorStakeDurationHoursString))
	minStakeDuration, err := app.Prompt.CaptureUint64Compare(
		"Minimum Stake Duration",
		[]prompts.Comparator{
			{
				Label: "0",
				Type:  prompts.MoreThan,
				Value: 0,
			},
			{
				Label: "Global Max Stake Duration",
				Type:  prompts.LessThanEq,
				Value: uint64(defaultMaxValidatorStakeDurationHours),
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	ux.Logger.PrintToUser("Select the Maximum Stake Duration")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Stake Duration is %d (%s)", defaultMaxValidatorStakeDurationHours, defaultMaxValidatorStakeDurationHoursString))
	maxStakeDuration, err := app.Prompt.CaptureUint64Compare(
		"Maximum Stake Duration",
		[]prompts.Comparator{
			{
				Label: "Minimum Stake Duration",
				Type:  prompts.MoreThanEq,
				Value: minStakeDuration,
			},
			{
				Label: "Global Max Stake Duration",
				Type:  prompts.LessThanEq,
				Value: uint64(defaultMaxValidatorStakeDurationHours),
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	return time.Duration(minStakeDuration) * time.Hour, time.Duration(maxStakeDuration) * time.Hour, nil
}

func getDelegatorStakeDuration(app *application.Odyssey) (time.Duration, time.Duration, error) {
	ux.Logger.PrintToUser("Select the Minimum Stake Duration. Please enter in units of hours")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Stake Duration is %d (%s)", defaultMinDelegatorStakeDurationHours, defaultMinDelegatorStakeDurationHoursString))
	minStakeDuration, err := app.Prompt.CaptureUint64Compare(
		"Minimum Stake Duration",
		[]prompts.Comparator{
			{
				Label: "0",
				Type:  prompts.MoreThan,
				Value: 0,
			},
			{
				Label: "Global Max Stake Duration",
				Type:  prompts.LessThanEq,
				Value: uint64(defaultMaxDelegatorStakeDurationHours),
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	ux.Logger.PrintToUser("Select the Maximum Stake Duration")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Stake Duration is %d (%s)", defaultMaxDelegatorStakeDurationHours, defaultMaxDelegatorStakeDurationHoursString))
	maxStakeDuration, err := app.Prompt.CaptureUint64Compare(
		"Maximum Stake Duration",
		[]prompts.Comparator{
			{
				Label: "Minimum Stake Duration",
				Type:  prompts.MoreThanEq,
				Value: minStakeDuration,
			},
			{
				Label: "Global Max Stake Duration",
				Type:  prompts.LessThanEq,
				Value: uint64(defaultMaxDelegatorStakeDurationHours),
			},
		},
	)
	if err != nil {
		return 0, 0, err
	}

	return time.Duration(minStakeDuration) * time.Hour, time.Duration(maxStakeDuration) * time.Hour, nil
}

func getMinDelegationFee(app *application.Odyssey) (uint32, error) {
	ux.Logger.PrintToUser("Select the Minimum Delegation Fee. Please denominate your percentage in PercentDenominator")
	ux.Logger.PrintToUser("To denominate your percentage in PercentDenominator just multiply it by 10_000. For example, 1 percent corresponds to 10_000")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Delegation Fee is %s", ux.ConvertToStringWithThousandSeparator(uint64(defaultMinDelegationFee))))
	minDelegationFee, err := app.Prompt.CaptureUint64Compare(
		"Minimum Delegation Fee",
		[]prompts.Comparator{
			{
				Label: "Percent Denominator(1_0000_0000)",
				Type:  prompts.LessThanEq,
				Value: reward.PercentDenominator,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	if minDelegationFee > math.MaxInt32 {
		return 0, fmt.Errorf("minimum Delegation Fee needs to be unsigned 32-bit integer")
	}
	return uint32(minDelegationFee), nil
}

func getMinDelegatorStake(app *application.Odyssey) (uint64, error) {
	ux.Logger.PrintToUser("Select the Minimum Delegator Stake")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Minimum Delegator Stake is %d", defaultMinDelegatorStake))
	minDelegatorStake, err := app.Prompt.CaptureUint64Compare(
		"Minimum Delegator Stake",
		[]prompts.Comparator{
			{
				Label: "0",
				Type:  prompts.MoreThan,
				Value: 0,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	return minDelegatorStake, nil
}

func getMaxValidatorWeightFactor(app *application.Odyssey) (byte, error) {
	ux.Logger.PrintToUser("Select the Maximum Validator Weight Factor. A value of 1 effectively disables delegation")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Maximum Validator Weight Factor is %d", defaultMaxValidatorWeightFactor))
	maxValidatorWeightFactor, err := app.Prompt.CaptureUint64Compare(
		"Maximum Validator Weight Factor",
		[]prompts.Comparator{
			{
				Label: "0",
				Type:  prompts.MoreThan,
				Value: 0,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	if maxValidatorWeightFactor > math.MaxInt8 {
		return 0, fmt.Errorf("maximum Validator Weight Factor needs to be unsigned 8-bit integer")
	}
	return byte(maxValidatorWeightFactor), nil
}

func getUptimeRequirement(app *application.Odyssey) (uint32, error) {
	ux.Logger.PrintToUser("Select the Uptime Requirement. Please denominate your percentage in PercentDenominator")
	ux.Logger.PrintToUser("To denominate your percentage in PercentDenominator just multiply it by 10_000. For example, 1 percent corresponds to 10_000")
	ux.Logger.PrintToUser(fmt.Sprintf("Mainnet Uptime Requirement is %s", ux.ConvertToStringWithThousandSeparator(uint64(defaultUptimeRequirement*reward.PercentDenominator))))
	uptimeReq, err := app.Prompt.CaptureUint64Compare(
		"Uptime Requirement",
		[]prompts.Comparator{
			{
				Label: "Percent Denominator(1_0000_0000)",
				Type:  prompts.LessThanEq,
				Value: reward.PercentDenominator,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	if uptimeReq > math.MaxInt32 {
		return 0, fmt.Errorf("uptime Requirement needs to be unsigned 32-bit integer")
	}
	return uint32(uptimeReq), nil
}
