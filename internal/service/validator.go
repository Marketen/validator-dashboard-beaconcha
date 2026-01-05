// Package service provides business logic for the validator-dashboard API.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/beaconcha"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/models"
)

// ValidatorService handles validator data aggregation.
type ValidatorService struct {
	beaconchainClient *beaconcha.Client
}

// NewValidatorService creates a new validator service.
func NewValidatorService(client *beaconcha.Client) *ValidatorService {
	return &ValidatorService{
		beaconchainClient: client,
	}
}

// GetValidatorData fetches and aggregates data for the given validator IDs.
func (s *ValidatorService) GetValidatorData(ctx context.Context, chain string, validatorIds []int) (models.ValidatorResponse, error) {
	if len(validatorIds) == 0 {
		return models.ValidatorResponse{}, nil
	}

	slog.Debug("fetching validator data", "validators", len(validatorIds))

	// Fetch data from Beaconcha
	return s.fetchAndAggregate(ctx, chain, validatorIds)
}

// fetchAndAggregate fetches all required data from Beaconcha and aggregates it.
func (s *ValidatorService) fetchAndAggregate(ctx context.Context, chain string, validatorIds []int) (models.ValidatorResponse, error) {
	// Fetch validator overview data (per-validator)
	validators, err := s.beaconchainClient.GetValidators(ctx, chain, validatorIds)
	if err != nil {
		return models.ValidatorResponse{}, fmt.Errorf("fetch validators: %w", err)
	}

	// Fetch aggregated rewards (combined for all validators)
	rewards, err := s.beaconchainClient.GetRewardsAggregate(ctx, chain, validatorIds)
	if err != nil {
		return models.ValidatorResponse{}, fmt.Errorf("fetch rewards: %w", err)
	}

	// Fetch aggregated performance (combined for all validators)
	performance, err := s.beaconchainClient.GetPerformanceAggregate(ctx, chain, validatorIds)
	if err != nil {
		return models.ValidatorResponse{}, fmt.Errorf("fetch performance: %w", err)
	}

	// Build per-validator overview map
	validatorOverviews := make(map[string]models.ValidatorOverview)
	for _, v := range validators {
		if v.Validator.Index != nil {
			idStr := strconv.Itoa(*v.Validator.Index)
			validatorOverviews[idStr] = s.buildOverview(v)
		}
	}

	// Build response with per-validator overviews and single aggregated rewards/performance
	response := models.ValidatorResponse{
		Validators:  validatorOverviews,
		Rewards:     s.buildRewards(rewards),
		Performance: s.buildPerformance(performance),
	}

	return response, nil
}

// buildOverview constructs the overview section from validator data.
func (s *ValidatorService) buildOverview(v models.BeaconchainValidatorData) models.ValidatorOverview {
	// Parse balances from wei strings
	currentBalance := v.Balances.Current
	effectiveBalance := v.Balances.Effective

	// Get activation and exit epochs
	var activationEpoch, exitEpoch int64
	if v.LifeCycleEpochs.Activation != nil {
		activationEpoch = *v.LifeCycleEpochs.Activation
	}
	if v.LifeCycleEpochs.Exit != nil {
		exitEpoch = *v.LifeCycleEpochs.Exit
	} else {
		exitEpoch = 0 // not scheduled for exit or exited
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

// buildRewards constructs the rewards section from aggregated rewards response.
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

// buildPerformance constructs the performance section from aggregated performance response.
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
