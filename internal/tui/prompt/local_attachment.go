package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxAttachments mirrors the chat editor cap.
const MaxAttachments = 5

// LocalAttachment is a validated file attached to the draft.
type LocalAttachment struct {
	Path string
	Name string
	Size int64
}

// AttachmentList is a deduped, capped attachment set.
type AttachmentList struct {
	items []LocalAttachment
}

func (l *AttachmentList) Add(cwd Cwd, mention string) error {
	abs := cwd.Resolve(mention)
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("attachment %q: %w", mention, err)
	}
	if info.IsDir() {
		return fmt.Errorf("attachment %q: is a directory", mention)
	}
	for _, existing := range l.items {
		if existing.Path == abs {
			return nil
		}
	}
	if len(l.items) >= MaxAttachments {
		return fmt.Errorf("cannot attach more than %d files", MaxAttachments)
	}
	l.items = append(l.items, LocalAttachment{Path: abs, Name: filepath.Base(abs), Size: info.Size()})
	return nil
}

func (l *AttachmentList) RemoveAt(i int) {
	if i < 0 || i >= len(l.items) {
		return
	}
	l.items = append(l.items[:i], l.items[i+1:]...)
}

func (l *AttachmentList) Clear() { l.items = nil }

func (l AttachmentList) Items() []LocalAttachment {
	return append([]LocalAttachment(nil), l.items...)
}

func (l AttachmentList) Names() string {
	names := make([]string, 0, len(l.items))
	for _, a := range l.items {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}
