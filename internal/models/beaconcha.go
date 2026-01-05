// Package models contains Beaconcha API response structures.
package models

// BeaconchainValidatorsRequest represents the request body for POST /api/v2/ethereum/validators.
type BeaconchainValidatorsRequest struct {
	Chain     string                       `json:"chain,omitempty"`
	Validator BeaconchainValidatorSelector `json:"validator"`
	PageSize  int                          `json:"page_size,omitempty"`
	Cursor    string                       `json:"cursor,omitempty"`
}

// BeaconchainValidatorsResponse represents the response from POST /api/v2/ethereum/validators.
type BeaconchainValidatorsResponse struct {
	Data   []BeaconchainValidatorData `json:"data"`
	Range  BeaconchainResultRange     `json:"range,omitempty"`
	Paging *BeaconchainPaging         `json:"paging,omitempty"`
}

// BeaconchainPaging contains pagination information from the response.
type BeaconchainPaging struct {
	NextCursor string `json:"next_cursor,omitempty"`
}

// BeaconchainValidatorData represents a single validator's data from Beaconcha v2 API.
type BeaconchainValidatorData struct {
	Validator             BeaconchainValidatorInfo     `json:"validator"`
	Slashed               bool                         `json:"slashed"`
	Status                string                       `json:"status"`
	Online                *bool                        `json:"online,omitempty"`
	WithdrawalCredentials BeaconchainWithdrawalCreds   `json:"withdrawal_credentials"`
	LifeCycleEpochs       BeaconchainLifeCycleEpochs   `json:"life_cycle_epochs"`
	Balances              BeaconchainValidatorBalances `json:"balances"`
	Finality              string                       `json:"finality,omitempty"`
}

// BeaconchainValidatorInfo contains the validator index and public key.
type BeaconchainValidatorInfo struct {
	Index     *int   `json:"index"`
	PublicKey string `json:"public_key"`
}

// BeaconchainWithdrawalCreds contains withdrawal credential information.
type BeaconchainWithdrawalCreds struct {
	Type       string  `json:"type"`
	Prefix     string  `json:"prefix"`
	Credential string  `json:"credential"`
	Address    *string `json:"address,omitempty"`
}

// BeaconchainLifeCycleEpochs contains lifecycle epoch information.
type BeaconchainLifeCycleEpochs struct {
	ActivationEligibility *int64 `json:"activation_eligibility"`
	Activation            *int64 `json:"activation"`
	Exit                  *int64 `json:"exit"`
	Withdrawable          *int64 `json:"withdrawable"`
}

// BeaconchainValidatorBalances contains current and effective balance.
type BeaconchainValidatorBalances struct {
	Current   string `json:"current"`
	Effective string `json:"effective"`
}

// BeaconchainRewardsAggregateRequest represents the request body for rewards aggregate endpoint.
type BeaconchainRewardsAggregateRequest struct {
	Chain     string                       `json:"chain,omitempty"`
	Validator BeaconchainValidatorSelector `json:"validator"`
	Range     BeaconchainTimeRangeSelector `json:"range"`
}

// BeaconchainPerformanceAggregateRequest represents the request body for performance aggregate endpoint.
type BeaconchainPerformanceAggregateRequest struct {
	Chain     string                       `json:"chain,omitempty"`
	Validator BeaconchainValidatorSelector `json:"validator"`
	Range     BeaconchainTimeRangeSelector `json:"range"`
}

// BeaconchainValidatorSelector selects validators by identifiers.
type BeaconchainValidatorSelector struct {
	ValidatorIdentifiers []int `json:"validator_identifiers"`
}

// BeaconchainTimeRangeSelector specifies the time range for aggregation.
type BeaconchainTimeRangeSelector struct {
	EvaluationWindow string `json:"evaluation_window"`
}

// BeaconchainRewardsAggregateResponse represents the response from rewards aggregate endpoint.
// Returns aggregated rewards for all requested validators combined.
type BeaconchainRewardsAggregateResponse struct {
	Data  BeaconchainRewardsData `json:"data"`
	Range BeaconchainResultRange `json:"range"`
}

// BeaconchainRewardsData contains the rewards breakdown.
type BeaconchainRewardsData struct {
	Total         string                          `json:"total"`
	TotalReward   string                          `json:"total_reward"`
	TotalPenalty  string                          `json:"total_penalty"`
	TotalMissed   string                          `json:"total_missed"`
	Attestation   BeaconchainAttestationRewards   `json:"attestation"`
	SyncCommittee BeaconchainSyncCommitteeRewards `json:"sync_committee"`
	Proposal      BeaconchainProposalRewards      `json:"proposal"`
	Finality      string                          `json:"finality,omitempty"`
}

// BeaconchainAttestationRewards contains attestation reward breakdown.
type BeaconchainAttestationRewards struct {
	Total                 string                     `json:"total"`
	Head                  BeaconchainRewardBreakdown `json:"head"`
	Source                BeaconchainRewardBreakdown `json:"source"`
	Target                BeaconchainRewardBreakdown `json:"target"`
	InactivityLeakPenalty string                     `json:"inactivity_leak_penalty"`
	InclusionDelay        *BeaconchainInclusionDelay `json:"inclusion_delay,omitempty"`
}

