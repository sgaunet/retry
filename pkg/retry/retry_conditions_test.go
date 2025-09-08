package retry_test

import (
	"testing"

	"github.com/sgaunet/retry/pkg/retry"
	"github.com/stretchr/testify/assert"
)

func TestRetryOnExitCode(t *testing.T) {
	t.Run("should retry on specified exit codes", func(t *testing.T) {
		condition := retry.NewRetryOnExitCode([]int{1, 2, 124})
		
		// Should retry on code 1
		condition.SetLastExitCode(1)
		assert.False(t, condition.IsLimitReached(), "should retry on exit code 1")
		
		// Should retry on code 2
		condition.SetLastExitCode(2)
		assert.False(t, condition.IsLimitReached(), "should retry on exit code 2")
		
		// Should not retry on code 0
		condition.SetLastExitCode(0)
		assert.True(t, condition.IsLimitReached(), "should not retry on exit code 0")
		
		// Should not retry on code 3
		condition.SetLastExitCode(3)
		assert.True(t, condition.IsLimitReached(), "should not retry on exit code 3")
	})
}

func TestRetryIfContains(t *testing.T) {
	t.Run("should retry when output contains pattern", func(t *testing.T) {
		condition, err := retry.NewRetryIfContains("temporary error")
		assert.NoError(t, err)
		
		// Should retry when pattern found
		condition.SetLastOutput("Connection failed: temporary error", "")
		assert.False(t, condition.IsLimitReached(), "should retry when pattern found")
		
		// Should not retry when pattern not found
		condition.SetLastOutput("Success", "")
		assert.True(t, condition.IsLimitReached(), "should not retry when pattern not found")
		
		// Should check stderr too
		condition.SetLastOutput("", "temporary error occurred")
		assert.False(t, condition.IsLimitReached(), "should retry when pattern found in stderr")
	})
	
	t.Run("should support simple regex patterns", func(t *testing.T) {
		condition, err := retry.NewRetryIfContains("error [0-9]+")
		assert.NoError(t, err)
		
		// Should match regex pattern
		condition.SetLastOutput("Got error 500", "")
		assert.False(t, condition.IsLimitReached(), "should retry on regex match")
		
		// Should not match when pattern doesn't match
		condition.SetLastOutput("Got error ABC", "")
		assert.True(t, condition.IsLimitReached(), "should not retry when regex doesn't match")
	})
}

func TestRetryRegex(t *testing.T) {
	t.Run("should retry on regex match", func(t *testing.T) {
		condition, err := retry.NewRetryRegex("HTTP/[0-9]\\.[0-9] 5[0-9][0-9]")
		assert.NoError(t, err)
		
		// Should retry on 500 errors
		condition.SetLastOutput("HTTP/1.1 500 Internal Server Error", "")
		assert.False(t, condition.IsLimitReached(), "should retry on 500")
		
		condition.SetLastOutput("HTTP/1.1 503 Service Unavailable", "")
		assert.False(t, condition.IsLimitReached(), "should retry on 503")
		
		// Should not retry on success
		condition.SetLastOutput("HTTP/1.1 200 OK", "")
		assert.True(t, condition.IsLimitReached(), "should not retry on 200")
	})
	
	t.Run("should reject invalid regex", func(t *testing.T) {
		_, err := retry.NewRetryRegex("[invalid regex")
		assert.Error(t, err, "should error on invalid regex")
	})
}