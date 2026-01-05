// Package models contains data structures for the validator-dashboard API.
package models

// ValidatorRequest represents the incoming request body for POST /validator.
type ValidatorRequest struct {
	// ValidatorIds is a list of unique validator indices.
	// Minimum: 1, Maximum: 100
	ValidatorIds []int `json:"validatorIds"`
	// Chain is the target chain for the request. Allowed values: "mainnet", "hoodi".
	Chain string `json:"chain"`
}

// ValidatorResponse contains per-validator overviews and aggregated rewards/performance.
type ValidatorResponse struct {
	// Validators contains per-validator overview data keyed by validator ID.
	Validators map[string]ValidatorOverview `json:"validators"`
	// Rewards contains aggregated rewards for all requested validators.
	Rewards ValidatorRewards `json:"rewards"`
	// Performance contains aggregated performance for all requested validators.
	Performance ValidatorPerformance `json:"performance"`
}

// ValidatorOverview contains basic validator state information.
type ValidatorOverview struct {
	Slashed               bool                  `json:"slashed"`
	Status                string                `json:"status"`
	WithdrawalCredentials WithdrawalCredentials `json:"withdrawalCredentials"`
	ActivationEpoch       int64                 `json:"activationEpoch"`
	ExitEpoch             int64                 `json:"exitEpoch"`
	CurrentBalance        string                `json:"currentBalance"`   // in wei
	EffectiveBalance      string                `json:"effectiveBalance"` //in wei
	Online                bool                  `json:"online"`
}

// WithdrawalCredentials contains the type and address for withdrawals.
type WithdrawalCredentials struct {
	Type    string `json:"type"`    // "0x00" (BLS) or "0x01" (execution address)
	Address string `json:"address"` // The full credentials hex string or derived address
}

// ValidatorRewards contains all-time reward/penalty information.
type ValidatorRewards struct {
	Total          string               `json:"total"`        // Net rewards (rewards - penalties) in wei
	TotalReward    string               `json:"totalReward"`  // Total rewards earned in wei
	TotalPenalty   string               `json:"totalPenalty"` // Total penalties in wei
	TotalMissed    string               `json:"totalMissed"`  // Total missed rewards in wei
	Proposals      ProposalRewards      `json:"proposals"`
	Attestations   AttestationRewards   `json:"attestations"`
	SyncCommittees SyncCommitteeRewards `json:"syncCommittees"`
}

// ProposalRewards contains reward breakdown for block proposals.
type ProposalRewards struct {
	Total                      string `json:"total"`                      // Total proposal rewards in wei
	ExecutionLayerReward       string `json:"executionLayerReward"`       // EL rewards in wei
	AttestationInclusionReward string `json:"attestationInclusionReward"` // Attestation inclusion in wei
	SyncInclusionReward        string `json:"syncInclusionReward"`        // Sync inclusion in wei
	SlashingInclusionReward    string `json:"slashingInclusionReward"`    // Slashing inclusion in wei
	MissedCLReward             string `json:"missedClReward"`             // Missed CL rewards in wei
	MissedELReward             string `json:"missedElReward"`             // Missed EL rewards in wei
}

// AttestationRewards contains reward breakdown for attestations.
type AttestationRewards struct {
	Total                 string `json:"total"`                 // Total attestation rewards in wei
	Head                  string `json:"head"`                  // Head vote rewards in wei
	Source                string `json:"source"`                // Source vote rewards in wei
	Target                string `json:"target"`                // Target vote rewards in wei
	InactivityLeakPenalty string `json:"inactivityLeakPenalty"` // Inactivity leak penalty in wei
}

// SyncCommitteeRewards contains reward breakdown for sync committee duties.
type SyncCommitteeRewards struct {
	Total        string `json:"total"`        // Net sync committee rewards in wei
	Reward       string `json:"reward"`       // Sync committee rewards in wei
	Penalty      string `json:"penalty"`      // Sync committee penalties in wei
	MissedReward string `json:"missedReward"` // Missed sync committee rewards in wei
}

// ValidatorPerformance contains all-time performance metrics.
type ValidatorPerformance struct {
	Beaconscore    *float64            `json:"beaconscore"` // Overall BeaconScore (0-1)
	Attestations   AttestationDuties   `json:"attestations"`
	SyncCommittees SyncCommitteeDuties `json:"syncCommittees"`
	Proposals      ProposalDuties      `json:"proposals"`
}

// AttestationDuties contains attestation performance metrics.
type AttestationDuties struct {
	Assigned          int      `json:"assigned"`          // Total attestation duties assigned
	Included          int      `json:"included"`          // Attestations included on-chain
	Missed            int      `json:"missed"`            // Missed attestations
	CorrectHead       int      `json:"correctHead"`       // Correct head votes
	CorrectSource     int      `json:"correctSource"`     // Correct source votes
	CorrectTarget     int      `json:"correctTarget"`     // Correct target votes
	AvgInclusionDelay float64  `json:"avgInclusionDelay"` // Average inclusion delay in slots
	Beaconscore       *float64 `json:"beaconscore"`       // Attestation-specific BeaconScore
}

// SyncCommitteeDuties contains sync committee performance metrics.
type SyncCommitteeDuties struct {
	Assigned    int      `json:"assigned"`    // Total sync committee duties assigned
	Successful  int      `json:"successful"`  // Successfully performed duties
	Missed      int      `json:"missed"`      // Missed duties
	Beaconscore *float64 `json:"beaconscore"` // Sync committee-specific BeaconScore
}

// ProposalDuties contains block proposal performance metrics.
type ProposalDuties struct {
	Assigned          int      `json:"assigned"`          // Total proposal duties assigned
	Successful        int      `json:"successful"`        // Successfully proposed blocks
	Missed            int      `json:"missed"`            // Missed/orphaned proposals
	IncludedSlashings int      `json:"includedSlashings"` // Slashing proofs included
	Beaconscore       *float64 `json:"beaconscore"`       // Proposal-specific BeaconScore
}

// APIError represents an error response from the API.
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}
