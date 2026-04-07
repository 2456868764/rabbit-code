package bashtool

import (
	"context"
	"os/exec"
)

// KernelSandboxArgv wraps a shell invocation for optional OS-level sandboxing (firejail, then bwrap).
// When no helper is installed, ok is false and callers should run argv as-is.
func KernelSandboxArgv(shell string, shellArgs []string) (argv []string, ok bool) {
	if path, err := exec.LookPath("firejail"); err == nil {
		out := []string{path, "--quiet", "--noprofile", "--private-tmp", shell}
		out = append(out, shellArgs...)
		return out, true
	}
	if path, err := exec.LookPath("bwrap"); err == nil {
		// Minimal namespace isolation; full filesystem policy is environment-specific.
		out := []string{
			path, "--unshare-pid", "--die-with-parent",
			"--dev-bind", "/", "/",
			"--proc", "/proc",
			"--tmpfs", "/tmp",
			shell,
		}
		out = append(out, shellArgs...)
		return out, true
	}
	return nil, false
}

// ExecCommandContextWithKernelSandbox returns exec.Cmd like exec.CommandContext(ctx, shell, shellArgs...)
// or wrapped with firejail/bwrap when useSandbox is true and a helper exists.
func ExecCommandContextWithKernelSandbox(ctx context.Context, useSandbox bool, shell string, shellArgs ...string) *exec.Cmd {
	if !useSandbox {
		return exec.CommandContext(ctx, shell, shellArgs...)
	}
	if argv, ok := KernelSandboxArgv(shell, shellArgs); ok {
		return exec.CommandContext(ctx, argv[0], argv[1:]...)
	}
	return exec.CommandContext(ctx, shell, shellArgs...)
}
