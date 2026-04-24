package handler

import "testing"

// Unit tests for containsDateToken.
//
// This helper is the deciding factor for the event_without_date lint: if it
// returns true, the lint assumes the iOS DateCard will be able to extract a
// date from the title/body via NSDataDetector and renders the badge.
//
// False positives here mean a skill's evergreen post gets no warning and
// ships with an empty date badge on older clients. False negatives mean a
// legitimate dated post nags the skill with a spurious warning.
//
// The previous implementation had two classes of bugs the cases below lock
// down:
//
//  1. Short month tokens were matched with a trailing space ("feb "), so
//     "Feb." / "Feb," / "ends Feb" all slipped past.
//  2. "may" was matched unconditionally as a month, so "trail may be wet"
//     and "you may also like" were treated as dated.
func TestContainsDateToken(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		// --- positives: real dates the iOS client would render ---
		{"full month + day", "Join us on April 16 in the park.", true},
		{"short month + period", "Event runs through Feb.", true},
		{"short month + comma", "Coming Feb, 2027.", true},
		{"short month + digit", "Starts Feb 3 at 7pm.", true},
		{"short month + end of string", "Meeting is in Dec", true},
		{"sept variant", "Back in Sept for round two.", true},
		{"weekday", "Saturday block party on Elm.", true},
		{"tonight", "Doors open tonight at 8.", true},
		{"this weekend", "Pop-up this weekend only.", true},
		{"next weekend", "Farmers market returns next weekend.", true},
		{"may + day (real date)", "Recital on May 10 at 2pm.", true},
		{"case-insensitive", "FRIDAY night market", true},

		// --- negatives: evergreen text that must NOT match ---
		{"empty", "", false},
		{"no date at all", "A year-round loop worth doing.", false},
		{"may as modal verb", "The trail may be wet after rain.", false},
		{"may also (common phrase)", "You may also like these hikes.", false},
		{"feb inside facebook", "Share on Facebook for updates.", false},
		{"mar inside smartphones", "Great for smartphones and tablets.", false},
		{"jan inside japan", "Ramen trip through Japan.", false},
		// Bare full month names count as a date mention. NSDataDetector
		// sometimes resolves them (it assumes the 1st of the month), and
		// even when it doesn't, a skill using "event" with only "April" in
		// the body is ambiguous enough that warning them isn't useful.
		// The narrow fix here is just that "may" alone does NOT count.
		{"bare full month name", "Join us in April for the next meetup.", true},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := containsDateToken(c.in)
			if got != c.want {
				t.Errorf("containsDateToken(%q) = %v; want %v", c.in, got, c.want)
			}
		})
	}
}
