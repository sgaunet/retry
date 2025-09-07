package retry

import (
	"bytes"
	"io"
	"strings"
)

// PrefixWriter wraps an io.Writer to add prefixes to each line and handle logging.
type PrefixWriter struct {
	logger   *Logger
	isStderr bool
	buffer   bytes.Buffer
}

// NewPrefixWriter creates a new PrefixWriter.
func NewPrefixWriter(logger *Logger, isStderr bool) *PrefixWriter {
	return &PrefixWriter{
		logger:   logger,
		isStderr: isStderr,
	}
}

// Write implements io.Writer, processing lines and passing them to the logger.
func (pw *PrefixWriter) Write(p []byte) (n int, err error) {
	// Add new data to buffer
	pw.buffer.Write(p)
	
	// Process complete lines from the buffer
	for {
		line, err := pw.buffer.ReadString('\n')
		if err == io.EOF {
			// No complete line, put the partial line back and wait for more data
			if line != "" {
				remaining := pw.buffer.Bytes()
				pw.buffer.Reset()
				pw.buffer.WriteString(line)
				pw.buffer.Write(remaining)
			}
			break
		}
		
		// Remove trailing newline and process the line
		line = strings.TrimSuffix(line, "\n")
		if line != "" {
			pw.logger.LogCommandOutput(line, pw.isStderr)
		}
	}
	
	return len(p), nil
}

// Flush processes any remaining data in the buffer.
func (pw *PrefixWriter) Flush() {
	remaining := pw.buffer.String()
	if remaining != "" {
		pw.logger.LogCommandOutput(remaining, pw.isStderr)
		pw.buffer.Reset()
	}
}