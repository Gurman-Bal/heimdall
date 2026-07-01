package truenas

import (
	"bufio"
	"os"
	"sync"

	"heimdall/internal/core"
)

type OffsetStore interface {
	GetOffset(source, path string) (int64, bool, error)
	SetOffset(source, path string, offset int64) error
}

type fileState struct {
	path   string
	offset int64
}

type Source struct {
	mu     sync.Mutex
	states []*fileState
	store  OffsetStore
}

func New(paths []string, store OffsetStore) *Source {
	s := &Source{store: store}
	for _, p := range paths {
		s.states = append(s.states, &fileState{path: p})
	}
	return s
}

func (s *Source) Name() string { return "truenas" }

func (s *Source) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range s.states {
		s.seedOffset(st)
	}
	return nil
}

func (s *Source) seedOffset(st *fileState) {
	if s.store != nil {
		if offset, found, err := s.store.GetOffset(s.Name(), st.path); err == nil && found {
			st.offset = offset
			return
		}
	}
	if fi, err := os.Stat(st.path); err == nil {
		st.offset = fi.Size() // start at EOF, don't replay history
	}
}

// AddPath registers a new file to tail while the process is already running.
func (s *Source) AddPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, st := range s.states {
		if st.path == path {
			return // already tracked
		}
	}

	st := &fileState{path: path}
	s.seedOffset(st)
	s.states = append(s.states, st)
}

// RemovePath stops tailing a file. The saved offset row is left in storage
// in case the same path gets re-added later.
func (s *Source) RemovePath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, st := range s.states {
		if st.path == path {
			s.states = append(s.states[:i], s.states[i+1:]...)
			return
		}
	}
}

func (s *Source) Poll() ([]core.Event, error) {
	s.mu.Lock()
	states := make([]*fileState, len(s.states))
	copy(states, s.states)
	s.mu.Unlock()

	var events []core.Event

	for _, st := range states {
		fi, err := os.Stat(st.path)
		if err != nil {
			continue
		}
		if fi.Size() < st.offset {
			st.offset = 0 // rotated/truncated
		}

		lines, newOffset, err := readNewLines(st.path, st.offset)
		if err != nil {
			continue
		}

		for _, line := range lines {
			events = append(events, parseLine(line))
		}
		st.offset = newOffset

		if s.store != nil {
			_ = s.store.SetOffset(s.Name(), st.path, st.offset)
		}
	}

	return events, nil
}

func readNewLines(path string, offset int64) ([]string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return nil, offset, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	newOffset, err := file.Seek(0, 1) // current position (start + bytes read)
	if err != nil {
		return lines, offset, err
	}

	return lines, newOffset, nil
}
