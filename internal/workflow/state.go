package workflow

import "time"

const SchemaVersion = 1

type Phase string

const (
	PhasePlanning    Phase = "PLANNING"
	PhasePlanReview  Phase = "PLAN_REVIEW"
	PhaseExecuting   Phase = "EXECUTING"
	PhaseCodeReview  Phase = "CODE_REVIEW"
	PhaseQA          Phase = "QA"
	PhaseFinalReview Phase = "FINAL_REVIEW"
	PhaseDone        Phase = "DONE"
)

type Plan struct {
	Revision int        `json:"revision"`
	Path     string     `json:"path"`
	Snapshot string     `json:"snapshot,omitempty"`
	SHA256   string     `json:"sha256,omitempty"`
	SealedAt *time.Time `json:"sealed_at,omitempty"`
}

type State struct {
	SchemaVersion  int       `json:"schema_version"`
	ChangeID       string    `json:"change_id"`
	Title          string    `json:"title"`
	Phase          Phase     `json:"phase"`
	Plan           Plan      `json:"plan"`
	ExecutionRound int       `json:"execution_round"`
	QARound        int       `json:"qa_round"`
	Events         []Event   `json:"events"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Project struct {
	SchemaVersion  int       `json:"schema_version"`
	ProjectName    string    `json:"project_name"`
	RepositoryRoot string    `json:"repository_root"`
	CreatedAt      time.Time `json:"created_at"`
	ElGordoVersion string    `json:"elgordo_version"`
}

type Event struct {
	Sequence   int       `json:"seq"`
	At         time.Time `json:"at"`
	Command    string    `json:"command"`
	From       Phase     `json:"from,omitempty"`
	To         Phase     `json:"to"`
	PlanSHA256 string    `json:"plan_sha256,omitempty"`
	Route      string    `json:"route,omitempty"`
	Verdict    string    `json:"verdict,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}
