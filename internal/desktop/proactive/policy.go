package proactive

import "fmt"

type PolicyInput struct {
	Autonomy              int    `json:"autonomy"`
	Action                string `json:"action"`
	Approved              bool   `json:"approved,omitempty"`
	ExplicitAuthorization bool   `json:"explicitAuthorization,omitempty"`
}

type PolicyDecision struct {
	Allowed          bool   `json:"allowed"`
	RequiresApproval bool   `json:"requiresApproval"`
	Level            int    `json:"level"`
	Action           string `json:"action"`
	Reason           string `json:"reason,omitempty"`
}

// Evaluate mirrors the task policy, not the Host's tool permission system.
// A task approval never changes account, shell or tool permissions by itself.
func Evaluate(in PolicyInput) (PolicyDecision, error) {
	d := PolicyDecision{Level: in.Autonomy, Action: in.Action}
	if in.Autonomy < 1 || in.Autonomy > 5 {
		return d, fmt.Errorf("autonomy must be an integer from 1 to 5")
	}
	if in.Action == "" {
		return d, fmt.Errorf("action is required")
	}
	switch in.Action {
	case "read", "summarize", "inspect":
		d.Allowed = true
		return d, nil
	}
	if in.Autonomy == 1 {
		d.Reason = "L1 is read-only"
		return d, nil
	}
	switch in.Action {
	case "draft", "local_write", "test":
		d.Allowed = true
		return d, nil
	case "external_message", "publish", "account_change", "permission_change", "payment", "irreversible_delete":
		d.RequiresApproval, d.Allowed = true, in.Approved
		if !d.Allowed {
			d.Reason = "Human approval is required"
		}
		return d, nil
	}
	if in.Autonomy < 4 {
		d.RequiresApproval = true
		d.Reason = "Action exceeds the autonomy level"
		return d, nil
	}
	if !in.ExplicitAuthorization {
		d.RequiresApproval = true
		d.Reason = "L4/L5 still needs task authorization"
		return d, nil
	}
	d.Allowed = true
	return d, nil
}
