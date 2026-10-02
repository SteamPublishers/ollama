package proxy

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
)

// rewriteAccessErrors changes only Ollama access-error messages. Successful
// events and unrelated errors retain their original bytes and status codes.
func (h *CodexDesktop) rewriteAccessErrors(resp *http.Response) error {
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		resp.Body = &codexAccessErrorStream{
			ReadCloser: resp.Body,
			reader:     bufio.NewReader(resp.Body),
			rewrite:    func(body []byte) ([]byte, bool) { return body, false },
			limit:      h.maxBodyBytes,
		}
		resp.ContentLength = -1
		resp.Header.Del("Content-Length")
		return nil
	}
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, h.maxBodyBytes+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > h.maxBodyBytes {
		resp.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(body), resp.Body), resp.Body}
		return nil
	}
	resp.Body.Close()
	// rewritten, changed := h.rewriteAccessErrorJSON(body, resp.StatusCode)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	// if changed {
	// 	resp.ContentLength = int64(len(rewritten))
	// 	resp.Header.Del("Content-Length")
	// }
	return nil
}

// Buffer one SSE frame, not the whole response. Oversized frames fall back to
// passthrough so cosmetic error changes cannot interrupt a valid model stream.
type codexAccessErrorStream struct {
	io.ReadCloser
	reader      *bufio.Reader
	rewrite     func([]byte) ([]byte, bool)
	limit       int64
	pending     []byte
	err         error
	passthrough bool
}

func (s *codexAccessErrorStream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(s.pending) == 0 {
		if s.err != nil {
			return 0, s.err
		}
		if s.passthrough {
			return s.reader.Read(p)
		}
	}
	if len(s.pending) == 0 && s.err == nil {
		var frame []byte
		partialLine := false
		for {
			line, err := s.reader.ReadSlice('\n')
			frame = append(frame, line...)
			if err != nil && err != bufio.ErrBufferFull {
				s.err = err
			}
			if int64(len(frame)) > s.limit {
				s.passthrough = true
				break
			}
			if s.err != nil {
				break
			}
			if !partialLine && (bytes.Equal(line, []byte("\n")) || bytes.Equal(line, []byte("\r\n"))) {
				break
			}
			partialLine = err == bufio.ErrBufferFull
		}
		s.pending = frame
		if !s.passthrough {
			s.pending = rewriteCodexErrorFrame(frame, s.rewrite)
		}
	}
	if len(s.pending) > 0 {
		n := copy(p, s.pending)
		s.pending = s.pending[n:]
		return n, nil
	}
	return 0, s.err
}

func rewriteCodexErrorFrame(frame []byte, rewrite func([]byte) ([]byte, bool)) []byte {
	lines := bytes.SplitAfter(frame, []byte("\n"))
	var data [][]byte
	for _, line := range lines {
		if value, ok := bytes.CutPrefix(line, []byte("data:")); ok {
			data = append(data, bytes.TrimSpace(value))
		}
	}
	rewritten, changed := rewrite(bytes.Join(data, []byte("\n")))
	if !changed {
		return frame
	}
	var result bytes.Buffer
	written := false
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("data:")) {
			if !written {
				result.WriteString("data: ")
				result.Write(rewritten)
				result.WriteByte('\n')
				written = true
			}
		} else {
			result.Write(line)
		}
	}
	return result.Bytes()
}
