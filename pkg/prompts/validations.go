// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prompts

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DioneProtocol/odysseygo/genesis"

	"github.com/DioneProtocol/odyssey-cli/pkg/constants"
	"github.com/DioneProtocol/odyssey-cli/pkg/models"
	"github.com/DioneProtocol/odyssey-cli/pkg/ux"
	"github.com/DioneProtocol/odysseygo/ids"
	odygoconstants "github.com/DioneProtocol/odysseygo/utils/constants"
	"github.com/DioneProtocol/odysseygo/utils/formatting/address"
	"github.com/ethereum/go-ethereum/common"
)

func validateEmail(input string) error {
	_, err := mail.ParseAddress(input)
	return err
}

func validatePositiveBigInt(input string) error {
	n := new(big.Int)
	n, ok := n.SetString(input, 10)
	if !ok {
		return errors.New("invalid number")
	}
	if n.Cmp(big.NewInt(0)) == -1 {
		return errors.New("invalid number")
	}
	return nil
}

func validateMainnetValidatorStakingDuration(input string) error {
	d, err := time.ParseDuration(input)
	if err != nil {
		return err
	}
	if d > genesis.MainnetParams.MaxValidatorStakeDuration {
		return fmt.Errorf("exceeds maximum validator staking duration of %s", ux.FormatDuration(genesis.MainnetParams.MaxValidatorStakeDuration))
	}
	if d < genesis.MainnetParams.MinValidatorStakeDuration {
		return fmt.Errorf("below the minimum validator staking duration of %s", ux.FormatDuration(genesis.MainnetParams.MinValidatorStakeDuration))
	}
	return nil
}

func validateTestnetValidatorStakingDuration(input string) error {
	d, err := time.ParseDuration(input)
	if err != nil {
		return err
	}
	if d > genesis.TestnetParams.MaxValidatorStakeDuration {
		return fmt.Errorf("exceeds maximum validator staking duration of %s", ux.FormatDuration(genesis.TestnetParams.MaxValidatorStakeDuration))
	}
	if d < genesis.TestnetParams.MinValidatorStakeDuration {
		return fmt.Errorf("below the minimum validator staking duration of %s", ux.FormatDuration(genesis.TestnetParams.MinValidatorStakeDuration))
	}
	return nil
}

func validateMainnetDelegatorStakingDuration(input string) error {
	d, err := time.ParseDuration(input)
	if err != nil {
		return err
	}
	if d > genesis.MainnetParams.MaxDelegatorStakeDuration {
		return fmt.Errorf("exceeds maximum delegator staking duration of %s", ux.FormatDuration(genesis.MainnetParams.MaxDelegatorStakeDuration))
	}
	if d < genesis.MainnetParams.MinDelegatorStakeDuration {
		return fmt.Errorf("below the minimum delegator staking duration of %s", ux.FormatDuration(genesis.MainnetParams.MinDelegatorStakeDuration))
	}
	return nil
}

func validateTestnetDelegatorStakingDuration(input string) error {
	d, err := time.ParseDuration(input)
	if err != nil {
		return err
	}
	if d > genesis.TestnetParams.MaxDelegatorStakeDuration {
		return fmt.Errorf("exceeds maximum delegator staking duration of %s", ux.FormatDuration(genesis.TestnetParams.MaxDelegatorStakeDuration))
	}
	if d < genesis.TestnetParams.MinDelegatorStakeDuration {
		return fmt.Errorf("below the minimum delegator staking duration of %s", ux.FormatDuration(genesis.TestnetParams.MinDelegatorStakeDuration))
	}
	return nil
}

func validateTime(input string) error {
	t, err := time.Parse(constants.TimeParseLayout, input)
	if err != nil {
		return err
	}
	if t.Before(time.Now().Add(constants.StakingStartLeadTime)) {
		return fmt.Errorf("time should be at least start from now + %s", constants.StakingStartLeadTime)
	}
	return err
}

func validateNodeID(input string) error {
	_, err := ids.NodeIDFromString(input)
	return err
}

