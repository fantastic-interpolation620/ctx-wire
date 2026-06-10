package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"ctx-wire/internal/paths"
)

// The deny loop-breaker. Hook invocations are one-shot processes, so "the
// agent is retrying a request we just denied" can only be detected through
// persisted state. Entries expire after denyTTL, which also rescues the
// Edit-precondition detour: deny -> shell read -> Edit refused -> the re-Read
// arrives within the TTL and is allowed through.
const denyTTL = 60 * time.Second

const denyStateName = "filetool-denies.json"

func denyStatePath() (string, error) {
	base, err := paths.DataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ctx-wire", denyStateName), nil
}

// recordDenyOnce reports whether a deny may be issued for this exact request.
// It returns false when the same request was denied within the TTL (the agent
// is retrying: let it through) or when the deny cannot be recorded (NEVER deny
// without recorded state, or the loop-breaker goes blind). On success the deny
// is recorded before the caller emits it.
func recordDenyOnce(sessionID, tool string, input []byte) bool {
	path, err := denyStatePath()
	if err != nil {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false
	}
	// Claude can issue tool calls in parallel, so two hook processes can race
	// this read/modify/write. An O_EXCL lock serializes them; failing to get
	// the lock fails OPEN (no deny), because an unrecorded deny is the one
	// state the loop-breaker must never produce.
	unlock, ok := lockDenyState(path)
	if !ok {
		return false
	}
	defer unlock()

	sum := sha256.Sum256([]byte(sessionID + "\x00" + tool + "\x00" + string(input)))
	key := hex.EncodeToString(sum[:])
	now := time.Now()

	entries := map[string]int64{}
	if data, rerr := os.ReadFile(path); rerr == nil {
		// Corrupt state starts fresh: worst case one extra redirect attempt.
		_ = json.Unmarshal(data, &entries)
	}
	for k, ts := range entries {
		if now.Sub(time.Unix(ts, 0)) > denyTTL {
			delete(entries, k)
		}
	}
	if ts, ok := entries[key]; ok && now.Sub(time.Unix(ts, 0)) <= denyTTL {
		return false // recently denied: the retry goes through
	}
	entries[key] = now.Unix()

	data, err := json.Marshal(entries)
	if err != nil {
		return false
	}
	// Unique temp file: a fixed name could be clobbered by a process that has
	// not yet observed the lock (or one racing a stale-lock takeover).
	tmp, err := os.CreateTemp(filepath.Dir(path), denyStateName+".*.tmp")
	if err != nil {
		return false
	}
	tmpName := tmp.Name()
	if _, werr := tmp.Write(data); werr != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return false
	}
	if cerr := tmp.Close(); cerr != nil {
		_ = os.Remove(tmpName)
		return false
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return false
	}
	return true
}

// lockDenyState acquires an exclusive advisory lock via O_CREATE|O_EXCL with a
// short bounded wait. A lock left behind by a crashed hook is taken over once
// it is clearly stale (hooks finish in milliseconds; 2s is generous).
func lockDenyState(path string) (func(), bool) {
	lock := path + ".lock"
	for i := 0; i < 20; i++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { _ = os.Remove(lock) }, true
		}
		if fi, serr := os.Stat(lock); serr == nil && time.Since(fi.ModTime()) > 2*time.Second {
			_ = os.Remove(lock) // stale: owner crashed; the next loop retries the create
			continue
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, false
}
