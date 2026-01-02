// Package service provides business logic for the validator-dashboard API.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/beaconcha"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/cache"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/models"
)

// ValidatorService handles validator data aggregation.
type ValidatorService struct {
	beaconchainClient *beaconcha.Client
	cache             *cache.Cache
}

// NewValidatorService creates a new validator service.
func NewValidatorService(client *beaconcha.Client, cache *cache.Cache) *ValidatorService {
	return &ValidatorService{
		beaconchainClient: client,
		cache:             cache,
	}
}

// GetValidatorData fetches and aggregates data for the given validator IDs.
func (s *ValidatorService) GetValidatorData(ctx context.Context, validatorIds []int) (models.ValidatorResponse, error) {
	if len(validatorIds) == 0 {
		return models.ValidatorResponse{}, nil
	}

	// Generate cache key from sorted validator IDs
	cacheKey := s.generateCacheKey(validatorIds)

	// Check cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		slog.Debug("cache hit", "key", cacheKey)
		return cached.(models.ValidatorResponse), nil
	}

	slog.Debug("cache miss", "key", cacheKey, "validators", len(validatorIds))

	// Fetch data from Beaconcha
	response, err := s.fetchAndAggregate(ctx, validatorIds)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.Set(cacheKey, response)

	return response, nil
}

// fetchAndAggregate fetches all required data from Beaconcha and aggregates it.
func (s *ValidatorService) fetchAndAggregate(ctx context.Context, validatorIds []int) (models.ValidatorResponse, error) {
	// Fetch validator overview data
	validators, err := s.beaconchainClient.GetValidators(ctx, validatorIds)
	if err != nil {
		return nil, fmt.Errorf("fetch validators: %w", err)
	}

	// Fetch aggregated rewards
	rewards, err := s.beaconchainClient.GetRewardsAggregate(ctx, validatorIds)
	if err != nil {
		return nil, fmt.Errorf("fetch rewards: %w", err)
	}

	// Fetch aggregated performance
	performance, err := s.beaconchainClient.GetPerformanceAggregate(ctx, validatorIds)
	if err != nil {
		return nil, fmt.Errorf("fetch performance: %w", err)
	}

	// Build response
	response := make(models.ValidatorResponse)

	// Create a map for quick lookup
	validatorMap := make(map[int]models.BeaconchainValidatorData)
	for _, v := range validators {
		if v.Validator.Index != nil {
			validatorMap[*v.Validator.Index] = v
		}
	}

	// Build individual validator responses
	for _, id := range validatorIds {
		validator, exists := validatorMap[id]
		if !exists {
			slog.Warn("validator not found in response", "id", id)
			continue
		}

		validatorData := models.ValidatorData{
			Overview:    s.buildOverview(validator),
			Rewards:     s.buildRewards(rewards),
			Performance: s.buildPerformance(performance),
		}

		response[strconv.Itoa(id)] = validatorData
	}

	return response, nil
}

// buildOverview constructs the overview section from validator data.
func (s *ValidatorService) buildOverview(v models.BeaconchainValidatorData) models.ValidatorOverview {
	// Parse balances from wei strings
	currentBalance := parseWeiToGwei(v.Balances.Current)
	effectiveBalance := parseWeiToGwei(v.Balances.Effective)

	// Get activation and exit epochs
	var activationEpoch, exitEpoch int64
	if v.LifeCycleEpochs.Activation != nil {
		activationEpoch = *v.LifeCycleEpochs.Activation
	}
	if v.LifeCycleEpochs.Exit != nil {
		exitEpoch = *v.LifeCycleEpochs.Exit
	} else {
		exitEpoch = 9223372036854775807 // FAR_FUTURE_EPOCH
	}

	// Determine online status
	online := false
	if v.Online != nil {
		online = *v.Online
	}

	return models.ValidatorOverview{
		Slashed:               v.Slashed,
		Status:                v.Status,
		WithdrawalCredentials: s.buildWithdrawalCredentials(v.WithdrawalCredentials),
		ActivationEpoch:       activationEpoch,
		ExitEpoch:             exitEpoch,
		CurrentBalance:        currentBalance,
		EffectiveBalance:      effectiveBalance,
		Online:                online,
	}
}

// parseWeiToGwei converts a wei string to gwei (divide by 10^9).
func parseWeiToGwei(weiStr string) int64 {
	if weiStr == "" {
		return 0
	}

	// Use big.Int for proper handling of large wei values
	wei := new(big.Int)
	_, ok := wei.SetString(weiStr, 10)
	if !ok {
		return 0
	}

	// Divide by 10^9 to get gwei
	gwei := new(big.Int).Div(wei, big.NewInt(1000000000))

	return gwei.Int64()
}

