package retry_test

import (
	"testing"

	"github.com/sgaunet/retry/pkg/retry"
	"github.com/stretchr/testify/assert"
)

func TestSuccessOnExitCode(t *testing.T) {
	t.Run("should succeed on specified exit codes", func(t *testing.T) {
		condition := retry.NewSuccessOnExitCode([]int{0, 2})
		
		// Should succeed on code 0
		condition.SetLastExitCode(0)
		assert.True(t, condition.IsLimitReached(), "should succeed on exit code 0")
		
		// Should succeed on code 2
		condition.SetLastExitCode(2)
		assert.True(t, condition.IsLimitReached(), "should succeed on exit code 2")
		
		// Should not succeed on code 1
		condition.SetLastExitCode(1)
		assert.False(t, condition.IsLimitReached(), "should not succeed on exit code 1")
		
		// Should not succeed on code 3
		condition.SetLastExitCode(3)
		assert.False(t, condition.IsLimitReached(), "should not succeed on exit code 3")
	})
}

func TestSuccessContains(t *testing.T) {
	t.Run("should succeed when output contains pattern", func(t *testing.T) {
		condition, err := retry.NewSuccessContains("200 OK")
		assert.NoError(t, err)
		
		// Should succeed when pattern found
		condition.SetLastOutput("HTTP/1.1 200 OK", "")
		assert.True(t, condition.IsLimitReached(), "should succeed when pattern found")
		
		// Should not succeed when pattern not found
		condition.SetLastOutput("HTTP/1.1 404 Not Found", "")
		assert.False(t, condition.IsLimitReached(), "should not succeed when pattern not found")
		
		// Should check stderr too
		condition.SetLastOutput("", "Response: 200 OK")
		assert.True(t, condition.IsLimitReached(), "should succeed when pattern found in stderr")
	})
	
	t.Run("should support simple regex patterns", func(t *testing.T) {
		condition, err := retry.NewSuccessContains("success|complete")
		assert.NoError(t, err)
		
		// Should match either pattern
		condition.SetLastOutput("Operation success", "")
		assert.True(t, condition.IsLimitReached(), "should succeed on 'success'")
		
		condition.SetLastOutput("Task complete", "")
		assert.True(t, condition.IsLimitReached(), "should succeed on 'complete'")
		
		condition.SetLastOutput("Failed", "")
		assert.False(t, condition.IsLimitReached(), "should not succeed on 'Failed'")
	})
}

func TestFailIfContains(t *testing.T) {
	t.Run("should fail immediately when pattern found", func(t *testing.T) {
		condition, err := retry.NewFailIfContains("fatal error")
		assert.NoError(t, err)
		
		// Should fail when pattern found
		condition.SetLastOutput("A fatal error occurred", "")
		assert.True(t, condition.IsLimitReached(), "should fail when pattern found")
		
		// Should not fail when pattern not found
		condition.SetLastOutput("All good", "")
		assert.False(t, condition.IsLimitReached(), "should not fail when pattern not found")
		
		// Should check stderr too
		condition.SetLastOutput("", "fatal error: disk full")
		assert.True(t, condition.IsLimitReached(), "should fail when pattern found in stderr")
	})
}

func TestSuccessRegex(t *testing.T) {
	t.Run("should succeed on regex match", func(t *testing.T) {
		condition, err := retry.NewSuccessRegex("HTTP/[0-9]\\.[0-9] [23][0-9][0-9]")
		assert.NoError(t, err)
		
		// Should succeed on 2xx codes
		condition.SetLastOutput("HTTP/1.1 200 OK", "")
		assert.True(t, condition.IsLimitReached(), "should succeed on 200")
		
		condition.SetLastOutput("HTTP/1.1 201 Created", "")
		assert.True(t, condition.IsLimitReached(), "should succeed on 201")
		
		// Should succeed on 3xx codes
		condition.SetLastOutput("HTTP/1.1 301 Moved Permanently", "")
		assert.True(t, condition.IsLimitReached(), "should succeed on 301")
		
		// Should not succeed on 4xx/5xx codes
		condition.SetLastOutput("HTTP/1.1 404 Not Found", "")
		assert.False(t, condition.IsLimitReached(), "should not succeed on 404")
		
		condition.SetLastOutput("HTTP/1.1 500 Internal Server Error", "")
		assert.False(t, condition.IsLimitReached(), "should not succeed on 500")
	})
	
	t.Run("should reject invalid regex", func(t *testing.T) {
		_, err := retry.NewSuccessRegex("[invalid regex")
		assert.Error(t, err, "should error on invalid regex")
	})
}