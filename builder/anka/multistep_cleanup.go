package anka

import "github.com/hashicorp/packer-plugin-sdk/multistep"

// shouldDeleteAnkaTemplateAfterFailedBuild reports whether the clone/create steps should run
// `anka delete` from Cleanup. It returns false when Packer's -on-error=ask flow chose
// "abort without cleanup" (state key "aborted" from packer-plugin-sdk commonsteps).
// See https://github.com/veertuinc/packer-plugin-veertu-anka/issues/94
func shouldDeleteAnkaTemplateAfterFailedBuild(state multistep.StateBag) bool {
	if _, aborted := state.GetOk("aborted"); aborted {
		return false
	}
	_, halted := state.GetOk(multistep.StateHalted)
	_, canceled := state.GetOk(multistep.StateCancelled)
	return halted || canceled
}