// BeaconchainRewardBreakdown contains detailed reward/penalty breakdown.
type BeaconchainRewardBreakdown struct {
	Total        string `json:"total"`
	Reward       string `json:"reward"`
	Penalty      string `json:"penalty"`
	MissedReward string `json:"missed_reward"`
}

// BeaconchainInclusionDelay contains inclusion delay rewards (pre-Altair only).
type BeaconchainInclusionDelay struct {
	Total        string `json:"total"`
	MissedReward string `json:"missed_reward"`
}

// BeaconchainSyncCommitteeRewards contains sync committee reward breakdown.
type BeaconchainSyncCommitteeRewards struct {
	Total        string `json:"total"`
	Reward       string `json:"reward"`
	Penalty      string `json:"penalty"`
	MissedReward string `json:"missed_reward"`
}

// BeaconchainProposalRewards contains proposal reward breakdown.
type BeaconchainProposalRewards struct {
	Total                      string `json:"total"`
	ExecutionLayerReward       string `json:"execution_layer_reward"`
	AttestationInclusionReward string `json:"attestation_inclusion_reward"`
	SyncInclusionReward        string `json:"sync_inclusion_reward"`
	SlashingInclusionReward    string `json:"slashing_inclusion_reward"`
	MissedCLReward             string `json:"missed_cl_reward"`
	MissedELReward             string `json:"missed_el_reward"`
}

// BeaconchainResultRange contains the range of data returned.
type BeaconchainResultRange struct {
	Slot      BeaconchainSlotRange      `json:"slot"`
	Epoch     BeaconchainEpochRange     `json:"epoch"`
	Timestamp BeaconchainTimestampRange `json:"timestamp"`
}

// BeaconchainSlotRange defines slot boundaries.
type BeaconchainSlotRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// BeaconchainEpochRange defines epoch boundaries.
type BeaconchainEpochRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// BeaconchainTimestampRange defines timestamp boundaries.
type BeaconchainTimestampRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// BeaconchainPerformanceAggregateResponse represents the response from performance aggregate endpoint.
// Returns aggregated performance for all requested validators combined.
type BeaconchainPerformanceAggregateResponse struct {
	Data  BeaconchainPerformanceData `json:"data"`
	Range BeaconchainResultRange     `json:"range"`
}

// BeaconchainPerformanceData contains performance metrics.
type BeaconchainPerformanceData struct {
	Beaconscore BeaconchainBeaconscore       `json:"beaconscore"`
	Duties      BeaconchainPerformanceDuties `json:"duties"`
	Finality    string                       `json:"finality"`
}

// BeaconchainBeaconscore contains BeaconScore values.
type BeaconchainBeaconscore struct {
	Total         *float64 `json:"total"`
	Attestation   *float64 `json:"attestation"`
	Proposal      *float64 `json:"proposal"`
	SyncCommittee *float64 `json:"sync_committee"`
}

// BeaconchainPerformanceDuties contains duty performance breakdown.
type BeaconchainPerformanceDuties struct {
	Attestation   BeaconchainAttestationDuties   `json:"attestation"`
	SyncCommittee BeaconchainSyncCommitteeDuties `json:"sync_committee"`
	Proposal      BeaconchainProposalDuties      `json:"proposal"`
}

// BeaconchainAttestationDuties contains attestation duty metrics.
type BeaconchainAttestationDuties struct {
	Included                              int     `json:"included"`
	Assigned                              int     `json:"assigned"`
	CorrectHead                           int     `json:"correct_head"`
	CorrectSource                         int     `json:"correct_source"`
	CorrectTarget                         int     `json:"correct_target"`
	ValuableCorrectHead                   int     `json:"valuable_correct_head"`
	ValuableCorrectSource                 int     `json:"valuable_correct_source"`
	ValuableCorrectTarget                 int     `json:"valuable_correct_target"`
	AvgInclusionDelay                     float64 `json:"avg_inclusion_delay"`
	AvgInclusionDelayExcludingMissedSlots float64 `json:"avg_inclusion_delay_excluding_missed_slots"`
	Missed                                int     `json:"missed"`
}

// BeaconchainSyncCommitteeDuties contains sync committee duty metrics.
type BeaconchainSyncCommitteeDuties struct {
	Successful int `json:"successful"`
	Assigned   int `json:"assigned"`
	Missed     int `json:"missed"`
}

// BeaconchainProposalDuties contains proposal duty metrics.
type BeaconchainProposalDuties struct {
	Successful        int `json:"successful"`
	Assigned          int `json:"assigned"`
	Missed            int `json:"missed"`
	IncludedSlashings int `json:"included_slashings"`
}

// BeaconchainErrorResponse represents an error response from Beaconcha.
type BeaconchainErrorResponse struct {
	Message string `json:"message"`
}
