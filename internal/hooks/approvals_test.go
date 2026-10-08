package hooks

import "testing"

func TestApproveAll_ReusesDecisionByPath(t *testing.T) {
	approvals := NewHookApprovals()
	approvals["/ws/.gws/hooks/pre-clone"] = hookApproval{approved: false}
	approvals["/ws/.gws/hooks/post-clone"] = hookApproval{approved: true, trusted: true}

	denied := &HookInfo{Name: HookPreClone, Path: "/ws/.gws/hooks/pre-clone", Origin: OriginGlobal}
	allowed := &HookInfo{Name: HookPostClone, Path: "/ws/.gws/hooks/post-clone"}

	kept, err := approveAll([]*HookInfo{denied, allowed, nil}, "/ws", approvals)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0] != allowed {
		t.Fatalf("kept = %v, want only the cached approved hook", kept)
	}
	if !allowed.Trusted {
		t.Fatal("cached trusted state not restored on the new HookInfo")
	}
}

func TestApproveAll_RecordsNewDecision(t *testing.T) {
	approvals := NewHookApprovals()
	global := &HookInfo{Name: HookPreClone, Path: "/home/.gws/hooks/pre-clone", Origin: OriginGlobal}

	if _, err := approveAll([]*HookInfo{global}, "/ws", approvals); err != nil {
		t.Fatal(err)
	}
	if decision, ok := approvals[global.Path]; !ok || !decision.approved {
		t.Fatalf("decision = %+v, %v", decision, ok)
	}
}
