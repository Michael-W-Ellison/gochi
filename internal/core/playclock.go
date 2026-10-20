package core

import "time"

// The play clock times what waits between rewards: Friendship Points for
// each kind of care (CareRewardCooldown), the hide-and-seek bonus, care
// counting less when repeated, and how soon a new day's goals can start.
// It runs only while the game does, measured by the system's monotonic
// clock, so changing the computer's clock can't hurry rewards, and moving
// it back doesn't hold them up. It is kept in the save and resumes from
// there.

// playNowLocked returns the pet's play clock. Caller must hold p.mu.
func (p *DigitalPet) playNowLocked() time.Time {
	t := now()
	switch {
	case p.PlayClock.IsZero():
		p.PlayClock = t
	case !p.playRef.IsZero():
		// Real readings from time.Now carry the monotonic clock, which
		// changing the computer's clock doesn't affect
		if elapsed := t.Sub(p.playRef); elapsed > 0 {
			p.PlayClock = p.PlayClock.Add(elapsed)
		}
	}
	p.playRef = t
	return p.PlayClock
}