// buildWithdrawalCredentials builds withdrawal credentials from v2 API response.
func (s *ValidatorService) buildWithdrawalCredentials(creds models.BeaconchainWithdrawalCreds) models.WithdrawalCredentials {
	result := models.WithdrawalCredentials{
		Type: creds.Type,
	}

	if creds.Address != nil {
		result.Address = *creds.Address
	} else {
		result.Address = creds.Credential
	}

	return result
}

// buildRewards constructs the rewards section from aggregated rewards data.
func (s *ValidatorService) buildRewards(r *models.BeaconchainRewardsAggregateResponse) models.ValidatorRewards {
	if r == nil {
		return models.ValidatorRewards{}
	}

	return models.ValidatorRewards{
		Total:        r.Data.Total,
		TotalReward:  r.Data.TotalReward,
		TotalPenalty: r.Data.TotalPenalty,
		TotalMissed:  r.Data.TotalMissed,
		Proposals: models.ProposalRewards{
			Total:                      r.Data.Proposal.Total,
			ExecutionLayerReward:       r.Data.Proposal.ExecutionLayerReward,
			AttestationInclusionReward: r.Data.Proposal.AttestationInclusionReward,
			SyncInclusionReward:        r.Data.Proposal.SyncInclusionReward,
			SlashingInclusionReward:    r.Data.Proposal.SlashingInclusionReward,
			MissedCLReward:             r.Data.Proposal.MissedCLReward,
			MissedELReward:             r.Data.Proposal.MissedELReward,
		},
		Attestations: models.AttestationRewards{
			Total:                 r.Data.Attestation.Total,
			Head:                  r.Data.Attestation.Head.Total,
			Source:                r.Data.Attestation.Source.Total,
			Target:                r.Data.Attestation.Target.Total,
			InactivityLeakPenalty: r.Data.Attestation.InactivityLeakPenalty,
		},
		SyncCommittees: models.SyncCommitteeRewards{
			Total:        r.Data.SyncCommittee.Total,
			Reward:       r.Data.SyncCommittee.Reward,
			Penalty:      r.Data.SyncCommittee.Penalty,
			MissedReward: r.Data.SyncCommittee.MissedReward,
		},
	}
}

// buildPerformance constructs the performance section from aggregated performance data.
func (s *ValidatorService) buildPerformance(p *models.BeaconchainPerformanceAggregateResponse) models.ValidatorPerformance {
	if p == nil {
		return models.ValidatorPerformance{}
	}

	return models.ValidatorPerformance{
		Beaconscore: p.Data.Beaconscore.Total,
		Attestations: models.AttestationDuties{
			Assigned:          p.Data.Duties.Attestation.Assigned,
			Included:          p.Data.Duties.Attestation.Included,
			Missed:            p.Data.Duties.Attestation.Missed,
			CorrectHead:       p.Data.Duties.Attestation.CorrectHead,
			CorrectSource:     p.Data.Duties.Attestation.CorrectSource,
			CorrectTarget:     p.Data.Duties.Attestation.CorrectTarget,
			AvgInclusionDelay: p.Data.Duties.Attestation.AvgInclusionDelay,
			Beaconscore:       p.Data.Beaconscore.Attestation,
		},
		SyncCommittees: models.SyncCommitteeDuties{
			Assigned:    p.Data.Duties.SyncCommittee.Assigned,
			Successful:  p.Data.Duties.SyncCommittee.Successful,
			Missed:      p.Data.Duties.SyncCommittee.Missed,
			Beaconscore: p.Data.Beaconscore.SyncCommittee,
		},
		Proposals: models.ProposalDuties{
			Assigned:          p.Data.Duties.Proposal.Assigned,
			Successful:        p.Data.Duties.Proposal.Successful,
			Missed:            p.Data.Duties.Proposal.Missed,
			IncludedSlashings: p.Data.Duties.Proposal.IncludedSlashings,
			Beaconscore:       p.Data.Beaconscore.Proposal,
		},
	}
}

// generateCacheKey creates a unique cache key from validator IDs.
func (s *ValidatorService) generateCacheKey(validatorIds []int) string {
	// Sort IDs for consistent cache keys
	sorted := make([]int, len(validatorIds))
	copy(sorted, validatorIds)
	sort.Ints(sorted)

	// Create string representation
	ids := make([]string, len(sorted))
	for i, id := range sorted {
		ids[i] = strconv.Itoa(id)
	}
	key := strings.Join(ids, ",")

	// Hash for shorter key if many validators
	if len(validatorIds) > 10 {
		hash := sha256.Sum256([]byte(key))
		return "validators:" + hex.EncodeToString(hash[:8])
	}

	return "validators:" + key
}
