package anka

import (
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"gotest.tools/v3/assert"
)

func TestShouldDeleteAnkaTemplateAfterFailedBuild(t *testing.T) {
	t.Run("halted without aborted means cleanup delete", func(t *testing.T) {
		sb := new(multistep.BasicStateBag)
		sb.Put(multistep.StateHalted, true)
		assert.Assert(t, shouldDeleteAnkaTemplateAfterFailedBuild(sb))
	})

	t.Run("cancelled without aborted means cleanup delete", func(t *testing.T) {
		sb := new(multistep.BasicStateBag)
		sb.Put(multistep.StateCancelled, true)
		assert.Assert(t, shouldDeleteAnkaTemplateAfterFailedBuild(sb))
	})

	t.Run("aborted skips delete even when halted (on-error ask, option a)", func(t *testing.T) {
		sb := new(multistep.BasicStateBag)
		sb.Put(multistep.StateHalted, true)
		sb.Put("aborted", true)
		assert.Assert(t, !shouldDeleteAnkaTemplateAfterFailedBuild(sb))
	})

	t.Run("aborted skips delete when cancelled", func(t *testing.T) {
		sb := new(multistep.BasicStateBag)
		sb.Put(multistep.StateCancelled, true)
		sb.Put("aborted", true)
		assert.Assert(t, !shouldDeleteAnkaTemplateAfterFailedBuild(sb))
	})

	t.Run("success path has no halt cancel so no delete from this predicate", func(t *testing.T) {
		sb := new(multistep.BasicStateBag)
		assert.Assert(t, !shouldDeleteAnkaTemplateAfterFailedBuild(sb))
	})
}
