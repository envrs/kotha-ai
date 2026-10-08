package permission

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutoApprove(t *testing.T) {
	s := NewPermissionService()
	s.AutoApproveSession("s1")
	require.True(t, s.Request(CreatePermissionRequest{SessionID: "s1", ToolName: "bash", Action: "exec", Path: "/tmp/x"}))
}

func TestGrantPersistentThenAutoAllow(t *testing.T) {
	svc := NewPermissionService()
	s := svc.(*permissionService)

	p := PermissionRequest{ID: "id-1", SessionID: "s", ToolName: "edit", Action: "write", Path: "/tmp"}
	respCh := make(chan bool, 1)
	s.pendingRequests.Store("id-1", respCh)
	s.GrantPersistant(p)
	require.True(t, <-respCh)
	require.Len(t, s.sessionPermissions, 1)

	// second identical request is auto-allowed via sessionPermissions
	done := make(chan bool, 1)
	go func() {
		done <- svc.Request(CreatePermissionRequest{SessionID: "s", ToolName: "edit", Action: "write", Path: "/tmp/f.txt"})
	}()
	select {
	case got := <-done:
		require.True(t, got)
	case <-time.After(2 * time.Second):
		t.Fatal("request not auto-approved")
	}
}

func TestGrantDenyDirect(t *testing.T) {
	s := NewPermissionService().(*permissionService)
	for id, fn := range map[string]func(PermissionRequest){
		"g": s.Grant, "d": s.Deny,
	} {
		ch := make(chan bool, 1)
		s.pendingRequests.Store(id, ch)
		fn(PermissionRequest{ID: id})
		got := <-ch
		if id == "g" {
			require.True(t, got)
		} else {
			require.False(t, got)
		}
	}
	// unknown IDs are no-ops
	require.NotPanics(t, func() {
		s.Grant(PermissionRequest{ID: "nope"})
		s.Deny(PermissionRequest{ID: "nope"})
		s.GrantPersistant(PermissionRequest{ID: "nope2"})
	})
}