func validateAddress(input string) error {
	if !common.IsHexAddress(input) {
		return errors.New("invalid address")
	}
	return nil
}

func validateExistingFilepath(input string) error {
	if fileInfo, err := os.Stat(input); err == nil && !fileInfo.IsDir() {
		return nil
	}
	return errors.New("file doesn't exist")
}

func validateWeight(input string) error {
	val, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return err
	}
	if val < constants.MinStakeWeight {
		return errors.New("the weight must be an integer between 1 and 100")
	}
	return nil
}

func validateBiggerThanZero(input string) error {
	val, err := strconv.ParseUint(input, 0, 64)
	if err != nil {
		return err
	}
	if val == 0 {
		return errors.New("the value must be bigger than zero")
	}
	return nil
}

func validateURLFormat(input string) error {
	_, err := url.ParseRequestURI(input)
	if err != nil {
		return err
	}
	return nil
}

func validateOChainAddress(input string) (string, error) {
	chainID, hrp, _, err := address.Parse(input)
	if err != nil {
		return "", err
	}

	if chainID != "O" {
		return "", errors.New("this is not a OChain address")
	}
	return hrp, nil
}

func validateOChainTestnetAddress(input string) error {
	hrp, err := validateOChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != odygoconstants.TestnetHRP {
		return errors.New("this is not a testnet address")
	}
	return nil
}

func validateOChainMainAddress(input string) error {
	hrp, err := validateOChainAddress(input)
	if err != nil {
		return err
	}
	if hrp != odygoconstants.MainnetHRP {
		return errors.New("this is not a mainnet address")
	}
	return nil
}

func validateOChainLocalAddress(input string) error {
	hrp, err := validateOChainAddress(input)
	if err != nil {
		return err
	}
	// ONR uses the `custom` HRP for local networks,
	// but the `local` HRP also exists...
	if hrp != odygoconstants.LocalHRP && hrp != odygoconstants.FallbackHRP {
		return errors.New("this is not a local nor custom address")
	}
	return nil
}

func getOChainValidationFunc(network models.Network) func(string) error {
	switch network.Kind {
	case models.Testnet:
		return validateOChainTestnetAddress
	case models.Mainnet:
		return validateOChainMainAddress
	case models.Local:
		return validateOChainLocalAddress
	default:
		return func(string) error {
			return errors.New("unsupported network")
		}
	}
}

func validateID(input string) error {
	_, err := ids.FromString(input)
	return err
}

func validateNewFilepath(input string) error {
	if _, err := os.Stat(input); err != nil && os.IsNotExist(err) {
		return nil
	}
	return errors.New("file already exists")
}

func validateNonEmpty(input string) error {
	if input == "" {
		return errors.New("string cannot be empty")
	}
	return nil
}

func RequestURL(url string) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for url %s: %w", url, err)
	}
	token := os.Getenv(constants.GithubAPITokenEnvVarName)
	if token != "" {
		// avoid rate limitation issues at CI
		request.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status code: %d", resp.StatusCode)
	}
	return resp, nil
}

func ValidateURL(url string) error {
	if err := validateURLFormat(url); err != nil {
		return err
	}
	resp, err := RequestURL(url)
	if err != nil {
		return err
	}
	// will just ignore this error, url is already validated
	_ = resp.Body.Close()
	return nil
}

func ValidateRepoBranch(repo string, branch string) error {
	url := repo + "/tree/" + branch
	return ValidateURL(url)
}

func ValidateRepoFile(repo string, branch string, file string) error {
	url := repo + "/blob/" + branch + "/" + file
	return ValidateURL(url)
}

func ValidateHexa(input string) error {
	if input == "" {
		return errors.New("string cannot be empty")
	}
	if len(input) < 2 || strings.ToLower(input[:2]) != "0x" {
		return errors.New("hexa string has not 0x prefix")
	}
	if len(input) == 2 {
		return errors.New("no hexa digits in string")
	}
	_, err := hex.DecodeString(input[2:])
	if err != nil {
		return errors.New("string not in hexa format")
	}
	return err
}
